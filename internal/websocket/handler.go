package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/template"
	"github.com/dop251/goja"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for mock server
	},
}

// Handler manages WebSocket connections and message handling
type Handler struct {
	mock             *models.Mock
	templateRenderer *template.Renderer
	connections      map[*websocket.Conn]bool
	mu               sync.RWMutex
	broadcast        chan []byte
}

// NewHandler creates a new WebSocket handler
func NewHandler(mock *models.Mock, templateRenderer *template.Renderer) *Handler {
	h := &Handler{
		mock:             mock,
		templateRenderer: templateRenderer,
		connections:      make(map[*websocket.Conn]bool),
		broadcast:        make(chan []byte, 256),
	}

	// Start broadcast handler if in broadcast mode
	if mock.WebSocket != nil && mock.WebSocket.Mode == "broadcast" {
		go h.handleBroadcast()
	}

	return h
}

// HandleConnection handles a WebSocket upgrade and connection
func (h *Handler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	// Check max connections
	if h.mock.WebSocket != nil && h.mock.WebSocket.MaxConnections > 0 {
		h.mu.RLock()
		connCount := len(h.connections)
		h.mu.RUnlock()

		if connCount >= h.mock.WebSocket.MaxConnections {
			http.Error(w, "Max connections reached", http.StatusServiceUnavailable)
			log.Printf("WebSocket: Max connections reached (%d)\n", h.mock.WebSocket.MaxConnections)
			return
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	log.Printf("WebSocket: Connection established from %s\n", r.RemoteAddr)

	// Register connection
	h.mu.Lock()
	h.connections[conn] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.connections, conn)
		h.mu.Unlock()
		if err := conn.Close(); err != nil {
			log.Printf("WebSocket: Error closing connection: %v\n", err)
		}
		log.Printf("WebSocket: Connection closed from %s\n", r.RemoteAddr)
	}()

	// Create request data for templates
	requestData := template.NewRequestData(r, "")

	// Send on-connect message if configured
	if h.mock.WebSocket != nil && h.mock.WebSocket.OnConnect != "" {
		message := h.mock.WebSocket.OnConnect
		if h.mock.WebSocket.Template {
			rendered, err := h.templateRenderer.Render(message, requestData)
			if err != nil {
				log.Printf("WebSocket: Error rendering on_connect template: %v\n", err)
			} else {
				message = rendered
			}
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			log.Printf("WebSocket: Error sending on_connect message: %v\n", err)
			return
		}
	}

	// Handle different modes
	if h.mock.WebSocket == nil {
		// No WebSocket config - just echo mode by default
		h.handleEchoMode(conn, requestData)
		return
	}

	switch h.mock.WebSocket.Mode {
	case "echo":
		h.handleEchoMode(conn, requestData)
	case "sequence":
		h.handleSequenceMode(conn, requestData)
	case "broadcast":
		h.handleBroadcastMode(conn, requestData)
	case "javascript":
		h.handleJavaScriptMode(conn, requestData)
	default:
		h.handleEchoMode(conn, requestData)
	}
}

// handleEchoMode echoes received messages back to the client
func (h *Handler) handleEchoMode(conn *websocket.Conn, requestData *template.RequestData) {
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v\n", err)
			}
			break
		}

		log.Printf("WebSocket: Received message: %s\n", string(message))

		// Echo the message back
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Printf("WebSocket write error: %v\n", err)
			break
		}
	}
}

