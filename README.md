# PMP Mock HTTP

A flexible and powerful HTTP mock server written in Go. Mock any HTTP API endpoint with support for regex matching, hot-reloading, and easy YAML configuration.

Part of the Poor Man's Platform (PMP) ecosystem - if a dependency of your app uses HTTP, we can mock it for you.

## Features

- ✅ **HTTP Server**: Listens on configurable port (default: 8083)
- ✅ **UI Dashboard**: Real-time web dashboard on port 8081 to monitor requests, matches, and responses
- ✅ **YAML Configuration**: Define mocks in simple YAML files
- ✅ **Hot Reloading**: Automatically reload mocks when files change
- ✅ **Recursive Loading**: Load mock files from nested subdirectories
- ✅ **Plugin System**: Load mocks from external git repositories
- ✅ **Advanced Matching**: Match requests by URI, HTTP Method, Headers, and Body
- ✅ **Regex Support**: Use regular expressions for flexible matching on any field
- ✅ **JSON Path Matching**: Use GJSON paths to match specific JSON fields in request bodies
- ✅ **JavaScript Evaluation**: Write custom JavaScript logic for complex matching and dynamic responses
- ✅ **Global State**: Persistent JavaScript state for stateful mock APIs (CRUD, sessions, rate limiting)
- ✅ **Priority System**: Control which mocks match first
- ✅ **Response Control**: Configure status codes, headers, body, and delays
- ✅ **Template Responses**: Use Go templates to generate dynamic responses with access to request data
- ✅ **Fake Data Generation**: Built-in template functions for generating realistic fake data (names, emails, UUIDs, etc.)
- ✅ **HTTP Callbacks**: Trigger HTTP callbacks to external URLs when mocks match (webhooks)
- ✅ **Sequential Responses**: Return different responses in sequence (cycle or once mode)
- ✅ **Request Recording**: Record real requests/responses and export as mocks
- ✅ **Proxy Passthrough**: Forward unmatched requests to a backend server
- ✅ **TLS Support**: Serve mocks over HTTPS with custom certificates

## Installation

### From Source

```bash
git clone https://github.com/comfortablynumb/pmp-mock-http.git
cd pmp-mock-http
go build -o pmp-mock-http ./cmd/server
```

### Docker

The Docker image comes with the following default environment variables:
- `PORT=8080` (mock server)
- `UI_PORT=8081` (dashboard)
- `MOCKS_DIR=/mocks`
- `PLUGINS_DIR=/plugins`

```bash
# Build the Docker image
docker build -t pmp-mock-http .

# Run with default settings
docker run -p 8080:8080 -p 8081:8081 -v $(pwd)/mocks:/mocks pmp-mock-http

# Run with custom port using environment variables
docker run -p 9000:9000 -p 8081:8081 -e PORT=9000 -v $(pwd)/mocks:/mocks pmp-mock-http

# Run with custom mocks directory using environment variables
docker run -p 8080:8080 -p 8081:8081 -e MOCKS_DIR=/custom/mocks -v $(pwd)/mocks:/custom/mocks pmp-mock-http

# Override environment variables with flags
docker run -p 9000:9000 -p 8081:8081 -e PORT=8080 -v $(pwd)/mocks:/mocks pmp-mock-http --port 9000
```

**Note:** The Docker image includes `git` for any git-based operations you might need.

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

### Configuration

Configuration values can be set via environment variables or command-line flags. **Command-line flags take precedence over environment variables.**

#### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8083 | HTTP server port |
| `UI_PORT` | 8081 | UI dashboard port |
| `MOCKS_DIR` | mocks | Directory containing mock YAML files |
| `PLUGINS_DIR` | plugins | Directory to store plugin repositories |
| `PLUGINS` | "" | Comma-separated list of git repository URLs to clone as plugins |
| `PLUGIN_INCLUDE_ONLY` | "" | Space-separated list of subdirectories from pmp-mock-http to include |
| `PROXY_TARGET` | "" | Target URL for proxy passthrough (e.g., "http://api.example.com") |
| `PROXY_PRESERVE_HOST` | false | Preserve the original Host header when proxying |
| `PROXY_TIMEOUT` | 30 | Proxy request timeout in seconds |
| `TLS_ENABLED` | false | Enable TLS/HTTPS |
| `TLS_CERT_FILE` | "" | Path to TLS certificate file |
| `TLS_KEY_FILE` | "" | Path to TLS private key file |

