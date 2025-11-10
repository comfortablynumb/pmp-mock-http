package server

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/callback"
	"github.com/comfortablynumb/pmp-mock-http/internal/matcher"
	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/proxy"
	"github.com/comfortablynumb/pmp-mock-http/internal/recorder"
	"github.com/comfortablynumb/pmp-mock-http/internal/sse"
	"github.com/comfortablynumb/pmp-mock-http/internal/template"
	"github.com/comfortablynumb/pmp-mock-http/internal/tracker"
	"github.com/comfortablynumb/pmp-mock-http/internal/websocket"
	"github.com/quic-go/quic-go/http3"
	"gopkg.in/yaml.v3"
)

// CORSConfig represents CORS configuration
type CORSConfig struct {
	Enabled bool
	Origins string
	Methods string
	Headers string
}

// Server represents the mock HTTP server
type Server struct {
	port             int
	matcher          *matcher.Matcher
	tracker          *tracker.Tracker
	templateRenderer *template.Renderer
	callbackExecutor *callback.Executor
	proxyClient      *proxy.Client
	recorder         *recorder.Recorder
	corsConfig       *CORSConfig
	wsHandlers       map[string]*websocket.Handler // Cache WebSocket handlers by mock name
	sseHandlers      map[string]*sse.Handler       // Cache SSE handlers by mock name
	mu               sync.RWMutex
}

// NewServer creates a new mock server
func NewServer(port int, mocks []models.Mock, proxyConfig *proxy.Config, corsConfig *CORSConfig) *Server {
	var proxyClient *proxy.Client
	if proxyConfig != nil {
		var err error
		proxyClient, err = proxy.NewClient(proxyConfig)
		if err != nil {
			log.Printf("Warning: failed to create proxy client: %v\n", err)
		}
	}

	return &Server{
		port:             port,
		matcher:          matcher.NewMatcher(mocks),
		tracker:          nil,
		templateRenderer: template.NewRenderer(),
		callbackExecutor: callback.NewExecutor(),
		proxyClient:      proxyClient,
		recorder:         recorder.NewRecorder(),
		corsConfig:       corsConfig,
		wsHandlers:       make(map[string]*websocket.Handler),
		sseHandlers:      make(map[string]*sse.Handler),
	}
}

