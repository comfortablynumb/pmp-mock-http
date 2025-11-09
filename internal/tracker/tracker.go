package tracker

import (
	"sync"
	"time"
)

type RequestLog struct {
	ID          int64             `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Method      string            `json:"method"`
	URI         string            `json:"uri"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	Matched     bool              `json:"matched"`
	MockName    string            `json:"mock_name,omitempty"`
	StatusCode  int               `json:"status_code"`
	Response    string            `json:"response"`
	RemoteAddr  string            `json:"remote_addr"`
}

type Tracker struct {
	logs    []RequestLog
	mu      sync.RWMutex
	nextID  int64
	maxLogs int
}

func NewTracker(maxLogs int) *Tracker {
	if maxLogs <= 0 {
		maxLogs = 1000
	}
	return &Tracker{
		logs:    make([]RequestLog, 0, maxLogs),
		maxLogs: maxLogs,
		nextID:  1,
	}
}

func (t *Tracker) Log(log RequestLog) {
	t.mu.Lock()
	defer t.mu.Unlock()
	log.ID = t.nextID
	t.nextID++
	log.Timestamp = time.Now()
	t.logs = append(t.logs, log)
	if len(t.logs) > t.maxLogs {
		t.logs = t.logs[len(t.logs)-t.maxLogs:]
	}
}

func (t *Tracker) GetLogs() []RequestLog {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]RequestLog, len(t.logs))
	for i, j := 0, len(t.logs)-1; j >= 0; i, j = i+1, j-1 {
		result[i] = t.logs[j]
	}
	return result
}

func (t *Tracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.logs = make([]RequestLog, 0, t.maxLogs)
}

func (t *Tracker) Count() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.logs)
}
