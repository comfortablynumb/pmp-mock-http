package graphql

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	gql "github.com/graphql-go/graphql"
)

// Handler handles GraphQL requests
type Handler struct {
	schema        *gql.Schema
	introspection bool
	operations    []GraphQLOperation
	validatorMode string
}

// NewHandler creates a new GraphQL handler
func NewHandler(config *GraphQLConfig) (*Handler, error) {
	var schema *gql.Schema

	if config.Schema != "" {
		// Parse schema if provided
		parsedSchema, err := parseSchema(config.Schema)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GraphQL schema: %w", err)
		}
		schema = parsedSchema
	}

	return &Handler{
		schema:        schema,
		introspection: config.Introspection,
		operations:    config.Operations,
		validatorMode: config.ValidationMode,
	}, nil
}

// ServeHTTP handles GraphQL HTTP requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only accept POST and GET requests
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req GraphQLRequest

	if r.Method == http.MethodPost {
		contentType := r.Header.Get("Content-Type")

		if strings.Contains(contentType, "application/json") {
			// JSON request
			body, err := io.ReadAll(r.Body)
			if err != nil {
				h.sendError(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			// Check for batch request
			if strings.TrimSpace(string(body))[0] == '[' {
				var batchReq GraphQLBatchRequest
				if err := json.Unmarshal(body, &batchReq); err != nil {
					h.sendError(w, "Invalid JSON", http.StatusBadRequest)
					return
				}
				h.handleBatch(w, batchReq)
				return
			}

			if err := json.Unmarshal(body, &req); err != nil {
				h.sendError(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		} else if strings.Contains(contentType, "application/graphql") {
			// Raw GraphQL query
			body, err := io.ReadAll(r.Body)
			if err != nil {
				h.sendError(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			req.Query = string(body)
		} else {
			h.sendError(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
			return
		}
	} else if r.Method == http.MethodGet {
		// GET request with query parameters
		req.Query = r.URL.Query().Get("query")
		req.OperationName = r.URL.Query().Get("operationName")

		if variables := r.URL.Query().Get("variables"); variables != "" {
			if err := json.Unmarshal([]byte(variables), &req.Variables); err != nil {
				h.sendError(w, "Invalid variables JSON", http.StatusBadRequest)
				return
			}
		}
	}

	// Handle introspection query
	if h.introspection && isIntrospectionQuery(req.Query) {
		if h.schema != nil {
			h.executeSchema(w, req)
			return
		}
		h.sendIntrospectionResponse(w)
		return
	}

	// Find matching operation
	response := h.findMatchingOperation(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBatch handles batched GraphQL requests
func (h *Handler) handleBatch(w http.ResponseWriter, requests GraphQLBatchRequest) {
	responses := make(GraphQLBatchResponse, len(requests))

	for i, req := range requests {
		responses[i] = h.findMatchingOperation(req)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// findMatchingOperation finds a matching GraphQL operation
func (h *Handler) findMatchingOperation(req GraphQLRequest) GraphQLResponse {
	for _, op := range h.operations {
		if h.matchesOperation(req, op) {
			return GraphQLResponse{
				Data:       op.Response,
				Errors:     op.Errors,
				Extensions: op.Extensions,
			}
		}
	}

	// No match found
	return GraphQLResponse{
		Errors: []GraphQLError{
			{
				Message: "No matching GraphQL operation found",
			},
		},
	}
}

// matchesOperation checks if a request matches an operation
func (h *Handler) matchesOperation(req GraphQLRequest, op GraphQLOperation) bool {
	// Check operation name
	if req.OperationName != "" && req.OperationName != op.Name {
		return false
	}

	// Check query matching
	switch op.MatchMode {
	case "exact":
		if normalizeQuery(req.Query) != normalizeQuery(op.Query) {
			return false
		}
	case "partial":
		if !strings.Contains(normalizeQuery(req.Query), normalizeQuery(op.Query)) {
			return false
		}
	case "regex":
		// TODO: Implement regex matching
	default:
		// Default to exact matching
		if normalizeQuery(req.Query) != normalizeQuery(op.Query) {
			return false
		}
	}

	// Check variables
	if len(op.Variables) > 0 && !matchVariables(req.Variables, op.Variables) {
		return false
	}

	return true
}

// matchVariables checks if request variables match expected variables
func matchVariables(reqVars, expectedVars map[string]interface{}) bool {
	for key, expectedValue := range expectedVars {
		reqValue, exists := reqVars[key]
		if !exists {
			return false
		}

		// Simple equality check (could be enhanced)
		if fmt.Sprintf("%v", reqValue) != fmt.Sprintf("%v", expectedValue) {
			return false
		}
	}
	return true
}

// normalizeQuery normalizes a GraphQL query by removing extra whitespace
func normalizeQuery(query string) string {
	// Remove comments
	lines := strings.Split(query, "\n")
	var filtered []string
	for _, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			filtered = append(filtered, line)
		}
	}
	query = strings.Join(filtered, "\n")

	// Normalize whitespace
	query = strings.Join(strings.Fields(query), " ")
	return strings.TrimSpace(query)
}

// isIntrospectionQuery checks if the query is an introspection query
func isIntrospectionQuery(query string) bool {
	normalized := normalizeQuery(query)
	return strings.Contains(normalized, "__schema") ||
		strings.Contains(normalized, "__type") ||
		strings.Contains(normalized, "IntrospectionQuery")
}

// executeSchema executes a query against the schema
func (h *Handler) executeSchema(w http.ResponseWriter, req GraphQLRequest) {
	result := gql.Do(gql.Params{
		Schema:         *h.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
	})

	response := GraphQLResponse{
		Data: result.Data,
	}

	if len(result.Errors) > 0 {
		response.Errors = make([]GraphQLError, len(result.Errors))
		for i, err := range result.Errors {
			response.Errors[i] = GraphQLError{
				Message: err.Message,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendIntrospectionResponse sends a default introspection response
func (h *Handler) sendIntrospectionResponse(w http.ResponseWriter) {
	// Basic introspection response
	response := GraphQLResponse{
		Data: map[string]interface{}{
			"__schema": map[string]interface{}{
				"queryType": map[string]interface{}{
					"name": "Query",
				},
				"mutationType": map[string]interface{}{
					"name": "Mutation",
				},
				"subscriptionType": map[string]interface{}{
					"name": "Subscription",
				},
				"types":      []interface{}{},
				"directives": []interface{}{},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendError sends an error response
func (h *Handler) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := GraphQLResponse{
		Errors: []GraphQLError{
			{Message: message},
		},
	}

	json.NewEncoder(w).Encode(response)
}

// parseSchema parses a GraphQL schema string
func parseSchema(schemaStr string) (*gql.Schema, error) {
	// This is a simplified schema parser
	// In a real implementation, you would parse the schema string
	// and build a proper GraphQL schema

	// For now, create a basic schema
	queryType := gql.NewObject(gql.ObjectConfig{
		Name: "Query",
		Fields: gql.Fields{
			"hello": &gql.Field{
				Type: gql.String,
				Resolve: func(p gql.ResolveParams) (interface{}, error) {
					return "world", nil
				},
			},
		},
	})

	schema, err := gql.NewSchema(gql.SchemaConfig{
		Query: queryType,
	})

	return &schema, err
}
