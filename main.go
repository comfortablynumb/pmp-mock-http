package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/comfortablynumb/pmp-mock-http/internal/loader"
	"github.com/comfortablynumb/pmp-mock-http/internal/plugins"
	"github.com/comfortablynumb/pmp-mock-http/internal/server"
	"github.com/comfortablynumb/pmp-mock-http/internal/watcher"
)

var (
	port       = flag.Int("port", 8083, "HTTP server port")
	mocksDir   = flag.String("mocks-dir", "mocks", "Directory containing mock YAML files")
	pluginsDir = flag.String("plugins-dir", "plugins", "Directory to store plugin repositories")
	pluginList = flag.String("plugins", "", "Comma-separated list of git repository URLs to clone as plugins")
)

func main() {
	flag.Parse()

	log.Printf("Starting PMP Mock HTTP Server...\n")
	log.Printf("Port: %d\n", *port)
	log.Printf("Mocks directory: %s\n", *mocksDir)

	// Parse plugin repositories
	var pluginRepos []string
	if *pluginList != "" {
		pluginRepos = strings.Split(*pluginList, ",")
		for i := range pluginRepos {
			pluginRepos[i] = strings.TrimSpace(pluginRepos[i])
		}
		log.Printf("Plugins: %d repositories configured\n", len(pluginRepos))
	}

	// Set up plugins (clone/update repositories)
	var pluginDirs []string
	if len(pluginRepos) > 0 {
		pluginManager := plugins.NewManager(*pluginsDir, pluginRepos)
		var err error
		pluginDirs, err = pluginManager.SetupPlugins()
		if err != nil {
			log.Printf("Warning: failed to setup plugins: %v\n", err)
		}
		log.Printf("Loaded %d plugin(s)\n", len(pluginDirs))
	}

	// Create directories to load (mocks dir + plugin dirs)
	loadDirs := append([]string{*mocksDir}, pluginDirs...)

	// Create the loader with all directories
	mockLoader := loader.NewLoader(loadDirs...)

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

	// Create and start file watchers for all directories
	var watchers []*watcher.Watcher
	for _, dir := range loadDirs {
		w, err := watcher.NewWatcher(dir, reloadFn)
		if err != nil {
			log.Printf("Warning: failed to create watcher for %s: %v\n", dir, err)
			continue
		}
		defer w.Close() //nolint:errcheck // cleanup operation

		if err := w.Start(); err != nil {
			log.Printf("Warning: failed to start watcher for %s: %v\n", dir, err)
			continue
		}
		watchers = append(watchers, w)
	}

	log.Printf("Watching %d directory(ies) for changes\n", len(watchers))

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
