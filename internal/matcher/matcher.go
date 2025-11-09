package matcher

import (
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
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
