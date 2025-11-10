package matcher

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestMatcherGlobalState(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Create User",
			Request: models.Request{
				URI:    "/api/users",
				Method: "POST",
				JavaScript: `
					(function() {
						var body = JSON.parse(request.body);

						// Initialize users array if it doesn't exist
						if (!global.users) {
							global.users = [];
						}

						// Add user to global state
						var newUser = {
							id: global.users.length + 1,
							name: body.name,
							email: body.email
						};
						global.users.push(newUser);

						return {
							matches: true,
							response: {
								status_code: 201,
								body: JSON.stringify(newUser)
							}
						};
					})()
				`,
			},
		},
		{
			Name: "Get All Users",
			Request: models.Request{
				URI:    "/api/users",
				Method: "GET",
				JavaScript: `
					(function() {
						var users = global.users || [];
						return {
							matches: true,
							response: {
								status_code: 200,
								body: JSON.stringify(users)
							}
						};
					})()
				`,
			},
		},
	}

	matcher := NewMatcher(mocks)

	// First, create a user
	body1 := []byte(`{"name": "John Doe", "email": "john@example.com"}`)
	req1 := createRequest("POST", "/api/users", nil, body1)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match1 == nil {
		t.Fatal("Expected match for create user")
	}
	if match1.Response.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", match1.Response.StatusCode)
	}

	// Get all users - should include the one we just created
	req2 := createRequest("GET", "/api/users", nil, nil)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match2 == nil {
		t.Fatal("Expected match for get users")
	}
	if !strings.Contains(match2.Response.Body, "John Doe") {
		t.Errorf("Expected response to contain created user, got: %s", match2.Response.Body)
	}

	// Create another user
	body3 := []byte(`{"name": "Jane Smith", "email": "jane@example.com"}`)
	req3 := createRequest("POST", "/api/users", nil, body3)
	match3, err := matcher.FindMatch(req3)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match3 == nil {
		t.Fatal("Expected match for create second user")
	}

	// Get all users again - should have both
	req4 := createRequest("GET", "/api/users", nil, nil)
	match4, err := matcher.FindMatch(req4)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if match4 == nil {
		t.Fatal("Expected match for get users")
	}
	if !strings.Contains(match4.Response.Body, "John Doe") || !strings.Contains(match4.Response.Body, "Jane Smith") {
		t.Errorf("Expected response to contain both users, got: %s", match4.Response.Body)
	}
}

