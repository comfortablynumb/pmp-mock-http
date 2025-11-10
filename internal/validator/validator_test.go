package validator

import (
	"testing"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
)

func TestValidateJavaScript(t *testing.T) {
	validator := NewValidator()

	// Test valid JavaScript with request object
	mocks := []models.Mock{
		{
			Name: "Valid JS with request",
			Request: models.Request{
				URI:    "/test",
				Method: "GET",
				JavaScript: `
					(function() {
						return {
							matches: request.method === "GET"
						};
					})()
				`,
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "OK",
			},
		},
	}

	result := validator.ValidateMocks(mocks)
	if !result.Valid {
		t.Errorf("Expected validation to pass for valid JavaScript with request object, got errors: %v", result.Errors)
	}
}

func TestValidateJavaScriptWithGlobal(t *testing.T) {
	validator := NewValidator()

	// Test valid JavaScript with global object
	mocks := []models.Mock{
		{
			Name: "Valid JS with global",
			Request: models.Request{
				URI:    "/test",
				Method: "GET",
				JavaScript: `
					(function() {
						if (!global.counter) {
							global.counter = 0;
						}
						global.counter++;
						return { matches: true };
					})()
				`,
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "OK",
			},
		},
	}

	result := validator.ValidateMocks(mocks)
	if !result.Valid {
		t.Errorf("Expected validation to pass for valid JavaScript with global object, got errors: %v", result.Errors)
	}
}

func TestValidateInvalidJavaScript(t *testing.T) {
	validator := NewValidator()

	// Test invalid JavaScript syntax
	mocks := []models.Mock{
		{
			Name: "Invalid JS",
			Request: models.Request{
				URI:        "/test",
				Method:     "GET",
				JavaScript: "this is not valid javascript {{{",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "OK",
			},
		},
	}

	result := validator.ValidateMocks(mocks)
	if result.Valid {
		t.Error("Expected validation to fail for invalid JavaScript")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for invalid JavaScript")
	}
}

func TestValidateChaosConfig(t *testing.T) {
	validator := NewValidator()

	// Test invalid chaos configuration
	mocks := []models.Mock{
		{
			Name: "Invalid Chaos",
			Request: models.Request{
				URI:    "/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "OK",
				Chaos: &models.ChaosConfig{
					Enabled:     true,
					FailureRate: 1.5, // Invalid: should be between 0 and 1
					ErrorCodes:  []int{500},
				},
			},
		},
	}

	result := validator.ValidateMocks(mocks)
	if result.Valid {
		t.Error("Expected validation to fail for invalid chaos failure rate")
	}
}

func TestValidateLatencyConfig(t *testing.T) {
	validator := NewValidator()

	// Test invalid latency configuration
	mocks := []models.Mock{
		{
			Name: "Invalid Latency",
			Request: models.Request{
				URI:    "/test",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "OK",
				Latency: &models.LatencyConfig{
					Type: "invalid_type",
					Min:  100,
					Max:  200,
				},
			},
		},
	}

	result := validator.ValidateMocks(mocks)
	if result.Valid {
		t.Error("Expected validation to fail for invalid latency type")
	}
}

func TestValidateDuplicateNames(t *testing.T) {
	validator := NewValidator()

	mocks := []models.Mock{
		{
			Name: "Duplicate",
			Request: models.Request{
				URI:    "/test1",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "OK",
			},
		},
		{
			Name: "Duplicate",
			Request: models.Request{
				URI:    "/test2",
				Method: "GET",
			},
			Response: models.Response{
				StatusCode: 200,
				Body:       "OK",
			},
		},
	}

	result := validator.ValidateMocks(mocks)
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for duplicate mock names")
	}
}
