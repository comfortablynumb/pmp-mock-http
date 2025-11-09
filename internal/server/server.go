package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/matcher"
	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/tracker"
)

// Server represents the mock HTTP server
type Server struct {
	port    int
	matcher *matcher.Matcher
	tracker *tracker.Tracker
	mu      sync.RWMutex
}

// NewServer creates a new mock server
func NewServer(port int, mocks []models.Mock) *Server {
	return &Server{
		port:    port,
		matcher: matcher.NewMatcher(mocks),
		tracker: nil,
	}
}

// NewServerWithTracker creates a new mock server with request tracking
func NewServerWithTracker(port int, mocks []models.Mock, t *tracker.Tracker) *Server {
	return &Server{
		port:    port,
		matcher: matcher.NewMatcher(mocks),
		tracker: t,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	http.HandleFunc("/", s.handleRequest)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Mock server listening on http://localhost%s\n", addr)

	return http.ListenAndServe(addr, nil)
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

	// Write response body
	responseBody := ""
	if mock.Response.Body != "" {
		responseBody = mock.Response.Body
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
}

// UpdateMocks updates the server's matcher with new mocks
func (s *Server) UpdateMocks(mocks []models.Mock) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.matcher.UpdateMocks(mocks)
}
