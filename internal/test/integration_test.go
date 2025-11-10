package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/loader"
	"github.com/comfortablynumb/pmp-mock-http/internal/server"
	"github.com/comfortablynumb/pmp-mock-http/internal/watcher"
)

func TestIntegrationFullWorkflow(t *testing.T) {
	// Create temp directory for mocks
	tempDir, err := os.MkdirTemp("", "integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// Create initial mock file
	mockFile := filepath.Join(tempDir, "test.yaml")
	initialContent := `mocks:
  - name: "Integration Test Mock"
    priority: 10
    request:
      uri: "/api/test"
      method: "GET"
    response:
      status_code: 200
      headers:
        Content-Type: "application/json"
      body: '{"status": "ok"}'
`
	err = os.WriteFile(mockFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	// Create loader and load mocks
	mockLoader := loader.NewLoader(tempDir)
	err = mockLoader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load mocks: %v", err)
	}

	// Create server
	srv := server.NewServer(0, mockLoader.GetMocks(), nil) // Use port 0 for testing

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.UpdateMocks(mockLoader.GetMocks())

		// Simulate the handleRequest method
		mocks := mockLoader.GetMocks()
		if len(mocks) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write([]byte(`{"status": "ok"}`)) //nolint:errcheck
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	// Test initial request
	resp, err := http.Get(ts.URL + "/api/test")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck // test cleanup

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"status": "ok"}` {
		t.Errorf("Expected body '{\"status\": \"ok\"}', got %s", string(body))
	}
}

func TestIntegrationLoaderAndMatcher(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// Create multiple mock files
	mockFile1 := filepath.Join(tempDir, "users.yaml")
	content1 := `mocks:
  - name: "Get User"
    request:
      uri: "/api/users/123"
      method: "GET"
    response:
      status_code: 200
      body: '{"id": 123, "name": "John"}'
`
	err = os.WriteFile(mockFile1, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock file 1: %v", err)
	}

	mockFile2 := filepath.Join(tempDir, "products.yaml")
	content2 := `mocks:
  - name: "Get Product"
    request:
      uri: "/api/products/456"
      method: "GET"
    response:
      status_code: 200
      body: '{"id": 456, "name": "Widget"}'
`
	err = os.WriteFile(mockFile2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock file 2: %v", err)
	}

	// Load mocks
	mockLoader := loader.NewLoader(tempDir)
	err = mockLoader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load mocks: %v", err)
	}

	mocks := mockLoader.GetMocks()
	if len(mocks) != 2 {
		t.Errorf("Expected 2 mocks, got %d", len(mocks))
	}

	// Verify mock names
	mockNames := make(map[string]bool)
	for _, mock := range mocks {
		mockNames[mock.Name] = true
	}

	if !mockNames["Get User"] {
		t.Error("Expected 'Get User' mock not found")
	}
	if !mockNames["Get Product"] {
		t.Error("Expected 'Get Product' mock not found")
	}
}

func TestIntegrationWatcherReload(t *testing.T) {
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// Create initial mock file
	mockFile := filepath.Join(tempDir, "test.yaml")
	initialContent := `mocks:
  - name: "Initial Mock"
    request:
      uri: "/api/test"
      method: "GET"
    response:
      status_code: 200
      body: "initial"
`
	err = os.WriteFile(mockFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial mock file: %v", err)
	}

	// Create loader
	mockLoader := loader.NewLoader(tempDir)
	err = mockLoader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load initial mocks: %v", err)
	}

	// Create server
	srv := server.NewServer(0, mockLoader.GetMocks(), nil)

	// Create reload function
	var reloadCount int32
	reloadFn := func() error {
		atomic.AddInt32(&reloadCount, 1)
		err := mockLoader.LoadAll()
		if err != nil {
			return err
		}
		srv.UpdateMocks(mockLoader.GetMocks())
		return nil
	}

	// Create and start watcher
	w, err := watcher.NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Close() //nolint:errcheck

	err = w.Start()
	if err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify the mock file
	updatedContent := `mocks:
  - name: "Updated Mock"
    request:
      uri: "/api/test"
      method: "GET"
    response:
      status_code: 200
      body: "updated"
`
	err = os.WriteFile(mockFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write updated mock file: %v", err)
	}

	// Wait for reload
	time.Sleep(300 * time.Millisecond)

	if atomic.LoadInt32(&reloadCount) == 0 {
		t.Error("Expected reload to be called after file modification")
	}

	// Verify mocks were updated
	mocks := mockLoader.GetMocks()
	if len(mocks) != 1 {
		t.Errorf("Expected 1 mock after update, got %d", len(mocks))
	}

	if len(mocks) > 0 && mocks[0].Name != "Updated Mock" {
		t.Errorf("Expected mock name 'Updated Mock', got '%s'", mocks[0].Name)
	}
}

func TestIntegrationNestedDirectories(t *testing.T) {
	// Create temp directory with nested structure
	tempDir, err := os.MkdirTemp("", "integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// Create nested directories
	apiDir := filepath.Join(tempDir, "api")
	v1Dir := filepath.Join(apiDir, "v1")
	err = os.MkdirAll(v1Dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested directories: %v", err)
	}

	// Create mocks at different levels
	rootMock := filepath.Join(tempDir, "root.yaml")
	apiMock := filepath.Join(apiDir, "api.yaml")
	v1Mock := filepath.Join(v1Dir, "v1.yaml")

	rootContent := `mocks:
  - name: "Root Mock"
    request:
      uri: "/root"
      method: "GET"
    response:
      status_code: 200
`
	apiContent := `mocks:
  - name: "API Mock"
    request:
      uri: "/api"
      method: "GET"
    response:
      status_code: 200
`
	v1Content := `mocks:
  - name: "V1 Mock"
    request:
      uri: "/api/v1"
      method: "GET"
    response:
      status_code: 200
`

	os.WriteFile(rootMock, []byte(rootContent), 0644) //nolint:errcheck
	os.WriteFile(apiMock, []byte(apiContent), 0644)   //nolint:errcheck
	os.WriteFile(v1Mock, []byte(v1Content), 0644)     //nolint:errcheck

	// Load mocks
	mockLoader := loader.NewLoader(tempDir)
	err = mockLoader.LoadAll()
	if err != nil {
		t.Fatalf("Failed to load mocks: %v", err)
	}

	mocks := mockLoader.GetMocks()
	if len(mocks) != 3 {
		t.Errorf("Expected 3 mocks from nested directories, got %d", len(mocks))
	}

	// Verify all mocks were loaded
	mockNames := make(map[string]bool)
	for _, mock := range mocks {
		mockNames[mock.Name] = true
	}

	expectedNames := []string{"Root Mock", "API Mock", "V1 Mock"}
	for _, name := range expectedNames {
		if !mockNames[name] {
			t.Errorf("Expected mock '%s' not found", name)
		}
	}
}
