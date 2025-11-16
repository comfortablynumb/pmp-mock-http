package grpc

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server represents a gRPC mock server
type Server struct {
	config   *GRPCConfig
	grpcServer *grpc.Server
	listener net.Listener
	services map[string]*ServiceConfig
	mu       sync.RWMutex
}

// NewServer creates a new gRPC mock server
func NewServer(config *GRPCConfig) (*Server, error) {
	s := &Server{
		config:   config,
		services: make(map[string]*ServiceConfig),
	}

	// Index services by name
	for i := range config.Services {
		s.services[config.Services[i].Name] = &config.Services[i]
	}

	// Create gRPC server options
	opts := []grpc.ServerOption{
		grpc.UnknownServiceHandler(s.handleUnknownService),
	}

	// Add max message size options
	if config.MaxRecvSize > 0 {
		opts = append(opts, grpc.MaxRecvMsgSize(config.MaxRecvSize))
	}
	if config.MaxSendSize > 0 {
		opts = append(opts, grpc.MaxSendMsgSize(config.MaxSendSize))
	}

	// Add TLS credentials if configured
	if config.TLS != nil && config.TLS.Enabled {
		cert, err := tls.LoadX509KeyPair(config.TLS.CertFile, config.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		creds := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
		})
		opts = append(opts, grpc.Creds(creds))
	}

	s.grpcServer = grpc.NewServer(opts...)

	// Register reflection service if enabled
	if config.Reflection {
		reflection.Register(s.grpcServer)
	}

	// Register health check service if enabled
	if config.HealthCheck {
		healthServer := health.NewServer()
		grpc_health_v1.RegisterHealthServer(s.grpcServer, healthServer)

		// Set all services as serving
		for serviceName := range s.services {
			healthServer.SetServingStatus(serviceName, grpc_health_v1.HealthCheckResponse_SERVING)
		}
	}

	return s, nil
}

// Start starts the gRPC server
func (s *Server) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = listener
	return s.grpcServer.Serve(listener)
}

// Stop stops the gRPC server gracefully
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
}

// handleUnknownService handles requests to unknown services
func (s *Server) handleUnknownService(srv interface{}, stream grpc.ServerStream) error {
	// Get call info
	method, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return status.Error(codes.Internal, "failed to get method name")
	}

	// Parse method name (format: /package.Service/Method)
	parts := strings.Split(method, "/")
	if len(parts) != 3 {
		return status.Error(codes.InvalidArgument, "invalid method name")
	}

	serviceName := parts[1]
	methodName := parts[2]

	// Find service config
	s.mu.RLock()
	serviceConfig, exists := s.services[serviceName]
	s.mu.RUnlock()

	if !exists {
		return status.Error(codes.Unimplemented, fmt.Sprintf("service %s not found", serviceName))
	}

	// Find method config
	var methodConfig *MethodConfig
	for i := range serviceConfig.Methods {
		if serviceConfig.Methods[i].Name == methodName {
			methodConfig = &serviceConfig.Methods[i]
			break
		}
	}

	if methodConfig == nil {
		return status.Error(codes.Unimplemented, fmt.Sprintf("method %s not found", methodName))
	}

	// Get metadata
	md, _ := metadata.FromIncomingContext(stream.Context())

	// Handle based on stream type
	switch methodConfig.StreamType {
	case string(StreamTypeUnary):
		return s.handleUnary(stream, methodConfig, md)
	case string(StreamTypeServerStream):
		return s.handleServerStream(stream, methodConfig, md)
	case string(StreamTypeClientStream):
		return s.handleClientStream(stream, methodConfig, md)
	case string(StreamTypeBidirectional):
		return s.handleBidirectional(stream, methodConfig, md)
	default:
		return s.handleUnary(stream, methodConfig, md)
	}
}

// handleUnary handles unary RPC calls
func (s *Server) handleUnary(stream grpc.ServerStream, method *MethodConfig, md metadata.MD) error {
	// Receive request
	var req MockMessage
	if err := stream.RecvMsg(&req); err != nil {
		return err
	}

	// Check if request matches
	if method.Request != nil && !s.matchesRequest(&req, method.Request) {
		return status.Error(codes.InvalidArgument, "request does not match expected pattern")
	}

	// Apply delay
	if method.Delay > 0 {
		time.Sleep(time.Duration(method.Delay) * time.Millisecond)
	}

	// Send metadata if configured
	if method.Response != nil && len(method.Response.Metadata) > 0 {
		respMd := metadata.New(method.Response.Metadata)
		_ = stream.SendHeader(respMd)
	}

	// Send response
	if method.Response != nil {
		resp := &MockMessage{
			Fields: method.Response.Body,
		}
		if err := stream.SendMsg(resp); err != nil {
			return err
		}
	}

	// Send trailers if configured
	if method.Response != nil && len(method.Response.Trailers) > 0 {
		trailerMd := metadata.New(method.Response.Trailers)
		stream.SetTrailer(trailerMd)
	}

	// Return status
	if method.StatusCode != 0 {
		return status.Error(codes.Code(method.StatusCode), method.StatusMessage)
	}

	return nil
}