#### Command Line Flags

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `-port` | `PORT` | HTTP server port |
| `-ui-port` | `UI_PORT` | UI dashboard port |
| `-mocks-dir` | `MOCKS_DIR` | Directory containing mock YAML files |
| `-plugins-dir` | `PLUGINS_DIR` | Directory to store plugin repositories |
| `-plugins` | `PLUGINS` | Comma-separated list of git repository URLs to clone as plugins |
| `-plugin-include-only` | `PLUGIN_INCLUDE_ONLY` | Space-separated list of subdirectories from pmp-mock-http to include |
| `-proxy-target` | `PROXY_TARGET` | Target URL for proxy passthrough |
| `-proxy-preserve-host` | `PROXY_PRESERVE_HOST` | Preserve original Host header when proxying |
| `-proxy-timeout` | `PROXY_TIMEOUT` | Proxy request timeout in seconds |
| `-tls` | `TLS_ENABLED` | Enable TLS/HTTPS |
| `-tls-cert` | `TLS_CERT_FILE` | Path to TLS certificate file |
| `-tls-key` | `TLS_KEY_FILE` | Path to TLS private key file |

**Examples:**

```bash
# Using environment variables
export PORT=9000
export MOCKS_DIR=/custom/mocks
./pmp-mock-http

# Using command-line flags (overrides environment variables)
PORT=9000 ./pmp-mock-http -port 8080  # Will use port 8080

# Docker with environment variables
docker run -e PORT=9000 -e MOCKS_DIR=/mocks -v $(pwd)/mocks:/mocks ironedge/pmp-mock-http
```

### UI Dashboard

The server automatically starts a web dashboard on port 8081 that provides real-time monitoring of all HTTP requests. Access it at **http://localhost:8081**

**Features:**
- Real-time request tracking with 2-second auto-refresh
- Visual color-coded indicators (green=matched, red=unmatched)
- Complete request details: method, URI, headers, body
- Response inspection: status code, headers, body
- Shows which mock matched (if any)
- Statistics: total, matched, and unmatched requests
- Clear all logs button

```bash
# Custom UI port
./pmp-mock-http --ui-port 9000
```

### Proxy Passthrough Mode

When a request doesn't match any mock, you can optionally forward it to a real backend server. This is useful for:

- **Partial mocking**: Mock some endpoints while proxying others to the real API
- **Development**: Test against a real backend for non-mocked endpoints
- **Gradual migration**: Progressively add mocks without breaking existing flows

#### Basic Proxy Usage

```bash
# Forward unmatched requests to a backend
./pmp-mock-http --proxy-target http://api.example.com

# Preserve the original Host header
./pmp-mock-http --proxy-target http://api.example.com --proxy-preserve-host

# Set custom timeout (default: 30 seconds)
./pmp-mock-http --proxy-target http://api.example.com --proxy-timeout 60

# Using environment variables
export PROXY_TARGET=http://api.example.com
export PROXY_PRESERVE_HOST=true
./pmp-mock-http
```

#### How Proxy Works

1. Request arrives at the mock server
2. Mocks are checked in priority order
3. If a mock matches → mock response is returned
4. If no mock matches:
   - **With proxy enabled** → request is forwarded to the proxy target
   - **Without proxy** → 404 response is returned

#### Proxy Headers

The proxy automatically adds standard forwarding headers:
- `X-Forwarded-For`: Client IP address
- `X-Forwarded-Proto`: Original request protocol (http/https)
- `X-Forwarded-Host`: Original Host header

#### Docker with Proxy

