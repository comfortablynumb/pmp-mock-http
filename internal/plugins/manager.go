package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles cloning and managing plugin repositories
type Manager struct {
	pluginsDir string
	repos      []string
	gitClient  GitClient
}

// NewManager creates a new plugin manager with a real git client
func NewManager(pluginsDir string, repos []string) *Manager {
	return NewManagerWithGitClient(pluginsDir, repos, NewRealGitClient())
}

// NewManagerWithGitClient creates a new plugin manager with a custom git client
func NewManagerWithGitClient(pluginsDir string, repos []string, gitClient GitClient) *Manager {
	return &Manager{
		pluginsDir: pluginsDir,
		repos:      repos,
		gitClient:  gitClient,
	}
}

// SetupPlugins clones all plugin repositories and returns directories to watch
func (m *Manager) SetupPlugins() ([]string, error) {
	if len(m.repos) == 0 {
		return []string{}, nil
	}

	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugins directory: %w", err)
	}

	var pluginDirs []string

	for _, repoURL := range m.repos {
		// Extract repository name from URL
		repoName := extractRepoName(repoURL)
		if repoName == "" {
			log.Printf("Warning: invalid repository URL: %s\n", repoURL)
			continue
		}

		pluginPath := filepath.Join(m.pluginsDir, repoName)

		// Check if plugin already exists
		if _, err := os.Stat(pluginPath); err == nil {
			// Plugin exists, update it
			log.Printf("Plugin '%s' already exists, updating...\n", repoName)
			if err := m.updateRepo(pluginPath); err != nil {
				log.Printf("Warning: failed to update plugin '%s': %v\n", repoName, err)
				// Continue using existing version
			}
		} else {
			// Clone the repository
			log.Printf("Cloning plugin from %s...\n", repoURL)
			if err := m.cloneRepo(repoURL, pluginPath); err != nil {
				log.Printf("Warning: failed to clone plugin '%s': %v\n", repoName, err)
				continue
			}
			log.Printf("Plugin '%s' cloned successfully\n", repoName)
		}

		pluginDirs = append(pluginDirs, pluginPath)
	}

	return pluginDirs, nil
}

// cloneRepo clones a git repository to the specified path
func (m *Manager) cloneRepo(repoURL, destPath string) error {
	return m.gitClient.Clone(repoURL, destPath)
}

// updateRepo updates an existing git repository
func (m *Manager) updateRepo(repoPath string) error {
	return m.gitClient.Pull(repoPath)
}

// extractRepoName extracts the repository name from a git URL
// Examples:
//   - https://github.com/user/repo.git -> repo
//   - https://github.com/user/repo -> repo
//   - git@github.com:user/repo.git -> repo
func extractRepoName(repoURL string) string {
	// Remove .git suffix if present
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Handle SSH format (git@github.com:user/repo)
	if strings.Contains(repoURL, ":") && strings.Contains(repoURL, "@") {
		parts := strings.Split(repoURL, ":")
		if len(parts) >= 2 {
			repoURL = parts[len(parts)-1]
		}
	}

	// Extract the last part of the path
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}