// handleServerStream handles server streaming RPC calls
func (s *Server) handleServerStream(stream grpc.ServerStream, method *MethodConfig, md metadata.MD) error {
	// Receive request
	var req MockMessage
	if err := stream.RecvMsg(&req); err != nil {
		return err
	}

	// Check if request matches
	if method.Request != nil && !s.matchesRequest(&req, method.Request) {
		return status.Error(codes.InvalidArgument, "request does not match expected pattern")
	}

	// Send metadata if configured
	if len(method.Responses) > 0 && len(method.Responses[0].Metadata) > 0 {
		respMd := metadata.New(method.Responses[0].Metadata)
		_ = stream.SendHeader(respMd)
	}

	// Send stream responses
	for _, respConfig := range method.Responses {
		// Apply stream delay
		if respConfig.StreamDelay > 0 {
			time.Sleep(time.Duration(respConfig.StreamDelay) * time.Millisecond)
		}

		resp := &MockMessage{
			Fields: respConfig.Body,
		}

		if err := stream.SendMsg(resp); err != nil {
			return err
		}
	}

	return nil
}

// handleClientStream handles client streaming RPC calls
func (s *Server) handleClientStream(stream grpc.ServerStream, method *MethodConfig, md metadata.MD) error {
	// Receive all client messages
	var messages []MockMessage

	for {
		var msg MockMessage
		err := stream.RecvMsg(&msg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		messages = append(messages, msg)
	}

	// Process messages (could aggregate, validate, etc.)
	// For now, just send configured response
	// Note: messages variable is collected but not yet processed in this implementation
	_ = messages

	if method.Response != nil {
		resp := &MockMessage{
			Fields: method.Response.Body,
		}
		if err := stream.SendMsg(resp); err != nil {
			return err
		}
	}

	return nil
}

// handleBidirectional handles bidirectional streaming RPC calls
func (s *Server) handleBidirectional(stream grpc.ServerStream, method *MethodConfig, md metadata.MD) error {
	// Handle bidirectional streaming
	responseIndex := 0

	for {
		var req MockMessage
		err := stream.RecvMsg(&req)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// Send corresponding response
		if responseIndex < len(method.Responses) {
			respConfig := method.Responses[responseIndex]

			if respConfig.StreamDelay > 0 {
				time.Sleep(time.Duration(respConfig.StreamDelay) * time.Millisecond)
			}

			resp := &MockMessage{
				Fields: respConfig.Body,
			}

			if err := stream.SendMsg(resp); err != nil {
				return err
			}

			responseIndex++
		}
	}
}

// matchesRequest checks if a request matches the expected pattern
func (s *Server) matchesRequest(req *MockMessage, matcher *RequestMatcher) bool {
	if matcher.Body == nil {
		return true
	}

	// Convert request to JSON for matching
	reqJSON, _ := json.Marshal(req.Fields)
	expectedJSON, _ := json.Marshal(matcher.Body)

	switch matcher.MatchMode {
	case "exact":
		return string(reqJSON) == string(expectedJSON)
	case "partial":
		return strings.Contains(string(reqJSON), string(expectedJSON))
	default:
		return string(reqJSON) == string(expectedJSON)
	}
}

// MockMessage implements proto.Message interface methods
func (m *MockMessage) Reset()         { m.Fields = nil }
func (m *MockMessage) String() string { return fmt.Sprintf("%v", m.Fields) }
func (m *MockMessage) ProtoMessage()  {}

// UnmarshalJSON implements json.Unmarshaler
func (m *MockMessage) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.Fields)
}

// MarshalJSON implements json.Marshaler
func (m *MockMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Fields)
}

// GetService returns a service config by name
func (s *Server) GetService(name string) (*ServiceConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, exists := s.services[name]
	return service, exists
}

// ListServices returns all registered service names
func (s *Server) ListServices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.services))
	for name := range s.services {
		names = append(names, name)
	}
	return names
}
