package grpc

import (
	"google.golang.org/grpc/metadata"
)

// GRPCConfig represents gRPC-specific mock configuration
type GRPCConfig struct {
	Services      []ServiceConfig      `yaml:"services"`       // gRPC services
	ProtoFiles    []string             `yaml:"proto_files"`    // Proto file paths
	Reflection    bool                 `yaml:"reflection"`     // Enable gRPC reflection
	HealthCheck   bool                 `yaml:"health_check"`   // Enable health checking
	Interceptors  []string             `yaml:"interceptors"`   // Custom interceptors
	TLS           *TLSConfig           `yaml:"tls"`            // TLS configuration
	MaxRecvSize   int                  `yaml:"max_recv_size"`  // Max receive message size
	MaxSendSize   int                  `yaml:"max_send_size"`  // Max send message size
	Compression   string               `yaml:"compression"`    // gzip, snappy
	Web           *GRPCWebConfig       `yaml:"web"`            // gRPC-Web configuration
}

// ServiceConfig represents a gRPC service configuration
type ServiceConfig struct {
	Name    string          `yaml:"name"`    // Service name (e.g., "helloworld.Greeter")
	Methods []MethodConfig  `yaml:"methods"` // Service methods
}

// MethodConfig represents a gRPC method configuration
type MethodConfig struct {
	Name          string                 `yaml:"name"`           // Method name
	StreamType    string                 `yaml:"stream_type"`    // unary, server_stream, client_stream, bidirectional
	Request       *RequestMatcher        `yaml:"request"`        // Request matching
	Response      *ResponseConfig        `yaml:"response"`       // Response configuration
	Responses     []ResponseConfig       `yaml:"responses"`      // Multiple responses for streaming
	Metadata      map[string]string      `yaml:"metadata"`       // Expected metadata
	StatusCode    int                    `yaml:"status_code"`    // gRPC status code (0 = OK)
	StatusMessage string                 `yaml:"status_message"` // Status message
	Delay         int                    `yaml:"delay"`          // Response delay in ms
	Template      bool                   `yaml:"template"`       // Use Go templates
	JavaScript    string                 `yaml:"javascript"`     // JavaScript handler
}

// RequestMatcher represents request matching configuration
type RequestMatcher struct {
	Body          map[string]interface{} `yaml:"body"`           // Expected request body
	MatchMode     string                 `yaml:"match_mode"`     // exact, partial, regex
	JSONPath      []JSONPathMatcher      `yaml:"json_path"`      // JSON path matchers
	BodyContains  string                 `yaml:"body_contains"`  // Body contains string
}

// JSONPathMatcher represents a JSON path matcher
type JSONPathMatcher struct {
	Path  string      `yaml:"path"`
	Value interface{} `yaml:"value"`
	Regex bool        `yaml:"regex"`
}

// ResponseConfig represents a gRPC response configuration
type ResponseConfig struct {
	Body          map[string]interface{} `yaml:"body"`      // Response body
	Metadata      map[string]string      `yaml:"metadata"`  // Response metadata
	Trailers      map[string]string      `yaml:"trailers"`  // Response trailers
	Delay         int                    `yaml:"delay"`     // Delay before sending
	StreamDelay   int                    `yaml:"stream_delay"` // Delay between stream messages
	StreamCount   int                    `yaml:"stream_count"` // Number of stream messages
	Template      bool                   `yaml:"template"`  // Use Go templates
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	CAFile   string `yaml:"ca_file"`
	ClientAuth bool `yaml:"client_auth"`
}

// GRPCWebConfig represents gRPC-Web configuration
type GRPCWebConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// StreamType represents the type of gRPC stream
type StreamType string

const (
	StreamTypeUnary          StreamType = "unary"
	StreamTypeServerStream   StreamType = "server_stream"
	StreamTypeClientStream   StreamType = "client_stream"
	StreamTypeBidirectional  StreamType = "bidirectional"
)

// CallInfo represents information about a gRPC call
type CallInfo struct {
	FullMethod string
	Service    string
	Method     string
	StreamType StreamType
	Metadata   metadata.MD
}

// MockMessage represents a generic gRPC message
type MockMessage struct {
	Fields map[string]interface{}
}

// ProtoFile represents a loaded protocol buffer file
type ProtoFile struct {
	Path     string
	Content  []byte
	Services []string
	Messages []string
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status string `json:"status"` // SERVING, NOT_SERVING, UNKNOWN
}

// ReflectionService represents gRPC reflection service data
type ReflectionService struct {
	Services []string
	Methods  map[string][]string
}

// StreamMessage represents a message in a stream
type StreamMessage struct {
	Data     map[string]interface{}
	Metadata map[string]string
	Trailers map[string]string
	Delay    int
}

// GRPCError represents a gRPC error
type GRPCError struct {
	Code    int    // gRPC status code
	Message string
	Details []interface{}
}

// Common gRPC status codes
const (
	StatusOK                 = 0
	StatusCancelled          = 1
	StatusUnknown            = 2
	StatusInvalidArgument    = 3
	StatusDeadlineExceeded   = 4
	StatusNotFound           = 5
	StatusAlreadyExists      = 6
	StatusPermissionDenied   = 7
	StatusResourceExhausted  = 8
	StatusFailedPrecondition = 9
	StatusAborted            = 10
	StatusOutOfRange         = 11
	StatusUnimplemented      = 12
	StatusInternal           = 13
	StatusUnavailable        = 14
	StatusDataLoss           = 15
	StatusUnauthenticated    = 16
)
