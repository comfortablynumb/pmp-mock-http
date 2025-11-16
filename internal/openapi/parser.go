package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"gopkg.in/yaml.v3"
)

// OpenAPISpec represents an OpenAPI 3.x specification
type OpenAPISpec struct {
	OpenAPI    string                       `json:"openapi" yaml:"openapi"`
	Info       Info                         `json:"info" yaml:"info"`
	Servers    []Server                     `json:"servers,omitempty" yaml:"servers,omitempty"`
	Paths      map[string]PathItem          `json:"paths" yaml:"paths"`
	Components *Components                  `json:"components,omitempty" yaml:"components,omitempty"`
}

// SwaggerSpec represents a Swagger 2.0 specification
type SwaggerSpec struct {
	Swagger     string                  `json:"swagger" yaml:"swagger"`
	Info        Info                    `json:"info" yaml:"info"`
	Host        string                  `json:"host,omitempty" yaml:"host,omitempty"`
	BasePath    string                  `json:"basePath,omitempty" yaml:"basePath,omitempty"`
	Schemes     []string                `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	Paths       map[string]PathItem     `json:"paths" yaml:"paths"`
	Definitions map[string]interface{}  `json:"definitions,omitempty" yaml:"definitions,omitempty"`
}

// Info contains API metadata
type Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

// Server represents an OpenAPI server
type Server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItem describes operations available on a path
type PathItem struct {
	Get     *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Patch   *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Delete  *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Head    *Operation `json:"head,omitempty" yaml:"head,omitempty"`
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
}

// Operation describes a single API operation
type Operation struct {
	Summary     string              `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string              `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string              `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses" yaml:"responses"`
	Tags        []string            `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Parameter describes a parameter
type Parameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"` // query, header, path, cookie
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      interface{} `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

// RequestBody describes a request body
type RequestBody struct {
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                  `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]MediaType  `json:"content" yaml:"content"`
}

