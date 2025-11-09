package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoadAll(t *testing.T) {
	testDir := "testdata"
	loader := NewLoader(testDir)

	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	mocks := loader.GetMocks()

	// We expect 4 mocks:
	// - 2 from valid-mock.yaml
	// - 1 from subdir/nested-mock.yaml
	// - 1 from defaults.yaml
	// (invalid.yaml should fail to parse but not stop loading)
	// (readme.txt should be ignored)
	if len(mocks) != 4 {
		t.Errorf("Expected 4 mocks, got %d", len(mocks))
	}

	// Verify mock names
	mockNames := make(map[string]bool)
	for _, mock := range mocks {
		mockNames[mock.Name] = true
	}

	expectedNames := []string{"Test Mock 1", "Test Mock 2", "Nested Mock", "Mock with defaults"}
	for _, name := range expectedNames {
		if !mockNames[name] {
			t.Errorf("Expected mock '%s' not found", name)
		}
	}
}

func TestLoaderDefaults(t *testing.T) {
	testDir := "testdata"
	loader := NewLoader(testDir)

	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	mocks := loader.GetMocks()

	// Find the mock with defaults
	var defaultMock *struct {
		Name        string
		StatusCode  int
		Priority    int
	}

	for _, mock := range mocks {
		if mock.Name == "Mock with defaults" {
			defaultMock = &struct {
				Name        string
				StatusCode  int
				Priority    int
			}{
				Name:       mock.Name,
				StatusCode: mock.Response.StatusCode,
				Priority:   mock.Priority,
			}
			break
		}
	}

	if defaultMock == nil {
		t.Fatal("Mock with defaults not found")
	}

	// Check that default status code is set to 200
	if defaultMock.StatusCode != 200 {
		t.Errorf("Expected default status code 200, got %d", defaultMock.StatusCode)
	}
}

func TestLoaderNonexistentDirectory(t *testing.T) {
	loader := NewLoader("nonexistent-directory")

	err := loader.LoadAll()
	// Should not error - just returns with no mocks
	if err != nil {
		t.Errorf("Expected no error for nonexistent directory, got: %v", err)
	}

	mocks := loader.GetMocks()
	if len(mocks) != 0 {
		t.Errorf("Expected 0 mocks for nonexistent directory, got %d", len(mocks))
	}
}

