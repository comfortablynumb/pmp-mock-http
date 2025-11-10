package recorder

import (
	"sync"
	"time"

	"github.com/comfortablynumb/pmp-mock-http/internal/models"
)

// RecordedRequest represents a recorded request/response pair
type RecordedRequest struct {
	Timestamp time.Time         `yaml:"timestamp" json:"timestamp"`
	Method    string            `yaml:"method" json:"method"`
	URI       string            `yaml:"uri" json:"uri"`
	Headers   map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body      string            `yaml:"body,omitempty" json:"body,omitempty"`
	Response  RecordedResponse  `yaml:"response" json:"response"`
}

// RecordedResponse represents a recorded response
type RecordedResponse struct {
	StatusCode int               `yaml:"status_code" json:"status_code"`
	Headers    map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	Body       string            `yaml:"body,omitempty" json:"body,omitempty"`
}

// Recorder handles recording of requests and responses
type Recorder struct {
	enabled   bool
	recordings []RecordedRequest
	mu        sync.RWMutex
}

// NewRecorder creates a new recorder
func NewRecorder() *Recorder {
	return &Recorder{
		enabled:    false,
		recordings: make([]RecordedRequest, 0),
	}
}

// IsEnabled returns whether recording is currently enabled
func (r *Recorder) IsEnabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled
}

// Start enables recording
func (r *Recorder) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = true
	r.recordings = make([]RecordedRequest, 0) // Clear previous recordings
}

// Stop disables recording
func (r *Recorder) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = false
}

// Record records a request/response pair
func (r *Recorder) Record(method, uri string, reqHeaders map[string]string, reqBody string,
	statusCode int, respHeaders map[string]string, respBody string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.enabled {
		return
	}

	recording := RecordedRequest{
		Timestamp: time.Now(),
		Method:    method,
		URI:       uri,
		Headers:   reqHeaders,
		Body:      reqBody,
		Response: RecordedResponse{
			StatusCode: statusCode,
			Headers:    respHeaders,
			Body:       respBody,
		},
	}

	r.recordings = append(r.recordings, recording)
}

// GetRecordings returns all recorded requests
func (r *Recorder) GetRecordings() []RecordedRequest {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid race conditions
	recordings := make([]RecordedRequest, len(r.recordings))
	copy(recordings, r.recordings)
	return recordings
}

// Clear clears all recordings
func (r *Recorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recordings = make([]RecordedRequest, 0)
}

// Count returns the number of recordings
func (r *Recorder) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.recordings)
}

// ExportAsMocks converts recordings to mock specifications
func (r *Recorder) ExportAsMocks(groupByURI bool) models.MockSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mocks := make([]models.Mock, 0)

	if groupByURI {
		// Group recordings by URI and create sequence mocks
		uriGroups := make(map[string][]RecordedRequest)
		for _, rec := range r.recordings {
			key := rec.Method + " " + rec.URI
			uriGroups[key] = append(uriGroups[key], rec)
		}

		for key, recs := range uriGroups {
			if len(recs) == 1 {
				// Single response
				rec := recs[0]
				mock := models.Mock{
					Name: "Recorded: " + key,
					Request: models.Request{
						URI:    rec.URI,
						Method: rec.Method,
					},
					Response: models.Response{
						StatusCode: rec.Response.StatusCode,
						Headers:    rec.Response.Headers,
						Body:       rec.Response.Body,
					},
				}
				mocks = append(mocks, mock)
			} else {
				// Multiple responses - create sequence
				sequence := make([]models.ResponseItem, 0, len(recs))
				for _, rec := range recs {
					sequence = append(sequence, models.ResponseItem{
						StatusCode: rec.Response.StatusCode,
						Headers:    rec.Response.Headers,
						Body:       rec.Response.Body,
					})
				}

				mock := models.Mock{
					Name: "Recorded: " + key + " (sequence)",
					Request: models.Request{
						URI:    recs[0].URI,
						Method: recs[0].Method,
					},
					Response: models.Response{
						Sequence:     sequence,
						SequenceMode: "cycle",
					},
				}
				mocks = append(mocks, mock)
			}
		}
	} else {
		// Create individual mocks for each recording
		for i, rec := range r.recordings {
			mock := models.Mock{
				Name: rec.Method + " " + rec.URI + " #" + string(rune(i+1)),
				Request: models.Request{
					URI:    rec.URI,
					Method: rec.Method,
				},
				Response: models.Response{
					StatusCode: rec.Response.StatusCode,
					Headers:    rec.Response.Headers,
					Body:       rec.Response.Body,
				},
			}
			mocks = append(mocks, mock)
		}
	}

	return models.MockSpec{
		Mocks: mocks,
	}
}
