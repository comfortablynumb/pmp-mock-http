package sse

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/template"
	"github.com/dop251/goja"
)

// Handler manages Server-Sent Events streaming
type Handler struct {
	mock             *models.Mock
	templateRenderer *template.Renderer
}

// NewHandler creates a new SSE handler
func NewHandler(mock *models.Mock, templateRenderer *template.Renderer) *Handler {
	return &Handler{
		mock:             mock,
		templateRenderer: templateRenderer,
	}
}

// HandleStream handles an SSE stream
func (h *Handler) HandleStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create flusher to push data immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	log.Printf("SSE: Stream started for %s\n", r.RemoteAddr)

	// Create request data for templates
	requestData := template.NewRequestData(r, "")

	// Send initial retry value if configured
	if h.mock.SSE != nil && h.mock.SSE.Retry > 0 {
		if _, err := fmt.Fprintf(w, "retry: %d\n\n", h.mock.SSE.Retry); err != nil {
			log.Printf("SSE: Error writing retry value: %v\n", err)
			return
		}
		flusher.Flush()
	}

	// Handle JavaScript mode
	if h.mock.SSE != nil && h.mock.SSE.JavaScript != "" {
		h.handleJavaScriptMode(w, flusher, requestData)
		return
	}

	// Handle event sequence mode
	if h.mock.SSE != nil && len(h.mock.SSE.Events) > 0 {
		h.handleEventSequence(w, flusher, r, requestData)
		return
	}

	// Default: send a simple message
	h.sendEvent(w, flusher, "message", "SSE connection established", "", 0)
	log.Println("SSE: No events configured, sent default message")
}

// handleEventSequence sends a sequence of events
func (h *Handler) handleEventSequence(w http.ResponseWriter, flusher http.Flusher, r *http.Request, requestData *template.RequestData) {
	eventsSent := 0
	mode := "cycle" // default mode

	if h.mock.SSE.Mode != "" {
		mode = h.mock.SSE.Mode
	}

	// Set up keep-alive ticker if configured
	var keepAliveTicker *time.Ticker
	var keepAliveStop chan bool
	if h.mock.SSE.KeepAlive > 0 {
		keepAliveTicker = time.NewTicker(time.Duration(h.mock.SSE.KeepAlive) * time.Millisecond)
		keepAliveStop = make(chan bool)
		defer func() {
			keepAliveTicker.Stop()
			close(keepAliveStop)
		}()

		// Start keep-alive goroutine
		go func() {
			for {
				select {
				case <-keepAliveTicker.C:
					if _, err := fmt.Fprintf(w, ": keep-alive\n\n"); err != nil {
						log.Printf("SSE: Error writing keep-alive: %v\n", err)
						return
					}
					flusher.Flush()
				case <-keepAliveStop:
					return
				}
			}
		}()
	}

	// Context for detecting client disconnect
	ctx := r.Context()

	for {
		for _, event := range h.mock.SSE.Events {
			// Check if client disconnected
			select {
			case <-ctx.Done():
				log.Println("SSE: Client disconnected")
				return
			default:
			}

			// Apply delay if specified
			if event.Delay > 0 {
				time.Sleep(time.Duration(event.Delay) * time.Millisecond)
			}

			// Render template if enabled
			data := event.Data
			if event.Template || (h.mock.SSE.Template && event.Template) {
				rendered, err := h.templateRenderer.Render(data, requestData)
				if err != nil {
					log.Printf("SSE: Error rendering event template: %v\n", err)
				} else {
					data = rendered
				}
			}

			// Send the event
			h.sendEvent(w, flusher, event.Event, data, event.ID, event.Retry)

			eventsSent++

			// Check if we should close after this event
			if h.mock.SSE.CloseAfter > 0 && eventsSent >= h.mock.SSE.CloseAfter {
				log.Printf("SSE: Closing after %d events\n", eventsSent)
				return
			}

			// Apply interval between events if configured
			if h.mock.SSE.Interval > 0 {
				time.Sleep(time.Duration(h.mock.SSE.Interval) * time.Millisecond)
			}
		}

		// If mode is "once", stop after sending all events
		if mode == "once" {
			log.Println("SSE: All events sent (once mode), closing stream")
			return
		}

		// In cycle mode, continue from the beginning
		log.Println("SSE: Cycling events")
	}
}

