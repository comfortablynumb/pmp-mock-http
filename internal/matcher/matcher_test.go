package matcher

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
)

func TestMatcherExactURIMatch(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:     "Test Mock",
			Priority: 10,
			Request: models.Request{
				URI:    "/api/users/123",
				Method: "GET",
				IsRegex: models.RegexConfig{
					URI:    false,
					Method: false,
				},
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "success",
			},
		},
	}

	matcher := NewMatcher(mocks)
	req := createRequest("GET", "/api/users/123", nil, nil)

	match, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match == nil {
		t.Fatal("Expected match, got nil")
	}
	if match.Name != "Test Mock" {
		t.Errorf("Expected 'Test Mock', got '%s'", match.Name)
	}
}

func TestMatcherNoMatch(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:     "Test Mock",
			Priority: 10,
			Request: models.Request{
				URI:    "/api/users/123",
				Method: "GET",
			},
		},
	}

	matcher := NewMatcher(mocks)
	req := createRequest("GET", "/api/users/456", nil, nil)

	match, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match != nil {
		t.Errorf("Expected no match, got '%s'", match.Name)
	}
}

func TestMatcherRegexURI(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:     "Regex Mock",
			Priority: 5,
			Request: models.Request{
				URI:    `^/api/users/\d+$`,
				Method: "GET",
				IsRegex: models.RegexConfig{
					URI: true,
				},
			},
			Response: models.Response{
				StatusCode: 200,
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching URI
	req1 := createRequest("GET", "/api/users/123", nil, nil)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match for /api/users/123")
	}

	// Test another matching URI
	req2 := createRequest("GET", "/api/users/999", nil, nil)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 == nil {
		t.Error("Expected match for /api/users/999")
	}

	// Test non-matching URI
	req3 := createRequest("GET", "/api/users/abc", nil, nil)
	match3, err := matcher.FindMatch(req3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match3 != nil {
		t.Error("Expected no match for /api/users/abc")
	}
}

func TestMatcherPriority(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:     "Low Priority",
			Priority: 5,
			Request: models.Request{
				URI:    `^/api/users/\d+$`,
				Method: "GET",
				IsRegex: models.RegexConfig{
					URI: true,
				},
			},
		},
		{
			Name:     "High Priority",
			Priority: 10,
			Request: models.Request{
				URI:    "/api/users/123",
				Method: "GET",
			},
		},
	}

	matcher := NewMatcher(mocks)
	req := createRequest("GET", "/api/users/123", nil, nil)

	match, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match == nil {
		t.Fatal("Expected match")
	}
	if match.Name != "High Priority" {
		t.Errorf("Expected 'High Priority' to match first, got '%s'", match.Name)
	}
}

func TestMatcherMethodMatch(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "GET Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
		},
		{
			Name: "POST Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "POST",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test GET
	req1 := createRequest("GET", "/api/test", nil, nil)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil || match1.Name != "GET Mock" {
		t.Error("Expected GET Mock to match")
	}

	// Test POST
	req2 := createRequest("POST", "/api/test", nil, nil)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 == nil || match2.Name != "POST Mock" {
		t.Error("Expected POST Mock to match")
	}
}

func TestMatcherHeadersExact(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Header Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
				Headers: map[string]string{
					"Content-Type": "application/json",
					"X-API-Key":    "secret123",
				},
				IsRegex: models.RegexConfig{
					Headers: false,
				},
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching headers
	headers := map[string]string{
		"Content-Type": "application/json",
		"X-API-Key":    "secret123",
	}
	req1 := createRequest("GET", "/api/test", headers, nil)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match with correct headers")
	}

	// Test non-matching headers
	headers2 := map[string]string{
		"Content-Type": "application/json",
		"X-API-Key":    "wrong",
	}
	req2 := createRequest("GET", "/api/test", headers2, nil)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match with wrong headers")
	}
}

