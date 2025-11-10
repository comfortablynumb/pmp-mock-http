package proxy

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Config holds proxy configuration
type Config struct {
	Target       string
	PreserveHost bool
	Timeout      time.Duration
}

// Client handles proxying requests to a backend
type Client struct {
	config     *Config
	httpClient *http.Client
	targetURL  *url.URL
}

// NewClient creates a new proxy client
func NewClient(config *Config) (*Client, error) {
	if config == nil || config.Target == "" {
		return nil, fmt.Errorf("proxy target is required")
	}

	targetURL, err := url.Parse(config.Target)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy target URL: %w", err)
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects, return them to the client
				return http.ErrUseLastResponse
			},
		},
		targetURL: targetURL,
	}, nil
}

// Forward forwards a request to the proxy target
func (c *Client) Forward(w http.ResponseWriter, r *http.Request) error {
	// Build the target URL
	targetURL := *c.targetURL
	targetURL.Path = r.URL.Path
	targetURL.RawQuery = r.URL.RawQuery

	// Create the proxy request
	proxyReq, err := http.NewRequest(r.Method, targetURL.String(), r.Body)
	if err != nil {
		return fmt.Errorf("failed to create proxy request: %w", err)
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Set Host header
	if c.config.PreserveHost {
		proxyReq.Host = r.Host
	} else {
		proxyReq.Host = c.targetURL.Host
	}

	// Add X-Forwarded headers
	if clientIP := getClientIP(r); clientIP != "" {
		proxyReq.Header.Set("X-Forwarded-For", clientIP)
	}
	proxyReq.Header.Set("X-Forwarded-Proto", getScheme(r))
	proxyReq.Header.Set("X-Forwarded-Host", r.Host)

	log.Printf("Proxying %s %s to %s\n", r.Method, r.URL.Path, targetURL.String())

	// Execute the proxy request
	resp, err := c.httpClient.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("proxy request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // cleanup

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying proxy response body: %v\n", err)
		return err
	}

	log.Printf("Proxied response: %d\n", resp.StatusCode)
	return nil
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, get the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}

// getScheme returns the request scheme (http or https)
func getScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}
