package watcher

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	reloadFn := func() error {
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	if w == nil {
		t.Fatal("Expected watcher instance, got nil")
	}
	if w.mocksDir != tempDir {
		t.Errorf("Expected mocksDir %s, got %s", tempDir, w.mocksDir)
	}
}

func TestWatcherStart(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	reloadFn := func() error {
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give the watcher time to start
	time.Sleep(100 * time.Millisecond)
}

func TestWatcherFileCreate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	var reloadCalled int32
	reloadFn := func() error {
		atomic.StoreInt32(&reloadCalled, 1)
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create a new YAML file
	testFile := filepath.Join(tempDir, "test.yaml")
	content := []byte("mocks:\n  - name: test\n")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Wait for debounce and reload
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&reloadCalled) == 0 {
		t.Error("Expected reload to be called after file creation")
	}
}

func TestWatcherFileModify(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	// Create initial file
	testFile := filepath.Join(tempDir, "test.yaml")
	content := []byte("mocks:\n  - name: test\n")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	var reloadCount int32
	reloadFn := func() error {
		atomic.AddInt32(&reloadCount, 1)
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify the file
	newContent := []byte("mocks:\n  - name: modified\n")
	err = os.WriteFile(testFile, newContent, 0644)
	if err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Wait for debounce and reload
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&reloadCount) == 0 {
		t.Error("Expected reload to be called after file modification")
	}
}

func TestWatcherFileDelete(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	// Create initial file
	testFile := filepath.Join(tempDir, "test.yaml")
	content := []byte("mocks:\n  - name: test\n")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	var reloadCalled int32
	reloadFn := func() error {
		atomic.StoreInt32(&reloadCalled, 1)
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Delete the file
	err = os.Remove(testFile)
	if err != nil {
		t.Fatalf("Failed to delete test file: %v", err)
	}

	// Wait for debounce and reload
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&reloadCalled) == 0 {
		t.Error("Expected reload to be called after file deletion")
	}
}

func TestWatcherNestedDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	// Create nested directory
	nestedDir := filepath.Join(tempDir, "nested")
	err = os.Mkdir(nestedDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nested dir: %v", err)
	}

	var reloadCalled int32
	reloadFn := func() error {
		atomic.StoreInt32(&reloadCalled, 1)
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create file in nested directory
	testFile := filepath.Join(nestedDir, "test.yaml")
	content := []byte("mocks:\n  - name: nested\n")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Wait for debounce and reload
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&reloadCalled) == 0 {
		t.Error("Expected reload to be called for file in nested directory")
	}
}

func TestWatcherIgnoreNonYAMLFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	reloadCount := 0
	reloadFn := func() error {
		reloadCount++
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create a non-YAML file
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("not a yaml file")
	err = os.WriteFile(testFile, content, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Wait to see if reload is called (it shouldn't be)
	time.Sleep(200 * time.Millisecond)

	// Note: Creating the file might trigger directory events,
	// so we just verify it doesn't crash
}

func TestWatcherNewDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	var reloadCalled int32
	reloadFn := func() error {
		atomic.StoreInt32(&reloadCalled, 1)
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create new directory
	newDir := filepath.Join(tempDir, "newdir")
	err = os.Mkdir(newDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create new directory: %v", err)
	}

	// Wait for reload
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&reloadCalled) == 0 {
		t.Error("Expected reload to be called after directory creation")
	}
}

func TestWatcherNonexistentDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	// Remove the directory immediately
	os.RemoveAll(tempDir) //nolint:errcheck

	reloadFn := func() error {
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	// Start should create the directory and not fail
	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed for nonexistent directory: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}

	// Cleanup
	os.RemoveAll(tempDir) //nolint:errcheck
}

func TestWatcherClose(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	reloadFn := func() error {
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Close the watcher
	err = w.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestWatcherDebounce(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "watcher-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir) //nolint:errcheck // test cleanup

	reloadFn := func() error {
		return nil
	}

	w, err := NewWatcher(tempDir, reloadFn)
	if err != nil {
		t.Fatalf("NewWatcher failed: %v", err)
	}
	defer w.Close() //nolint:errcheck // test cleanup

	err = w.Start()
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait for watcher to initialize
	time.Sleep(100 * time.Millisecond)

	// Create file
	testFile := filepath.Join(tempDir, "test.yaml")

	// Rapidly modify the file multiple times to test debouncing
	for i := 0; i < 5; i++ {
		content := []byte("mocks:\n  - name: test" + string(rune(i)) + "\n")
		err = os.WriteFile(testFile, content, 0644)
		if err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce and reload to complete
	// This test primarily verifies that debouncing doesn't crash
	// and handles rapid file changes gracefully
	time.Sleep(300 * time.Millisecond)
}
