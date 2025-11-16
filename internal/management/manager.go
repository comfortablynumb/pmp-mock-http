package management

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"gopkg.in/yaml.v3"
)

// Manager handles mock lifecycle and versioning
type Manager struct {
	mocks     map[string]*ManagedMock
	versions  map[string][]MockVersion
	templates map[string]*MockTemplate
	mu        sync.RWMutex
	nextID    int
}

// NewManager creates a new mock manager
func NewManager() *Manager {
	return &Manager{
		mocks:     make(map[string]*ManagedMock),
		versions:  make(map[string][]MockVersion),
		templates: make(map[string]*MockTemplate),
		nextID:    1,
	}
}

// CreateMock creates a new managed mock
func (m *Manager) CreateMock(req CreateMockRequest) (*ManagedMock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.generateID()

	metadata := MockMetadata{
		ID:          id,
		Name:        req.Mock.Name,
		Version:     1,
		Tags:        req.Tags,
		Labels:      req.Labels,
		Description: req.Description,
		Author:      req.Author,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Source:      "api",
		Template:    req.Template,
	}

	managed := &ManagedMock{
		Metadata: metadata,
		Mock:     req.Mock,
	}

	m.mocks[id] = managed

	// Create initial version
	m.versions[id] = []MockVersion{
		{
			Version:   1,
			Mock:      req.Mock,
			ChangedBy: req.Author,
			Timestamp: time.Now(),
			Comment:   "Initial version",
		},
	}

	return managed, nil
}

// GetMock retrieves a mock by ID
func (m *Manager) GetMock(id string) (*ManagedMock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	mock, exists := m.mocks[id]
	if !exists {
		return nil, fmt.Errorf("mock not found: %s", id)
	}

	return mock, nil
}

// UpdateMock updates an existing mock
func (m *Manager) UpdateMock(id string, req UpdateMockRequest) (*ManagedMock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	managed, exists := m.mocks[id]
	if !exists {
		return nil, fmt.Errorf("mock not found: %s", id)
	}

	// Create new version
	newVersion := managed.Metadata.Version + 1

	// Update mock if provided
	if req.Mock != nil {
		managed.Mock = *req.Mock
	}

	// Update tags if provided
	if req.Tags != nil {
		managed.Metadata.Tags = *req.Tags
	}

	// Update labels if provided
	if req.Labels != nil {
		managed.Metadata.Labels = *req.Labels
	}

	// Update description if provided
	if req.Description != nil {
		managed.Metadata.Description = *req.Description
	}

	// Update metadata
	managed.Metadata.Version = newVersion
	managed.Metadata.UpdatedAt = time.Now()

	// Add version history
	m.versions[id] = append(m.versions[id], MockVersion{
		Version:   newVersion,
		Mock:      managed.Mock,
		ChangedBy: req.Author,
		Timestamp: time.Now(),
		Comment:   req.Comment,
	})

	return managed, nil
}

// DeleteMock deletes a mock
func (m *Manager) DeleteMock(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.mocks[id]; !exists {
		return fmt.Errorf("mock not found: %s", id)
	}

	delete(m.mocks, id)
	delete(m.versions, id)

	return nil
}

// ListMocks lists all mocks, optionally filtered
func (m *Manager) ListMocks(filter *MockFilter) ([]*ManagedMock, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*ManagedMock

	for _, mock := range m.mocks {
		if filter == nil || m.matchesFilter(mock, filter) {
			result = append(result, mock)
		}
	}

	return result, nil
}

// GetVersionHistory retrieves version history for a mock
func (m *Manager) GetVersionHistory(id string) ([]MockVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions, exists := m.versions[id]
	if !exists {
		return nil, fmt.Errorf("mock not found: %s", id)
	}

	return versions, nil
}

