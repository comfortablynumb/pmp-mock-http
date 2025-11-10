package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/loader"
	"github.com/comfortablynumb/pmp-mock-http/internal/plugins"
	"github.com/comfortablynumb/pmp-mock-http/internal/proxy"
	"github.com/comfortablynumb/pmp-mock-http/internal/server"
	"github.com/comfortablynumb/pmp-mock-http/internal/tracker"
	"github.com/comfortablynumb/pmp-mock-http/internal/ui"
	"github.com/comfortablynumb/pmp-mock-http/internal/validator"
	"github.com/comfortablynumb/pmp-mock-http/internal/watcher"
)

// getEnvInt gets an integer value from environment variable, or returns the default
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

// getEnvString gets a string value from environment variable, or returns the default
func getEnvString(key string, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvBool gets a boolean value from environment variable, or returns the default
func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

var (
	port                = flag.Int("port", getEnvInt("PORT", 8083), "HTTP server port")
	uiPort              = flag.Int("ui-port", getEnvInt("UI_PORT", 8081), "UI dashboard port")
	mocksDir            = flag.String("mocks-dir", getEnvString("MOCKS_DIR", "mocks"), "Directory containing mock YAML files")
	pluginsDir          = flag.String("plugins-dir", getEnvString("PLUGINS_DIR", "plugins"), "Directory to store plugin repositories")
	pluginList          = flag.String("plugins", getEnvString("PLUGINS", ""), "Comma-separated list of git repository URLs to clone as plugins")
	pluginIncludeOnly   = flag.String("plugin-include-only", getEnvString("PLUGIN_INCLUDE_ONLY", ""), "Space-separated list of subdirectories from pmp-mock-http to include (e.g., 'openai stripe')")
	proxyTarget         = flag.String("proxy-target", getEnvString("PROXY_TARGET", ""), "Target URL for proxy passthrough (e.g., 'http://api.example.com')")
	proxyPreserveHost   = flag.Bool("proxy-preserve-host", getEnvBool("PROXY_PRESERVE_HOST", false), "Preserve the original Host header when proxying")
	proxyTimeout        = flag.Int("proxy-timeout", getEnvInt("PROXY_TIMEOUT", 30), "Proxy request timeout in seconds")
	tlsEnabled          = flag.Bool("tls", getEnvBool("TLS_ENABLED", false), "Enable TLS/HTTPS with HTTP/2")
	tlsCertFile         = flag.String("tls-cert", getEnvString("TLS_CERT_FILE", ""), "Path to TLS certificate file")
	tlsKeyFile          = flag.String("tls-key", getEnvString("TLS_KEY_FILE", ""), "Path to TLS private key file")
	http3Enabled        = flag.Bool("http3", getEnvBool("HTTP3_ENABLED", false), "Enable HTTP/3 with QUIC (requires TLS)")
	dualStack           = flag.Bool("dual-stack", getEnvBool("DUAL_STACK", false), "Enable both HTTP/2 and HTTP/3 (requires TLS)")
	enableCORS          = flag.Bool("enable-cors", getEnvBool("ENABLE_CORS", false), "Enable CORS support")
	corsOrigins         = flag.String("cors-origins", getEnvString("CORS_ORIGINS", "*"), "CORS allowed origins")
	corsMethods         = flag.String("cors-methods", getEnvString("CORS_METHODS", "GET,POST,PUT,DELETE,PATCH,OPTIONS"), "CORS allowed methods")
	corsHeaders         = flag.String("cors-headers", getEnvString("CORS_HEADERS", "Content-Type,Authorization"), "CORS allowed headers")
	validateMocks       = flag.Bool("validate-mocks", getEnvBool("VALIDATE_MOCKS", true), "Validate mock configurations on startup")
)

func main() {
	flag.Parse()

	log.Printf("Starting PMP Mock HTTP Server...\n")
	log.Printf("Mock server port: %d\n", *port)
	log.Printf("UI dashboard port: %d\n", *uiPort)
	log.Printf("Mocks directory: %s\n", *mocksDir)
	log.Printf("TLS enabled: %v\n", *tlsEnabled)
	if *proxyTarget != "" {
		log.Printf("Proxy target: %s\n", *proxyTarget)
		log.Printf("Proxy preserve host: %v\n", *proxyPreserveHost)
		log.Printf("Proxy timeout: %ds\n", *proxyTimeout)
	}

	// Parse plugin repositories
	var pluginRepos []string
	if *pluginList != "" {
		pluginRepos = strings.Split(*pluginList, ",")
		for i := range pluginRepos {
			pluginRepos[i] = strings.TrimSpace(pluginRepos[i])
		}
		log.Printf("Plugins: %d repositories configured\n", len(pluginRepos))
	}

	// Parse plugin include filter
	var pluginIncludeFilter []string
	if *pluginIncludeOnly != "" {
		pluginIncludeFilter = strings.Fields(*pluginIncludeOnly)
		log.Printf("Plugin include filter: %v\n", pluginIncludeFilter)
	}

	// Set up plugins (clone/update repositories)
	var pluginDirs []string
	if len(pluginRepos) > 0 {
		var pluginManager *plugins.Manager
		if len(pluginIncludeFilter) > 0 {
			pluginManager = plugins.NewManagerWithIncludeFilter(*pluginsDir, pluginRepos, pluginIncludeFilter)
		} else {
			pluginManager = plugins.NewManager(*pluginsDir, pluginRepos)
		}
		var err error
		pluginDirs, err = pluginManager.SetupPlugins()
		if err != nil {
			log.Printf("Warning: failed to setup plugins: %v\n", err)
		}
		log.Printf("Loaded %d plugin directory(ies)\n", len(pluginDirs))
	}

	// Create directories to load (mocks dir + plugin dirs)
	loadDirs := append([]string{*mocksDir}, pluginDirs...)

	// Create the loader with all directories
	mockLoader := loader.NewLoader(loadDirs...)

	// Load initial mocks
	if err := mockLoader.LoadAll(); err != nil {
		log.Printf("Warning: failed to load mocks: %v\n", err)
	}

	// Validate mocks if enabled
	if *validateMocks {
		mockValidator := validator.NewValidator()
		validationResult := mockValidator.ValidateMocks(mockLoader.GetMocks())
		mockValidator.PrintValidationResult(validationResult)

		// Exit if validation failed
		if !validationResult.Valid {
			log.Fatal("Mock validation failed. Fix errors and try again, or disable validation with --validate-mocks=false")
		}
	}

	// Create request tracker for UI dashboard
	requestTracker := tracker.NewTracker(1000) // Keep last 1000 requests

	// Create proxy configuration if proxy target is specified
	var proxyConfig *proxy.Config
	if *proxyTarget != "" {
		proxyConfig = &proxy.Config{
			Target:       *proxyTarget,
			PreserveHost: *proxyPreserveHost,
			Timeout:      time.Duration(*proxyTimeout) * time.Second,
		}
	}

	// Create CORS configuration if enabled
	var corsConfig *server.CORSConfig
	if *enableCORS {
		corsConfig = &server.CORSConfig{
			Enabled: true,
			Origins: *corsOrigins,
			Methods: *corsMethods,
			Headers: *corsHeaders,
		}
		log.Printf("CORS enabled: Origins=%s, Methods=%s, Headers=%s\n", *corsOrigins, *corsMethods, *corsHeaders)
	}

	// Create the mock server with tracker, proxy config, and CORS config
	srv := server.NewServerWithTracker(*port, mockLoader.GetMocks(), requestTracker, proxyConfig, corsConfig)

	// Create and start the UI server
	uiServer := ui.NewServer(*uiPort, requestTracker)
	go func() {
		if err := uiServer.Start(); err != nil {
			log.Fatalf("UI server error: %v\n", err)
		}
	}()

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
		var err error

		// Validate TLS configuration for HTTP/3 and dual-stack
		if (*http3Enabled || *dualStack) && !*tlsEnabled {
			log.Fatalf("HTTP/3 and dual-stack mode require TLS to be enabled (--tls)\n")
		}

		if *tlsEnabled || *http3Enabled || *dualStack {
			if *tlsCertFile == "" || *tlsKeyFile == "" {
				log.Fatalf("TLS/HTTP3 enabled but certificate or key file not specified\n")
			}

			// Choose server mode
			if *dualStack {
				log.Println("Starting server in dual-stack mode (HTTP/1.1, HTTP/2, HTTP/3)")
				err = srv.StartDualStack(*tlsCertFile, *tlsKeyFile)
			} else if *http3Enabled {
				log.Println("Starting server in HTTP/3 mode")
				err = srv.StartHTTP3(*tlsCertFile, *tlsKeyFile)
			} else {
				log.Println("Starting server in TLS mode (HTTP/1.1, HTTP/2)")
				err = srv.StartTLS(*tlsCertFile, *tlsKeyFile)
			}
		} else {
			log.Println("Starting server in HTTP/1.1 mode")
			err = srv.Start()
		}

		if err != nil {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutting down gracefully...")
}
