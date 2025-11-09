package models

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMockSpecUnmarshal(t *testing.T) {
	yamlData := `
mocks:
  - name: "Test Mock"
    priority: 10
    request:
      uri: "/api/test"
      method: "GET"
      headers:
        Content-Type: "application/json"
      body: "test body"
      regex:
        uri: true
        method: false
        headers: true
        body: false
    response:
      status_code: 200
      headers:
        Content-Type: "application/json"
      body: '{"result": "success"}'
      delay: 100
`

	var spec MockSpec
	err := yaml.Unmarshal([]byte(yamlData), &spec)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(spec.Mocks) != 1 {
		t.Fatalf("Expected 1 mock, got %d", len(spec.Mocks))
	}

	mock := spec.Mocks[0]

	// Test basic fields
	if mock.Name != "Test Mock" {
		t.Errorf("Expected name 'Test Mock', got '%s'", mock.Name)
	}
	if mock.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", mock.Priority)
	}

	// Test request fields
	if mock.Request.URI != "/api/test" {
		t.Errorf("Expected URI '/api/test', got '%s'", mock.Request.URI)
	}
	if mock.Request.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", mock.Request.Method)
	}
	if mock.Request.Body != "test body" {
		t.Errorf("Expected body 'test body', got '%s'", mock.Request.Body)
	}

	// Test headers
	if mock.Request.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected Content-Type header 'application/json', got '%s'", mock.Request.Headers["Content-Type"])
	}

	// Test regex config
	if !mock.Request.IsRegex.URI {
		t.Error("Expected regex.uri to be true")
	}
	if mock.Request.IsRegex.Method {
		t.Error("Expected regex.method to be false")
	}
	if !mock.Request.IsRegex.Headers {
		t.Error("Expected regex.headers to be true")
	}
	if mock.Request.IsRegex.Body {
		t.Error("Expected regex.body to be false")
	}

	// Test response fields
	if mock.Response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", mock.Response.StatusCode)
	}
	if mock.Response.Headers["Content-Type"] != "application/json" {
		t.Errorf("Expected response Content-Type 'application/json', got '%s'", mock.Response.Headers["Content-Type"])
	}
	if mock.Response.Body != `{"result": "success"}` {
		t.Errorf("Expected response body '{\"result\": \"success\"}', got '%s'", mock.Response.Body)
	}
	if mock.Response.Delay != 100 {
		t.Errorf("Expected delay 100, got %d", mock.Response.Delay)
	}
}

func TestMockSpecMultipleMocks(t *testing.T) {
	yamlData := `
mocks:
  - name: "Mock 1"
    request:
      uri: "/api/test1"
      method: "GET"
    response:
      status_code: 200
  - name: "Mock 2"
    request:
      uri: "/api/test2"
      method: "POST"
    response:
      status_code: 201
`

	var spec MockSpec
	err := yaml.Unmarshal([]byte(yamlData), &spec)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(spec.Mocks) != 2 {
		t.Fatalf("Expected 2 mocks, got %d", len(spec.Mocks))
	}

	if spec.Mocks[0].Name != "Mock 1" {
		t.Errorf("Expected first mock name 'Mock 1', got '%s'", spec.Mocks[0].Name)
	}
	if spec.Mocks[1].Name != "Mock 2" {
		t.Errorf("Expected second mock name 'Mock 2', got '%s'", spec.Mocks[1].Name)
	}
}

func TestMockSpecMinimal(t *testing.T) {
	yamlData := `
mocks:
  - name: "Minimal Mock"
    request:
      uri: "/test"
      method: "GET"
    response:
      status_code: 204
`

	var spec MockSpec
	err := yaml.Unmarshal([]byte(yamlData), &spec)
	if err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	if len(spec.Mocks) != 1 {
		t.Fatalf("Expected 1 mock, got %d", len(spec.Mocks))
	}

	mock := spec.Mocks[0]
	if mock.Priority != 0 {
		t.Errorf("Expected default priority 0, got %d", mock.Priority)
	}
	if mock.Response.Delay != 0 {
		t.Errorf("Expected default delay 0, got %d", mock.Response.Delay)
	}
}
