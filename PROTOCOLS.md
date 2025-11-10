# Protocol Support in PMP Mock HTTP

This document describes the advanced protocol support in PMP Mock HTTP Server, including WebSocket, Server-Sent Events (SSE), HTTP/2, and HTTP/3.

## Table of Contents

- [WebSocket Support](#websocket-support)
- [Server-Sent Events (SSE)](#server-sent-events-sse)
- [HTTP/2 Support](#http2-support)
- [HTTP/3 Support](#http3-support)

---

## WebSocket Support

PMP Mock HTTP supports mocking WebSocket connections with multiple modes of operation.

### Configuration

Add `protocol: "websocket"` to your mock configuration:

```yaml
mocks:
  - name: "My WebSocket Mock"
    protocol: "websocket"
    request:
      uri: "/ws/endpoint"
      method: "GET"
    websocket:
      mode: "echo"  # or "sequence", "broadcast", "javascript"
      # Additional WebSocket-specific configuration
```

### WebSocket Modes

#### 1. Echo Mode

Echoes received messages back to the client.

```yaml
websocket:
  mode: "echo"
  on_connect: "Welcome! Send me a message."
```

**Use Cases:**
- Simple echo servers
- Connection testing
- Message format validation

#### 2. Sequence Mode

Sends a predefined sequence of messages to the client.

```yaml
websocket:
  mode: "sequence"
  messages:
    - type: "text"
      data: '{"status": "connecting"}'
      delay: 0
    - type: "text"
      data: '{"status": "ready"}'
      delay: 1000
  interval: 500        # Time between messages (ms)
  close_after: 0       # Close after N messages (0 = keep open)
```

**Use Cases:**
- Simulating multi-step processes
- Testing client message handling
- Progress updates

#### 3. Broadcast Mode

Broadcasts received messages to all connected clients (chat room pattern).

```yaml
websocket:
  mode: "broadcast"
  max_connections: 100  # Limit concurrent connections
```

**Use Cases:**
- Chat applications
- Multi-user collaboration
- Real-time notifications

#### 4. JavaScript Mode

Custom WebSocket logic using JavaScript.

```yaml
websocket:
  mode: "javascript"
  javascript: |
    (function() {
      console.log("Connection established");

      // Send initial message
      connection.send("Welcome!");

      // Handle incoming messages
      onMessage(function(message) {
        if (message === "ping") {
          connection.send("pong");
        } else {
          connection.sendJSON({
            received: message,
            timestamp: new Date().toISOString()
          });
        }
      });
    })();
```

**Available JavaScript API:**
- `connection.send(message)` - Send text message
- `connection.sendJSON(object)` - Send JSON message
- `connection.close()` - Close connection
- `onMessage(callback)` - Register message handler
- `console.log(...)` - Log to server console
- `global` - Access to global state object
- `request` - Access to request data (uri, method, headers, remoteAddr)

**Use Cases:**
- Complex business logic
- Stateful interactions
- Custom protocols

### WebSocket Configuration Options

| Option | Type | Description |
|--------|------|-------------|
| `mode` | string | Operating mode: "echo", "sequence", "broadcast", "javascript" |
| `messages` | array | Messages to send in sequence mode |
| `interval` | int | Time between messages in ms |
| `close_after` | int | Close after N messages (0 = keep open) |
| `javascript` | string | JavaScript code for custom logic |
| `on_connect` | string | Message to send on connection |
| `template` | bool | Enable Go templates in messages |
| `max_connections` | int | Max concurrent connections (0 = unlimited) |

### Template Support

WebSocket messages support Go templates for dynamic content:

```yaml
websocket:
  mode: "sequence"
  template: true
  on_connect: 'Connected from {{.RemoteAddr}} at {{now}}'
  messages:
    - type: "text"
      data: '{"id": "{{uuid}}", "time": "{{timestamp}}"}'
      template: true
```

**Available template functions:** All standard PMP Mock HTTP template functions (uuid, randomString, now, timestamp, firstName, email, etc.)

### Examples

See `examples/websocket/` directory for complete examples:
- `echo.yaml` - Simple echo server
- `sequence.yaml` - Sequential message delivery
- `broadcast.yaml` - Chat room broadcast
- `javascript.yaml` - Custom JavaScript handler
- `template.yaml` - Dynamic templates

---

## Server-Sent Events (SSE)

Server-Sent Events allow servers to push real-time updates to clients over HTTP.

### Configuration

Add `protocol: "sse"` to your mock configuration:

```yaml
mocks:
  - name: "My SSE Stream"
    protocol: "sse"
    request:
      uri: "/sse/endpoint"
      method: "GET"
    sse:
      events:
        - event: "message"
          data: '{"status": "active"}'
          id: "1"
      mode: "cycle"  # or "once"
```

### SSE Modes

#### 1. Once Mode

Send events once and close the stream.

```yaml
sse:
  events:
    - event: "notification"
      data: '{"message": "Task complete"}'
      delay: 1000
  mode: "once"
  close_after: 1  # Close after 1 event
```

**Use Cases:**
- Single notifications
- Progress completion
- Event-driven updates

#### 2. Cycle Mode

Continuously cycle through events (default).

```yaml
sse:
  events:
    - event: "price"
      data: '{"symbol": "AAPL", "price": 150.25}'
      delay: 0
    - event: "price"
      data: '{"symbol": "GOOGL", "price": 2800.50}'
      delay: 500
  mode: "cycle"
  interval: 1000      # Time between cycles
  keep_alive: 15000   # Keep-alive comment interval
```

**Use Cases:**
- Real-time data feeds
- Stock tickers
- Live metrics

#### 3. JavaScript Mode

Custom SSE logic using JavaScript.

```yaml
sse:
  javascript: |
    (function() {
      console.log("SSE stream started");

      for (var i = 1; i <= 5; i++) {
        sleep(1000);
        sse.sendEvent("counter", JSON.stringify({
          count: i,
          timestamp: new Date().toISOString()
        }), "evt-" + i, 0);
      }

      sse.send("Stream complete");
    })();
```

**Available JavaScript API:**
- `sse.send(data)` - Send data event
- `sse.sendEvent(type, data, id, retry)` - Send event with full control
- `sleep(ms)` - Wait for milliseconds
- `console.log(...)` - Log to server console
- `global` - Access to global state object
- `request` - Access to request data

**Use Cases:**
- Complex event generation
- Dynamic data sources
- Stateful streams

### SSE Configuration Options

| Option | Type | Description |
|--------|------|-------------|
| `events` | array | Events to send |
| `mode` | string | "once" or "cycle" |
| `interval` | int | Time between events in ms |
| `retry` | int | Client retry interval in ms |
| `keep_alive` | int | Keep-alive comment interval in ms |
| `close_after` | int | Close after N events (0 = unlimited) |
| `template` | bool | Enable Go templates |
| `javascript` | string | JavaScript code for custom logic |

### Event Configuration

Each event in the `events` array can have:

| Field | Type | Description |
|-------|------|-------------|
| `event` | string | Event type (optional) |
| `data` | string | Event data |
| `id` | string | Event ID (optional) |
| `retry` | int | Retry interval for this event (optional) |
| `delay` | int | Delay before sending in ms |
| `template` | bool | Enable template for this event |

### Template Support

SSE events support Go templates:

```yaml
sse:
  template: true
  events:
    - event: "user"
      data: '{"name": "{{firstName}} {{lastName}}", "email": "{{email}}"}'
      template: true
```

### Examples

See `examples/sse/` directory for complete examples:
- `simple.yaml` - Basic SSE stream
- `realtime.yaml` - Real-time data feed
- `notifications.yaml` - Notification system
- `template.yaml` - Dynamic templates
- `javascript.yaml` - Custom JavaScript logic

---

## HTTP/2 Support

HTTP/2 is automatically enabled when using TLS.

### Starting with HTTP/2

```bash
# Start server with TLS (HTTP/2 automatically enabled)
./pmp-mock-http --tls --tls-cert cert.pem --tls-key key.pem
```

Or using environment variables:

```bash
export TLS_ENABLED=true
export TLS_CERT_FILE=cert.pem
export TLS_KEY_FILE=key.pem
./pmp-mock-http
```

### Features

- **Multiplexing** - Multiple requests over single connection
- **Server Push** - Not currently supported (requires additional configuration)
- **Header Compression** - HPACK compression
- **Binary Protocol** - Efficient binary framing
- **Backward Compatible** - Falls back to HTTP/1.1 for non-supporting clients

### Testing HTTP/2

```bash
# Using curl with HTTP/2
curl --http2 https://localhost:8083/api/test

# Check protocol version
curl -I --http2 https://localhost:8083/api/test
```

---

## HTTP/3 Support

HTTP/3 uses QUIC protocol for improved performance and reliability.

### Starting with HTTP/3

```bash
# HTTP/3 only
./pmp-mock-http --http3 --tls --tls-cert cert.pem --tls-key key.pem

# Dual-stack (HTTP/2 + HTTP/3)
./pmp-mock-http --dual-stack --tls --tls-cert cert.pem --tls-key key.pem
```

Or using environment variables:

```bash
# HTTP/3 only
export HTTP3_ENABLED=true
export TLS_ENABLED=true
export TLS_CERT_FILE=cert.pem
export TLS_KEY_FILE=key.pem
./pmp-mock-http

# Dual-stack
export DUAL_STACK=true
export TLS_ENABLED=true
export TLS_CERT_FILE=cert.pem
export TLS_KEY_FILE=key.pem
./pmp-mock-http
```

### Features

- **QUIC Protocol** - UDP-based transport
- **0-RTT Connection** - Faster connection establishment
- **Better Loss Recovery** - Independent stream recovery
- **Connection Migration** - Survive network changes
- **Built-in Encryption** - TLS 1.3 required

### Dual-Stack Mode

Dual-stack mode runs both HTTP/2 (over TLS/TCP) and HTTP/3 (over QUIC/UDP) simultaneously on the same port. Clients can negotiate which protocol to use.

**Benefits:**
- Maximum compatibility
- Automatic protocol selection
- Graceful fallback
- Performance optimization

### Testing HTTP/3

```bash
# Using curl with HTTP/3 (requires curl with HTTP/3 support)
curl --http3 https://localhost:8083/api/test

# Check protocol version
curl -I --http3 https://localhost:8083/api/test
```

**Note:** HTTP/3 support requires a curl build with QUIC/HTTP3 enabled.

### Generating Test Certificates

For development, you can generate self-signed certificates:

```bash
# Generate certificate and key
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem \
  -days 365 -nodes -subj "/CN=localhost"
```

---

## Protocol Selection Priority

When multiple protocols are available:

1. Client tries HTTP/3 (if advertised via Alt-Svc header)
2. Falls back to HTTP/2 (if TLS connection succeeds)
3. Falls back to HTTP/1.1 (if HTTP/2 negotiation fails)

---

## Command-Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--tls` | Enable TLS/HTTPS with HTTP/2 | false |
| `--tls-cert` | Path to TLS certificate file | "" |
| `--tls-key` | Path to TLS private key file | "" |
| `--http3` | Enable HTTP/3 with QUIC | false |
| `--dual-stack` | Enable HTTP/2 + HTTP/3 | false |

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TLS_ENABLED` | Enable TLS/HTTPS | false |
| `TLS_CERT_FILE` | TLS certificate file path | "" |
| `TLS_KEY_FILE` | TLS key file path | "" |
| `HTTP3_ENABLED` | Enable HTTP/3 | false |
| `DUAL_STACK` | Enable dual-stack mode | false |

---

## Best Practices

### WebSocket
- Use `max_connections` to prevent resource exhaustion
- Implement proper error handling in JavaScript mode
- Use `close_after` for finite sequences
- Monitor connection count in production

### SSE
- Set appropriate `retry` intervals for clients
- Use `keep_alive` for long-lived connections
- Handle client disconnections gracefully
- Implement `close_after` for finite streams

### HTTP/2
- Always use TLS in production
- Monitor multiplexed connection usage
- Test with HTTP/2-specific load patterns

### HTTP/3
- Test with multiple network conditions
- Implement fallback to HTTP/2
- Monitor QUIC connection statistics
- Use dual-stack for maximum compatibility

---

## Troubleshooting

### WebSocket Issues

**Connection fails:**
- Check that `protocol: "websocket"` is set
- Verify WebSocket upgrade headers
- Check firewall/proxy WebSocket support

**Messages not received:**
- Verify message format in logs
- Check `interval` and `delay` settings
- Test JavaScript code syntax

### SSE Issues

**Stream doesn't start:**
- Verify client accepts `text/event-stream`
- Check that `protocol: "sse"` is set
- Review server logs for errors

**Events not received:**
- Check `interval` settings
- Verify event data format
- Test with simple curl: `curl -N http://localhost:8083/sse/endpoint`

### HTTP/3 Issues

**Connection fails:**
- Ensure UDP port is open
- Verify TLS certificates are valid
- Check client HTTP/3 support
- Try dual-stack mode for debugging

**Performance issues:**
- Check network UDP throughput
- Verify QUIC isn't blocked
- Monitor packet loss rates

---

## Examples

All protocol examples are available in the `examples/` directory:

```
examples/
├── websocket/
│   ├── echo.yaml
│   ├── sequence.yaml
│   ├── broadcast.yaml
│   ├── javascript.yaml
│   └── template.yaml
└── sse/
    ├── simple.yaml
    ├── realtime.yaml
    ├── notifications.yaml
    ├── template.yaml
    └── javascript.yaml
```

To run examples:

```bash
# WebSocket examples
./pmp-mock-http --mocks-dir examples/websocket

# SSE examples
./pmp-mock-http --mocks-dir examples/sse

# With HTTP/3
./pmp-mock-http --mocks-dir examples/websocket --dual-stack \
  --tls --tls-cert cert.pem --tls-key key.pem
```

---

## Additional Resources

- [WebSocket Protocol (RFC 6455)](https://tools.ietf.org/html/rfc6455)
- [Server-Sent Events Specification](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [HTTP/2 Specification (RFC 7540)](https://tools.ietf.org/html/rfc7540)
- [HTTP/3 Specification (RFC 9114)](https://tools.ietf.org/html/rfc9114)
- [QUIC Protocol](https://www.chromium.org/quic/)

---

## License

See main project LICENSE file.
