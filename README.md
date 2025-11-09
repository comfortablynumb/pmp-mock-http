# PMP Mock HTTP

A flexible and powerful HTTP mock server written in Go. Mock any HTTP API endpoint with support for regex matching, hot-reloading, and easy YAML configuration.

Part of the Poor Man's Platform (PMP) ecosystem - if a dependency of your app uses HTTP, we can mock it for you.

## Features

- ✅ **HTTP Server**: Listens on configurable port (default: 8083)
- ✅ **YAML Configuration**: Define mocks in simple YAML files
- ✅ **Hot Reloading**: Automatically reload mocks when files change
- ✅ **Recursive Loading**: Load mock files from nested subdirectories
- ✅ **Advanced Matching**: Match requests by URI, HTTP Method, Headers, and Body
- ✅ **Regex Support**: Use regular expressions for flexible matching on any field
- ✅ **Priority System**: Control which mocks match first
- ✅ **Response Control**: Configure status codes, headers, body, and delays

## Installation

### From Source

```bash
git clone https://github.com/comfortablynumb/pmp-mock-http.git
cd pmp-mock-http
go build -o pmp-mock-http
```

## Usage

### Basic Usage

```bash
# Start the server with defaults (port 8083, mocks directory: ./mocks)
./pmp-mock-http

# Specify custom port
./pmp-mock-http -port 9000

# Specify custom mocks directory
./pmp-mock-http -mocks-dir /path/to/mocks

# Both custom port and directory
./pmp-mock-http -port 9000 -mocks-dir /path/to/mocks
```

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-port` | 8083 | HTTP server port |
| `-mocks-dir` | mocks | Directory containing mock YAML files |

## Mock Configuration

### YAML Structure

Mocks are defined in YAML files with the following structure:

```yaml
mocks:
  - name: "Mock Name"
    priority: 10              # Higher priority = matched first (optional, default: 0)
    request:
      uri: "/api/endpoint"    # URI to match
      method: "GET"           # HTTP method to match
      headers:                # Headers to match (optional)
        Content-Type: "application/json"
      body: "request body"    # Body content to match (optional)
      regex:                  # Enable regex matching for each field
        uri: false
        method: false
        headers: false
        body: false
    response:
      status_code: 200        # HTTP status code
      headers:                # Response headers (optional)
        Content-Type: "application/json"
      body: |                 # Response body (optional)
        {"message": "success"}
      delay: 0                # Response delay in milliseconds (optional)
```

### Simple Example

```yaml
mocks:
  - name: "Get User"
    request:
      uri: "/api/users/123"
      method: "GET"
      regex:
        uri: false
        method: false
    response:
      status_code: 200
      headers:
        Content-Type: "application/json"
      body: |
        {
          "id": 123,
          "name": "John Doe"
        }
```

### Regex Matching Examples

#### Match any user ID

```yaml
mocks:
  - name: "Get Any User"
    priority: 5
    request:
      uri: "^/api/users/\\d+$"
      method: "GET"
      regex:
        uri: true
        method: false
    response:
      status_code: 200
      body: '{"id": 999, "name": "Generic User"}'
```

#### Match Authorization header pattern

```yaml
mocks:
  - name: "Authorized Request"
    request:
      uri: "/api/protected"
      method: "GET"
      headers:
        Authorization: "^Bearer [A-Za-z0-9\\-._~+/]+=*$"
      regex:
        uri: false
        method: false
        headers: true
    response:
      status_code: 200
      body: '{"message": "Access granted"}'
```

#### Match request body with regex

```yaml
mocks:
  - name: "Create User with Email"
    request:
      uri: "/api/users"
      method: "POST"
      body: '.*"email"\\s*:\\s*"[^"]+".*'
      regex:
        uri: false
        method: false
        body: true
    response:
      status_code: 201
      body: '{"id": 124, "message": "User created"}'
```

### Priority System

When multiple mocks could match a request, the mock with the **highest priority** is chosen first. This allows you to create:

- Specific matches with high priority
- Catch-all/fallback matches with low priority

```yaml
mocks:
  # Specific match - high priority
  - name: "Get Specific User 123"
    priority: 10
    request:
      uri: "/api/users/123"
      method: "GET"
    response:
      status_code: 200
      body: '{"id": 123, "name": "John Doe"}'

  # Generic match - lower priority
  - name: "Get Any User"
    priority: 5
    request:
      uri: "^/api/users/\\d+$"
      method: "GET"
      regex:
        uri: true
    response:
      status_code: 200
      body: '{"id": 999, "name": "Generic User"}'
```

### Response Delays

Simulate slow APIs by adding a delay (in milliseconds):

```yaml
mocks:
  - name: "Slow API"
    request:
      uri: "/api/slow"
      method: "GET"
    response:
      status_code: 200
      body: "This took 2 seconds"
      delay: 2000
```

## Project Structure

```
pmp-mock-http/
├── main.go                    # Application entry point
├── internal/
│   ├── models/
│   │   └── mock.go           # Mock specification data structures
│   ├── loader/
│   │   └── loader.go         # YAML file loader
│   ├── watcher/
│   │   └── watcher.go        # File system watcher
│   ├── matcher/
│   │   └── matcher.go        # Request matching logic
│   └── server/
│       └── server.go         # HTTP server implementation
└── mocks/                     # Default mocks directory
    ├── basic-examples.yaml
    ├── regex-examples.yaml
    └── apis/
        └── external-service.yaml
```

## How It Works

1. **Startup**: The server loads all YAML files from the mocks directory (including subdirectories)
2. **Request Handling**: When an HTTP request arrives:
   - The server reads the request (URI, method, headers, body)
   - Mocks are checked in priority order
   - The first matching mock's response is returned
   - If no mock matches, a 404 is returned
3. **Hot Reloading**: The file watcher monitors the mocks directory
   - New files are automatically loaded
   - Modified files trigger a reload
   - Deleted files are removed from active mocks

## Testing

### Test with curl

```bash
# Start the server
./pmp-mock-http

# Test a simple GET request
curl http://localhost:8083/api/users/123

# Test with headers
curl -H "Authorization: Bearer abc123" http://localhost:8083/api/protected

# Test POST with body
curl -X POST http://localhost:8083/api/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Jane Doe", "email": "jane@example.com"}'
```

## Examples

The `mocks/` directory contains several example files demonstrating various features:

- `basic-examples.yaml`: Simple exact matching examples
- `regex-examples.yaml`: Advanced regex matching patterns
- `apis/external-service.yaml`: Examples in a subdirectory

## Development

### Build

```bash
go build -o pmp-mock-http
```

### Run

```bash
go run main.go
```

### Dependencies

- [fsnotify](https://github.com/fsnotify/fsnotify) - File system notifications
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
