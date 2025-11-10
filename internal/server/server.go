package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/callback"
	"github.com/comfortablynumb/pmp-mock-http/internal/matcher"
	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/proxy"
	"github.com/comfortablynumb/pmp-mock-http/internal/recorder"
	"github.com/comfortablynumb/pmp-mock-http/internal/template"
	"github.com/comfortablynumb/pmp-mock-http/internal/tracker"
	"gopkg.in/yaml.v3"
)

// Server represents the mock HTTP server
type Server struct {
	port             int
	matcher          *matcher.Matcher
	tracker          *tracker.Tracker
	templateRenderer *template.Renderer
	callbackExecutor *callback.Executor
	proxyClient      *proxy.Client
	recorder         *recorder.Recorder
	mu               sync.RWMutex
}

// NewServer creates a new mock server
func NewServer(port int, mocks []models.Mock, proxyConfig *proxy.Config) *Server {
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
	}
}

// NewServerWithTracker creates a new mock server with request tracking
func NewServerWithTracker(port int, mocks []models.Mock, t *tracker.Tracker, proxyConfig *proxy.Config) *Server {
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

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock server listening on http://localhost%s\n", addr)

	return http.ListenAndServe(addr, nil)
}

// StartTLS starts the HTTPS server with TLS
func (s *Server) StartTLS(certFile, keyFile string) error {
	http.HandleFunc("/", s.handleRequest)

	// Register recording control endpoints
	http.HandleFunc("/__recording/start", s.handleRecordingStart)
	http.HandleFunc("/__recording/stop", s.handleRecordingStop)
	http.HandleFunc("/__recording/status", s.handleRecordingStatus)
	http.HandleFunc("/__recording/clear", s.handleRecordingClear)
	http.HandleFunc("/__recording/export", s.handleRecordingExport)
	http.HandleFunc("/__recording/list", s.handleRecordingList)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock server listening on https://localhost%s (TLS enabled)\n", addr)

	return http.ListenAndServeTLS(addr, certFile, keyFile, nil)
}

// handleRequest handles incoming HTTP requests
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s from %s\n", r.Method, r.URL.Path, r.RemoteAddr)

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

	// Create request data for templates and callbacks
	requestData := template.NewRequestData(r, string(bodyBytes))

	// Execute callback if specified
	if mock.Response.Callback != nil {
		s.callbackExecutor.Execute(mock.Response.Callback, requestData)
	}

	// Apply response delay if specified
	if mock.Response.Delay > 0 {
		time.Sleep(time.Duration(mock.Response.Delay) * time.Millisecond)
	}

	// Set response headers
	for key, value := range mock.Response.Headers {
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "recording",
		"message": "Recording started",
	})
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "stopped",
		"message": "Recording stopped",
		"count":   count,
	})
}

// handleRecordingStatus handles getting the recording status
func (s *Server) handleRecordingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"enabled": s.recorder.IsEnabled(),
		"count":   s.recorder.Count(),
	})
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "cleared",
		"message": "All recordings cleared",
	})
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
		json.NewEncoder(w).Encode(mockSpec)
	} else {
		// Default to YAML
		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Content-Disposition", "attachment; filename=recorded-mocks.yaml")
		yaml.NewEncoder(w).Encode(mockSpec)
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
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":      len(recordings),
		"recordings": recordings,
	})
}