```bash
# Proxy to a backend service
docker run -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/mocks:/mocks \
  -e PROXY_TARGET=http://backend-api:8080 \
  pmp-mock-http

# With host preservation
docker run -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/mocks:/mocks \
  -e PROXY_TARGET=http://backend-api:8080 \
  -e PROXY_PRESERVE_HOST=true \
  pmp-mock-http
```

### TLS/HTTPS Support

Enable TLS to serve mocks over HTTPS with your own certificates.

#### Generating Self-Signed Certificates (for testing)

```bash
# Generate a self-signed certificate
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
```

#### Basic TLS Usage

```bash
# Enable TLS with certificate files
./pmp-mock-http --tls --tls-cert cert.pem --tls-key key.pem

# Server will now listen on https://localhost:8083
curl -k https://localhost:8083/api/test

# Using environment variables
export TLS_ENABLED=true
export TLS_CERT_FILE=cert.pem
export TLS_KEY_FILE=key.pem
./pmp-mock-http
```

#### Docker with TLS

```bash
# Mount certificate files
docker run -p 8443:8080 -p 8081:8081 \
  -v $(pwd)/mocks:/mocks \
  -v $(pwd)/certs:/certs \
  -e TLS_ENABLED=true \
  -e TLS_CERT_FILE=/certs/cert.pem \
  -e TLS_KEY_FILE=/certs/key.pem \
  pmp-mock-http

# Access via HTTPS
curl -k https://localhost:8443/api/test
```

#### Production TLS

For production, use certificates from a trusted Certificate Authority (CA) like Let's Encrypt:

```bash
# Using Let's Encrypt certificates
./pmp-mock-http \
  --tls \
  --tls-cert /etc/letsencrypt/live/yourdomain.com/fullchain.pem \
  --tls-key /etc/letsencrypt/live/yourdomain.com/privkey.pem \
  --port 443
```

#### Combining Proxy and TLS

You can use both features together:

```bash
# HTTPS mock server with proxy fallback
./pmp-mock-http \
  --tls \
  --tls-cert cert.pem \
  --tls-key key.pem \
  --proxy-target https://api.example.com \
  --port 443
```

### Template Responses

Response bodies can use Go templates to generate dynamic content based on the incoming request. Enable templates by setting `template: true` in the response configuration.

#### Accessing Request Data

Templates have access to the following request data:

- `.Method` - HTTP method (GET, POST, etc.)
- `.URI` - Full request URI
- `.Path` - URL path
- `.RawQuery` - Query string
- `.Headers` - Request headers as a map
- `.Body` - Request body as a string
- `.RemoteAddr` - Client IP address

#### Fake Data Functions

Built-in functions for generating realistic fake data:

**Identifiers:**
- `uuid` - Generate a UUID
- `randomString <length>` - Random alphanumeric string
- `randomInt <min> <max>` - Random integer in range
- `randomFloat <min> <max>` - Random float in range
- `randomBool` - Random boolean

**Names:**
- `firstName` - Random first name
- `lastName` - Random last name
- `fullName` - Random full name
- `username` - Random username
- `email` - Random email address

**Addresses:**
- `city` - Random city
- `country` - Random country
- `zipCode` - Random ZIP code
- `address` - Random street address

**Business:**
- `company` - Random company name
- `jobTitle` - Random job title

**Internet:**
- `ipAddress` - Random IP address
- `domain` - Random domain name
- `url` - Random URL

**Time:**
- `now` - Current time (time.Time)
- `timestamp` - Current Unix timestamp
- `date` - Current date (YYYY-MM-DD)
- `datetime` - Current datetime (RFC3339)

**String utilities:**
- `upper` - Convert to uppercase
- `lower` - Convert to lowercase

#### Template Example

```yaml
mocks:
  - name: "Dynamic User Response"
    request:
      uri: "/api/users"
      method: "POST"
    response:
      status_code: 201
      headers:
        Content-Type: "application/json"
      template: true
      body: |
        {
          "id": "{{ uuid }}",
          "username": "{{ username }}",
          "email": "{{ email }}",
          "created_at": "{{ datetime }}",
          "request_from": "{{ .RemoteAddr }}",
          "original_body": {{ .Body | printf "%q" }}
        }
```

