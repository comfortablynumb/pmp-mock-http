package graphql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// SubscriptionHandler handles GraphQL subscriptions over WebSocket
type SubscriptionHandler struct {
	config   *SubscriptionConfig
	upgrader websocket.Upgrader
	clients  map[*websocket.Conn]*subscriptionClient
	mu       sync.RWMutex
}

// subscriptionClient represents a connected subscription client
type subscriptionClient struct {
	conn          *websocket.Conn
	subscriptions map[string]*subscription
	mu            sync.RWMutex
}

// subscription represents an active subscription
type subscription struct {
	id        string
	operation GraphQLOperation
	stopChan  chan struct{}
}

// WebSocket message types for graphql-ws protocol
const (
	MessageTypeConnectionInit      = "connection_init"
	MessageTypeConnectionAck       = "connection_ack"
	MessageTypeConnectionKeepAlive = "ka"
	MessageTypeStart               = "start"
	MessageTypeStop                = "stop"
	MessageTypeConnectionTerminate = "connection_terminate"
	MessageTypeData                = "data"
	MessageTypeError               = "error"
	MessageTypeComplete            = "complete"
)

// WebSocket message types for graphql-transport-ws protocol
const (
	MessageTypeConnectionInitTransport = "connection_init"
	MessageTypeConnectionAckTransport  = "connection_ack"
	MessageTypePing                    = "ping"
	MessageTypePong                    = "pong"
	MessageTypeSubscribe               = "subscribe"
	MessageTypeNext                    = "next"
	MessageTypeError2                  = "error"
	MessageTypeComplete2               = "complete"
)