// handleJavaScriptMode handles custom JavaScript logic for SSE
func (h *Handler) handleJavaScriptMode(w http.ResponseWriter, flusher http.Flusher, requestData *template.RequestData) {
	// Create JavaScript VM
	vm := goja.New()

	// Set up console
	if err := vm.Set("console", map[string]interface{}{
		"log": func(args ...interface{}) {
			log.Println("SSE JS:", fmt.Sprint(args...))
		},
	}); err != nil {
		log.Printf("SSE: Error setting console in JavaScript VM: %v\n", err)
		h.sendEvent(w, flusher, "error", fmt.Sprintf("JavaScript setup error: %v", err), "", 0)
		return
	}

	// Create SSE object with send methods
	sseObj := map[string]interface{}{
		"send": func(data string) {
			h.sendEvent(w, flusher, "", data, "", 0)
		},
		"sendEvent": func(eventType, data, id string, retry int) {
			h.sendEvent(w, flusher, eventType, data, id, retry)
		},
		"close": func() {
			// Signal to close the connection
		},
	}
	if err := vm.Set("sse", sseObj); err != nil {
		log.Printf("SSE: Error setting sse object in JavaScript VM: %v\n", err)
		h.sendEvent(w, flusher, "error", fmt.Sprintf("JavaScript setup error: %v", err), "", 0)
		return
	}

	// Set up request object
	if err := vm.Set("request", map[string]interface{}{
		"uri":        requestData.URI,
		"method":     requestData.Method,
		"headers":    requestData.Headers,
		"remoteAddr": requestData.RemoteAddr,
	}); err != nil {
		log.Printf("SSE: Error setting request object in JavaScript VM: %v\n", err)
		h.sendEvent(w, flusher, "error", fmt.Sprintf("JavaScript setup error: %v", err), "", 0)
		return
	}

	// Set up global state object
	if err := vm.Set("global", vm.NewObject()); err != nil {
		log.Printf("SSE: Error setting global object in JavaScript VM: %v\n", err)
		h.sendEvent(w, flusher, "error", fmt.Sprintf("JavaScript setup error: %v", err), "", 0)
		return
	}

	// Sleep function for JavaScript
	if err := vm.Set("sleep", func(ms int) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}); err != nil {
		log.Printf("SSE: Error setting sleep function in JavaScript VM: %v\n", err)
		h.sendEvent(w, flusher, "error", fmt.Sprintf("JavaScript setup error: %v", err), "", 0)
		return
	}

	// Execute JavaScript code
	_, err := vm.RunString(h.mock.SSE.JavaScript)
	if err != nil {
		log.Printf("SSE: JavaScript error: %v\n", err)
		h.sendEvent(w, flusher, "error", fmt.Sprintf("JavaScript error: %v", err), "", 0)
	}
}

// sendEvent sends a single SSE event
func (h *Handler) sendEvent(w http.ResponseWriter, flusher http.Flusher, eventType, data, id string, retry int) {
	// Send event type if specified
	if eventType != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", eventType); err != nil {
			log.Printf("SSE: Error writing event type: %v\n", err)
			return
		}
	}

	// Send ID if specified
	if id != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", id); err != nil {
			log.Printf("SSE: Error writing event ID: %v\n", err)
			return
		}
	}

	// Send retry if specified
	if retry > 0 {
		if _, err := fmt.Fprintf(w, "retry: %d\n", retry); err != nil {
			log.Printf("SSE: Error writing retry: %v\n", err)
			return
		}
	}

	// Send data (can be multiline)
	if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
		log.Printf("SSE: Error writing event data: %v\n", err)
		return
	}

	// Flush to send immediately
	flusher.Flush()

	log.Printf("SSE: Sent event (type=%s, id=%s): %s\n", eventType, id, data)
}
