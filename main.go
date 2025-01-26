package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ksauraj/ksau-oned-api/api"
)

const (
	defaultReadTimeout  = 10 * time.Minute
	defaultWriteTimeout = 10 * time.Minute
	defaultIdleTimeout  = 120 * time.Second
)

func getEnvWithDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	strValue := getEnvWithDefault(key, "")
	if strValue == "" {
		return defaultValue
	}

	// Try to parse as seconds
	if seconds, err := strconv.Atoi(strValue); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try to parse as duration string
	if duration, err := time.ParseDuration(strValue); err == nil {
		return duration
	}

	return defaultValue
}

type maxBytesHandler struct {
	h http.Handler
	n int64
}

func (h *maxBytesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.ContentLength > h.n {
		http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.n)
	h.h.ServeHTTP(w, r)
}

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting server initialization...")

	// Create a new serve mux
	mux := http.NewServeMux()

	// Set up routes
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		api.Handler(w, r)
	})

	// Add system info endpoints
	mux.HandleFunc("/system", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received system info request: %s %s", r.Method, r.URL.Path)
		api.SystemHandler(w, r)
	})

	mux.HandleFunc("/neofetch", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received neofetch request: %s %s", r.Method, r.URL.Path)
		api.NeofetchHandler(w, r)
	})

	mux.HandleFunc("/quota", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received quota info request: %s %s", r.Method, r.URL.Path)
		api.QuotaHandler(w, r)
	})

	// Get server timeouts from environment variables
	readTimeout := getEnvDurationWithDefault("SERVER_READ_TIMEOUT", defaultReadTimeout)
	writeTimeout := getEnvDurationWithDefault("SERVER_WRITE_TIMEOUT", defaultWriteTimeout)
	idleTimeout := getEnvDurationWithDefault("SERVER_IDLE_TIMEOUT", defaultIdleTimeout)

	// Create server with timeouts
	addr := getEnvWithDefault("SERVER_ADDR", "0.0.0.0:8080")

	// Set maximum request size to 5GB
	maxRequestSize := int64(5 * 1024 * 1024 * 1024)

	handler := &maxBytesHandler{
		h: mux,
		n: maxRequestSize,
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		// Increase maximum header size to 10MB
		MaxHeaderBytes: 10 << 20,
	}

	log.Printf("Server configuration:")
	log.Printf("- Address: %s", addr)
	log.Printf("- Read Timeout: %v", readTimeout)
	log.Printf("- Write Timeout: %v", writeTimeout)
	log.Printf("- Idle Timeout: %v", idleTimeout)
	log.Printf("- Max Request Size: %d bytes", maxRequestSize)

	// Channel to receive errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on %s...\n", addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Channel to handle shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting server: %v", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		log.Printf("Start shutdown... Signal: %v", sig)

		// Create context with timeout for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
			// Force shutdown
			if err := server.Close(); err != nil {
				log.Printf("Error during forced server close: %v", err)
			}
			os.Exit(1)
		}
	}

	log.Printf("Server shutdown complete")
}