// SubscriptionMessage represents a WebSocket message
type SubscriptionMessage struct {
	ID      string                 `json:"id,omitempty"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// NewSubscriptionHandler creates a new subscription handler
func NewSubscriptionHandler(config *SubscriptionConfig) *SubscriptionHandler {
	if config.Protocol == "" {
		config.Protocol = "graphql-ws"
	}

	return &SubscriptionHandler{
		config: config,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins (configure as needed)
			},
			Subprotocols: []string{"graphql-ws", "graphql-transport-ws"},
		},
		clients: make(map[*websocket.Conn]*subscriptionClient),
	}
}

// ServeHTTP handles WebSocket upgrade for subscriptions
func (h *SubscriptionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &subscriptionClient{
		conn:          conn,
		subscriptions: make(map[string]*subscription),
	}

	h.mu.Lock()
	h.clients[conn] = client
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, conn)
		h.mu.Unlock()
		conn.Close()
	}()

	// Handle messages
	h.handleClient(client)
}

// handleClient handles messages from a WebSocket client
func (h *SubscriptionHandler) handleClient(client *subscriptionClient) {
	initTimeout := time.Duration(h.config.InitTimeout) * time.Millisecond
	if initTimeout == 0 {
		initTimeout = 10 * time.Second
	}

	initialized := false
	initTimer := time.NewTimer(initTimeout)

	// Start keep-alive if configured
	if h.config.KeepAlive > 0 {
		go h.sendKeepAlive(client)
	}

	for {
		select {
		case <-initTimer.C:
			if !initialized {
				client.conn.Close()
				return
			}
		default:
			var msg SubscriptionMessage
			err := client.conn.ReadJSON(&msg)
			if err != nil {
				return
			}

			switch msg.Type {
			case MessageTypeConnectionInit, MessageTypeConnectionInitTransport:
				initialized = true
				initTimer.Stop()
				h.sendConnectionAck(client)

			case MessageTypeStart, MessageTypeSubscribe:
				h.handleSubscribe(client, msg)

			case MessageTypeStop:
				h.handleUnsubscribe(client, msg.ID)

			case MessageTypeConnectionTerminate:
				return

			case MessageTypePing:
				h.sendPong(client, msg.Payload)
			}
		}
	}
}

// handleSubscribe handles subscription start
func (h *SubscriptionHandler) handleSubscribe(client *subscriptionClient, msg SubscriptionMessage) {
	// Parse subscription request
	var req GraphQLRequest
	if payload := msg.Payload; payload != nil {
		if query, ok := payload["query"].(string); ok {
			req.Query = query
		}
		if opName, ok := payload["operationName"].(string); ok {
			req.OperationName = opName
		}
		if vars, ok := payload["variables"].(map[string]interface{}); ok {
			req.Variables = vars
		}
	}

	// Find matching subscription operation
	// In a real implementation, match against configured operations
	sub := &subscription{
		id:       msg.ID,
		stopChan: make(chan struct{}),
	}

	client.mu.Lock()
	client.subscriptions[msg.ID] = sub
	client.mu.Unlock()

	// Start emitting events
	go h.emitEvents(client, sub)
}

// handleUnsubscribe handles subscription stop
func (h *SubscriptionHandler) handleUnsubscribe(client *subscriptionClient, id string) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if sub, exists := client.subscriptions[id]; exists {
		close(sub.stopChan)
		delete(client.subscriptions, id)
		h.sendComplete(client, id)
	}
}

// emitEvents emits subscription events
func (h *SubscriptionHandler) emitEvents(client *subscriptionClient, sub *subscription) {
	interval := time.Duration(h.config.Interval) * time.Millisecond
	if interval == 0 {
		interval = 1000 * time.Millisecond
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	eventCount := 0
	maxEvents := h.config.MaxEvents

	for i := 0; i < len(h.config.Events); i++ {
		select {
		case <-sub.stopChan:
			return
		case <-ticker.C:
			if maxEvents > 0 && eventCount >= maxEvents {
				h.sendComplete(client, sub.id)
				return
			}

			event := h.config.Events[i]

			// Apply delay if specified
			if event.Delay > 0 {
				time.Sleep(time.Duration(event.Delay) * time.Millisecond)
			}

			// Send event
			response := GraphQLResponse{
				Data:       event.Data,
				Errors:     event.Errors,
				Extensions: event.Extensions,
			}

			h.sendData(client, sub.id, response)
			eventCount++

			// Loop back to start if needed
			if i == len(h.config.Events)-1 {
				i = -1 // Will be incremented to 0
			}
		}
	}

	h.sendComplete(client, sub.id)
}

// sendConnectionAck sends connection acknowledgment
func (h *SubscriptionHandler) sendConnectionAck(client *subscriptionClient) {
	msg := SubscriptionMessage{
		Type: MessageTypeConnectionAck,
	}
	client.conn.WriteJSON(msg)
}

// sendKeepAlive sends periodic keep-alive messages
func (h *SubscriptionHandler) sendKeepAlive(client *subscriptionClient) {
	ticker := time.NewTicker(time.Duration(h.config.KeepAlive) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		msg := SubscriptionMessage{
			Type: MessageTypeConnectionKeepAlive,
		}
		if err := client.conn.WriteJSON(msg); err != nil {
			return
		}
	}
}

// sendData sends subscription data
func (h *SubscriptionHandler) sendData(client *subscriptionClient, id string, response GraphQLResponse) {
	msgType := MessageTypeData
	if h.config.Protocol == "graphql-transport-ws" {
		msgType = MessageTypeNext
	}

	msg := SubscriptionMessage{
		ID:   id,
		Type: msgType,
		Payload: map[string]interface{}{
			"data":   response.Data,
			"errors": response.Errors,
		},
	}

	if err := client.conn.WriteJSON(msg); err != nil {
		if h.config.CloseOnError {
			client.conn.Close()
		}
	}
}

// sendComplete sends subscription complete
func (h *SubscriptionHandler) sendComplete(client *subscriptionClient, id string) {
	msg := SubscriptionMessage{
		ID:   id,
		Type: MessageTypeComplete,
	}
	client.conn.WriteJSON(msg)
}

// sendPong sends pong response
func (h *SubscriptionHandler) sendPong(client *subscriptionClient, payload map[string]interface{}) {
	msg := SubscriptionMessage{
		Type:    MessageTypePong,
		Payload: payload,
	}
	client.conn.WriteJSON(msg)
}

// Broadcast sends an event to all subscribed clients
func (h *SubscriptionHandler) Broadcast(event SubscriptionEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	response := GraphQLResponse{
		Data:       event.Data,
		Errors:     event.Errors,
		Extensions: event.Extensions,
	}

	for _, client := range h.clients {
		client.mu.RLock()
		for id := range client.subscriptions {
			h.sendData(client, id, response)
		}
		client.mu.RUnlock()
	}
}

// GetActiveConnectionCount returns the number of active connections
func (h *SubscriptionHandler) GetActiveConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// CloseAll closes all active connections
func (h *SubscriptionHandler) CloseAll() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	var lastErr error
	for conn := range h.clients {
		if err := conn.Close(); err != nil {
			lastErr = fmt.Errorf("error closing connection: %w", err)
		}
	}

	h.clients = make(map[*websocket.Conn]*subscriptionClient)
	return lastErr
}
