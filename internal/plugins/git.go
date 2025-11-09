package plugins

import (
	"fmt"
	"os"
	"os/exec"
)

// GitClient defines the interface for git operations
type GitClient interface {
	Clone(repoURL, destPath string) error
	Pull(repoPath string) error
}

// RealGitClient implements GitClient using actual git commands
type RealGitClient struct{}

// NewRealGitClient creates a new real git client
func NewRealGitClient() *RealGitClient {
	return &RealGitClient{}
}

// Clone clones a git repository to the specified path
func (g *RealGitClient) Clone(repoURL, destPath string) error {
	cmd := exec.Command("git", "clone", repoURL, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

// Pull updates an existing git repository
func (g *RealGitClient) Pull(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	return nil
}
