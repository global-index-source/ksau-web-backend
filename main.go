package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ksauraj/ksau-oned-api/api"
)

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

	// Create server with timeouts
	port := "8080"
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Channel to receive errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("Local development server starting on port %s...\n", port)
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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