// NewServerWithTracker creates a new mock server with request tracking
func NewServerWithTracker(port int, mocks []models.Mock, t *tracker.Tracker, proxyConfig *proxy.Config, corsConfig *CORSConfig) *Server {
	var proxyClient *proxy.Client
	if proxyConfig != nil {
		var err error
		proxyClient, err = proxy.NewClient(proxyConfig)
		if err != nil {
			log.Printf("Warning: failed to create proxy client: %v\n", err)
		}
	}

	return &Server{
		port:             port,
		matcher:          matcher.NewMatcher(mocks),
		tracker:          t,
		templateRenderer: template.NewRenderer(),
		callbackExecutor: callback.NewExecutor(),
		proxyClient:      proxyClient,
		recorder:         recorder.NewRecorder(),
		corsConfig:       corsConfig,
		wsHandlers:       make(map[string]*websocket.Handler),
		sseHandlers:      make(map[string]*sse.Handler),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	http.HandleFunc("/", s.handleRequest)

	// Register recording control endpoints
	http.HandleFunc("/__recording/start", s.handleRecordingStart)
	http.HandleFunc("/__recording/stop", s.handleRecordingStop)
	http.HandleFunc("/__recording/status", s.handleRecordingStatus)
	http.HandleFunc("/__recording/clear", s.handleRecordingClear)
	http.HandleFunc("/__recording/export", s.handleRecordingExport)
	http.HandleFunc("/__recording/list", s.handleRecordingList)

	// Register scenario control endpoints
	http.HandleFunc("/__scenario/list", s.handleScenarioList)
	http.HandleFunc("/__scenario/active", s.handleScenarioActive)
	http.HandleFunc("/__scenario/set", s.handleScenarioSet)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock server listening on http://localhost%s\n", addr)

	return http.ListenAndServe(addr, nil)
}

// StartTLS starts the HTTPS server with TLS and HTTP/2 support
func (s *Server) StartTLS(certFile, keyFile string) error {
	http.HandleFunc("/", s.handleRequest)

	// Register recording control endpoints
	http.HandleFunc("/__recording/start", s.handleRecordingStart)
	http.HandleFunc("/__recording/stop", s.handleRecordingStop)
	http.HandleFunc("/__recording/status", s.handleRecordingStatus)
	http.HandleFunc("/__recording/clear", s.handleRecordingClear)
	http.HandleFunc("/__recording/export", s.handleRecordingExport)
	http.HandleFunc("/__recording/list", s.handleRecordingList)

	// Register scenario control endpoints
	http.HandleFunc("/__scenario/list", s.handleScenarioList)
	http.HandleFunc("/__scenario/active", s.handleScenarioActive)
	http.HandleFunc("/__scenario/set", s.handleScenarioSet)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock server listening on https://localhost%s (TLS with HTTP/2 enabled)\n", addr)

	// Create server with explicit HTTP/2 support
	server := &http.Server{
		Addr:         addr,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // Enable HTTP/2
	}

	return server.ListenAndServeTLS(certFile, keyFile)
}

// StartHTTP3 starts the HTTP/3 server with QUIC
func (s *Server) StartHTTP3(certFile, keyFile string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	// Register recording control endpoints
	mux.HandleFunc("/__recording/start", s.handleRecordingStart)
	mux.HandleFunc("/__recording/stop", s.handleRecordingStop)
	mux.HandleFunc("/__recording/status", s.handleRecordingStatus)
	mux.HandleFunc("/__recording/clear", s.handleRecordingClear)
	mux.HandleFunc("/__recording/export", s.handleRecordingExport)
	mux.HandleFunc("/__recording/list", s.handleRecordingList)

	// Register scenario control endpoints
	mux.HandleFunc("/__scenario/list", s.handleScenarioList)
	mux.HandleFunc("/__scenario/active", s.handleScenarioActive)
	mux.HandleFunc("/__scenario/set", s.handleScenarioSet)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock server listening on https://localhost%s (HTTP/3 with QUIC enabled)\n", addr)

	// Create HTTP/3 server
	server := &http3.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server.ListenAndServeTLS(certFile, keyFile)
}

// StartDualStack starts both HTTP/2 (TLS) and HTTP/3 (QUIC) servers on the same port
func (s *Server) StartDualStack(certFile, keyFile string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	// Register recording control endpoints
	mux.HandleFunc("/__recording/start", s.handleRecordingStart)
	mux.HandleFunc("/__recording/stop", s.handleRecordingStop)
	mux.HandleFunc("/__recording/status", s.handleRecordingStatus)
	mux.HandleFunc("/__recording/clear", s.handleRecordingClear)
	mux.HandleFunc("/__recording/export", s.handleRecordingExport)
	mux.HandleFunc("/__recording/list", s.handleRecordingList)

	// Register scenario control endpoints
	mux.HandleFunc("/__scenario/list", s.handleScenarioList)
	mux.HandleFunc("/__scenario/active", s.handleScenarioActive)
	mux.HandleFunc("/__scenario/set", s.handleScenarioSet)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock server listening on https://localhost%s (HTTP/2 + HTTP/3 dual-stack)\n", addr)

	// Create HTTP/3 server
	http3Server := &http3.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start HTTP/3 server in background
	go func() {
		if err := http3Server.ListenAndServeTLS(certFile, keyFile); err != nil {
			log.Printf("HTTP/3 server error: %v\n", err)
		}
	}()

	// Create and start HTTP/2 server (also serves HTTP/1.1)
	http2Server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)), // Enable HTTP/2
	}

	return http2Server.ListenAndServeTLS(certFile, keyFile)
}