func TestLoaderGetMocksThreadSafe(t *testing.T) {
	testDir := "testdata"
	loader := NewLoader(testDir)

	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	// Call GetMocks multiple times concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			mocks := loader.GetMocks()
			if len(mocks) == 0 {
				t.Error("Expected mocks, got empty slice")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLoaderReload(t *testing.T) {
	// Create a temporary directory for this test
	tempDir, err := os.MkdirTemp("", "loader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	// Create initial mock file
	mockFile := filepath.Join(tempDir, "test.yaml")
	initialContent := `mocks:
  - name: "Initial Mock"
    request:
      uri: "/api/test"
      method: "GET"
    response:
      status_code: 200
`
	err = os.WriteFile(mockFile, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write initial mock file: %v", err)
	}

	loader := NewLoader(tempDir)

	// Initial load
	err = loader.LoadAll()
	if err != nil {
		t.Fatalf("Initial LoadAll failed: %v", err)
	}

	mocks := loader.GetMocks()
	if len(mocks) != 1 {
		t.Fatalf("Expected 1 mock initially, got %d", len(mocks))
	}
	if mocks[0].Name != "Initial Mock" {
		t.Errorf("Expected 'Initial Mock', got '%s'", mocks[0].Name)
	}

	// Update mock file
	updatedContent := `mocks:
  - name: "Updated Mock"
    request:
      uri: "/api/test"
      method: "GET"
    response:
      status_code: 200
  - name: "New Mock"
    request:
      uri: "/api/new"
      method: "POST"
    response:
      status_code: 201
`
	err = os.WriteFile(mockFile, []byte(updatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write updated mock file: %v", err)
	}

	// Reload
	err = loader.LoadAll()
	if err != nil {
		t.Fatalf("Reload LoadAll failed: %v", err)
	}

	mocks = loader.GetMocks()
	if len(mocks) != 2 {
		t.Fatalf("Expected 2 mocks after reload, got %d", len(mocks))
	}

	// Check that we have the new mocks
	mockNames := make(map[string]bool)
	for _, mock := range mocks {
		mockNames[mock.Name] = true
	}

	if !mockNames["Updated Mock"] {
		t.Error("Expected 'Updated Mock' after reload")
	}
	if !mockNames["New Mock"] {
		t.Error("Expected 'New Mock' after reload")
	}
	if mockNames["Initial Mock"] {
		t.Error("Did not expect 'Initial Mock' after reload")
	}
}

func TestLoaderIgnoresNonYAMLFiles(t *testing.T) {
	testDir := "testdata"
	loader := NewLoader(testDir)

	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	mocks := loader.GetMocks()

	// Make sure no mock was loaded from readme.txt
	for _, mock := range mocks {
		if mock.Request.URI == "/readme" {
			t.Error("Non-YAML file should not be loaded")
		}
	}
}

func TestLoaderNestedDirectories(t *testing.T) {
	testDir := "testdata"
	loader := NewLoader(testDir)

	err := loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	mocks := loader.GetMocks()

	// Check that nested mock was loaded
	found := false
	for _, mock := range mocks {
		if mock.Name == "Nested Mock" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Mock from nested directory was not loaded")
	}
}

func TestLoaderMultipleDirectories(t *testing.T) {
	// Create two temp directories with different mocks
	tempDir1, err := os.MkdirTemp("", "loader-test-1-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tempDir1) //nolint:errcheck // test cleanup

	tempDir2, err := os.MkdirTemp("", "loader-test-2-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tempDir2) //nolint:errcheck // test cleanup

	// Create mock file in first directory
	mockFile1 := filepath.Join(tempDir1, "mocks1.yaml")
	content1 := `mocks:
  - name: "Mock from Dir1"
    request:
      uri: "/api/dir1"
      method: "GET"
    response:
      status_code: 200
`
	err = os.WriteFile(mockFile1, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock file 1: %v", err)
	}

	// Create mock file in second directory
	mockFile2 := filepath.Join(tempDir2, "mocks2.yaml")
	content2 := `mocks:
  - name: "Mock from Dir2"
    request:
      uri: "/api/dir2"
      method: "POST"
    response:
      status_code: 201
  - name: "Another Mock from Dir2"
    request:
      uri: "/api/dir2/extra"
      method: "GET"
    response:
      status_code: 200
`
	err = os.WriteFile(mockFile2, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock file 2: %v", err)
	}

	// Create loader with multiple directories
	loader := NewLoader(tempDir1, tempDir2)

	err = loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

	mocks := loader.GetMocks()

	// Should have 3 mocks total (1 from dir1, 2 from dir2)
	if len(mocks) != 3 {
		t.Errorf("Expected 3 mocks from multiple directories, got %d", len(mocks))
	}

	// Verify all mocks are loaded
	mockNames := make(map[string]bool)
	for _, mock := range mocks {
		mockNames[mock.Name] = true
	}

	expectedNames := []string{"Mock from Dir1", "Mock from Dir2", "Another Mock from Dir2"}
	for _, name := range expectedNames {
		if !mockNames[name] {
			t.Errorf("Expected mock '%s' not found", name)
		}
	}
}

func TestLoaderMultipleDirectoriesWithNonexistent(t *testing.T) {
	// Create one temp directory
	tempDir, err := os.MkdirTemp("", "loader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	mockFile := filepath.Join(tempDir, "mocks.yaml")
	content := `mocks:
  - name: "Existing Mock"
    request:
      uri: "/api/test"
      method: "GET"
    response:
      status_code: 200
`
	err = os.WriteFile(mockFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write mock file: %v", err)
	}

	// Create loader with one existing and one nonexistent directory
	loader := NewLoader(tempDir, "/nonexistent/directory")

	err = loader.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll should not fail with nonexistent directory: %v", err)
	}

	mocks := loader.GetMocks()

	// Should have 1 mock from existing directory
	if len(mocks) != 1 {
		t.Errorf("Expected 1 mock, got %d", len(mocks))
	}

	if mocks[0].Name != "Existing Mock" {
		t.Errorf("Expected 'Existing Mock', got '%s'", mocks[0].Name)
	}
}

func TestLoaderEmptyDirectoryList(t *testing.T) {
	// Create loader with no directories
	loader := NewLoader()

	err := loader.LoadAll()
	if err != nil {
		t.Errorf("LoadAll should not fail with empty directory list: %v", err)
	}

	mocks := loader.GetMocks()
	if len(mocks) != 0 {
		t.Errorf("Expected 0 mocks with empty directory list, got %d", len(mocks))
	}
}
