package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/comfortablynumb/pmp-mock-http/internal/loader"
	"github.com/comfortablynumb/pmp-mock-http/internal/server"
	"github.com/comfortablynumb/pmp-mock-http/internal/watcher"
)

var (
	port      = flag.Int("port", 8083, "HTTP server port")
	mocksDir  = flag.String("mocks-dir", "mocks", "Directory containing mock YAML files")
)

func main() {
	flag.Parse()

	log.Printf("Starting PMP Mock HTTP Server...\n")
	log.Printf("Port: %d\n", *port)
	log.Printf("Mocks directory: %s\n", *mocksDir)

	// Create the loader
	mockLoader := loader.NewLoader(*mocksDir)

	// Load initial mocks
	if err := mockLoader.LoadAll(); err != nil {
		log.Printf("Warning: failed to load mocks: %v\n", err)
	}

	// Create the server
	srv := server.NewServer(*port, mockLoader.GetMocks())

	// Create reload function for the watcher
	reloadFn := func() error {
		if err := mockLoader.LoadAll(); err != nil {
			return err
		}
		srv.UpdateMocks(mockLoader.GetMocks())
		return nil
	}

	// Create and start the file watcher
	w, err := watcher.NewWatcher(*mocksDir, reloadFn)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v\n", err)
	}
	defer w.Close()

	if err := w.Start(); err != nil {
		log.Fatalf("Failed to start watcher: %v\n", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutting down gracefully...")
}
