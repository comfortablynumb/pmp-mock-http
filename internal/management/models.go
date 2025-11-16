package management

import (
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
)

// MockMetadata represents metadata for a mock
type MockMetadata struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Version     int               `json:"version" yaml:"version"`
	Tags        []string          `json:"tags" yaml:"tags"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Description string            `json:"description" yaml:"description"`
	Author      string            `json:"author" yaml:"author"`
	CreatedAt   time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" yaml:"updated_at"`
	Source      string            `json:"source" yaml:"source"` // file, api, template
	Template    string            `json:"template,omitempty" yaml:"template,omitempty"`
}

// ManagedMock represents a mock with management metadata
type ManagedMock struct {
	Metadata MockMetadata `json:"metadata" yaml:"metadata"`
	Mock     models.Mock  `json:"mock" yaml:"mock"`
}

// MockVersion represents a version of a mock
type MockVersion struct {
	Version   int         `json:"version" yaml:"version"`
	Mock      models.Mock `json:"mock" yaml:"mock"`
	ChangedBy string      `json:"changed_by" yaml:"changed_by"`
	Timestamp time.Time   `json:"timestamp" yaml:"timestamp"`
	Comment   string      `json:"comment" yaml:"comment"`
}

// MockTemplate represents a reusable mock template
type MockTemplate struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Category    string                 `json:"category" yaml:"category"` // API provider (stripe, github, aws, etc.)
	Tags        []string               `json:"tags" yaml:"tags"`
	Mock        models.Mock            `json:"mock" yaml:"mock"`
	Parameters  []TemplateParameter    `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty" yaml:"variables,omitempty"`
	CreatedAt   time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" yaml:"updated_at"`
}

// TemplateParameter represents a configurable parameter in a template
type TemplateParameter struct {
	Name         string      `json:"name" yaml:"name"`
	Type         string      `json:"type" yaml:"type"` // string, number, boolean, object
	Description  string      `json:"description" yaml:"description"`
	Required     bool        `json:"required" yaml:"required"`
	DefaultValue interface{} `json:"default_value,omitempty" yaml:"default_value,omitempty"`
}

// MockFilter represents filter criteria for searching mocks
type MockFilter struct {
	Tags       []string          `json:"tags,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	Source     string            `json:"source,omitempty"`
	Template   string            `json:"template,omitempty"`
	Search     string            `json:"search,omitempty"` // Search in name, description
	CreatedAfter  *time.Time     `json:"created_after,omitempty"`
	CreatedBefore *time.Time     `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time     `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time     `json:"updated_before,omitempty"`
}

// MockStats represents statistics about mocks
type MockStats struct {
	TotalMocks      int                `json:"total_mocks"`
	MocksBySource   map[string]int     `json:"mocks_by_source"`
	MocksByTemplate map[string]int     `json:"mocks_by_template"`
	MocksByTag      map[string]int     `json:"mocks_by_tag"`
	TotalVersions   int                `json:"total_versions"`
	Templates       int                `json:"templates"`
}

// CreateMockRequest represents a request to create a mock
type CreateMockRequest struct {
	Mock        models.Mock       `json:"mock"`
	Tags        []string          `json:"tags,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Description string            `json:"description,omitempty"`
	Author      string            `json:"author,omitempty"`
	Template    string            `json:"template,omitempty"`
}

// UpdateMockRequest represents a request to update a mock
type UpdateMockRequest struct {
	Mock        *models.Mock       `json:"mock,omitempty"`
	Tags        *[]string          `json:"tags,omitempty"`
	Labels      *map[string]string `json:"labels,omitempty"`
	Description *string            `json:"description,omitempty"`
	Comment     string             `json:"comment,omitempty"`
	Author      string             `json:"author,omitempty"`
}

// CreateTemplateRequest represents a request to create a template
type CreateTemplateRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags,omitempty"`
	Mock        models.Mock            `json:"mock"`
	Parameters  []TemplateParameter    `json:"parameters,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
}

// InstantiateTemplateRequest represents a request to create a mock from a template
type InstantiateTemplateRequest struct {
	TemplateID string                 `json:"template_id"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Tags       []string               `json:"tags,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
}

// ExportFormat represents the format for exporting mocks
type ExportFormat string

const (
	ExportFormatYAML    ExportFormat = "yaml"
	ExportFormatJSON    ExportFormat = "json"
	ExportFormatOpenAPI ExportFormat = "openapi"
)

// ImportRequest represents a request to import mocks
type ImportRequest struct {
	Format ExportFormat `json:"format"`
	Data   string       `json:"data"`
	Source string       `json:"source,omitempty"`
	Tags   []string     `json:"tags,omitempty"`
}

// ExportRequest represents a request to export mocks
type ExportRequest struct {
	Format ExportFormat `json:"format"`
	Filter *MockFilter  `json:"filter,omitempty"`
}

// Common template categories
const (
	TemplateStripe  = "stripe"
	TemplateGitHub  = "github"
	TemplateAWS     = "aws"
	TemplateTwilio  = "twilio"
	TemplateSlack   = "slack"
	TemplateOpenAI  = "openai"
	TemplateGoogle  = "google"
	TemplatePayPal  = "paypal"
)
