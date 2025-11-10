package models

// MockSpec represents a complete mock specification loaded from a YAML file
type MockSpec struct {
	Mocks []Mock `yaml:"mocks"`
}

// Mock represents a single mock endpoint definition
type Mock struct {
	Name        string            `yaml:"name"`
	Scenarios   []string          `yaml:"scenarios"`  // Scenarios this mock belongs to (empty means all scenarios)
	Request     Request           `yaml:"request"`
	Response    Response          `yaml:"response"`
	Priority    int               `yaml:"priority"` // Higher priority mocks are matched first
}

// Request defines the matching criteria for incoming requests
type Request struct {
	URI            string                 `yaml:"uri"`             // Can be exact match or regex
	Method         string                 `yaml:"method"`          // Can be exact match or regex
	Headers        map[string]string      `yaml:"headers"`         // Can be exact match or regex (both key and value)
	Body           string                 `yaml:"body"`            // Can be exact match or regex
	IsRegex        RegexConfig            `yaml:"regex"`           // Specify which fields use regex
	JSONPath       []JSONPathMatcher      `yaml:"json_path"`       // GJSON path matchers for JSON bodies
	JavaScript     string                 `yaml:"javascript"`      // JavaScript code for custom matching logic
	ValidateSchema map[string]interface{} `yaml:"validate_schema"` // JSON Schema for request body validation
}

// RegexConfig specifies which request fields should use regex matching
type RegexConfig struct {
	URI     bool `yaml:"uri"`
	Method  bool `yaml:"method"`
	Headers bool `yaml:"headers"` // If true, both header names and values are treated as regex
	Body    bool `yaml:"body"`
}

// JSONPathMatcher defines a GJSON path-based matcher for JSON bodies
type JSONPathMatcher struct {
	Path  string `yaml:"path"`  // GJSON path expression
	Value string `yaml:"value"` // Expected value (supports exact match or regex)
	Regex bool   `yaml:"regex"` // If true, value is treated as regex
}

// Response defines what to return when a request matches
type Response struct {
	StatusCode      int               `yaml:"status_code"`
	Headers         map[string]string `yaml:"headers"`
	Body            string            `yaml:"body"`
	Delay           int               `yaml:"delay"`           // Response delay in milliseconds (fixed)
	Template        bool              `yaml:"template"`        // If true, body is a Go template
	HeaderTemplates bool              `yaml:"header_templates"` // If true, headers support Go templates
	Callback        *Callback         `yaml:"callback"`        // Optional callback to trigger
	Sequence        []ResponseItem    `yaml:"sequence"`        // Sequential responses
	SequenceMode    string            `yaml:"sequence_mode"`   // "cycle" or "once" (default: cycle)
	Chaos           *ChaosConfig      `yaml:"chaos"`           // Chaos engineering configuration
	Latency         *LatencyConfig    `yaml:"latency"`         // Advanced latency simulation
}

// ChaosConfig defines chaos engineering behavior
type ChaosConfig struct {
	Enabled     bool    `yaml:"enabled"`      // Enable chaos mode
	FailureRate float64 `yaml:"failure_rate"` // Probability of failure (0.0 to 1.0)
	ErrorCodes  []int   `yaml:"error_codes"`  // Status codes to randomly return on failure
	LatencyMin  int     `yaml:"latency_min"`  // Minimum latency to inject (ms)
	LatencyMax  int     `yaml:"latency_max"`  // Maximum latency to inject (ms)
}

// LatencyConfig defines advanced latency simulation
type LatencyConfig struct {
	Type string `yaml:"type"` // "fixed", "random", "percentile"
	Min  int    `yaml:"min"`  // Minimum latency for random (ms)
	Max  int    `yaml:"max"`  // Maximum latency for random (ms)
	P50  int    `yaml:"p50"`  // 50th percentile latency (ms)
	P95  int    `yaml:"p95"`  // 95th percentile latency (ms)
	P99  int    `yaml:"p99"`  // 99th percentile latency (ms)
}

// ResponseItem represents a single response in a sequence
type ResponseItem struct {
	StatusCode      int               `yaml:"status_code"`
	Headers         map[string]string `yaml:"headers"`
	Body            string            `yaml:"body"`
	Delay           int               `yaml:"delay"`
	Template        bool              `yaml:"template"`
	HeaderTemplates bool              `yaml:"header_templates"`
	Callback        *Callback         `yaml:"callback"`
	Chaos           *ChaosConfig      `yaml:"chaos"`
	Latency         *LatencyConfig    `yaml:"latency"`
}

// Callback defines an HTTP callback to trigger when a mock matches
type Callback struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`  // HTTP method (default: POST)
	Headers map[string]string `yaml:"headers"` // Headers to send
	Body    string            `yaml:"body"`    // Body to send (can be a template)
}

// JavaScriptResponse represents a custom response from JavaScript evaluation
type JavaScriptResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Delay      int               `json:"delay"`
}
