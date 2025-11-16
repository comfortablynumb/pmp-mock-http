package graphql

// GraphQLConfig represents GraphQL-specific mock configuration
type GraphQLConfig struct {
	Schema         string                  `yaml:"schema"`          // GraphQL schema definition
	Operations     []GraphQLOperation      `yaml:"operations"`      // Query/Mutation/Subscription operations
	Introspection  bool                    `yaml:"introspection"`   // Enable introspection
	ValidationMode string                  `yaml:"validation_mode"` // strict, permissive, none
	Subscriptions  *SubscriptionConfig     `yaml:"subscriptions"`   // WebSocket subscription config
	Resolvers      map[string]ResolverFunc `yaml:"-"`               // Custom resolver functions
}

// GraphQLOperation represents a mocked GraphQL operation
type GraphQLOperation struct {
	Name          string                 `yaml:"name"`           // Operation name
	Type          string                 `yaml:"type"`           // query, mutation, subscription
	Query         string                 `yaml:"query"`          // GraphQL query/mutation text
	Variables     map[string]interface{} `yaml:"variables"`      // Expected variables
	Response      interface{}            `yaml:"response"`       // Response data
	Errors        []GraphQLError         `yaml:"errors"`         // GraphQL errors
	Extensions    map[string]interface{} `yaml:"extensions"`     // Extensions data
	Template      bool                   `yaml:"template"`       // Use Go templates in response
	MatchMode     string                 `yaml:"match_mode"`     // exact, partial, regex
	VariableMatch map[string]string      `yaml:"variable_match"` // Variable matching rules
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message    string                 `yaml:"message"`
	Path       []interface{}          `yaml:"path,omitempty"`
	Locations  []GraphQLLocation      `yaml:"locations,omitempty"`
	Extensions map[string]interface{} `yaml:"extensions,omitempty"`
}

// GraphQLLocation represents error location in query
type GraphQLLocation struct {
	Line   int `yaml:"line"`
	Column int `yaml:"column"`
}

// SubscriptionConfig represents GraphQL subscription configuration
type SubscriptionConfig struct {
	Events       []SubscriptionEvent `yaml:"events"`        // Events to emit
	Interval     int                 `yaml:"interval"`      // Emission interval in ms
	MaxEvents    int                 `yaml:"max_events"`    // Max events per subscription
	KeepAlive    int                 `yaml:"keep_alive"`    // Keep-alive interval in ms
	Protocol     string              `yaml:"protocol"`      // graphql-ws, graphql-transport-ws
	InitTimeout  int                 `yaml:"init_timeout"`  // Connection init timeout in ms
	CloseOnError bool                `yaml:"close_on_error"` // Close connection on error
}

// SubscriptionEvent represents a subscription event
type SubscriptionEvent struct {
	Data       interface{}            `yaml:"data"`
	Errors     []GraphQLError         `yaml:"errors,omitempty"`
	Extensions map[string]interface{} `yaml:"extensions,omitempty"`
	Delay      int                    `yaml:"delay"` // Delay before emitting in ms
}

// ResolverFunc is a custom resolver function type
type ResolverFunc func(params interface{}) (interface{}, error)

// GraphQLRequest represents an incoming GraphQL request
type GraphQLRequest struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName,omitempty"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	Extensions    map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLResponse represents a GraphQL response
type GraphQLResponse struct {
	Data       interface{}            `json:"data,omitempty"`
	Errors     []GraphQLError         `json:"errors,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// GraphQLBatchRequest represents a batched GraphQL request
type GraphQLBatchRequest []GraphQLRequest

// GraphQLBatchResponse represents a batched GraphQL response
type GraphQLBatchResponse []GraphQLResponse

// IntrospectionQuery is the standard GraphQL introspection query
const IntrospectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`
