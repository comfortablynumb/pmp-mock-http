package callback

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
	"github.com/comfortablynumb/pmp-mock-http/internal/template"
)

// Executor handles executing callbacks
type Executor struct {
	client   *http.Client
	renderer *template.Renderer
}

// NewExecutor creates a new callback executor
func NewExecutor() *Executor {
	return &Executor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		renderer: template.NewRenderer(),
	}
}

// Execute executes a callback asynchronously
func (e *Executor) Execute(callback *models.Callback, requestData *template.RequestData) {
	if callback == nil || callback.URL == "" {
		return
	}

	// Execute callback in a goroutine to not block the response
	go e.executeCallback(callback, requestData)
}

func (e *Executor) executeCallback(callback *models.Callback, requestData *template.RequestData) {
	method := callback.Method
	if method == "" {
		method = "POST"
	}

	// Render the callback body if it's a template
	body := callback.Body
	if body != "" {
		rendered, err := e.renderer.Render(body, requestData)
		if err != nil {
			log.Printf("Error rendering callback body template: %v\n", err)
			return
		}
		body = rendered
	}

	// Create the request
	req, err := http.NewRequest(method, callback.URL, bytes.NewBufferString(body))
	if err != nil {
		log.Printf("Error creating callback request: %v\n", err)
		return
	}

	// Set headers
	for key, value := range callback.Headers {
		req.Header.Set(key, value)
	}

	// Set default Content-Type if not specified
	if req.Header.Get("Content-Type") == "" && body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute the callback
	log.Printf("Executing callback to %s %s\n", method, callback.URL)
	resp, err := e.client.Do(req)
	if err != nil {
		log.Printf("Error executing callback: %v\n", err)
		return
	}
	defer resp.Body.Close() //nolint:errcheck // cleanup

	log.Printf("Callback completed with status: %d\n", resp.StatusCode)

	if resp.StatusCode >= 400 {
		log.Printf("Warning: callback returned error status code: %d\n", resp.StatusCode)
	}
}

// ExecuteSync executes a callback synchronously (useful for testing)
func (e *Executor) ExecuteSync(callback *models.Callback, requestData *template.RequestData) error {
	if callback == nil || callback.URL == "" {
		return fmt.Errorf("callback is nil or URL is empty")
	}

	method := callback.Method
	if method == "" {
		method = "POST"
	}

	// Render the callback body if it's a template
	body := callback.Body
	if body != "" {
		rendered, err := e.renderer.Render(body, requestData)
		if err != nil {
			return fmt.Errorf("error rendering callback body template: %w", err)
		}
		body = rendered
	}

	// Create the request
	req, err := http.NewRequest(method, callback.URL, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("error creating callback request: %w", err)
	}

	// Set headers
	for key, value := range callback.Headers {
		req.Header.Set(key, value)
	}

	// Set default Content-Type if not specified
	if req.Header.Get("Content-Type") == "" && body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute the callback
	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("error executing callback: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // cleanup

	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback returned error status code: %d", resp.StatusCode)
	}

	return nil
}
