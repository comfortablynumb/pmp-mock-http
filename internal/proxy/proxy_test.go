package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantErr   bool
		errString string
	}{
		{
			name:      "nil config",
			config:    nil,
			wantErr:   true,
			errString: "proxy target is required",
		},
		{
			name: "empty target",
			config: &Config{
				Target: "",
			},
			wantErr:   true,
			errString: "proxy target is required",
		},
		{
			name: "invalid URL",
			config: &Config{
				Target: "://invalid",
			},
			wantErr:   true,
			errString: "invalid proxy target URL",
		},
		{
			name: "valid config",
			config: &Config{
				Target:  "http://example.com",
				Timeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid config with default timeout",
			config: &Config{
				Target: "http://example.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("NewClient() error = %v, want error containing %v", err, tt.errString)
				}
			} else {
				if err != nil {
					t.Errorf("NewClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewClient() returned nil client")
				}
			}
		})
	}
}

func TestClientForward(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back request info
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Backend", "true")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"proxied": true, "path": "` + r.URL.Path + `"}`))
	}))
	defer backend.Close()

	config := &Config{
		Target:  backend.URL,
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/test/path?query=value", nil)
	req.Header.Set("X-Custom-Header", "test-value")
	w := httptest.NewRecorder()

	// Forward the request
	err = client.Forward(w, req)
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}

	resp := w.Result()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Forward() status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Check headers
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Forward() Content-Type = %s, want application/json", resp.Header.Get("Content-Type"))
	}

	if resp.Header.Get("X-Backend") != "true" {
		t.Errorf("Forward() X-Backend header not found")
	}

	// Check body
	body, _ := io.ReadAll(resp.Body)
	expectedBody := `{"proxied": true, "path": "/test/path"}`
	if string(body) != expectedBody {
		t.Errorf("Forward() body = %s, want %s", string(body), expectedBody)
	}
}

func TestClientForwardWithBody(t *testing.T) {
	// Create a test backend server that echoes the request body
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer backend.Close()

	config := &Config{
		Target: backend.URL,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}

	// Create a test request with body
	requestBody := `{"test": "data"}`
	req := httptest.NewRequest("POST", "/api/test", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Forward the request
	err = client.Forward(w, req)
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if string(body) != requestBody {
		t.Errorf("Forward() body = %s, want %s", string(body), requestBody)
	}
}

func TestClientForwardPreserveHost(t *testing.T) {
	receivedHost := ""

	// Create a test backend server that captures the Host header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHost = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	tests := []struct {
		name         string
		preserveHost bool
		requestHost  string
		wantHost     string
	}{
		{
			name:         "preserve host enabled",
			preserveHost: true,
			requestHost:  "original.example.com",
			wantHost:     "original.example.com",
		},
		{
			name:         "preserve host disabled",
			preserveHost: false,
			requestHost:  "original.example.com",
			wantHost:     strings.TrimPrefix(backend.URL, "http://"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Target:       backend.URL,
				PreserveHost: tt.preserveHost,
			}

			client, err := NewClient(config)
			if err != nil {
				t.Fatalf("Failed to create proxy client: %v", err)
			}

			req := httptest.NewRequest("GET", "/test", nil)
			req.Host = tt.requestHost
			w := httptest.NewRecorder()

			err = client.Forward(w, req)
			if err != nil {
				t.Fatalf("Forward() error = %v", err)
			}

			if receivedHost != tt.wantHost {
				t.Errorf("Forward() Host = %s, want %s", receivedHost, tt.wantHost)
			}
		})
	}
}

func TestClientForwardXForwardedHeaders(t *testing.T) {
	var receivedHeaders http.Header

	// Create a test backend server that captures headers
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	config := &Config{
		Target: backend.URL,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Host = "original.example.com"
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()

	err = client.Forward(w, req)
	if err != nil {
		t.Fatalf("Forward() error = %v", err)
	}

	// Check X-Forwarded headers
	if receivedHeaders.Get("X-Forwarded-For") == "" {
		t.Error("Forward() X-Forwarded-For header not set")
	}

	if receivedHeaders.Get("X-Forwarded-Proto") == "" {
		t.Error("Forward() X-Forwarded-Proto header not set")
	}

	if receivedHeaders.Get("X-Forwarded-Host") != "original.example.com" {
		t.Errorf("Forward() X-Forwarded-Host = %s, want original.example.com", receivedHeaders.Get("X-Forwarded-Host"))
	}
}

func TestClientForwardError(t *testing.T) {
	// Create a config with an unreachable target
	config := &Config{
		Target:  "http://localhost:0", // Port 0 should be unreachable
		Timeout: 1 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create proxy client: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Forward should return an error
	err = client.Forward(w, req)
	if err == nil {
		t.Error("Forward() expected error for unreachable target, got nil")
	}
}