More examples available in `pmp-mock-http/examples/templates.yaml`.

### HTTP Callbacks (Webhooks)

Trigger HTTP callbacks to external URLs when a mock matches. This is useful for:

- Simulating webhooks and async notifications
- Testing event-driven architectures
- Integrating with external services during testing

#### Callback Configuration

```yaml
response:
  status_code: 200
  body: "..."
  callback:
    url: "http://localhost:8082/webhook/endpoint"
    method: "POST"  # Optional, defaults to POST
    headers:
      Content-Type: "application/json"
      X-Custom-Header: "value"
    body: |
      {
        "event": "order.created",
        "data": "..."
      }
```

Callback bodies support template syntax just like response bodies:

```yaml
callback:
  url: "http://localhost:8082/webhook"
  body: |
    {
      "event_id": "{{ uuid }}",
      "timestamp": {{ timestamp }},
      "user": "{{ fullName }}",
      "original_request": {
        "method": "{{ .Method }}",
        "uri": "{{ .URI }}",
        "body": {{ .Body | printf "%q" }}
      }
    }
```

#### Callback Behavior

- Callbacks are executed **asynchronously** (non-blocking)
- Callbacks are executed **after** the response delay (if any)
- Callback failures are logged but don't affect the mock response
- Callbacks have a 30-second timeout

#### Callback Example

```yaml
mocks:
  - name: "Order with Webhook"
    request:
      uri: "/api/orders"
      method: "POST"
    response:
      status_code: 202
      headers:
        Content-Type: "application/json"
      body: |
        {
          "status": "accepted",
          "message": "Order received"
        }
      callback:
        url: "http://localhost:8082/webhook/order-received"
        method: "POST"
        headers:
          Content-Type: "application/json"
          X-Event-Type: "order.created"
        body: |
          {
            "event": "order.created",
            "order_id": "{{ uuid }}",
            "timestamp": {{ timestamp }},
            "customer_ip": "{{ .RemoteAddr }}"
          }
```

More examples available in `pmp-mock-http/examples/callbacks.yaml`.

### Sequential Responses

Return different responses in sequence for the same endpoint. Perfect for simulating multi-step processes, status changes, and progressive workflows.

#### Cycle Mode (default)

Responses repeat from the beginning after the last one:

```yaml
mocks:
  - name: "Status Polling"
    request:
      uri: "/api/task/status"
      method: "GET"
    response:
      sequence:
        - status_code: 202
          body: '{"status": "pending"}'
        - status_code: 200
          body: '{"status": "processing"}'
        - status_code: 200
          body: '{"status": "completed"}'
      sequence_mode: "cycle"  # Repeats: pending -> processing -> completed -> pending ...
```

#### Once Mode

Responses stop at the last one:

```yaml
mocks:
  - name: "Account Setup"
    request:
      uri: "/api/onboarding/status"
      method: "GET"
    response:
      sequence:
        - status_code: 200
          body: '{"step": "email_verification"}'
        - status_code: 200
          body: '{"step": "profile_setup"}'
        - status_code: 200
          body: '{"step": "complete"}'
      sequence_mode: "once"  # Stays at "complete" after 3rd call
```

#### Use Cases

- **Status polling**: Simulate async operations (pending → processing → done)
- **Onboarding flows**: Multi-step user journeys
- **Rate limiting**: Return success N times, then errors
- **A/B testing**: Alternate between different responses
- **Degradation testing**: Gradually increase error rates

**Sequence Features:**
- Each response can have different status codes, headers, body, and delay
- Supports templates in sequence responses
- Sequence counter resets on mock file reload
- Thread-safe for concurrent requests

More examples available in `mocks/sequence-examples.yaml`.

### Request Recording & Replay

Record real API traffic and convert it into reusable mocks. Perfect for capturing production API behavior and creating test fixtures.