// handleRequest handles incoming HTTP requests
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s from %s\n", r.Method, r.URL.Path, r.RemoteAddr)

	// Handle CORS if enabled
	if s.corsConfig != nil && s.corsConfig.Enabled {
		w.Header().Set("Access-Control-Allow-Origin", s.corsConfig.Origins)
		w.Header().Set("Access-Control-Allow-Methods", s.corsConfig.Methods)
		w.Header().Set("Access-Control-Allow-Headers", s.corsConfig.Headers)
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read the body first so we can log it and use it for matching
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// Restore the body for the matcher to read
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Log request details (limit body size to avoid hanging on large payloads)
	bodyStr := ""
	if len(bodyBytes) > 0 {
		const maxLogSize = 1024 // Log up to 1KB of body
		if len(bodyBytes) <= maxLogSize {
			bodyStr = string(bodyBytes)
			log.Printf("Request body: %s\n", bodyStr)
		} else {
			bodyStr = string(bodyBytes[:maxLogSize]) + "..."
			log.Printf("Request body: %s (%d bytes total)\n", bodyStr, len(bodyBytes))
		}
	}

	// Extract headers for logging
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	// Find a matching mock
	mock, err := s.matcher.FindMatch(r)
	if err != nil {
		log.Printf("Error matching request: %v\n", err)
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		if s.tracker != nil {
			s.tracker.Log(tracker.RequestLog{
				Method: r.Method, URI: r.URL.RequestURI(), Headers: headers, Body: bodyStr,
				Matched: false, StatusCode: http.StatusInternalServerError,
				Response: "Error processing request", RemoteAddr: r.RemoteAddr,
			})
		}
		return
	}

	if mock == nil {
		log.Printf("No mock found for %s %s\n", r.Method, r.URL.Path)

		// If proxy is configured, forward the request
		if s.proxyClient != nil {
			// Restore the body for the proxy to read
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			log.Printf("Forwarding request to proxy\n")
			if err := s.proxyClient.Forward(w, r); err != nil {
				log.Printf("Proxy error: %v\n", err)
				http.Error(w, "Proxy error", http.StatusBadGateway)
				if s.tracker != nil {
					s.tracker.Log(tracker.RequestLog{
						Method: r.Method, URI: r.URL.RequestURI(), Headers: headers, Body: bodyStr,
						Matched: false, StatusCode: http.StatusBadGateway,
						Response: "Proxy error", RemoteAddr: r.RemoteAddr,
					})
				}
			}
			// Proxy handled the request, don't track it as not found
			return
		}

		// No proxy configured, return 404
		http.NotFound(w, r)
		if s.tracker != nil {
			s.tracker.Log(tracker.RequestLog{
				Method: r.Method, URI: r.URL.RequestURI(), Headers: headers, Body: bodyStr,
				Matched: false, StatusCode: http.StatusNotFound,
				Response: "404 page not found", RemoteAddr: r.RemoteAddr,
			})
		}
		return
	}

	log.Printf("Matched mock: %s\n", mock.Name)

	// Handle WebSocket protocol
	if mock.Protocol == "websocket" {
		log.Printf("Handling WebSocket connection for mock: %s\n", mock.Name)
		s.handleWebSocket(w, r, mock)
		return
	}

	// Handle SSE protocol
	if mock.Protocol == "sse" {
		log.Printf("Handling SSE stream for mock: %s\n", mock.Name)
		s.handleSSE(w, r, mock)
		return
	}

	// Create request data for templates and callbacks
	requestData := template.NewRequestData(r, string(bodyBytes))

	// Execute callback if specified
	if mock.Response.Callback != nil {
		s.callbackExecutor.Execute(mock.Response.Callback, requestData)
	}

	// Apply chaos engineering (if enabled)
	chaosStatusCode, shouldFail := s.applyChaos(mock.Response.Chaos)
	if shouldFail {
		// Chaos injected a failure - return error immediately
		w.WriteHeader(chaosStatusCode)
		chaosBody := fmt.Sprintf(`{"error":"Chaos engineering failure","status":%d}`, chaosStatusCode)
		if _, err := w.Write([]byte(chaosBody)); err != nil {
			log.Printf("Error writing chaos response: %v\n", err)
		}

		// Track the chaos response
		if s.tracker != nil {
			s.tracker.Log(tracker.RequestLog{
				Method: r.Method, URI: r.URL.RequestURI(), Headers: headers, Body: bodyStr,
				Matched: true, MockName: mock.Name + " (chaos)", MockConfig: mock,
				StatusCode: chaosStatusCode, Response: chaosBody, RemoteAddr: r.RemoteAddr,
			})
		}
		return
	}

	// Calculate latency (advanced latency or standard delay)
	latency := s.calculateLatency(mock.Response.Latency, mock.Response.Delay)
	if latency > 0 {
		time.Sleep(time.Duration(latency) * time.Millisecond)
	}

	// Render response headers (with templates if enabled)
	responseHeaders := s.renderHeaderTemplates(mock.Response.Headers, mock.Response.HeaderTemplates, requestData)

	// Set response headers
	for key, value := range responseHeaders {
		w.Header().Set(key, value)
	}

	// Set status code
	w.WriteHeader(mock.Response.StatusCode)

	// Render response body (with template if enabled)
	responseBody := ""
	if mock.Response.Body != "" {
		responseBody = mock.Response.Body
		if mock.Response.Template {
			rendered, err := s.templateRenderer.Render(mock.Response.Body, requestData)
			if err != nil {
				log.Printf("Error rendering response template: %v\n", err)
				// Fall back to the original body
			} else {
				responseBody = rendered
			}
		}
		if _, err := w.Write([]byte(responseBody)); err != nil {
			log.Printf("Error writing response body: %v\n", err)
		}
	}

	log.Printf("Returned %d response\n", mock.Response.StatusCode)

	// Track matched request
	if s.tracker != nil {
		s.tracker.Log(tracker.RequestLog{
			Method: r.Method, URI: r.URL.RequestURI(), Headers: headers, Body: bodyStr,
			Matched: true, MockName: mock.Name, MockConfig: mock, StatusCode: mock.Response.StatusCode,
			Response: responseBody, RemoteAddr: r.RemoteAddr,
		})
	}

	// Record request/response if recording is enabled
	if s.recorder.IsEnabled() {
		// Convert response headers to map
		respHeaders := make(map[string]string)
		for key, values := range w.Header() {
			if len(values) > 0 {
				respHeaders[key] = values[0]
			}
		}
		s.recorder.Record(r.Method, r.URL.Path, headers, bodyStr,
			mock.Response.StatusCode, respHeaders, responseBody)
	}
}

