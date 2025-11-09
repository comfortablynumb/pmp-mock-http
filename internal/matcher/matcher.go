package matcher

import (
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/dop251/goja"
	"github.com/tidwall/gjson"
)

// Matcher handles matching incoming requests to mock specifications
type Matcher struct {
	mocks []models.Mock
}

// NewMatcher creates a new request matcher
func NewMatcher(mocks []models.Mock) *Matcher {
	// Sort mocks by priority (higher priority first)
	sortedMocks := make([]models.Mock, len(mocks))
	copy(sortedMocks, mocks)
	sort.Slice(sortedMocks, func(i, j int) bool {
		return sortedMocks[i].Priority > sortedMocks[j].Priority
	})

	return &Matcher{
		mocks: sortedMocks,
	}
}

// FindMatch finds the first mock that matches the given request
func (m *Matcher) FindMatch(r *http.Request) (*models.Mock, error) {
	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	bodyStr := string(body)

	// Try to match each mock in priority order
	for _, mock := range m.mocks {
		// For JavaScript evaluation, we need special handling
		if mock.Request.JavaScript != "" {
			matches, customResponse := m.evaluateJavaScript(r, bodyStr, mock.Request.JavaScript)
			if matches {
				// Create a copy of the mock
				matchedMock := mock
				// If JavaScript returned a custom response, use it
				if customResponse != nil {
					matchedMock.Response = *customResponse
				}
				return &matchedMock, nil
			}
			continue
		}

		// Standard matching
		if m.matches(r, bodyStr, &mock) {
			return &mock, nil
		}
	}

	return nil, nil // No match found
}

// matches checks if a request matches a mock specification
func (m *Matcher) matches(r *http.Request, body string, mock *models.Mock) bool {
	// Match URI
	if !m.matchString(r.URL.Path, mock.Request.URI, mock.Request.IsRegex.URI) {
		return false
	}

	// Match method
	if !m.matchString(r.Method, mock.Request.Method, mock.Request.IsRegex.Method) {
		return false
	}

	// Match headers
	if !m.matchHeaders(r.Header, mock.Request.Headers, mock.Request.IsRegex.Headers) {
		return false
	}

	// Match body (if specified)
	if mock.Request.Body != "" {
		if !m.matchString(body, mock.Request.Body, mock.Request.IsRegex.Body) {
			return false
		}
	}

	// Match JSON path (if specified)
	if len(mock.Request.JSONPath) > 0 {
		if !m.matchJSONPath(body, mock.Request.JSONPath) {
			return false
		}
	}

	return true
}

// matchString matches a value against a pattern (exact or regex)
func (m *Matcher) matchString(value, pattern string, useRegex bool) bool {
	if pattern == "" {
		return true // Empty pattern matches anything
	}

	if useRegex {
		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			// If regex is invalid, treat as no match
			return false
		}
		return matched
	}

	// Exact match (case-insensitive for methods)
	return strings.EqualFold(value, pattern)
}

// matchHeaders matches request headers against mock header specifications
func (m *Matcher) matchHeaders(requestHeaders http.Header, mockHeaders map[string]string, useRegex bool) bool {
	if len(mockHeaders) == 0 {
		return true // No headers to match
	}

	for mockKey, mockValue := range mockHeaders {
		matched := false

		if useRegex {
			// Regex mode: match both header name and value using regex
			for reqKey, reqValues := range requestHeaders {
				// Try to match header key
				keyMatched, err := regexp.MatchString(mockKey, reqKey)
				if err != nil || !keyMatched {
					continue
				}

				// Try to match header value
				for _, reqValue := range reqValues {
					valueMatched, err := regexp.MatchString(mockValue, reqValue)
					if err == nil && valueMatched {
						matched = true
						break
					}
				}

				if matched {
					break
				}
			}
		} else {
			// Exact match mode
			reqValues := requestHeaders.Values(mockKey)
			for _, reqValue := range reqValues {
				if strings.EqualFold(reqValue, mockValue) {
					matched = true
					break
				}
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

// UpdateMocks updates the matcher with new mocks
func (m *Matcher) UpdateMocks(mocks []models.Mock) {
	// Sort mocks by priority (higher priority first)
	sortedMocks := make([]models.Mock, len(mocks))
	copy(sortedMocks, mocks)
	sort.Slice(sortedMocks, func(i, j int) bool {
		return sortedMocks[i].Priority > sortedMocks[j].Priority
	})

	m.mocks = sortedMocks
}

// matchJSONPath matches request body against GJSON path matchers
func (m *Matcher) matchJSONPath(body string, matchers []models.JSONPathMatcher) bool {
	// Validate that the body is valid JSON
	if !gjson.Valid(body) {
		return false
	}

	// Check each path matcher
	for _, matcher := range matchers {
		result := gjson.Get(body, matcher.Path)
		if !result.Exists() {
			return false
		}

		resultStr := result.String()
		if matcher.Regex {
			// Use regex matching
			matched, err := regexp.MatchString(matcher.Value, resultStr)
			if err != nil || !matched {
				return false
			}
		} else {
			// Exact match
			if resultStr != matcher.Value {
				return false
			}
		}
	}

	return true
}

// evaluateJavaScript evaluates JavaScript code to determine if request matches
// Returns (matches bool, customResponse *models.Response)
func (m *Matcher) evaluateJavaScript(r *http.Request, body string, script string) (bool, *models.Response) {
	vm := goja.New()

	// Prepare the request object for JavaScript
	headers := make(map[string]string)
	for key, values := range r.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	requestObj := map[string]interface{}{
		"uri":     r.URL.Path,
		"method":  r.Method,
		"headers": headers,
		"body":    body,
	}

	// Set the request object in the VM
	err := vm.Set("request", requestObj)
	if err != nil {
		return false, nil
	}

	// Execute the JavaScript code
	result, err := vm.RunString(script)
	if err != nil {
		return false, nil
	}

	// Parse the result
	resultObj := result.Export()
	if resultMap, ok := resultObj.(map[string]interface{}); ok {
		// Check if matches is true
		matches, matchesOk := resultMap["matches"].(bool)
		if !matchesOk || !matches {
			return false, nil
		}

		// Check for custom response
		if responseData, hasResponse := resultMap["response"]; hasResponse && responseData != nil {
			if responseMap, ok := responseData.(map[string]interface{}); ok {
				customResponse := &models.Response{}

				// Parse status code
				if statusCode, ok := responseMap["status_code"].(int64); ok {
					customResponse.StatusCode = int(statusCode)
				}

				// Parse headers
				if headersData, ok := responseMap["headers"].(map[string]interface{}); ok {
					customResponse.Headers = make(map[string]string)
					for k, v := range headersData {
						if strVal, ok := v.(string); ok {
							customResponse.Headers[k] = strVal
						}
					}
				}

				// Parse body
				if bodyData, ok := responseMap["body"].(string); ok {
					customResponse.Body = bodyData
				}

				// Parse delay
				if delay, ok := responseMap["delay"].(int64); ok {
					customResponse.Delay = int(delay)
				}

				return true, customResponse
			}
		}

		return true, nil
	}

	return false, nil
}
