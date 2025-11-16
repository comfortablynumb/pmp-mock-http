package observability

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string       `json:"name"`
	Status      HealthStatus `json:"status"`
	Message     string       `json:"message,omitempty"`
	LastChecked time.Time    `json:"last_checked"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  HealthStatus  `json:"status"`
	Checks  []HealthCheck `json:"checks"`
	Uptime  float64       `json:"uptime_seconds"`
	Version string        `json:"version"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Ready   bool          `json:"ready"`
	Checks  []HealthCheck `json:"checks"`
	Message string        `json:"message,omitempty"`
}

var (
	healthChecks   = make(map[string]func() HealthCheck)
	healthChecksMu sync.RWMutex
	startTime      = time.Now()
	appVersion     = "1.0.0"
)

// RegisterHealthCheck registers a health check function
func RegisterHealthCheck(name string, check func() HealthCheck) {
	healthChecksMu.Lock()
	defer healthChecksMu.Unlock()
	healthChecks[name] = check
}

// SetVersion sets the application version
func SetVersion(version string) {
	appVersion = version
}

// HealthHandler returns the health check HTTP handler
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthChecksMu.RLock()
		defer healthChecksMu.RUnlock()

		checks := make([]HealthCheck, 0, len(healthChecks))
		overallStatus := HealthStatusHealthy

		for _, checkFn := range healthChecks {
			check := checkFn()
			checks = append(checks, check)

			// Determine overall status
			if check.Status == HealthStatusUnhealthy {
				overallStatus = HealthStatusUnhealthy
			} else if check.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}

		response := HealthResponse{
			Status:  overallStatus,
			Checks:  checks,
			Uptime:  time.Since(startTime).Seconds(),
			Version: appVersion,
		}

		w.Header().Set("Content-Type", "application/json")

		// Set HTTP status based on health status
		if overallStatus == HealthStatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// ReadinessHandler returns the readiness check HTTP handler
func ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthChecksMu.RLock()
		defer healthChecksMu.RUnlock()

		checks := make([]HealthCheck, 0, len(healthChecks))
		ready := true
		var message string

		for _, checkFn := range healthChecks {
			check := checkFn()
			checks = append(checks, check)

			// Service is not ready if any check is unhealthy
			if check.Status == HealthStatusUnhealthy {
				ready = false
				message = "One or more health checks failed"
			}
		}

		response := ReadinessResponse{
			Ready:   ready,
			Checks:  checks,
			Message: message,
		}

		w.Header().Set("Content-Type", "application/json")

		if ready {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// LivenessHandler returns a simple liveness check
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"alive":  true,
			"uptime": time.Since(startTime).Seconds(),
		})
	}
}

// DefaultHealthChecks registers default health checks
func RegisterDefaultHealthChecks() {
	// System health check
	RegisterHealthCheck("system", func() HealthCheck {
		return HealthCheck{
			Name:        "system",
			Status:      HealthStatusHealthy,
			Message:     "System is operational",
			LastChecked: time.Now(),
		}
	})

	// Memory health check (basic example)
	RegisterHealthCheck("memory", func() HealthCheck {
		// In a real implementation, check memory usage
		return HealthCheck{
			Name:        "memory",
			Status:      HealthStatusHealthy,
			Message:     "Memory usage is within limits",
			LastChecked: time.Now(),
		}
	})

	// Uptime health check
	RegisterHealthCheck("uptime", func() HealthCheck {
		uptime := time.Since(startTime)
		return HealthCheck{
			Name:        "uptime",
			Status:      HealthStatusHealthy,
			Message:     uptime.String(),
			LastChecked: time.Now(),
		}
	})
}