#### Starting Recording

```bash
# Start recording
curl -X POST http://localhost:8083/__recording/start

# Make requests to your API
curl http://localhost:8083/api/users/123
curl -X POST http://localhost:8083/api/orders -d '{"item":"widget"}'

# Stop recording
curl -X POST http://localhost:8083/__recording/stop
```

#### Exporting Recordings

```bash
# Export as YAML (default)
curl http://localhost:8083/__recording/export > recorded-mocks.yaml

# Export as JSON
curl http://localhost:8083/__recording/export?format=json > recorded-mocks.json

# Group by URI to create sequences
curl "http://localhost:8083/__recording/export?group=uri" > recorded-sequences.yaml
```

#### Recording Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/__recording/start` | POST | Start recording requests/responses |
| `/__recording/stop` | POST | Stop recording |
| `/__recording/status` | GET | Get recording status and count |
| `/__recording/clear` | POST | Clear all recordings |
| `/__recording/export` | GET | Export as mocks (YAML or JSON) |
| `/__recording/list` | GET | List all recorded requests |

#### Export Options

**Query Parameters:**
- `format=json` - Export as JSON (default: YAML)
- `group=uri` - Group multiple recordings of same endpoint into sequences

**Grouping Example:**

Without grouping, 3 requests to `/api/status` create 3 separate mocks.

With `group=uri`, they're combined into a single mock with a sequence:

```yaml
mocks:
  - name: "Recorded: GET /api/status (sequence)"
    request:
      uri: "/api/status"
      method: "GET"
    response:
      sequence:
        - status_code: 202
          body: '{"status": "pending"}'
        - status_code: 200
          body: '{"status": "processing"}'
        - status_code: 200
          body: '{"status": "completed"}'
      sequence_mode: "cycle"
```

#### Use Cases

- **Production API capture**: Record real API behavior for testing
- **Test fixtures**: Convert manual test runs into automated mocks
- **API documentation**: Generate mock examples from real traffic
- **Regression testing**: Capture before/after behavior for comparisons
- **Quick mock creation**: Skip writing YAML by hand

**Recording Tips:**
- Recording is thread-safe and works with concurrent requests
- Recordings persist until cleared or server restarts
- Only matched mock responses are recorded (not 404s or proxy responses)
- Large request/response bodies are captured in full

### Plugins System

The plugin system allows you to load mock configurations from external git repositories. This is useful for:

- **Sharing mock configurations** across multiple projects
- **Versioning mock libraries** in separate repositories
- **Organizing mocks** by service or domain
- **Collaborating** on mock definitions with teams

#### Basic Plugin Usage

```bash
# Load mocks from a single plugin repository
./pmp-mock-http --plugins "https://github.com/user/api-mocks.git"

# Load mocks from multiple plugin repositories
./pmp-mock-http --plugins "https://github.com/user/api-mocks.git,https://github.com/org/service-mocks.git"

# Specify custom plugins directory
./pmp-mock-http --plugins-dir /tmp/plugins --plugins "https://github.com/user/mocks.git"
```

#### How Plugins Work

1. **Clone/Update**: On startup, the server clones each plugin repository to the plugins directory
2. **Auto-Update**: If a plugin already exists, it's updated with `git pull`
3. **Load Mocks**: All YAML files in the plugin repositories' `pmp-mock-http` directory are loaded as mocks
4. **Hot-Reload**: Plugin directories are watched for changes, just like the main mocks directory
5. **Priority**: Mocks from plugins are merged with local mocks, with priority determining match order

#### Plugin Structure

**IMPORTANT**: Plugin repositories **must** contain a `pmp-mock-http` directory where all mock YAML files reside:

```
my-api-mocks-repo/
└── pmp-mock-http/          # Required directory
    ├── openai/             # OpenAI API mocks
    │   ├── chat.yaml
    │   └── completions.yaml
    ├── stripe/             # Stripe API mocks
    │   ├── customers.yaml
    │   └── payments.yaml
    └── github/             # GitHub API mocks
        └── repos.yaml
```