func TestMatcherHeadersRegex(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Regex Header Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
				Headers: map[string]string{
					"Authorization": `^Bearer [A-Za-z0-9]+$`,
				},
				IsRegex: models.RegexConfig{
					Headers: true,
				},
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching header
	headers1 := map[string]string{
		"Authorization": "Bearer abc123XYZ",
	}
	req1 := createRequest("GET", "/api/test", headers1, nil)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match with Bearer token")
	}

	// Test non-matching header
	headers2 := map[string]string{
		"Authorization": "Basic abc123",
	}
	req2 := createRequest("GET", "/api/test", headers2, nil)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match with Basic auth")
	}
}

func TestMatcherBodyExact(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Body Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "POST",
				Body:   `{"name": "test"}`,
				IsRegex: models.RegexConfig{
					Body: false,
				},
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching body
	req1 := createRequest("POST", "/api/test", nil, []byte(`{"name": "test"}`))
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match with correct body")
	}

	// Test non-matching body
	req2 := createRequest("POST", "/api/test", nil, []byte(`{"name": "other"}`))
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match with different body")
	}
}

func TestMatcherBodyRegex(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Regex Body Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "POST",
				Body:   `.*"email"\s*:\s*"[^"]+@[^"]+\.[^"]+".*`,
				IsRegex: models.RegexConfig{
					Body: true,
				},
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching body with email
	req1 := createRequest("POST", "/api/test", nil, []byte(`{"name": "John", "email": "john@example.com"}`))
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match with email in body")
	}

	// Test non-matching body without email
	req2 := createRequest("POST", "/api/test", nil, []byte(`{"name": "John"}`))
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match without email")
	}
}

func TestMatcherUpdateMocks(t *testing.T) {
	initialMocks := []models.Mock{
		{
			Name: "Initial Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
		},
	}

	matcher := NewMatcher(initialMocks)

	// Initial state
	req := createRequest("GET", "/api/test", nil, nil)
	match, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match == nil || match.Name != "Initial Mock" {
		t.Error("Expected 'Initial Mock' to match")
	}

	// Update mocks
	newMocks := []models.Mock{
		{
			Name: "Updated Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
		},
	}
	matcher.UpdateMocks(newMocks)

	// After update
	match2, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 == nil || match2.Name != "Updated Mock" {
		t.Error("Expected 'Updated Mock' to match after update")
	}
}

func TestMatcherEmptyPattern(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Empty Pattern Mock",
			Request: models.Request{
				URI:    "",
				Method: "",
			},
		},
	}

	matcher := NewMatcher(mocks)
	req := createRequest("GET", "/any/path", nil, nil)

	match, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match == nil {
		t.Error("Expected match with empty patterns (should match anything)")
	}
}

func TestMatcherJSONPath(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "JSON Path Mock",
			Request: models.Request{
				URI:    "/api/users",
				Method: "POST",
				JSONPath: []models.JSONPathMatcher{
					{
						Path:  "user.email",
						Value: "test@example.com",
						Regex: false,
					},
					{
						Path:  "user.age",
						Value: "25",
						Regex: false,
					},
				},
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "matched",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching JSON path
	body1 := []byte(`{"user": {"email": "test@example.com", "age": 25}}`)
	req1 := createRequest("POST", "/api/users", nil, body1)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match with correct JSON path values")
	}

	// Test non-matching JSON path
	body2 := []byte(`{"user": {"email": "other@example.com", "age": 25}}`)
	req2 := createRequest("POST", "/api/users", nil, body2)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match with different email")
	}
}

func TestMatcherJSONPathRegex(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "JSON Path Regex Mock",
			Request: models.Request{
				URI:    "/api/users",
				Method: "POST",
				JSONPath: []models.JSONPathMatcher{
					{
						Path:  "user.email",
						Value: `^[a-z]+@example\.com$`,
						Regex: true,
					},
				},
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "matched",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching with regex
	body1 := []byte(`{"user": {"email": "test@example.com"}}`)
	req1 := createRequest("POST", "/api/users", nil, body1)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match with regex pattern")
	}

	// Test non-matching with regex
	body2 := []byte(`{"user": {"email": "Test123@example.com"}}`)
	req2 := createRequest("POST", "/api/users", nil, body2)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match with uppercase/numbers in email")
	}
}