// UpdateMocks updates the server's matcher with new mocks
func (s *Server) UpdateMocks(mocks []models.Mock) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.matcher.UpdateMocks(mocks)
}

// handleRecordingStart handles starting the recording
func (s *Server) handleRecordingStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.recorder.Start()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "recording",
		"message": "Recording started",
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// handleRecordingStop handles stopping the recording
func (s *Server) handleRecordingStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.recorder.Stop()
	count := s.recorder.Count()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "stopped",
		"message": "Recording stopped",
		"count":   count,
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// handleRecordingStatus handles getting the recording status
func (s *Server) handleRecordingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled": s.recorder.IsEnabled(),
		"count":   s.recorder.Count(),
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// handleRecordingClear handles clearing all recordings
func (s *Server) handleRecordingClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.recorder.Clear()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "cleared",
		"message": "All recordings cleared",
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// handleRecordingExport handles exporting recordings as mocks
func (s *Server) handleRecordingExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	format := r.URL.Query().Get("format") // "json" or "yaml"
	groupBy := r.URL.Query().Get("group")  // "uri" to group by URI

	groupByURI := groupBy == "uri"
	mockSpec := s.recorder.ExportAsMocks(groupByURI)

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=recorded-mocks.json")
		if err := json.NewEncoder(w).Encode(mockSpec); err != nil {
			log.Printf("Error encoding JSON response: %v\n", err)
		}
	} else {
		// Default to YAML
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Content-Disposition", "attachment; filename=recorded-mocks.yaml")
		if err := yaml.NewEncoder(w).Encode(mockSpec); err != nil {
			log.Printf("Error encoding YAML response: %v\n", err)
		}
	}
}

// handleRecordingList handles listing all recordings
func (s *Server) handleRecordingList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	recordings := s.recorder.GetRecordings()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(recordings),
		"recordings": recordings,
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// handleScenarioList handles listing all available scenarios
func (s *Server) handleScenarioList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	scenarios := s.matcher.GetAvailableScenarios()
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"scenarios": scenarios,
		"count":     len(scenarios),
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// handleScenarioActive handles getting the currently active scenario
func (s *Server) handleScenarioActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	activeScenario := s.matcher.GetActiveScenario()
	s.mu.RUnlock()

	// If no scenario is set, return empty string or "all"
	if activeScenario == "" {
		activeScenario = "all"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"active_scenario": activeScenario,
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// handleScenarioSet handles setting the active scenario
func (s *Server) handleScenarioSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get scenario from query parameter or request body
	scenario := r.URL.Query().Get("scenario")
	if scenario == "" {
		// Try to read from request body
		var requestBody map[string]string
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err == nil {
			scenario = requestBody["scenario"]
		}
	}

	// "all" means clear the scenario (show all mocks)
	if scenario == "all" {
		scenario = ""
	}

	s.mu.Lock()
	s.matcher.SetScenario(scenario)
	s.mu.Unlock()

	log.Printf("Active scenario set to: %s\n", scenario)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"active_scenario": scenario,
		"message":         fmt.Sprintf("Active scenario set to: %s", scenario),
	}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}

