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

	// Token generation endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received token request: %s %s", r.Method, r.URL.Path)
		api.TokenHandler(w, r)
	})

	// Get server timeouts from environment variables
	readTimeout := getEnvDurationWithDefault("SERVER_READ_TIMEOUT", defaultReadTimeout)
	writeTimeout := getEnvDurationWithDefault("SERVER_WRITE_TIMEOUT", defaultWriteTimeout)
	idleTimeout := getEnvDurationWithDefault("SERVER_IDLE_TIMEOUT", defaultIdleTimeout)

	// Create server with timeouts
	addr := getEnvWithDefault("SERVER_ADDR", "0.0.0.0:8080")
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
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
