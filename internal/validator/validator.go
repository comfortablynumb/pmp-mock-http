package validator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/dop251/goja"
	"github.com/xeipuuv/gojsonschema"
)

// ValidationResult represents the result of mock validation
type ValidationResult struct {
	Valid   bool
	Errors  []string
	Warnings []string
}

// Validator validates mock configurations
type Validator struct{}

// NewValidator creates a new mock validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateMocks validates all mocks and returns validation results
func (v *Validator) ValidateMocks(mocks []models.Mock) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Track mock names to detect duplicates
	nameCount := make(map[string]int)

	for i, mock := range mocks {
		mockPrefix := fmt.Sprintf("Mock #%d (%s)", i+1, mock.Name)

		// Validate mock name
		if mock.Name == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: mock has no name", mockPrefix))
		} else {
			nameCount[mock.Name]++
		}

		// Validate request patterns
		v.validateRequest(&mock.Request, mockPrefix, result)

		// Validate response
		v.validateResponse(&mock.Response, mockPrefix, result)
	}

	// Check for duplicate names
	for name, count := range nameCount {
		if count > 1 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Duplicate mock name '%s' (%d occurrences)", name, count))
		}
	}

	return result
}

// validateRequest validates request configuration
func (v *Validator) validateRequest(req *models.Request, prefix string, result *ValidationResult) {
	// Validate regex patterns
	if req.IsRegex.URI && req.URI != "" {
		if _, err := regexp.Compile(req.URI); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid URI regex: %v", prefix, err))
		}
	}

	if req.IsRegex.Method && req.Method != "" {
		if _, err := regexp.Compile(req.Method); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid Method regex: %v", prefix, err))
		}
	}

	if req.IsRegex.Body && req.Body != "" {
		if _, err := regexp.Compile(req.Body); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid Body regex: %v", prefix, err))
		}
	}

	if req.IsRegex.Headers {
		for key, value := range req.Headers {
			if _, err := regexp.Compile(key); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid header key regex '%s': %v", prefix, key, err))
			}
			if _, err := regexp.Compile(value); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid header value regex for '%s': %v", prefix, key, err))
			}
		}
	}

	// Validate JSON path matchers
	for j, matcher := range req.JSONPath {
		if matcher.Path == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: json_path[%d] has empty path", prefix, j))
			result.Valid = false
		}
		if matcher.Regex {
			if _, err := regexp.Compile(matcher.Value); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: json_path[%d] invalid regex: %v", prefix, j, err))
			}
		}
	}

	// Validate JavaScript
	if req.JavaScript != "" {
		vm := goja.New()

		// Set up mock request object for validation
		mockRequest := map[string]interface{}{
			"uri":     "/test",
			"method":  "GET",
			"headers": map[string]string{},
			"body":    "",
		}

		// Set up global object for validation
		if err := vm.Set("global", vm.NewObject()); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to initialize global object: %v", prefix, err))
		}

		if err := vm.Set("request", mockRequest); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to set request object: %v", prefix, err))
		}

		// Now validate the JavaScript code
		if _, err := vm.RunString(req.JavaScript); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid JavaScript: %v", prefix, err))
		}
	}

	// Validate JSON schema
	if len(req.ValidateSchema) > 0 {
		schemaJSON, err := json.Marshal(req.ValidateSchema)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid validate_schema: %v", prefix, err))
		} else {
			schemaLoader := gojsonschema.NewBytesLoader(schemaJSON)
			if _, err := gojsonschema.NewSchema(schemaLoader); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid JSON schema: %v", prefix, err))
			}
		}
	}
}

// validateResponse validates response configuration
func (v *Validator) validateResponse(resp *models.Response, prefix string, result *ValidationResult) {
	// Validate status code
	if resp.StatusCode < 100 || resp.StatusCode > 599 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s: unusual status code %d", prefix, resp.StatusCode))
	}

	// Validate chaos configuration
	if resp.Chaos != nil && resp.Chaos.Enabled {
		if resp.Chaos.FailureRate < 0 || resp.Chaos.FailureRate > 1 {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: chaos failure_rate must be between 0 and 1", prefix))
		}
		if len(resp.Chaos.ErrorCodes) == 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: chaos enabled but no error_codes specified", prefix))
		}
		for _, code := range resp.Chaos.ErrorCodes {
			if code < 100 || code > 599 {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid chaos error code %d", prefix, code))
				result.Valid = false
			}
		}
		if resp.Chaos.LatencyMin < 0 || resp.Chaos.LatencyMax < 0 {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: chaos latency values must be >= 0", prefix))
		}
		if resp.Chaos.LatencyMax > 0 && resp.Chaos.LatencyMin > resp.Chaos.LatencyMax {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: chaos latency_min must be <= latency_max", prefix))
		}
	}

	// Validate latency configuration
	if resp.Latency != nil {
		latencyType := strings.ToLower(resp.Latency.Type)
		switch latencyType {
		case "fixed":
			// Fixed uses the standard Delay field
		case "random":
			if resp.Latency.Min < 0 || resp.Latency.Max < 0 {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: latency min/max must be >= 0", prefix))
			}
			if resp.Latency.Max > 0 && resp.Latency.Min > resp.Latency.Max {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: latency min must be <= max", prefix))
			}
		case "percentile":
			if resp.Latency.P50 < 0 || resp.Latency.P95 < 0 || resp.Latency.P99 < 0 {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: latency percentiles must be >= 0", prefix))
			}
			if resp.Latency.P50 > resp.Latency.P95 || resp.Latency.P95 > resp.Latency.P99 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s: latency percentiles should be ordered p50 <= p95 <= p99", prefix))
			}
		default:
			if resp.Latency.Type != "" {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid latency type '%s' (must be: fixed, random, or percentile)", prefix, resp.Latency.Type))
			}
		}
	}

	// Validate sequence responses
	for j, item := range resp.Sequence {
		itemPrefix := fmt.Sprintf("%s sequence[%d]", prefix, j)
		itemResp := models.Response{
			StatusCode:      item.StatusCode,
			Headers:         item.Headers,
			Body:            item.Body,
			Delay:           item.Delay,
			Template:        item.Template,
			HeaderTemplates: item.HeaderTemplates,
			Callback:        item.Callback,
			Chaos:           item.Chaos,
			Latency:         item.Latency,
		}
		v.validateResponse(&itemResp, itemPrefix, result)
	}

	// Validate sequence mode
	if len(resp.Sequence) > 0 && resp.SequenceMode != "" {
		mode := strings.ToLower(resp.SequenceMode)
		if mode != "cycle" && mode != "once" {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: invalid sequence_mode '%s' (must be: cycle or once)", prefix, resp.SequenceMode))
		}
	}
}

// PrintValidationResult prints validation results in a user-friendly format
func (v *Validator) PrintValidationResult(result *ValidationResult) {
	if len(result.Errors) > 0 {
		fmt.Println("\n❌ Mock Validation FAILED:")
		for _, err := range result.Errors {
			fmt.Printf("  ERROR: %s\n", err)
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  Mock Validation Warnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  WARNING: %s\n", warn)
		}
	}

	if result.Valid && len(result.Warnings) == 0 {
		fmt.Println("\n✅ All mocks validated successfully!")
	} else if result.Valid {
		fmt.Printf("\n✅ Mocks are valid (with %d warnings)\n", len(result.Warnings))
	}
}