func TestMatcherGlobalStatePersistsAcrossUpdates(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Set Counter",
			Request: models.Request{
				URI:    "/api/counter",
				Method: "POST",
				JavaScript: `
					(function() {
						global.counter = (global.counter || 0) + 1;
						return {
							matches: true,
							response: {
								status_code: 200,
								body: JSON.stringify({counter: global.counter})
							}
						};
					})()
				`,
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Increment counter
	req1 := createRequest("POST", "/api/counter", nil, nil)
	match1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(match1.Response.Body, `"counter":1`) {
		t.Errorf("Expected counter to be 1, got: %s", match1.Response.Body)
	}

	// Update mocks (simulating a file reload)
	matcher.UpdateMocks(mocks)

	// Counter should persist
	req2 := createRequest("POST", "/api/counter", nil, nil)
	match2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !strings.Contains(match2.Response.Body, `"counter":2`) {
		t.Errorf("Expected counter to be 2 after mock update, got: %s", match2.Response.Body)
	}
}

func TestMatcherGlobalStateConcurrent(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Increment Counter",
			Request: models.Request{
				URI:    "/api/increment",
				Method: "POST",
				JavaScript: `
					(function() {
						global.counter = (global.counter || 0) + 1;
						return {
							matches: true,
							response: {
								status_code: 200,
								body: JSON.stringify({counter: global.counter})
							}
						};
					})()
				`,
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Make concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := createRequest("POST", "/api/increment", nil, nil)
			_, err := matcher.FindMatch(req)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}

	// Final counter check
	req := createRequest("POST", "/api/increment", nil, nil)
	match, err := matcher.FindMatch(req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	// Should be 11 (10 concurrent + 1 final)
	if !strings.Contains(match.Response.Body, `"counter":11`) {
		t.Errorf("Expected counter to be 11, got: %s", match.Response.Body)
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

func TestSequentialResponsesCycle(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Sequential Test",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				Sequence: []models.ResponseItem{
					{StatusCode: 200, Body: "first"},
					{StatusCode: 200, Body: "second"},
					{StatusCode: 200, Body: "third"},
				},
				SequenceMode: "cycle",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// First call
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1 == nil {
		t.Fatal("Expected mock to match")
	}
	if mock1.Response.Body != "first" {
		t.Errorf("Expected 'first', got '%s'", mock1.Response.Body)
	}

	// Second call
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2.Response.Body != "second" {
		t.Errorf("Expected 'second', got '%s'", mock2.Response.Body)
	}

	// Third call
	req3 := httptest.NewRequest("GET", "/api/test", nil)
	mock3, err := matcher.FindMatch(req3)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock3.Response.Body != "third" {
		t.Errorf("Expected 'third', got '%s'", mock3.Response.Body)
	}

	// Fourth call - should cycle back to first
	req4 := httptest.NewRequest("GET", "/api/test", nil)
	mock4, err := matcher.FindMatch(req4)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock4.Response.Body != "first" {
		t.Errorf("Expected 'first' (cycling), got '%s'", mock4.Response.Body)
	}
}

func TestSequentialResponsesOnce(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Sequential Once Test",
			Request: models.Request{
				URI:    "/api/once",
				Method: "GET",
			},
			Response: models.Response{
				Sequence: []models.ResponseItem{
					{StatusCode: 201, Body: "first"},
					{StatusCode: 200, Body: "second"},
					{StatusCode: 200, Body: "third"},
				},
				SequenceMode: "once",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// First call
	req1 := httptest.NewRequest("GET", "/api/once", nil)
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1.Response.StatusCode != 201 {
		t.Errorf("Expected status 201, got %d", mock1.Response.StatusCode)
	}
	if mock1.Response.Body != "first" {
		t.Errorf("Expected 'first', got '%s'", mock1.Response.Body)
	}

	// Second call
	req2 := httptest.NewRequest("GET", "/api/once", nil)
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2.Response.Body != "second" {
		t.Errorf("Expected 'second', got '%s'", mock2.Response.Body)
	}

	// Third call
	req3 := httptest.NewRequest("GET", "/api/once", nil)
	mock3, err := matcher.FindMatch(req3)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock3.Response.Body != "third" {
		t.Errorf("Expected 'third', got '%s'", mock3.Response.Body)
	}

	// Fourth call - should stay at last response
	req4 := httptest.NewRequest("GET", "/api/once", nil)
	mock4, err := matcher.FindMatch(req4)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock4.Response.Body != "third" {
		t.Errorf("Expected 'third' (staying at last), got '%s'", mock4.Response.Body)
	}

	// Fifth call - should still be at last response
	req5 := httptest.NewRequest("GET", "/api/once", nil)
	mock5, err := matcher.FindMatch(req5)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock5.Response.Body != "third" {
		t.Errorf("Expected 'third' (staying at last), got '%s'", mock5.Response.Body)
	}
}

func TestSequentialResponsesWithHeaders(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Sequential with Headers",
			Request: models.Request{
				URI:    "/api/headers",
				Method: "GET",
			},
			Response: models.Response{
				Sequence: []models.ResponseItem{
					{
						StatusCode: 200,
						Headers:    map[string]string{"X-Step": "1"},
						Body:       "step1",
					},
					{
						StatusCode: 200,
						Headers:    map[string]string{"X-Step": "2"},
						Body:       "step2",
					},
				},
				SequenceMode: "cycle",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// First call
	req1 := httptest.NewRequest("GET", "/api/headers", nil)
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1.Response.Headers["X-Step"] != "1" {
		t.Errorf("Expected header X-Step=1, got %s", mock1.Response.Headers["X-Step"])
	}

	// Second call
	req2 := httptest.NewRequest("GET", "/api/headers", nil)
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2.Response.Headers["X-Step"] != "2" {
		t.Errorf("Expected header X-Step=2, got %s", mock2.Response.Headers["X-Step"])
	}
}

func TestNoSequenceUsesDefaultResponse(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "No Sequence",
			Request: models.Request{
				URI:    "/api/normal",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "default response",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Multiple calls should all return the same default response
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/api/normal", nil)
		mock, err := matcher.FindMatch(req)
		if err != nil {
			t.Fatalf("FindMatch error: %v", err)
		}
		if mock.Response.Body != "default response" {
			t.Errorf("Call %d: Expected 'default response', got '%s'", i+1, mock.Response.Body)
		}
	}
}

func TestSequenceResetOnMockUpdate(t *testing.T) {
	mocks := []models.Mock{
		{
			Name: "Sequential Test",
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				Sequence: []models.ResponseItem{
					{StatusCode: 200, Body: "first"},
					{StatusCode: 200, Body: "second"},
				},
				SequenceMode: "cycle",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// First call
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1.Response.Body != "first" {
		t.Errorf("Expected 'first', got '%s'", mock1.Response.Body)
	}

	// Update mocks (simulating hot reload)
	matcher.UpdateMocks(mocks)

	// After update, sequence should reset to first
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2.Response.Body != "first" {
		t.Errorf("Expected 'first' (after reset), got '%s'", mock2.Response.Body)
	}
}

func TestScenarioFiltering(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:      "Happy Path Mock",
			Scenarios: []string{"happy_path"},
			Priority:  10,
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "success",
			},
		},
		{
			Name:      "Error Mock",
			Scenarios: []string{"error_state"},
			Priority:  10,
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 500,
				Body:       "error",
			},
		},
		{
			Name:     "Default Mock",
			Priority: 5, // Lower priority
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "default",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Test with happy_path scenario
	matcher.SetScenario("happy_path")
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1.Response.Body != "success" {
		t.Errorf("Expected 'success', got '%s'", mock1.Response.Body)
	}

	// Test with error_state scenario
	matcher.SetScenario("error_state")
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2.Response.Body != "error" {
		t.Errorf("Expected 'error', got '%s'", mock2.Response.Body)
	}

	// Test with no scenario (all mocks)
	matcher.SetScenario("")
	req3 := httptest.NewRequest("GET", "/api/test", nil)
	mock3, err := matcher.FindMatch(req3)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	// Should match the highest priority mock (Happy Path Mock)
	if mock3.Response.Body != "success" {
		t.Errorf("Expected 'success', got '%s'", mock3.Response.Body)
	}
}

func TestScenarioMultipleTags(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:      "Multi-Scenario Mock",
			Scenarios: []string{"scenario_a", "scenario_b"},
			Priority:  10,
			Request: models.Request{
				URI:    "/api/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "multi",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Should match in scenario_a
	matcher.SetScenario("scenario_a")
	req1 := httptest.NewRequest("GET", "/api/test", nil)
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1 == nil {
		t.Fatal("Expected match in scenario_a")
	}

	// Should match in scenario_b
	matcher.SetScenario("scenario_b")
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2 == nil {
		t.Fatal("Expected match in scenario_b")
	}

	// Should not match in scenario_c
	matcher.SetScenario("scenario_c")
	req3 := httptest.NewRequest("GET", "/api/test", nil)
	mock3, err := matcher.FindMatch(req3)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock3 != nil {
		t.Error("Expected no match in scenario_c")
	}
}

func TestGetAvailableScenarios(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:      "Mock 1",
			Scenarios: []string{"happy_path", "test"},
			Request:   models.Request{URI: "/test1"},
		},
		{
			Name:      "Mock 2",
			Scenarios: []string{"error_state"},
			Request:   models.Request{URI: "/test2"},
		},
		{
			Name:      "Mock 3",
			Scenarios: []string{"happy_path"},
			Request:   models.Request{URI: "/test3"},
		},
	}

	matcher := NewMatcher(mocks)
	scenarios := matcher.GetAvailableScenarios()

	// Should have 3 unique scenarios
	if len(scenarios) != 3 {
		t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
	}

	// Check that all scenarios are present
	scenarioMap := make(map[string]bool)
	for _, s := range scenarios {
		scenarioMap[s] = true
	}

	if !scenarioMap["happy_path"] {
		t.Error("Expected 'happy_path' scenario")
	}
	if !scenarioMap["error_state"] {
		t.Error("Expected 'error_state' scenario")
	}
	if !scenarioMap["test"] {
		t.Error("Expected 'test' scenario")
	}
}

func TestValidateSchemaBasic(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:     "Validated Mock",
			Priority: 10,
			Request: models.Request{
				URI:    "/api/users",
				Method: "POST",
				ValidateSchema: map[string]interface{}{
					"type": "object",
					"required": []interface{}{"name", "email"},
					"properties": map[string]interface{}{
						"name": map[string]interface{}{
							"type": "string",
						},
						"email": map[string]interface{}{
							"type": "string",
						},
					},
				},
			},
			Response: models.Response{
				StatusCode: 201,
				Body:       "created",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Valid request
	validBody := `{"name": "John", "email": "john@example.com"}`
	req1 := httptest.NewRequest("POST", "/api/users", strings.NewReader(validBody))
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1 == nil {
		t.Fatal("Expected match for valid request")
	}

	// Invalid request - missing required field
	invalidBody := `{"name": "John"}`
	req2 := httptest.NewRequest("POST", "/api/users", strings.NewReader(invalidBody))
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2 != nil {
		t.Error("Expected no match for invalid request")
	}
}

func TestValidateSchemaWithTypes(t *testing.T) {
	mocks := []models.Mock{
		{
			Name:     "Type Validation Mock",
			Priority: 10,
			Request: models.Request{
				URI:    "/api/data",
				Method: "POST",
				ValidateSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"age": map[string]interface{}{
							"type":    "integer",
							"minimum": float64(0),
							"maximum": float64(150),
						},
						"score": map[string]interface{}{
							"type": "number",
						},
					},
				},
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "ok",
			},
		},
	}

	matcher := NewMatcher(mocks)

	// Valid request
	validBody := `{"age": 25, "score": 95.5}`
	req1 := httptest.NewRequest("POST", "/api/data", strings.NewReader(validBody))
	mock1, err := matcher.FindMatch(req1)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock1 == nil {
		t.Fatal("Expected match for valid request")
	}

	// Invalid request - age is string
	invalidBody := `{"age": "25", "score": 95.5}`
	req2 := httptest.NewRequest("POST", "/api/data", strings.NewReader(invalidBody))
	mock2, err := matcher.FindMatch(req2)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock2 != nil {
		t.Error("Expected no match for invalid type")
	}

	// Invalid request - age out of range
	invalidRangeBody := `{"age": 200, "score": 95.5}`
	req3 := httptest.NewRequest("POST", "/api/data", strings.NewReader(invalidRangeBody))
	mock3, err := matcher.FindMatch(req3)
	if err != nil {
		t.Fatalf("FindMatch error: %v", err)
	}
	if mock3 != nil {
		t.Error("Expected no match for out of range value")
	}
}