// handleSequenceMode sends a sequence of predefined messages
func (h *Handler) handleSequenceMode(conn *websocket.Conn, requestData *template.RequestData) {
	if h.mock.WebSocket == nil || len(h.mock.WebSocket.Messages) == 0 {
		log.Println("WebSocket: No messages configured for sequence mode")
		return
	}

	messagesSent := 0

	// Send messages in sequence
	for _, msg := range h.mock.WebSocket.Messages {
		// Apply delay if specified
		if msg.Delay > 0 {
			time.Sleep(time.Duration(msg.Delay) * time.Millisecond)
		}

		// Render template if enabled
		data := msg.Data
		if msg.Template || h.mock.WebSocket.Template {
			rendered, err := h.templateRenderer.Render(data, requestData)
			if err != nil {
				log.Printf("WebSocket: Error rendering message template: %v\n", err)
			} else {
				data = rendered
			}
		}

		// Determine message type
		msgType := websocket.TextMessage
		if msg.Type == "binary" {
			msgType = websocket.BinaryMessage
		}

		// Send message
		if err := conn.WriteMessage(msgType, []byte(data)); err != nil {
			log.Printf("WebSocket: Error sending message: %v\n", err)
			return
		}

		log.Printf("WebSocket: Sent message (%s): %s\n", msg.Type, data)
		messagesSent++

		// Check if we should close after this message
		if h.mock.WebSocket.CloseAfter > 0 && messagesSent >= h.mock.WebSocket.CloseAfter {
			log.Printf("WebSocket: Closing after %d messages\n", messagesSent)
			return
		}

		// Apply interval between messages if not the last message
		if h.mock.WebSocket.Interval > 0 {
			time.Sleep(time.Duration(h.mock.WebSocket.Interval) * time.Millisecond)
		}
	}

	// Keep connection open and read messages to prevent errors
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handleBroadcastMode handles broadcast to all connected clients
func (h *Handler) handleBroadcastMode(conn *websocket.Conn, requestData *template.RequestData) {
	// Read messages and broadcast to all connections
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		log.Printf("WebSocket: Broadcasting message: %s\n", string(message))

		// Broadcast to all connections
		h.mu.RLock()
		for c := range h.connections {
			if c != conn { // Don't echo back to sender
				if err := c.WriteMessage(messageType, message); err != nil {
					log.Printf("WebSocket broadcast error: %v\n", err)
				}
			}
		}
		h.mu.RUnlock()
	}
}

// handleBroadcast manages the broadcast channel
func (h *Handler) handleBroadcast() {
	for message := range h.broadcast {
		h.mu.RLock()
		for conn := range h.connections {
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket broadcast error: %v\n", err)
			}
		}
		h.mu.RUnlock()
	}
}

// handleJavaScriptMode handles custom JavaScript logic
func (h *Handler) handleJavaScriptMode(conn *websocket.Conn, requestData *template.RequestData) {
	if h.mock.WebSocket == nil || h.mock.WebSocket.JavaScript == "" {
		log.Println("WebSocket: No JavaScript configured for javascript mode")
		h.handleEchoMode(conn, requestData)
		return
	}

	// Create JavaScript VM
	vm := goja.New()

	// Set up global objects
	if err := vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			log.Println("WebSocket JS:", fmt.Sprint(args...))
		},
	}); err != nil {
		log.Printf("WebSocket: Error setting console in JavaScript VM: %v\n", err)
		return
	}

	// Create connection object with send method
	connObj := map[string]interface{}{
		"send": func(message string) error {
			return conn.WriteMessage(websocket.TextMessage, []byte(message))
		},
		"sendJSON": func(data interface{}) error {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return err
			}
			return conn.WriteMessage(websocket.TextMessage, jsonData)
		},
		"close": func() error {
			return conn.Close()
		},
	}
	if err := vm.Set("connection", connObj); err != nil {
		log.Printf("WebSocket: Error setting connection object in JavaScript VM: %v\n", err)
		return
	}

	// Set up request object
	if err := vm.Set("request", map[string]interface{}{
		"uri":        requestData.URI,
		"method":     requestData.Method,
		"headers":    requestData.Headers,
		"remoteAddr": requestData.RemoteAddr,
	}); err != nil {
		log.Printf("WebSocket: Error setting request object in JavaScript VM: %v\n", err)
		return
	}

	// Set up global state object (shared across all connections)
	if err := vm.Set("global", vm.NewObject()); err != nil {
		log.Printf("WebSocket: Error setting global object in JavaScript VM: %v\n", err)
		return
	}

	// Message handler
	messageHandler := make(chan []byte, 256)

	// Start goroutine to read messages
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				close(messageHandler)
				break
			}
			messageHandler <- message
		}
	}()

	// Set up onMessage callback
	var onMessageCallback func(string)
	if err := vm.Set("onMessage", func(callback func(string)) {
		onMessageCallback = callback
	}); err != nil {
		log.Printf("WebSocket: Error setting onMessage callback in JavaScript VM: %v\n", err)
		return
	}

	// Execute JavaScript initialization code
	_, err := vm.RunString(h.mock.WebSocket.JavaScript)
	if err != nil {
		log.Printf("WebSocket: JavaScript error: %v\n", err)
		return
	}

	// Handle incoming messages
	for message := range messageHandler {
		if onMessageCallback != nil {
			onMessageCallback(string(message))
		}
	}
}

// GetConnectionCount returns the current number of connections
func (h *Handler) GetConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}