#### Selective Loading with Include Filter

Use the `--plugin-include-only` flag to load only specific subdirectories from plugin repositories:

```bash
# Only load OpenAI and Stripe mocks from the plugin
./pmp-mock-http \
  --plugins "https://github.com/user/api-mocks.git" \
  --plugin-include-only "openai stripe"

# Using environment variable
export PLUGIN_INCLUDE_ONLY="openai stripe"
./pmp-mock-http --plugins "https://github.com/user/api-mocks.git"
```

This is useful when:
- You only need mocks for specific services
- You want to reduce memory usage by not loading all mocks
- You want to avoid conflicts with local mock definitions

#### Docker with Plugins

```bash
# Run with plugins using environment variable
docker run -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/mocks:/mocks \
  -e PLUGINS="https://github.com/user/api-mocks.git" \
  pmp-mock-http

# With custom plugins directory and multiple plugins
docker run -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/mocks:/mocks \
  -v $(pwd)/plugins:/custom-plugins \
  -e PLUGINS_DIR=/custom-plugins \
  -e PLUGINS="https://github.com/user/mocks1.git,https://github.com/user/mocks2.git" \
  pmp-mock-http

# Using flags (override environment variables)
docker run -p 8080:8080 -p 8081:8081 \
  -v $(pwd)/mocks:/mocks \
  pmp-mock-http \
  --plugins "https://github.com/user/api-mocks.git"
```

**Note**: The Docker image includes git, so plugin cloning works out of the box.

#### Plugin Best Practices

- **Organize by service**: Create separate plugin repositories for each external service
- **Use semantic versioning**: Tag plugin releases for stable versions
- **Document mock behavior**: Include README files in plugin repositories
- **Test plugins independently**: Each plugin can have its own test suite
- **Share via private repos**: Private git repositories work with SSH authentication

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

### JSON Path Matching (GJSON)