func TestMatcherJSONPathInvalidJSON(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "JSON Path Mock",
			Request: models.Request{
				URI:    "/api/users",
				Method: "POST",
				JSONPath: []models.JSONPathMatcher{
					{
						Path:  "user.email",
						Value: "test@example.com",
					},
				},
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test with invalid JSON
	body := []byte(`{invalid json}`)
	req := createRequest("POST", "/api/users", nil, body)
	match, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match != nil {
		t.Error("Expected no match with invalid JSON")
	}
}

func TestMatcherJavaScript(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "JavaScript Mock",
			Request: models.Request{
				URI:    "/api/test",
				Method: "POST",
				JavaScript: `
					(function() {
						var body = JSON.parse(request.body);
						return {
							matches: body.user && body.user.role === "admin",
							response: null
						};
					})()
				`,
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "admin access granted",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching JavaScript condition
	body1 := []byte(`{"user": {"role": "admin"}}`)
	req1 := createRequest("POST", "/api/test", nil, body1)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match for admin user")
	}
	if match1 != nil && match1.Response.Body != "admin access granted" {
		t.Errorf("Expected 'admin access granted', got '%s'", match1.Response.Body)
	}

	// Test non-matching JavaScript condition
	body2 := []byte(`{"user": {"role": "user"}}`)
	req2 := createRequest("POST", "/api/test", nil, body2)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match for regular user")
	}
}

func TestMatcherJavaScriptCustomResponse(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "JavaScript Custom Response Mock",
			Request: models.Request{
				URI:    "/api/dynamic",
				Method: "POST",
				JavaScript: `
					(function() {
						var body = JSON.parse(request.body);
						if (body.type === "premium") {
							return {
								matches: true,
								response: {
									status_code: 200,
									headers: {"X-Premium": "true"},
									body: "Premium response",
									delay: 0
								}
							};
						}
						return {
							matches: true,
							response: {
								status_code: 200,
								body: "Standard response"
							}
						};
					})()
				`,
			},
			Response: models.Response{
				StatusCode: 500,
				Body:       "should not see this",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test custom response for premium type
	body1 := []byte(`{"type": "premium"}`)
	req1 := createRequest("POST", "/api/dynamic", nil, body1)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Fatal("Expected match for premium type")
	}
	if match1.Response.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", match1.Response.StatusCode)
	}
	if match1.Response.Body != "Premium response" {
		t.Errorf("Expected 'Premium response', got '%s'", match1.Response.Body)
	}
	if match1.Response.Headers["X-Premium"] != "true" {
		t.Errorf("Expected X-Premium header 'true', got '%s'", match1.Response.Headers["X-Premium"])
	}

	// Test standard response
	body2 := []byte(`{"type": "standard"}`)
	req2 := createRequest("POST", "/api/dynamic", nil, body2)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 == nil {
		t.Fatal("Expected match for standard type")
	}
	if match2.Response.Body != "Standard response" {
		t.Errorf("Expected 'Standard response', got '%s'", match2.Response.Body)
	}
}

func TestMatcherJavaScriptRequestObject(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "JavaScript Request Object Mock",
			Request: models.Request{
				JavaScript: `
					(function() {
						return {
							matches: request.uri === "/api/test" &&
							         request.method === "POST" &&
							         request.headers["Content-Type"] === "application/json",
							response: null
						};
					})()
				`,
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "matched",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test matching all conditions
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	req1 := createRequest("POST", "/api/test", headers, []byte(`{}`))
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Error("Expected match when all conditions are met")
	}

	// Test non-matching method
	req2 := createRequest("GET", "/api/test", headers, []byte(`{}`))
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 != nil {
		t.Error("Expected no match with GET method")
	}
}

// Helper function to create HTTP requests for testing
func createRequest(method, uri string, headers map[string]string, body []byte) *http.Request {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req, _ := http.NewRequest(method, uri, bodyReader)

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	return req
}
