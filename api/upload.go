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

	"github.com/golang-jwt/jwt/v5"
	"github.com/ksauraj/ksau-oned-api/azure"
	"github.com/ksauraj/ksau-oned-api/config"
)

// JWT related constants
const (
	AccessTokenDuration  = 1 * time.Hour
	RefreshTokenDuration = 24 * time.Hour
	JWTSecretKey         = "your-secret-key-change-this-in-production" // Change this in production
)

// TokenResponse represents the response for token generation
type TokenResponse struct {
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	ExpiresIn      int64  `json:"expires_in"` // in seconds
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	DriveID        string `json:"drive_id"`
	DriveType      string `json:"drive_type"`
	BaseURL        string `json:"base_url"`
	UploadRootPath string `json:"upload_root_path"`
}

// CustomClaims represents the claims in the JWT token
type CustomClaims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// generateToken creates a new JWT token
func generateToken(tokenType string, duration time.Duration) (string, error) {
	claims := CustomClaims{
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecretKey))
}

// TokenHandler handles token generation requests
func TokenHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only allow GET requests
	if r.Method != "GET" {
		sendErrorResponse(w, http.StatusMethodNotAllowed, fmt.Errorf("method %s not allowed", r.Method), "Method not allowed")
		return
	}

	// Get remote from query parameter
	remote := r.URL.Query().Get("remote")
	if remote == "" {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("remote parameter is required"), "Missing remote parameter")
		return
	}

	// Validate remote
	if _, ok := rootFolders[remote]; !ok {
		sendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid remote: %s", remote), "Invalid remote")
		return
	}

	// Get embedded config data
	configData := config.GetRcloneConfig()

	// Get Azure client for the remote
	client, err := azure.NewAzureClientFromRcloneConfigData(configData, remote)
	if err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Failed to initialize Azure client")
		return
	}

	// Ensure token is refreshed if needed
	if err := client.EnsureTokenValid(http.DefaultClient); err != nil {
		sendErrorResponse(w, http.StatusInternalServerError, err, "Failed to refresh token")
		return
	}

	response := TokenResponse{
		AccessToken:    client.AccessToken,
		RefreshToken:   client.RefreshToken,
		ExpiresIn:      int64(time.Until(client.Expiration).Seconds()),
		ClientID:       client.ClientID,
		ClientSecret:   client.ClientSecret,
		DriveID:        client.DriveID,
		DriveType:      client.DriveType,
		BaseURL:        baseURLs[remote],
		UploadRootPath: rootFolders[remote],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

const (
	// 5GB max file size
	MaxFileSize = 5 * 1024 * 1024 * 1024
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

	log.Printf("Getting embedded config data...")
	// Get embedded config data
	configData := config.GetRcloneConfig()

	var (
		err           error
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

	// Upload parameters with sequential chunk upload
	params := azure.UploadParams{
		FilePath:       tempFile.Name(),
		RemoteFilePath: remoteFilePath,
		ChunkSize:      chunkSize,
		ParallelChunks: 1,                // Disable parallel uploads to avoid eTag conflicts
		MaxRetries:     5,                // Increase retries
		RetryDelay:     10 * time.Second, // Increase delay between retries
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
