package watcher

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors the mocks directory for changes
type Watcher struct {
	mocksDir string
	watcher  *fsnotify.Watcher
	reloadFn func() error
}

// NewWatcher creates a new file watcher
func NewWatcher(mocksDir string, reloadFn func() error) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	w := &Watcher{
		mocksDir: mocksDir,
		watcher:  watcher,
		reloadFn: reloadFn,
	}

	return w, nil
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	// Add the mocks directory and all subdirectories to the watcher
	if err := w.addDirectories(); err != nil {
		return err
	}

	// Start watching in a goroutine
	go w.watch()

	log.Printf("Started watching %s for changes\n", w.mocksDir)
	return nil
}

// addDirectories recursively adds all directories to the watcher
func (w *Watcher) addDirectories() error {
	return filepath.Walk(w.mocksDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// If the directory doesn't exist yet, create it
			if os.IsNotExist(err) {
				if err := os.MkdirAll(w.mocksDir, 0755); err != nil {
					return fmt.Errorf("failed to create mocks directory: %w", err)
				}
				return w.watcher.Add(w.mocksDir)
			}
			return err
		}

		// Only watch directories
		if info.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch directory %s: %w", path, err)
			}
		}

		return nil
	})
}

// watch monitors for file system events
func (w *Watcher) watch() {
	// Debounce timer to avoid multiple reloads for rapid changes
	var debounceTimer *time.Timer
	debounceDuration := 100 * time.Millisecond

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Check if it's a YAML file or a directory
			isYAML := isYAMLFile(event.Name)
			isDir := isDirectory(event.Name)

			// Handle different event types
			if event.Op&fsnotify.Write == fsnotify.Write && isYAML {
				log.Printf("Modified file: %s\n", event.Name)
				w.scheduleReload(&debounceTimer, debounceDuration)
			} else if event.Op&fsnotify.Create == fsnotify.Create {
				if isDir {
					// New directory created, start watching it
					log.Printf("New directory created: %s\n", event.Name)
					w.watcher.Add(event.Name)
					w.scheduleReload(&debounceTimer, debounceDuration)
				} else if isYAML {
					log.Printf("New file: %s\n", event.Name)
					w.scheduleReload(&debounceTimer, debounceDuration)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				if isYAML {
					log.Printf("Deleted file: %s\n", event.Name)
					w.scheduleReload(&debounceTimer, debounceDuration)
				} else if isDir {
					log.Printf("Deleted directory: %s\n", event.Name)
					w.scheduleReload(&debounceTimer, debounceDuration)
				}
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				log.Printf("Renamed: %s\n", event.Name)
				w.scheduleReload(&debounceTimer, debounceDuration)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v\n", err)
		}
	}
}

// scheduleReload schedules a reload with debouncing
func (w *Watcher) scheduleReload(timer **time.Timer, duration time.Duration) {
	if *timer != nil {
		(*timer).Stop()
	}

	*timer = time.AfterFunc(duration, func() {
		log.Println("Reloading mocks...")
		if err := w.reloadFn(); err != nil {
			log.Printf("Failed to reload mocks: %v\n", err)
		} else {
			log.Println("Mocks reloaded successfully")
		}
	})
}

// Close stops the watcher
func (w *Watcher) Close() error {
	return w.watcher.Close()
}

// isYAMLFile checks if a file has a YAML extension
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}

// isDirectory checks if a path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
