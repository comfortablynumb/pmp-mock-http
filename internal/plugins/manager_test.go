package plugins

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractRepoName(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			repoURL:  "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			repoURL:  "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "SSH URL with .git",
			repoURL:  "git@github.com:user/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH URL without .git",
			repoURL:  "git@github.com:user/repo",
			expected: "repo",
		},
		{
			name:     "GitLab HTTPS URL",
			repoURL:  "https://gitlab.com/user/my-awesome-mocks.git",
			expected: "my-awesome-mocks",
		},
		{
			name:     "Nested path",
			repoURL:  "https://github.com/org/team/project.git",
			expected: "project",
		},
		{
			name:     "Empty URL",
			repoURL:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepoName(tt.repoURL)
			if result != tt.expected {
				t.Errorf("extractRepoName(%q) = %q, want %q", tt.repoURL, result, tt.expected)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	repos := []string{"https://github.com/user/repo1.git", "https://github.com/user/repo2.git"}
	manager := NewManager("/tmp/plugins", repos)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.pluginsDir != "/tmp/plugins" {
		t.Errorf("Expected pluginsDir to be /tmp/plugins, got %s", manager.pluginsDir)
	}

	if len(manager.repos) != 2 {
		t.Errorf("Expected 2 repos, got %d", len(manager.repos))
	}

	if manager.gitClient == nil {
		t.Error("Expected gitClient to be initialized")
	}
}

func TestNewManagerWithGitClient(t *testing.T) {
	mockGit := NewMockGitClient()
	repos := []string{"https://github.com/user/repo.git"}
	manager := NewManagerWithGitClient("/tmp/plugins", repos, mockGit)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.gitClient != mockGit {
		t.Error("Expected gitClient to be the provided mock")
	}
}

func TestSetupPluginsEmpty(t *testing.T) {
	mockGit := NewMockGitClient()
	manager := NewManagerWithGitClient("/tmp/plugins", []string{}, mockGit)

	dirs, err := manager.SetupPlugins()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(dirs) != 0 {
		t.Errorf("Expected 0 directories, got %d", len(dirs))
	}

	if mockGit.GetCloneCallCount() != 0 {
		t.Error("Expected Clone not to be called")
	}
}

func TestSetupPluginsCloneNew(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")

	mockGit := NewMockGitClient()
	repos := []string{"https://github.com/user/test-repo.git"}
	manager := NewManagerWithGitClient(pluginsDir, repos, mockGit)

	dirs, err := manager.SetupPlugins()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(dirs) != 1 {
		t.Fatalf("Expected 1 directory, got %d", len(dirs))
	}

	expectedPath := filepath.Join(pluginsDir, "test-repo")
	if dirs[0] != expectedPath {
		t.Errorf("Expected directory %s, got %s", expectedPath, dirs[0])
	}

	if mockGit.GetCloneCallCount() != 1 {
		t.Errorf("Expected Clone to be called once, got %d", mockGit.GetCloneCallCount())
	}

	if err := mockGit.AssertCloneCalled("https://github.com/user/test-repo.git", expectedPath); err != nil {
		t.Error(err)
	}

	if mockGit.GetPullCallCount() != 0 {
		t.Error("Expected Pull not to be called for new repository")
	}
}

func TestSetupPluginsUpdateExisting(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")

	// Create the plugin directory to simulate existing repository
	repoPath := filepath.Join(pluginsDir, "test-repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}

	mockGit := NewMockGitClient()
	repos := []string{"https://github.com/user/test-repo.git"}
	manager := NewManagerWithGitClient(pluginsDir, repos, mockGit)

	dirs, err := manager.SetupPlugins()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(dirs) != 1 {
		t.Fatalf("Expected 1 directory, got %d", len(dirs))
	}

	if mockGit.GetCloneCallCount() != 0 {
		t.Error("Expected Clone not to be called for existing repository")
	}

	if mockGit.GetPullCallCount() != 1 {
		t.Errorf("Expected Pull to be called once, got %d", mockGit.GetPullCallCount())
	}

	if err := mockGit.AssertPullCalled(repoPath); err != nil {
		t.Error(err)
	}
}

func TestSetupPluginsMultipleRepos(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")

	mockGit := NewMockGitClient()
	repos := []string{
		"https://github.com/user/repo1.git",
		"https://github.com/user/repo2.git",
		"https://github.com/user/repo3.git",
	}
	manager := NewManagerWithGitClient(pluginsDir, repos, mockGit)

	dirs, err := manager.SetupPlugins()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(dirs) != 3 {
		t.Errorf("Expected 3 directories, got %d", len(dirs))
	}

	if mockGit.GetCloneCallCount() != 3 {
		t.Errorf("Expected Clone to be called 3 times, got %d", mockGit.GetCloneCallCount())
	}

	// Verify all repos were cloned
	for i, repo := range repos {
		repoName := extractRepoName(repo)
		expectedPath := filepath.Join(pluginsDir, repoName)
		if err := mockGit.AssertCloneCalled(repo, expectedPath); err != nil {
			t.Errorf("Repo %d: %v", i, err)
		}
	}
}

func TestSetupPluginsCloneError(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")

	mockGit := NewMockGitClient()
	mockGit.SetCloneError(os.ErrPermission)

	repos := []string{"https://github.com/user/repo.git"}
	manager := NewManagerWithGitClient(pluginsDir, repos, mockGit)

	dirs, err := manager.SetupPlugins()

	// Should not return error, just log warning
	if err != nil {
		t.Errorf("Expected no error (warnings are logged), got %v", err)
	}

	// Should return empty list because clone failed
	if len(dirs) != 0 {
		t.Errorf("Expected 0 directories due to clone error, got %d", len(dirs))
	}
}

func TestSetupPluginsPullError(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")

	// Create existing repository
	repoPath := filepath.Join(pluginsDir, "repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}

	mockGit := NewMockGitClient()
	mockGit.SetPullError(os.ErrPermission)

	repos := []string{"https://github.com/user/repo.git"}
	manager := NewManagerWithGitClient(pluginsDir, repos, mockGit)

	dirs, err := manager.SetupPlugins()

	// Should not return error, just log warning
	if err != nil {
		t.Errorf("Expected no error (warnings are logged), got %v", err)
	}

	// Should still return the directory even if pull failed
	if len(dirs) != 1 {
		t.Errorf("Expected 1 directory (continue with existing version), got %d", len(dirs))
	}
}

func TestSetupPluginsInvalidURL(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")

	mockGit := NewMockGitClient()
	// Set clone error to simulate failure with invalid URL
	mockGit.SetCloneError(os.ErrInvalid)

	repos := []string{"not-a-valid-url"}
	manager := NewManagerWithGitClient(pluginsDir, repos, mockGit)

	dirs, err := manager.SetupPlugins()

	// Should not return error, just log warning
	if err != nil {
		t.Errorf("Expected no error (warnings are logged), got %v", err)
	}

	// Should return 0 directories because clone failed
	if len(dirs) != 0 {
		t.Errorf("Expected 0 directories due to clone failure, got %d", len(dirs))
	}

	// Verify clone was attempted (extractRepoName returns "not-a-valid-url")
	if mockGit.GetCloneCallCount() != 1 {
		t.Errorf("Expected Clone to be called once, got %d", mockGit.GetCloneCallCount())
	}
}