// GetVersion retrieves a specific version of a mock
func (m *Manager) GetVersion(id string, version int) (*MockVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions, exists := m.versions[id]
	if !exists {
		return nil, fmt.Errorf("mock not found: %s", id)
	}

	for _, v := range versions {
		if v.Version == version {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("version not found: %d", version)
}

// RollbackToVersion rolls back a mock to a specific version
func (m *Manager) RollbackToVersion(id string, version int, author string) (*ManagedMock, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	managed, exists := m.mocks[id]
	if !exists {
		return nil, fmt.Errorf("mock not found: %s", id)
	}

	versions := m.versions[id]
	var targetVersion *MockVersion

	for _, v := range versions {
		if v.Version == version {
			targetVersion = &v
			break
		}
	}

	if targetVersion == nil {
		return nil, fmt.Errorf("version not found: %d", version)
	}

	// Create new version with rolled back content
	newVersion := managed.Metadata.Version + 1
	managed.Mock = targetVersion.Mock
	managed.Metadata.Version = newVersion
	managed.Metadata.UpdatedAt = time.Now()

	m.versions[id] = append(m.versions[id], MockVersion{
		Version:   newVersion,
		Mock:      targetVersion.Mock,
		ChangedBy: author,
		Timestamp: time.Now(),
		Comment:   fmt.Sprintf("Rolled back to version %d", version),
	})

	return managed, nil
}

// CreateTemplate creates a new mock template
func (m *Manager) CreateTemplate(req CreateTemplateRequest) (*MockTemplate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.generateID()

	template := &MockTemplate{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Tags:        req.Tags,
		Mock:        req.Mock,
		Parameters:  req.Parameters,
		Variables:   req.Variables,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.templates[id] = template

	return template, nil
}

// GetTemplate retrieves a template by ID
func (m *Manager) GetTemplate(id string) (*MockTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	template, exists := m.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}

	return template, nil
}

// ListTemplates lists all templates, optionally by category
func (m *Manager) ListTemplates(category string) ([]*MockTemplate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*MockTemplate

	for _, template := range m.templates {
		if category == "" || template.Category == category {
			result = append(result, template)
		}
	}

	return result, nil
}

// InstantiateTemplate creates a mock from a template
func (m *Manager) InstantiateTemplate(req InstantiateTemplateRequest) (*ManagedMock, error) {
	template, err := m.GetTemplate(req.TemplateID)
	if err != nil {
		return nil, err
	}

	// Create mock from template
	mock := template.Mock

	// TODO: Apply parameters to mock using template engine

	createReq := CreateMockRequest{
		Mock:     mock,
		Tags:     append(req.Tags, template.Tags...),
		Labels:   req.Labels,
		Template: req.TemplateID,
	}

	return m.CreateMock(createReq)
}

// GetStats returns statistics about mocks
func (m *Manager) GetStats() MockStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := MockStats{
		TotalMocks:      len(m.mocks),
		MocksBySource:   make(map[string]int),
		MocksByTemplate: make(map[string]int),
		MocksByTag:      make(map[string]int),
		Templates:       len(m.templates),
	}

	for _, mock := range m.mocks {
		stats.MocksBySource[mock.Metadata.Source]++
		if mock.Metadata.Template != "" {
			stats.MocksByTemplate[mock.Metadata.Template]++
		}
		for _, tag := range mock.Metadata.Tags {
			stats.MocksByTag[tag]++
		}
	}

	for _, versions := range m.versions {
		stats.TotalVersions += len(versions)
	}

	return stats
}

// Export exports mocks in the specified format
func (m *Manager) Export(req ExportRequest) (string, error) {
	mocks, err := m.ListMocks(req.Filter)
	if err != nil {
		return "", err
	}

	switch req.Format {
	case ExportFormatYAML:
		data, err := yaml.Marshal(mocks)
		return string(data), err

	case ExportFormatJSON:
		data, err := json.MarshalIndent(mocks, "", "  ")
		return string(data), err

	case ExportFormatOpenAPI:
		// TODO: Convert to OpenAPI format
		return "", fmt.Errorf("OpenAPI export not yet implemented")

	default:
		return "", fmt.Errorf("unsupported export format: %s", req.Format)
	}
}

// Import imports mocks from the specified format
func (m *Manager) Import(req ImportRequest) (int, error) {
	var mocks []ManagedMock

	switch req.Format {
	case ExportFormatYAML:
		if err := yaml.Unmarshal([]byte(req.Data), &mocks); err != nil {
			return 0, err
		}

	case ExportFormatJSON:
		if err := json.Unmarshal([]byte(req.Data), &mocks); err != nil {
			return 0, err
		}

	default:
		return 0, fmt.Errorf("unsupported import format: %s", req.Format)
	}

	count := 0
	for _, mock := range mocks {
		createReq := CreateMockRequest{
			Mock:   mock.Mock,
			Tags:   append(mock.Metadata.Tags, req.Tags...),
			Labels: mock.Metadata.Labels,
		}

		if req.Source != "" {
			createReq.Labels["source"] = req.Source
		}

		if _, err := m.CreateMock(createReq); err == nil {
			count++
		}
	}

	return count, nil
}

// matchesFilter checks if a mock matches the filter criteria
func (m *Manager) matchesFilter(mock *ManagedMock, filter *MockFilter) bool {
	// Filter by tags
	if len(filter.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filter.Tags {
			for _, mockTag := range mock.Metadata.Tags {
				if mockTag == filterTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	// Filter by labels
	if len(filter.Labels) > 0 {
		for key, value := range filter.Labels {
			if mockValue, exists := mock.Metadata.Labels[key]; !exists || mockValue != value {
				return false
			}
		}
	}

	// Filter by source
	if filter.Source != "" && mock.Metadata.Source != filter.Source {
		return false
	}

	// Filter by template
	if filter.Template != "" && mock.Metadata.Template != filter.Template {
		return false
	}

	// Filter by search term
	if filter.Search != "" {
		searchLower := strings.ToLower(filter.Search)
		if !strings.Contains(strings.ToLower(mock.Metadata.Name), searchLower) &&
			!strings.Contains(strings.ToLower(mock.Metadata.Description), searchLower) {
			return false
		}
	}

	// Filter by created date
	if filter.CreatedAfter != nil && mock.Metadata.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}
	if filter.CreatedBefore != nil && mock.Metadata.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	// Filter by updated date
	if filter.UpdatedAfter != nil && mock.Metadata.UpdatedAt.Before(*filter.UpdatedAfter) {
		return false
	}
	if filter.UpdatedBefore != nil && mock.Metadata.UpdatedAt.After(*filter.UpdatedBefore) {
		return false
	}

	return true
}

// generateID generates a unique ID
func (m *Manager) generateID() string {
	id := fmt.Sprintf("mock-%d", m.nextID)
	m.nextID++
	return id
}

// GetAllMocks returns all mocks for integration with the server
func (m *Manager) GetAllMocks() []models.Mock {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]models.Mock, 0, len(m.mocks))
	for _, managed := range m.mocks {
		result = append(result, managed.Mock)
	}
	return result
}
