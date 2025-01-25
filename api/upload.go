package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ksauraj/ksau-oned-api/azure"
)

// Root folders for each remote configuration
var rootFolders = map[string]string{
	"hakimionedrive": "Public",
	"oned":           "",
	"saurajcf":       "MY_BOMT_STUFFS",
}

// Base URLs for each remote configuration
var baseURLs = map[string]string{
	"hakimionedrive": "https://onedrive-vercel-index-kohl-eight-30.vercel.app",
	"oned":           "https://index.sauraj.eu.org",
	"saurajcf":       "https://my-index-azure.vercel.app",
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// sendErrorResponse sends a JSON error response
func sendErrorResponse(w http.ResponseWriter, statusCode int, err error, message string) {
	log.Printf("Error: %v - %s", err, message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Details: err.Error(),
	})
}

// validateRequest validates the upload request
func validateRequest(r *http.Request) error {
	if r.ContentLength > 100<<20 { // 100MB limit
		return fmt.Errorf("file too large: %d bytes", r.ContentLength)
	}

	remote := r.FormValue("remote")
	if remote == "" {
		return fmt.Errorf("remote is required")
	}

	if _, ok := rootFolders[remote]; !ok {
		return fmt.Errorf("invalid remote: %s", remote)
	}

	chunkSizeStr := r.FormValue("chunkSize")
	chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
	if err != nil || chunkSize < 2 || chunkSize > 16 {
		return fmt.Errorf("invalid chunk size: must be between 2 and 16")
	}

	return nil
}

// Handler is the main API handler for file uploads
func Handler(w http.ResponseWriter, r *http.Request) {
	// Enable detailed logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	start := time.Now()
	log.Printf("Starting new request: %s %s", r.Method, r.URL.Path)
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic recovered: %v", err)
			sendErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("%v", err), "Internal server error")
		}
		log.Printf("Request completed in %v", time.Since(start))
	}()

	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow POST requests
	if r.Method != "POST" {
		sendErrorResponse(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s not allowed", r.Method), "Method not allowed")
		return
	}

	log.Printf("Reading config file...")
	// Read the config file
	configData, err := os.ReadFile("rclone.conf")
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Failed to read config file")
		return
	}

	log.Printf("Initializing Azure client...")
	// Initialize AzureClient for the default remote configuration
	remoteConfig := "oned" // Default remote configuration
	client, err := azure.NewAzureClientFromRcloneConfigData(configData, remoteConfig)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Failed to initialize Azure client")
		return
	}

	log.Printf("Parsing multipart form...")
	// Parse form data with a reasonable memory limit
	err = r.ParseMultipartForm(10 << 20) // 10 MB max memory
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err, "Unable to parse form data")
		return
	}
	defer r.MultipartForm.RemoveAll() // Clean up parsed files

	// Validate request
	if err := validateRequest(r); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err, "Invalid request")
		return
	}

	// Get form values
	remote := r.FormValue("remote")
	remoteFolder := r.FormValue("remoteFolder")
	remoteFileName := r.FormValue("remoteFileName")
	chunkSizeStr := r.FormValue("chunkSize")

	log.Printf("Processing upload for remote: %s, folder: %s", remote, remoteFolder)

	// Parse chunk size
	chunkSize, _ := strconv.ParseInt(chunkSizeStr, 10, 64)
	chunkSize *= 1024 * 1024 // Convert MB to bytes

	// Get the uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err, "Unable to read file")
		return
	}
	defer file.Close()

	log.Printf("Creating temporary file for: %s (%d bytes)", header.Filename, header.Size)
	// Create a temporary file with a meaningful prefix
	tempFile, err := os.CreateTemp("", fmt.Sprintf("upload-%s-*.tmp", filepath.Base(header.Filename)))
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Unable to create temp file")
		return
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
		log.Printf("Cleaned up temporary file: %s", tempFile.Name())
	}()

	// Copy the uploaded file content
	log.Printf("Copying file content...")
	written, err := io.Copy(tempFile, file)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Unable to save file")
		return
	}
	log.Printf("Copied %d bytes to temporary file", written)

	// Determine the remote file name
	finalRemoteFileName := header.Filename
	if remoteFileName != "" {
		finalRemoteFileName = remoteFileName
	}

	// Construct the remote file path
	remoteFilePath := filepath.Join(rootFolders[remote], remoteFolder, finalRemoteFileName)
	log.Printf("Remote file path: %s", remoteFilePath)

	// Upload parameters
	params := azure.UploadParams{
		FilePath:       tempFile.Name(),
		RemoteFilePath: remoteFilePath,
		ChunkSize:      chunkSize,
		ParallelChunks: 1, // Disable parallel uploads
		MaxRetries:     3,
		RetryDelay:     5 * time.Second,
		AccessToken:    client.AccessToken,
	}

	// Upload the file to OneDrive
	log.Printf("Starting OneDrive upload...")
	_, err = client.Upload(http.DefaultClient, params)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Failed to upload file")
		return
	}
	log.Printf("File uploaded successfully")

	// Generate the download URL
	baseURL := baseURLs[remote]
	downloadURL := fmt.Sprintf("%s/%s/%s", baseURL, remoteFolder, finalRemoteFileName)

	// Return success response
	response := map[string]interface{}{
		"status":      "success",
		"message":     "File uploaded successfully",
		"downloadURL": downloadURL,
		"fileSize":    written,
		"fileName":    finalRemoteFileName,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("Request completed successfully")
}