Match specific fields in JSON request bodies using [GJSON path syntax](https://github.com/tidwall/gjson#path-syntax). This provides a more precise and readable way to match JSON data compared to regex.

#### Basic JSON Path Example

```yaml
mocks:
  - name: "Create Admin User"
    request:
      uri: "/api/users"
      method: "POST"
      json_path:
        - path: "user.role"
          value: "admin"
          regex: false
    response:
      status_code: 201
      body: '{"message": "Admin user created"}'
```

This matches requests like:
```json
{
  "user": {
    "role": "admin",
    "name": "John"
  }
}
```

#### Multiple JSON Path Matchers

You can specify multiple path matchers - all must match for the request to match:

```yaml
mocks:
  - name: "Premium Subscription"
    request:
      uri: "/api/subscribe"
      method: "POST"
      json_path:
        - path: "plan.type"
          value: "premium"
          regex: false
        - path: "payment.method"
          value: "credit_card"
          regex: false
    response:
      status_code: 200
      body: '{"subscription_id": "sub_123", "status": "active"}'
```

#### JSON Path with Regex

Combine GJSON paths with regex patterns:

```yaml
mocks:
  - name: "Valid Email Format"
    request:
      uri: "/api/register"
      method: "POST"
      json_path:
        - path: "email"
          value: "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"
          regex: true
    response:
      status_code: 200
      body: '{"message": "Registration successful"}'
```

#### Advanced GJSON Features

GJSON supports powerful path syntax including:

- **Nested paths**: `user.profile.email`
- **Array access**: `items.0.name` (first item's name)
- **Array queries**: `items.#(price>100).name` (names of items over $100)
- **Wildcards**: `users.*.email` (all user emails)

See the [GJSON documentation](https://github.com/tidwall/gjson#path-syntax) for complete path syntax.

### JavaScript Evaluation

For complex matching logic or dynamic responses, use JavaScript code to evaluate requests. The JavaScript code receives a `request` object and must return an object with `matches` (boolean) and optionally a custom `response`.

#### Request Object

The JavaScript code has access to a `request` object with:

```javascript
{
  uri: "/api/endpoint",        // Request URI path
  method: "POST",              // HTTP method
  headers: {                   // Request headers
    "Content-Type": "application/json",
    "X-API-Key": "secret"
  },
  body: "{\"user\": \"data\"}"  // Request body as string
}
```

#### Simple Matching Example

```yaml
mocks:
  - name: "Admin Access Only"
    request:
      uri: "/api/admin"
      method: "POST"
      javascript: |
        (function() {
          var body = JSON.parse(request.body);
          return {
            matches: body.user && body.user.role === "admin",
            response: null
          };
        })()
    response:
      status_code: 200
      body: '{"message": "Welcome, administrator"}'
```

#### Dynamic Response Example

Return a custom response based on request data:

```yaml
mocks:
  - name: "Dynamic Pricing"
    request:
      uri: "/api/pricing"
      method: "POST"
      javascript: |
        (function() {
          var body = JSON.parse(request.body);
          var isPremium = body.tier === "premium";

          return {
            matches: true,
            response: {
              status_code: 200,
              headers: {
                "Content-Type": "application/json"
              },
              body: JSON.stringify({
                tier: body.tier,
                price: isPremium ? 79.99 : 99.99,
                discount: isPremium ? 20 : 0
              })
            }
          };
        })()
    response:
      status_code: 500
      body: "This fallback is not used when JS returns a custom response"
```

#### Complex Validation Example

```yaml
mocks:
  - name: "Order Validation"
    request:
      uri: "/api/orders"
      method: "POST"
      javascript: |
        (function() {
          var body = JSON.parse(request.body);
          var hasItems = body.items && body.items.length > 0;
          var hasValidTotal = body.total && body.total > 0;

          if (!hasItems || !hasValidTotal) {
            return {
              matches: true,
              response: {
                status_code: 400,
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({
                  error: "Invalid order",
                  missing: !hasItems ? "items" : "total"
                })
              }
            };
          }

          return {
            matches: true,
            response: {
              status_code: 201,
              body: JSON.stringify({
                order_id: "ord_12345",
                status: "confirmed"
              })
            }
          };
        })()
    response:
      status_code: 500
      body: "Fallback"
```

#### Authentication Example

Check headers and URI together:

```yaml
mocks:
  - name: "API Key Auth"
    request:
      javascript: |
        (function() {
          var hasValidKey = request.headers["X-API-Key"] === "secret-key-123";
          var isProtected = request.uri.indexOf("/api/protected") === 0;

          if (isProtected && !hasValidKey) {
            return {
              matches: true,
              response: {
                status_code: 401,
                body: JSON.stringify({error: "Unauthorized"})
              }
            };
          }

          return { matches: isProtected && hasValidKey, response: null };
        })()
    response:
      status_code: 200
      body: '{"message": "Access granted"}'
```

**Note**: When a mock has a `javascript` field, other matching criteria (uri, method, headers, body, json_path) are ignored. The JavaScript code has full control over matching.

### Global State (Stateful Mocks)

JavaScript mocks have access to a persistent `global` object that maintains state across requests. This enables creating stateful API simulations like in-memory databases, session management, and rate limiting.

#### Basic Example: In-Memory User Database

```yaml
mocks:
  # Create user - stores in global state
  - name: "Create User"
    request:
      uri: "/api/users"
      method: "POST"
      javascript: |
        (function() {
          var body = JSON.parse(request.body);

          // Initialize global state
          if (!global.users) {
            global.users = [];
            global.nextUserId = 1;
          }

          // Create and store user
          var newUser = {
            id: global.nextUserId++,
            name: body.name,
            email: body.email
          };
          global.users.push(newUser);

          return {
            matches: true,
            response: {
              status_code: 201,
              body: JSON.stringify(newUser)
            }
          };
        })()
    response:
      status_code: 500

  # Get all users - retrieves from global state
  - name: "Get Users"
    request:
      uri: "/api/users"
      method: "GET"
      javascript: |
        (function() {
          var users = global.users || [];
          return {
            matches: true,
            response: {
              status_code: 200,
              body: JSON.stringify(users)
            }
          };
        })()
    response:
      status_code: 500
```

**Usage:**
1. POST to `/api/users` with `{"name": "John", "email": "john@example.com"}`
2. GET `/api/users` returns the array of created users
3. State persists across requests until the server is restarted

#### Session Management Example

```yaml
mocks:
  - name: "Login"
    request:
      uri: "/api/login"
      method: "POST"
      javascript: |
        (function() {
          var body = JSON.parse(request.body);

          if (!global.sessions) {
            global.sessions = {};
            global.sessionCounter = 0;
          }

          // Simple auth check
          if (body.username === "admin" && body.password === "secret") {
            var sessionId = "sess_" + (global.sessionCounter++);
            global.sessions[sessionId] = {
              username: body.username,
              createdAt: new Date().toISOString()
            };

            return {
              matches: true,
              response: {
                status_code: 200,
                headers: {"Set-Cookie": "sessionId=" + sessionId},
                body: JSON.stringify({sessionId: sessionId})
              }
            };
          }

          return {
            matches: true,
            response: {
              status_code: 401,
              body: JSON.stringify({error: "Invalid credentials"})
            }
          };
        })()
    response:
      status_code: 500

  - name: "Protected Resource"
    request:
      uri: "/api/profile"
      method: "GET"
      javascript: |
        (function() {
          var cookie = request.headers["Cookie"] || "";
          var match = cookie.match(/sessionId=([^;]+)/);

          if (!match || !global.sessions || !global.sessions[match[1]]) {
            return {
              matches: true,
              response: {
                status_code: 401,
                body: JSON.stringify({error: "Not authenticated"})
              }
            };
          }

          var session = global.sessions[match[1]];
          return {
            matches: true,
            response: {
              status_code: 200,
              body: JSON.stringify({username: session.username})
            }
          };
        })()
    response:
      status_code: 500
```

#### Rate Limiting Example

```yaml
mocks:
  - name: "Rate Limited API"
    request:
      uri: "/api/limited"
      method: "GET"
      javascript: |
        (function() {
          if (!global.requestCounts) {
            global.requestCounts = {};
          }

          var clientIp = request.headers["X-Forwarded-For"] || "default";
          global.requestCounts[clientIp] = (global.requestCounts[clientIp] || 0) + 1;

          if (global.requestCounts[clientIp] > 10) {
            return {
              matches: true,
              response: {
                status_code: 429,
                headers: {"X-RateLimit-Remaining": "0"},
                body: JSON.stringify({error: "Too many requests"})
              }
            };
          }

          return {
            matches: true,
            response: {
              status_code: 200,
              headers: {"X-RateLimit-Remaining": String(10 - global.requestCounts[clientIp])},
              body: JSON.stringify({message: "Success"})
            }
          };
        })()
    response:
      status_code: 500
```

**Important Notes:**
- Global state persists across all JavaScript-enabled mocks
- State is thread-safe for concurrent requests
- State survives mock file reloads (hot-reload preserves state)
- State is cleared when the server restarts
- Use `global.propertyName` to store and access data
- Complex data structures (arrays, objects) are fully supported

See `mocks/stateful-examples.yaml` for complete CRUD and session examples.

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
- `jsonpath-examples.yaml`: GJSON path matching examples
- `javascript-examples.yaml`: JavaScript evaluation examples
- `stateful-examples.yaml`: Global state and stateful API simulations (CRUD, sessions, rate limiting)
- `apis/external-service.yaml`: Examples in a subdirectory

## Development

### Build

```bash
go build -o pmp-mock-http ./cmd/server
```

### Run

```bash
go run main.go
```

### Dependencies

- [fsnotify](https://github.com/fsnotify/fsnotify) - File system notifications
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing
- [gjson](https://github.com/tidwall/gjson) - JSON path matching
- [goja](https://github.com/dop251/goja) - JavaScript runtime for Go

## License

See [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