// Response describes a response
type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// MediaType describes a media type
type MediaType struct {
	Schema   interface{}            `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{}            `json:"example,omitempty" yaml:"example,omitempty"`
	Examples map[string]Example     `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// Example represents an example value
type Example struct {
	Summary string      `json:"summary,omitempty" yaml:"summary,omitempty"`
	Value   interface{} `json:"value,omitempty" yaml:"value,omitempty"`
}

// Header describes a response header
type Header struct {
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      interface{} `json:"schema,omitempty" yaml:"schema,omitempty"`
}

// Components holds reusable objects
type Components struct {
	Schemas map[string]interface{} `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

// Parser handles OpenAPI/Swagger spec parsing
type Parser struct {
	generateExamples bool
}

// NewParser creates a new OpenAPI parser
func NewParser(generateExamples bool) *Parser {
	return &Parser{
		generateExamples: generateExamples,
	}
}

// ParseFile parses an OpenAPI or Swagger spec file
func (p *Parser) ParseFile(filePath string) (*models.MockSpec, error) {
	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(data, filePath)
}

// ParseURL parses an OpenAPI or Swagger spec from a URL
func (p *Parser) ParseURL(url string) (*models.MockSpec, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch spec: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v\n", closeErr)
		}
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return p.Parse(data, url)
}

// Parse parses OpenAPI/Swagger spec data
func (p *Parser) Parse(data []byte, source string) (*models.MockSpec, error) {
	// Try to detect format
	isJSON := strings.HasSuffix(strings.ToLower(source), ".json")
	isYAML := strings.HasSuffix(strings.ToLower(source), ".yaml") ||
	          strings.HasSuffix(strings.ToLower(source), ".yml")

	// If not clear from extension, try JSON first
	if !isJSON && !isYAML {
		if err := json.Unmarshal(data, &map[string]interface{}{}); err == nil {
			isJSON = true
		}
	}

	// Try OpenAPI 3.x first
	var openAPISpec OpenAPISpec
	var swaggerSpec SwaggerSpec

	if isJSON {
		if err := json.Unmarshal(data, &openAPISpec); err == nil && openAPISpec.OpenAPI != "" {
			return p.convertOpenAPIToMocks(&openAPISpec), nil
		}
		if err := json.Unmarshal(data, &swaggerSpec); err == nil && swaggerSpec.Swagger != "" {
			return p.convertSwaggerToMocks(&swaggerSpec), nil
		}
	} else {
		if err := yaml.Unmarshal(data, &openAPISpec); err == nil && openAPISpec.OpenAPI != "" {
			return p.convertOpenAPIToMocks(&openAPISpec), nil
		}
		if err := yaml.Unmarshal(data, &swaggerSpec); err == nil && swaggerSpec.Swagger != "" {
			return p.convertSwaggerToMocks(&swaggerSpec), nil
		}
	}

	return nil, fmt.Errorf("failed to parse as OpenAPI 3.x or Swagger 2.0")
}

// convertOpenAPIToMocks converts OpenAPI 3.x spec to mocks
func (p *Parser) convertOpenAPIToMocks(spec *OpenAPISpec) *models.MockSpec {
	mockSpec := &models.MockSpec{
		Mocks: []models.Mock{},
	}

	log.Printf("Converting OpenAPI spec: %s v%s\n", spec.Info.Title, spec.Info.Version)

	priority := 100 // Start with high priority

	for path, pathItem := range spec.Paths {
		operations := map[string]*Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"PATCH":   pathItem.Patch,
			"DELETE":  pathItem.Delete,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			mock := p.createMockFromOperation(path, method, operation, priority)
			mockSpec.Mocks = append(mockSpec.Mocks, mock)
			priority--
		}
	}

	log.Printf("Generated %d mocks from OpenAPI spec\n", len(mockSpec.Mocks))
	return mockSpec
}

// convertSwaggerToMocks converts Swagger 2.0 spec to mocks
func (p *Parser) convertSwaggerToMocks(spec *SwaggerSpec) *models.MockSpec {
	mockSpec := &models.MockSpec{
		Mocks: []models.Mock{},
	}

	log.Printf("Converting Swagger spec: %s v%s\n", spec.Info.Title, spec.Info.Version)

	priority := 100
	basePath := spec.BasePath
	if basePath == "" {
		basePath = ""
	}

	for path, pathItem := range spec.Paths {
		fullPath := basePath + path

		operations := map[string]*Operation{
			"GET":     pathItem.Get,
			"POST":    pathItem.Post,
			"PUT":     pathItem.Put,
			"PATCH":   pathItem.Patch,
			"DELETE":  pathItem.Delete,
			"HEAD":    pathItem.Head,
			"OPTIONS": pathItem.Options,
		}

		for method, operation := range operations {
			if operation == nil {
				continue
			}

			mock := p.createMockFromOperation(fullPath, method, operation, priority)
			mockSpec.Mocks = append(mockSpec.Mocks, mock)
			priority--
		}
	}

	log.Printf("Generated %d mocks from Swagger spec\n", len(mockSpec.Mocks))
	return mockSpec
}

// createMockFromOperation creates a mock from an operation
func (p *Parser) createMockFromOperation(path, method string, operation *Operation, priority int) models.Mock {
	mockName := operation.OperationID
	if mockName == "" {
		mockName = fmt.Sprintf("%s %s", method, path)
	}

	// Get the best response (prefer 200, then 201, then first available)
	var statusCode int
	var response *Response

	for code, resp := range operation.Responses {
		if code == "200" {
			statusCode = 200
			response = &resp
			break
		} else if code == "201" && statusCode != 200 {
			statusCode = 201
			response = &resp
		} else if statusCode == 0 {
			// Parse status code
			if _, err := fmt.Sscanf(code, "%d", &statusCode); err != nil {
				// If parsing fails, skip this response
				continue
			}
			response = &resp
		}
	}

	if statusCode == 0 {
		statusCode = 200
	}

	// Extract response body example
	responseBody := p.extractResponseExample(response)

	// Create the mock
	mock := models.Mock{
		Name:     mockName,
		Priority: priority,
		Request: models.Request{
			URI:    path,
			Method: method,
		},
		Response: models.Response{
			StatusCode: statusCode,
			Headers:    p.extractResponseHeaders(response),
			Body:       responseBody,
		},
	}

	return mock
}

// extractResponseExample extracts an example from a response
func (p *Parser) extractResponseExample(response *Response) string {
	if response == nil || response.Content == nil {
		return ""
	}

	// Look for application/json content
	for contentType, mediaType := range response.Content {
		if strings.Contains(contentType, "json") {
			// Try to get example
			if mediaType.Example != nil {
				if jsonData, err := json.Marshal(mediaType.Example); err == nil {
					return string(jsonData)
				}
			}

			// Try examples
			if len(mediaType.Examples) > 0 {
				for _, example := range mediaType.Examples {
					if example.Value != nil {
						if jsonData, err := json.Marshal(example.Value); err == nil {
							return string(jsonData)
						}
					}
				}
			}

			// Generate example from schema if requested
			if p.generateExamples && mediaType.Schema != nil {
				return p.generateExampleFromSchema(mediaType.Schema)
			}
		}
	}

	return `{"message": "Mock response - add your own example"}`
}

// extractResponseHeaders extracts headers from a response
func (p *Parser) extractResponseHeaders(response *Response) map[string]string {
	headers := make(map[string]string)

	if response != nil && response.Headers != nil {
		for name := range response.Headers {
			headers[name] = "example-value"
		}
	}

	// Always add Content-Type for JSON
	if _, exists := headers["Content-Type"]; !exists {
		headers["Content-Type"] = "application/json"
	}

	return headers
}

// generateExampleFromSchema generates an example value from a JSON schema
func (p *Parser) generateExampleFromSchema(schema interface{}) string {
	// Simplified schema example generation
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return `{"example": "generated"}`
	}

	schemaType, _ := schemaMap["type"].(string)

	switch schemaType {
	case "object":
		properties, ok := schemaMap["properties"].(map[string]interface{})
		if !ok {
			return `{}`
		}

		result := make(map[string]interface{})
		for propName := range properties {
			result[propName] = "example"
		}

		if jsonData, err := json.Marshal(result); err == nil {
			return string(jsonData)
		}

	case "array":
		return `[{"example": "item"}]`

	case "string":
		return `"example string"`

	case "number", "integer":
		return `123`

	case "boolean":
		return `true`
	}

	return `{"example": "generated from schema"}`
}

// SaveMocks saves the generated mocks to a file
func SaveMocks(mockSpec *models.MockSpec, outputPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(mockSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal mocks: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("Saved %d mocks to %s\n", len(mockSpec.Mocks), outputPath)
	return nil
}