// applyChaos applies chaos engineering logic to the response
// Returns (statusCode, shouldFail)
func (s *Server) applyChaos(chaos *models.ChaosConfig) (int, bool) {
	if chaos == nil || !chaos.Enabled {
		return 0, false
	}

	// Check if we should inject failure
	if rand.Float64() < chaos.FailureRate {
		// Inject failure - pick random error code
		if len(chaos.ErrorCodes) > 0 {
			errorCode := chaos.ErrorCodes[rand.Intn(len(chaos.ErrorCodes))]
			log.Printf("Chaos: Injecting failure with status code %d\n", errorCode)
			return errorCode, true
		}
	}

	// Inject latency if configured
	if chaos.LatencyMax > 0 {
		latency := chaos.LatencyMin
		if chaos.LatencyMax > chaos.LatencyMin {
			latency = chaos.LatencyMin + rand.Intn(chaos.LatencyMax-chaos.LatencyMin)
		}
		if latency > 0 {
			log.Printf("Chaos: Injecting %dms latency\n", latency)
			time.Sleep(time.Duration(latency) * time.Millisecond)
		}
	}

	return 0, false
}

// calculateLatency calculates latency based on the latency configuration
func (s *Server) calculateLatency(latency *models.LatencyConfig, baseDelay int) int {
	if latency == nil {
		return baseDelay
	}

	switch latency.Type {
	case "random":
		if latency.Max > 0 {
			min := latency.Min
			max := latency.Max
			if max > min {
				return min + rand.Intn(max-min)
			}
			return min
		}
		return baseDelay

	case "percentile":
		// Use percentile-based latency distribution
		roll := rand.Float64()
		if roll < 0.50 {
			return latency.P50
		} else if roll < 0.95 {
			return latency.P95
		} else {
			return latency.P99
		}

	case "fixed":
		return baseDelay

	default:
		return baseDelay
	}
}

// renderHeaderTemplates renders templates in response headers
func (s *Server) renderHeaderTemplates(headers map[string]string, useTemplates bool, requestData *template.RequestData) map[string]string {
	if !useTemplates || len(headers) == 0 {
		return headers
	}

	rendered := make(map[string]string)
	for key, value := range headers {
		renderedValue, err := s.templateRenderer.Render(value, requestData)
		if err != nil {
			log.Printf("Error rendering header template for '%s': %v\n", key, err)
			rendered[key] = value // Fall back to original value
		} else {
			rendered[key] = renderedValue
		}
	}
	return rendered
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request, mock *models.Mock) {
	// Get or create WebSocket handler for this mock
	handler, exists := s.wsHandlers[mock.Name]
	if !exists {
		handler = websocket.NewHandler(mock, s.templateRenderer)
		s.wsHandlers[mock.Name] = handler
	}

	// Track the connection attempt
	if s.tracker != nil {
		headers := make(map[string]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		s.tracker.Log(tracker.RequestLog{
			Method:     r.Method,
			URI:        r.URL.RequestURI(),
			Headers:    headers,
			Body:       "",
			Matched:    true,
			MockName:   mock.Name + " (websocket)",
			MockConfig: mock,
			StatusCode: 101, // Switching Protocols
			Response:   "WebSocket connection established",
			RemoteAddr: r.RemoteAddr,
		})
	}

	// Handle the WebSocket connection
	handler.HandleConnection(w, r)
}

// handleSSE handles Server-Sent Events streams
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request, mock *models.Mock) {
	// Get or create SSE handler for this mock
	handler, exists := s.sseHandlers[mock.Name]
	if !exists {
		handler = sse.NewHandler(mock, s.templateRenderer)
		s.sseHandlers[mock.Name] = handler
	}

	// Track the SSE stream
	if s.tracker != nil {
		headers := make(map[string]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		s.tracker.Log(tracker.RequestLog{
			Method:     r.Method,
			URI:        r.URL.RequestURI(),
			Headers:    headers,
			Body:       "",
			Matched:    true,
			MockName:   mock.Name + " (sse)",
			MockConfig: mock,
			StatusCode: 200,
			Response:   "SSE stream established",
			RemoteAddr: r.RemoteAddr,
		})
	}

	// Handle the SSE stream
	handler.HandleStream(w, r)
}
