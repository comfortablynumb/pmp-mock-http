package plugins

import "fmt"

// MockGitClient is a mock implementation of GitClient for testing
type MockGitClient struct {
	CloneCalls []CloneCall
	PullCalls  []PullCall
	CloneError error
	PullError  error
}

// CloneCall records a call to Clone
type CloneCall struct {
	RepoURL  string
	DestPath string
}

// PullCall records a call to Pull
type PullCall struct {
	RepoPath string
}

// NewMockGitClient creates a new mock git client
func NewMockGitClient() *MockGitClient {
	return &MockGitClient{
		CloneCalls: make([]CloneCall, 0),
		PullCalls:  make([]PullCall, 0),
	}
}

// Clone records the call and returns the configured error
func (m *MockGitClient) Clone(repoURL, destPath string) error {
	m.CloneCalls = append(m.CloneCalls, CloneCall{
		RepoURL:  repoURL,
		DestPath: destPath,
	})
	if m.CloneError != nil {
		return m.CloneError
	}
	return nil
}

// Pull records the call and returns the configured error
func (m *MockGitClient) Pull(repoPath string) error {
	m.PullCalls = append(m.PullCalls, PullCall{
		RepoPath: repoPath,
	})
	if m.PullError != nil {
		return m.PullError
	}
	return nil
}

// SetCloneError sets the error to return from Clone
func (m *MockGitClient) SetCloneError(err error) {
	m.CloneError = err
}

// SetPullError sets the error to return from Pull
func (m *MockGitClient) SetPullError(err error) {
	m.PullError = err
}

// GetCloneCallCount returns the number of times Clone was called
func (m *MockGitClient) GetCloneCallCount() int {
	return len(m.CloneCalls)
}

// GetPullCallCount returns the number of times Pull was called
func (m *MockGitClient) GetPullCallCount() int {
	return len(m.PullCalls)
}

// AssertCloneCalled verifies Clone was called with expected parameters
func (m *MockGitClient) AssertCloneCalled(repoURL, destPath string) error {
	for _, call := range m.CloneCalls {
		if call.RepoURL == repoURL && call.DestPath == destPath {
			return nil
		}
	}
	return fmt.Errorf("Clone not called with repoURL=%s, destPath=%s", repoURL, destPath)
}

// AssertPullCalled verifies Pull was called with expected parameters
func (m *MockGitClient) AssertPullCalled(repoPath string) error {
	for _, call := range m.PullCalls {
		if call.RepoPath == repoPath {
			return nil
		}
	}
	return fmt.Errorf("Pull not called with repoPath=%s", repoPath)
}
