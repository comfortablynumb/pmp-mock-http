package matcher

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/dop251/goja"
	"github.com/tidwall/gjson"
	"github.com/xeipuuv/gojsonschema"
)

// Matcher handles matching incoming requests to mock specifications
type Matcher struct {
	mocks          []models.Mock
	globalVM       *goja.Runtime         // Persistent JS runtime for global state
	globalState    map[string]interface{} // Global state shared across JavaScript evaluations
	stateMu        sync.RWMutex           // Mutex to protect global state
	callCounts     map[string]int         // Track call counts for sequence responses
	countMu        sync.Mutex             // Mutex to protect call counts
	activeScenario string                 // Currently active scenario (empty means all mocks)
	scenarioMu     sync.RWMutex           // Mutex to protect scenario state
}

// NewMatcher creates a new request matcher
func NewMatcher(mocks []models.Mock) *Matcher {
	// Sort mocks by priority (higher priority first)
	sortedMocks := make([]models.Mock, len(mocks))
	copy(sortedMocks, mocks)
	sort.Slice(sortedMocks, func(i, j int) bool {
		return sortedMocks[i].Priority > sortedMocks[j].Priority
	})

	// Create a persistent VM for global state
	globalVM := goja.New()
	// Initialize global object in the VM
	if err := globalVM.Set("global", globalVM.NewObject()); err != nil {
		// This should never fail during initialization, but handle it defensively
		panic("failed to initialize global object in JavaScript VM: " + err.Error())
	}

	return &Matcher{
		mocks:       sortedMocks,
		globalVM:    globalVM,
		globalState: make(map[string]interface{}),
		callCounts:  make(map[string]int),
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

	// Get active scenario
	m.scenarioMu.RLock()
	activeScenario := m.activeScenario
	m.scenarioMu.RUnlock()

	// Try to match each mock in priority order
	for _, mock := range m.mocks {
		// Skip mocks that don't belong to the active scenario
		if !m.belongsToScenario(&mock, activeScenario) {
			continue
		}

		// For JavaScript evaluation, we need special handling
		if mock.Request.JavaScript != "" {
			matches, customResponse := m.evaluateJavaScript(r, bodyStr, mock.Request.JavaScript)
			if matches {
				// Create a copy of the mock
				matchedMock := mock
				// If JavaScript returned a custom response, use it
				if customResponse != nil {
					matchedMock.Response = *customResponse
				} else {
					// Use sequential response if defined
					matchedMock.Response = m.getSequentialResponse(&mock)
				}
				return &matchedMock, nil
			}
			continue
		}

		// Standard matching
		if m.matches(r, bodyStr, &mock) {
			// Create a copy of the mock
			matchedMock := mock
			// Get sequential response if defined
			matchedMock.Response = m.getSequentialResponse(&mock)
			return &matchedMock, nil
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

	// Validate JSON schema (if specified)
	if len(mock.Request.ValidateSchema) > 0 {
		if !m.validateSchema(body, mock.Request.ValidateSchema) {
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
// Note: This preserves the global state across mock reloads
func (m *Matcher) UpdateMocks(mocks []models.Mock) {
	// Sort mocks by priority (higher priority first)
	sortedMocks := make([]models.Mock, len(mocks))
	copy(sortedMocks, mocks)
	sort.Slice(sortedMocks, func(i, j int) bool {
		return sortedMocks[i].Priority > sortedMocks[j].Priority
	})

	m.mocks = sortedMocks

	// Reset call counts when mocks are updated
	m.countMu.Lock()
	m.callCounts = make(map[string]int)
	m.countMu.Unlock()

	// Note: We intentionally do NOT reset globalState here
	// This allows state to persist across mock file reloads
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

// validateSchema validates request body against a JSON schema
func (m *Matcher) validateSchema(body string, schema map[string]interface{}) bool {
	// Validate that the body is valid JSON
	if !gjson.Valid(body) {
		return false
	}

	// Convert schema map to JSON
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return false
	}

	// Create schema loader from the schema JSON
	schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)

	// Create document loader from the request body
	documentLoader := gojsonschema.NewStringLoader(body)

	// Validate
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return false
	}

	return result.Valid()
}

// evaluateJavaScript evaluates JavaScript code to determine if request matches
// Returns (matches bool, customResponse *models.Response)
func (m *Matcher) evaluateJavaScript(r *http.Request, body string, script string) (bool, *models.Response) {
	// Lock for thread-safe access to global state
	m.stateMu.Lock()
	defer m.stateMu.Unlock()

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

	// Set the request object in the global VM
	err := m.globalVM.Set("request", requestObj)
	if err != nil {
		return false, nil
	}

	// Execute the JavaScript code in the global VM
	// This allows the script to access and modify the persistent global object
	result, err := m.globalVM.RunString(script)
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

// getSequentialResponse returns the appropriate response based on the sequence and call count
func (m *Matcher) getSequentialResponse(mock *models.Mock) models.Response {
	// If no sequence is defined, return the default response
	if len(mock.Response.Sequence) == 0 {
		return mock.Response
	}

	// Get and increment call count
	m.countMu.Lock()
	callCount := m.callCounts[mock.Name]
	m.callCounts[mock.Name] = callCount + 1
	m.countMu.Unlock()

	// Determine which response to return
	sequenceLen := len(mock.Response.Sequence)
	var responseIndex int

	// Default mode is "cycle"
	mode := mock.Response.SequenceMode
	if mode == "" {
		mode = "cycle"
	}

	if mode == "once" {
		// Stop at the last response
		responseIndex = callCount
		if responseIndex >= sequenceLen {
			responseIndex = sequenceLen - 1
		}
	} else {
		// Cycle through responses
		responseIndex = callCount % sequenceLen
	}

	// Build the response from the sequence item
	item := mock.Response.Sequence[responseIndex]
	return models.Response{
		StatusCode: item.StatusCode,
		Headers:    item.Headers,
		Body:       item.Body,
		Delay:      item.Delay,
		Template:   item.Template,
		Callback:   item.Callback,
	}
}

// belongsToScenario checks if a mock belongs to the given scenario
func (m *Matcher) belongsToScenario(mock *models.Mock, scenario string) bool {
	// If no scenario is active (empty string), all mocks are included
	if scenario == "" {
		return true
	}

	// If mock has no scenarios specified, it belongs to all scenarios
	if len(mock.Scenarios) == 0 {
		return true
	}

	// Check if the mock's scenarios list contains the active scenario
	for _, s := range mock.Scenarios {
		if s == scenario {
			return true
		}
	}

	return false
}

// SetScenario sets the active scenario
func (m *Matcher) SetScenario(scenario string) {
	m.scenarioMu.Lock()
	defer m.scenarioMu.Unlock()
	m.activeScenario = scenario
}

// GetActiveScenario returns the currently active scenario
func (m *Matcher) GetActiveScenario() string {
	m.scenarioMu.RLock()
	defer m.scenarioMu.RUnlock()
	return m.activeScenario
}

// GetAvailableScenarios returns a list of all unique scenarios across all mocks
func (m *Matcher) GetAvailableScenarios() []string {
	scenarioSet := make(map[string]bool)

	for _, mock := range m.mocks {
		for _, scenario := range mock.Scenarios {
			if scenario != "" {
				scenarioSet[scenario] = true
			}
		}
	}

	scenarios := make([]string, 0, len(scenarioSet))
	for scenario := range scenarioSet {
		scenarios = append(scenarios, scenario)
	}

	return scenarios
}
