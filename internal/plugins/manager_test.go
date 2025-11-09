package plugins

import (
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
}
