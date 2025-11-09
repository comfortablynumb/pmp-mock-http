package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"gopkg.in/yaml.v3"
)

// Loader manages loading mock specifications from YAML files
type Loader struct {
	mocksDirs []string
	mocks     []models.Mock
	mu        sync.RWMutex
}

// NewLoader creates a new mock loader with one or more directories
func NewLoader(mocksDirs ...string) *Loader {
	return &Loader{
		mocksDirs: mocksDirs,
		mocks:     make([]models.Mock, 0),
	}
}

// LoadAll loads all mock files from all configured directories and subdirectories
func (l *Loader) LoadAll() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Clear existing mocks
	l.mocks = make([]models.Mock, 0)

	// Walk through each configured directory
	for _, mocksDir := range l.mocksDirs {
		err := filepath.Walk(mocksDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// If the directory doesn't exist, just return (it will be created later)
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Only process YAML files
			if !isYAMLFile(path) {
				return nil
			}

			// Load the mock file
			if err := l.loadFile(path); err != nil {
				fmt.Printf("Warning: failed to load mock file %s: %v\n", path, err)
				// Continue processing other files even if one fails
				return nil
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("failed to walk mocks directory %s: %w", mocksDir, err)
		}
	}

	fmt.Printf("Loaded %d total mock(s) from %d directory(ies)\n", len(l.mocks), len(l.mocksDirs))
	return nil
}

// loadFile loads a single YAML mock file
func (l *Loader) loadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var spec models.MockSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Add all mocks from this file
	for _, mock := range spec.Mocks {
		// Set default values if not specified
		if mock.Response.StatusCode == 0 {
			mock.Response.StatusCode = 200
		}
		l.mocks = append(l.mocks, mock)
	}

	fmt.Printf("Loaded %d mock(s) from %s\n", len(spec.Mocks), path)
	return nil
}

// GetMocks returns a copy of all loaded mocks
func (l *Loader) GetMocks() []models.Mock {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to avoid race conditions
	mocks := make([]models.Mock, len(l.mocks))
	copy(mocks, l.mocks)
	return mocks
}

// isYAMLFile checks if a file has a YAML extension
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
