package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pmp_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pmp_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	httpRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pmp_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)

	httpResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pmp_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path", "status"},
	)

	// Mock metrics
	mockMatchesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pmp_mock_matches_total",
			Help: "Total number of mock matches",
		},
		[]string{"mock_name"},
	)

	mockMatchFailuresTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "pmp_mock_match_failures_total",
			Help: "Total number of requests that failed to match any mock",
		},
	)

	// WebSocket metrics
	websocketConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pmp_websocket_connections_active",
			Help: "Number of active WebSocket connections",
		},
	)

	websocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pmp_websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"direction"}, // sent, received
	)

	// SSE metrics
	sseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pmp_sse_connections_active",
			Help: "Number of active SSE connections",
		},
	)

	sseEventsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "pmp_sse_events_total",
			Help: "Total number of SSE events sent",
		},
	)

	// Proxy metrics
	proxyRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pmp_proxy_requests_total",
			Help: "Total number of proxy requests",
		},
		[]string{"status"},
	)

	// Recorder metrics
	recordedRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "pmp_recorded_requests_total",
			Help: "Total number of recorded requests",
		},
	)
)

// MetricsMiddleware wraps an HTTP handler with metrics collection
func MetricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code and size
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Record request size
		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
		}

		// Call the next handler
		next(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)

		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
		httpResponseSize.WithLabelValues(r.Method, r.URL.Path, status).Observe(float64(wrapped.size))
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}

// Metric recording functions

// RecordMockMatch records a successful mock match
func RecordMockMatch(mockName string) {
	mockMatchesTotal.WithLabelValues(mockName).Inc()
}

// RecordMockMatchFailure records a failed mock match
func RecordMockMatchFailure() {
	mockMatchFailuresTotal.Inc()
}

// RecordWebSocketConnection records WebSocket connection changes
func RecordWebSocketConnection(delta int) {
	websocketConnectionsActive.Add(float64(delta))
}

// RecordWebSocketMessage records a WebSocket message
func RecordWebSocketMessage(direction string) {
	websocketMessagesTotal.WithLabelValues(direction).Inc()
}

// RecordSSEConnection records SSE connection changes
func RecordSSEConnection(delta int) {
	sseConnectionsActive.Add(float64(delta))
}

// RecordSSEEvent records an SSE event
func RecordSSEEvent() {
	sseEventsTotal.Inc()
}

// RecordProxyRequest records a proxy request
func RecordProxyRequest(status string) {
	proxyRequestsTotal.WithLabelValues(status).Inc()
}

// RecordRecordedRequest records a recorded request
func RecordRecordedRequest() {
	recordedRequestsTotal.Inc()
}

// MetricsHandler returns the Prometheus metrics HTTP handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
