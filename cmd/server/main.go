package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/graphql"
	"github.com/comfortablynumb/pmp-mock-http/internal/grpc"
	"github.com/comfortablynumb/pmp-mock-http/internal/loader"
	"github.com/comfortablynumb/pmp-mock-http/internal/management"
	"github.com/comfortablynumb/pmp-mock-http/internal/observability"
	"github.com/comfortablynumb/pmp-mock-http/internal/plugins"
	"github.com/comfortablynumb/pmp-mock-http/internal/proxy"
	"github.com/comfortablynumb/pmp-mock-http/internal/server"
	"github.com/comfortablynumb/pmp-mock-http/internal/tracker"
	"github.com/comfortablynumb/pmp-mock-http/internal/ui"
	"github.com/comfortablynumb/pmp-mock-http/internal/validator"
	"github.com/comfortablynumb/pmp-mock-http/internal/watcher"
	"go.uber.org/zap"
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

	// Observability flags
	logLevel            = flag.String("log-level", getEnvString("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	enableMetrics       = flag.Bool("enable-metrics", getEnvBool("ENABLE_METRICS", true), "Enable Prometheus metrics")
	enableTracing       = flag.Bool("enable-tracing", getEnvBool("ENABLE_TRACING", false), "Enable OpenTelemetry tracing")
	otlpEndpoint        = flag.String("otlp-endpoint", getEnvString("OTLP_ENDPOINT", "localhost:4317"), "OTLP collector endpoint")
	enableHealthCheck   = flag.Bool("enable-health", getEnvBool("ENABLE_HEALTH", true), "Enable health check endpoints")
	healthPort          = flag.Int("health-port", getEnvInt("HEALTH_PORT", 8080), "Health check and metrics endpoints port")

	// Management API flags
	enableManagementAPI = flag.Bool("enable-management", getEnvBool("ENABLE_MANAGEMENT", true), "Enable management API")
	managementPort      = flag.Int("management-port", getEnvInt("MANAGEMENT_PORT", 8082), "Management API port")
	loadTemplates       = flag.Bool("load-templates", getEnvBool("LOAD_TEMPLATES", true), "Load default mock templates")

	// GraphQL flags
	enableGraphQL       = flag.Bool("enable-graphql", getEnvBool("ENABLE_GRAPHQL", false), "Enable GraphQL support")
	graphqlPort         = flag.Int("graphql-port", getEnvInt("GRAPHQL_PORT", 8084), "GraphQL server port")

	// gRPC flags
	enableGRPC          = flag.Bool("enable-grpc", getEnvBool("ENABLE_GRPC", false), "Enable gRPC support")
	grpcPort            = flag.Int("grpc-port", getEnvInt("GRPC_PORT", 9000), "gRPC server port")
)

func main() {
	flag.Parse()

	// Initialize observability (structured logging)
	isDevelopment := *logLevel == "debug"
	if err := observability.InitLogger(*logLevel, isDevelopment); err != nil {
		log.Fatalf("Failed to initialize logger: %v\n", err)
	}
	defer observability.Sync()

	observability.Info("Starting PMP Mock HTTP Server",
		zap.Int("port", *port),
		zap.Int("ui_port", *uiPort),
		zap.String("mocks_dir", *mocksDir),
		zap.Bool("tls_enabled", *tlsEnabled),
	)

	// Initialize tracing if enabled
	var tracingShutdown func(context.Context) error
	if *enableTracing {
		var err error
		tracingShutdown, err = observability.InitTracing("pmp-mock-http", *otlpEndpoint)
		if err != nil {
			observability.Warn("Failed to initialize tracing", zap.Error(err))
		} else {
			observability.Info("Tracing enabled", zap.String("otlp_endpoint", *otlpEndpoint))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := tracingShutdown(ctx); err != nil {
					observability.Error("Failed to shutdown tracing", zap.Error(err))
				}
			}()
		}
	}

	// Register default health checks
	if *enableHealthCheck {
		observability.RegisterDefaultHealthChecks()
		observability.Info("Health checks enabled", zap.Int("health_port", *healthPort))
	}

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

	// Initialize and start Management API
	var mockManager *management.Manager
	if *enableManagementAPI {
		mockManager = management.NewManager()

		// Load default templates if enabled
		if *loadTemplates {
			if err := management.LoadDefaultTemplates(mockManager); err != nil {
				observability.Warn("Failed to load default templates", zap.Error(err))
			} else {
				observability.Info("Loaded default mock templates")
			}
		}

		// Create management API handler
		managementHandler := management.NewAPIHandler(mockManager)
		managementMux := http.NewServeMux()
		managementHandler.RegisterRoutes(managementMux)

		// Start management API server
		managementServer := &http.Server{
			Addr:    ":" + strconv.Itoa(*managementPort),
			Handler: managementMux,
		}

		go func() {
			observability.Info("Starting Management API server", zap.Int("port", *managementPort))
			if err := managementServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				observability.Error("Management API server error", zap.Error(err))
			}
		}()

		log.Printf("Management API running on port %d\n", *managementPort)
	}

	// Initialize and start Health/Metrics server
	if *enableHealthCheck || *enableMetrics {
		healthMux := http.NewServeMux()

		if *enableHealthCheck {
			healthMux.HandleFunc("/health", observability.HealthHandler())
			healthMux.HandleFunc("/ready", observability.ReadinessHandler())
			healthMux.HandleFunc("/live", observability.LivenessHandler())
		}

		if *enableMetrics {
			healthMux.Handle("/metrics", observability.MetricsHandler())
		}

		healthServer := &http.Server{
			Addr:    ":" + strconv.Itoa(*healthPort),
			Handler: healthMux,
		}

		go func() {
			observability.Info("Starting Health/Metrics server", zap.Int("port", *healthPort))
			if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				observability.Error("Health/Metrics server error", zap.Error(err))
			}
		}()

		log.Printf("Health/Metrics endpoints running on port %d\n", *healthPort)
	}

	// Initialize and start GraphQL server
	if *enableGraphQL {
		graphqlConfig := &graphql.GraphQLConfig{
			Introspection: true,
			Operations:    []graphql.GraphQLOperation{},
		}

		graphqlHandler, err := graphql.NewHandler(graphqlConfig)
		if err != nil {
			observability.Error("Failed to create GraphQL handler", zap.Error(err))
		} else {
			graphqlMux := http.NewServeMux()
			graphqlMux.Handle("/graphql", graphqlHandler)

			graphqlServer := &http.Server{
				Addr:    ":" + strconv.Itoa(*graphqlPort),
				Handler: graphqlMux,
			}

			go func() {
				observability.Info("Starting GraphQL server", zap.Int("port", *graphqlPort))
				if err := graphqlServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					observability.Error("GraphQL server error", zap.Error(err))
				}
			}()

			log.Printf("GraphQL server running on port %d\n", *graphqlPort)
		}
	}

	// Initialize and start gRPC server
	if *enableGRPC {
		grpcConfig := &grpc.GRPCConfig{
			Services:    []grpc.ServiceConfig{},
			Reflection:  true,
			HealthCheck: true,
		}

		grpcServer, err := grpc.NewServer(grpcConfig)
		if err != nil {
			observability.Error("Failed to create gRPC server", zap.Error(err))
		} else {
			go func() {
				addr := ":" + strconv.Itoa(*grpcPort)
				observability.Info("Starting gRPC server", zap.Int("port", *grpcPort))
				if err := grpcServer.Start(addr); err != nil {
					observability.Error("gRPC server error", zap.Error(err))
				}
			}()

			log.Printf("gRPC server running on port %d\n", *grpcPort)
		}
	}

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
