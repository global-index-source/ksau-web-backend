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

const (
	// 5GB max file size
	MaxFileSize = 5 * 1024 * 1024 * 1024
	// Maximum parallel chunks for upload
	MaxParallelChunks = 4
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

	var (
		remote        string
		remoteFolder  string
		filename      string
		chunkSizeStr  string
		file          io.ReadCloser
		contentLength int64
	)

	contentType := r.Header.Get("Content-Type")
	if contentType == "application/octet-stream" {
		// Binary upload mode
		remote = r.Header.Get("X-Remote")
		remoteFolder = r.Header.Get("X-Remote-Folder")
		filename = r.Header.Get("X-Filename")
		chunkSizeStr = r.Header.Get("X-Chunk-Size")
		file = r.Body
		contentLength = r.ContentLength
	} else {
		// Traditional form upload mode
		err = r.ParseMultipartForm(10 << 20) // 10MB for form data
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, err, "Unable to parse form data")
			return
		}
		defer r.MultipartForm.RemoveAll()

		remote = r.FormValue("remote")
		remoteFolder = r.FormValue("remoteFolder")
		chunkSizeStr = r.FormValue("chunkSize")

		uploadedFile, header, err := r.FormFile("file")
		if err != nil {
			sendErrorResponse(w, http.StatusBadRequest, err, "Unable to read file")
			return
		}
		defer uploadedFile.Close()

		file = uploadedFile
		filename = header.Filename
		contentLength = header.Size
	}

	// Validate parameters
	if remote == "" {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("remote is required"), "Invalid request")
		return
	}

	if _, ok := rootFolders[remote]; !ok {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid remote: %s", remote), "Invalid request")
		return
	}

	if filename == "" {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("filename is required"), "Invalid request")
		return
	}

	if chunkSizeStr == "" {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("chunk size is required"), "Invalid request")
		return
	}

	chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
	if err != nil || chunkSize < 2 || chunkSize > 32 {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid chunk size: must be between 2 and 32"), "Invalid request")
		return
	}
	chunkSize *= 1024 * 1024 // Convert MB to bytes

	log.Printf("Initializing Azure client...")
	// Initialize AzureClient for the remote configuration
	client, err := azure.NewAzureClientFromRcloneConfigData(configData, remote)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Failed to initialize Azure client")
		return
	}

	log.Printf("Processing upload for remote: %s, folder: %s, file: %s", remote, remoteFolder, filename)

	// Create a temporary file with a meaningful prefix
	tempFile, err := os.CreateTemp("", fmt.Sprintf("upload-%s-*.tmp", filepath.Base(filename)))
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Unable to create temp file")
		return
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name())
		log.Printf("Cleaned up temporary file: %s", tempFile.Name())
	}()

	// Copy the file content with progress tracking
	log.Printf("Copying file content...")
	written, err := io.Copy(tempFile, io.TeeReader(file, &progressWriter{
		total:     contentLength,
		processed: 0,
	}))
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Unable to save file")
		return
	}
	log.Printf("Copied %d bytes to temporary file", written)

	// Construct the remote file path
	remoteFilePath := filepath.Join(rootFolders[remote], remoteFolder, filename)
	log.Printf("Remote file path: %s", remoteFilePath)

	// Upload parameters with parallel chunk support
	params := azure.UploadParams{
		FilePath:       tempFile.Name(),
		RemoteFilePath: remoteFilePath,
		ChunkSize:      chunkSize,
		ParallelChunks: MaxParallelChunks,
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
	downloadURL := fmt.Sprintf("%s/%s/%s", baseURL, remoteFolder, filename)

	// Return success response
	response := map[string]interface{}{
		"status":      "success",
		"message":     "File uploaded successfully",
		"downloadURL": downloadURL,
		"fileSize":    written,
		"fileName":    filename,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("Request completed successfully")
}

// progressWriter tracks upload progress
type progressWriter struct {
	total     int64
	processed int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.processed += int64(n)
	progress := float64(pw.processed) / float64(pw.total) * 100
	log.Printf("Upload progress: %.2f%% (%d/%d bytes)", progress, pw.processed, pw.total)
	return n, nil
}
