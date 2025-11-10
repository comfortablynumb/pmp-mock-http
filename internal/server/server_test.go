package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/proxy"
)

func TestServerBasicRequest(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Test Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"result": "success"}`,
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", resp.Header.Get("Content-Type"))
	}

	expectedBody := `{"result": "success"}`
	if string(body) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(body))
	}
}

func TestServerNoMatch(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Test Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	req := httptest.NewRequest("GET", "/api/other", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()

	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404 for no match, got %d", resp.StatusCode)
	}
}

func TestServerPOSTRequest(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "POST Mock",
			Request: models.Request{
				URI:    "/api/users",
				Method: "POST",
			},
			Response: models.Response{
				StatusCode: 201,
				Headers: map[string]string{
					"Location": "/api/users/123",
				},
				Body: `{"id": 123}`,
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	body := bytes.NewBufferString(`{"name": "John"}`)
	req := httptest.NewRequest("POST", "/api/users", body)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Location") != "/api/users/123" {
		t.Errorf("Expected Location header '/api/users/123', got '%s'", resp.Header.Get("Location"))
	}

	expectedBody := `{"id": 123}`
	if string(respBody) != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, string(respBody))
	}
}

func TestServerResponseDelay(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Delayed Mock",
			Request: models.Request{
				URI:    "/api/slow",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "delayed response",
				Delay:      100, // 100ms delay
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	req := httptest.NewRequest("GET", "/api/slow", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	srv.handleRequest(w, req)
	elapsed := time.Since(start)

	// Should take at least 100ms
	if elapsed < 100*time.Millisecond {
		t.Errorf("Expected delay of at least 100ms, got %v", elapsed)
	}

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestServerMultipleHeaders(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Multi-Header Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"X-Custom-1":    "value1",
					"X-Custom-2":    "value2",
					"Cache-Control": "no-cache",
				},
				Body: "test",
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()

	expectedHeaders := map[string]string{
		"Content-Type":  "application/json",
		"X-Custom-1":    "value1",
		"X-Custom-2":    "value2",
		"Cache-Control": "no-cache",
	}

	for key, expectedValue := range expectedHeaders {
		actualValue := resp.Header.Get(key)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s='%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

func TestServerUpdateMocks(t *testing.T) {
	initialMocks := []models.Mock{
		{
			Name: "Initial Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "initial",
			},
		},
	}

	srv := NewServer(8080, initialMocks, nil)

	// Test initial state
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	w1 := httptest.NewRecorder()
	srv.handleRequest(w1, req1)

	resp1 := w1.Result()
	body1, _ := io.ReadAll(resp1.Body)

	if string(body1) != "initial" {
		t.Errorf("Expected 'initial', got '%s'", string(body1))
	}

	// Update mocks
	updatedMocks := []models.Mock{
		{
			Name: "Updated Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "updated",
			},
		},
	}
	srv.UpdateMocks(updatedMocks)

	// Test updated state
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	w2 := httptest.NewRecorder()
	srv.handleRequest(w2, req2)

	resp2 := w2.Result()
	body2, _ := io.ReadAll(resp2.Body)

	if string(body2) != "updated" {
		t.Errorf("Expected 'updated', got '%s'", string(body2))
	}
}

func TestServerEmptyBody(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Empty Body Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 204, // No Content
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 204 {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	if len(body) != 0 {
		t.Errorf("Expected empty body, got '%s'", string(body))
	}
}

func TestServerDifferentMethods(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "GET Mock",
			Request: models.Request{
				URI:    "/api/resource",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "GET response",
			},
		},
		{
			Name: "POST Mock",
			Request: models.Request{
				URI:    "/api/resource",
				Method: "POST",
			},
			Response: models.Response{
				StatusCode: 201,
				Body:       "POST response",
			},
		},
		{
			Name: "DELETE Mock",
			Request: models.Request{
				URI:    "/api/resource",
				Method: "DELETE",
			},
			Response: models.Response{
				StatusCode: 204,
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	tests := []struct {
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{"GET", 200, "GET response"},
		{"POST", 201, "POST response"},
		{"DELETE", 204, ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/api/resource", nil)
		w := httptest.NewRecorder()

		srv.handleRequest(w, req)

		resp := w.Result()
		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode != tt.expectedStatus {
			t.Errorf("%s: expected status %d, got %d", tt.method, tt.expectedStatus, resp.StatusCode)
		}

		if string(body) != tt.expectedBody {
			t.Errorf("%s: expected body '%s', got '%s'", tt.method, tt.expectedBody, string(body))
		}
	}
}

func TestServerConcurrentRequests(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Concurrent Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "concurrent response",
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	// Make concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()

			srv.handleRequest(w, req)

			resp := w.Result()
			if resp.StatusCode != 200 {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestServerLargeRequestBody(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Large Body Mock",
			Request: models.Request{
				URI:    "/api/upload",
				Method: "POST",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "received",
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	// Create a large body (1MB)
	largeBody := bytes.Repeat([]byte("a"), 1024*1024)
	req := httptest.NewRequest("POST", "/api/upload", bytes.NewReader(largeBody))
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 for large body, got %d", resp.StatusCode)
	}
}

func TestServerVariousStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"OK", 200},
		{"Created", 201},
		{"Accepted", 202},
		{"No Content", 204},
		{"Bad Request", 400},
		{"Unauthorized", 401},
		{"Forbidden", 403},
		{"Not Found", 404},
		{"Internal Server Error", 500},
		{"Service Unavailable", 503},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mocks := []models.Mock{
				{
					Name: tt.name,
					Request: models.Request{
						URI:    "/api/test",
						Method: "GET",
					},
					Response: models.Response{
						StatusCode: tt.statusCode,
						Body:       tt.name,
					},
				},
			}

			srv := NewServer(8080, mocks, nil)

			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()

			srv.handleRequest(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, resp.StatusCode)
			}
		})
	}
}

func TestServerConcurrentUpdates(t *testing.T) {
	initialMocks := []models.Mock{
		{
			Name: "Initial",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "initial",
			},
		},
	}

	srv := NewServer(8080, initialMocks, nil)

	// Concurrently update mocks and make requests
	done := make(chan bool, 20)

	// Start requests
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()
			srv.handleRequest(w, req)
			done <- true
		}()
	}

	// Concurrent updates
	for i := 0; i < 10; i++ {
		go func() {
			newMocks := []models.Mock{
				{
					Name: "Updated",
					Request: models.Request{
						URI:    "/api/test",
						Method: "GET",
					},
					Response: models.Response{
						StatusCode: 200,
						Body:       "updated",
					},
				},
			}
			srv.UpdateMocks(newMocks)
			done <- true
		}()
	}

	// Wait for all operations
	for i := 0; i < 20; i++ {
		<-done
	}
}

func TestServerQueryParameters(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "With Query Params",
			Request: models.Request{
				URI:    "/api/search",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "search results",
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	// Request with query parameters
	req := httptest.NewRequest("GET", "/api/search?q=test&limit=10", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200 with query params, got %d", resp.StatusCode)
	}
}

func TestServerSpecialCharactersInBody(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Special Chars Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "POST",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       `{"message": "success"}`,
			},
		},
	}

	srv := NewServer(8080, mocks, nil)

	body := bytes.NewBufferString(`{"test": "special chars"}`)
	req := httptest.NewRequest("POST", "/api/test", body)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestServerProxyPassthrough(t *testing.T) {
	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"proxied": true}`))
	}))
	defer backend.Close()

	// Create a mock that won't match our request
	mocks := []models.Mock{
		{
			Name: "Test Mock",
			Request: models.Request{
				URI:    "/api/matched",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       `{"matched": true}`,
			},
		},
	}

	// Create proxy config
	proxyConfig := &proxy.Config{
		Target: backend.URL,
	}

	srv := NewServer(8080, mocks, proxyConfig)

	// Make a request that doesn't match any mock
	req := httptest.NewRequest("GET", "/api/unmatched", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	// Should be proxied to backend
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	expectedBody := `{"proxied": true}`
	if string(body) != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, string(body))
	}
}

func TestServerProxyDisabled(t *testing.T) {
	// Create a mock that won't match our request
	mocks := []models.Mock{
		{
			Name: "Test Mock",
			Request: models.Request{
				URI:    "/api/matched",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       `{"matched": true}`,
			},
		},
	}

	// No proxy config
	srv := NewServer(8080, mocks, nil)

	// Make a request that doesn't match any mock
	req := httptest.NewRequest("GET", "/api/unmatched", nil)
	w := httptest.NewRecorder()

	srv.handleRequest(w, req)

	resp := w.Result()

	// Should return 404 when proxy is disabled
	if resp.StatusCode != 404 {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}
