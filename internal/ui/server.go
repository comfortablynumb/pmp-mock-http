package ui

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"github.com/comfortablynumb/pmp-mock-http/internal/tracker"
)

type Server struct {
	port    int
	tracker *tracker.Tracker
}

func NewServer(port int, tracker *tracker.Tracker) *Server {
	return &Server{port: port, tracker: tracker}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/api/requests", s.handleRequests)
	mux.HandleFunc("/api/clear", s.handleClear)
	log.Printf("Starting UI server on port %d\n", s.port)
	log.Printf("Dashboard available at http://localhost:%d\n", s.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), mux)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(dashboardHTML)); err != nil {
		log.Printf("Error writing dashboard: %v\n", err)
	}
}

func (s *Server) handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	logs := s.tracker.GetLogs()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		log.Printf("Error encoding logs: %v\n", err)
	}
}

func (s *Server) handleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.tracker.Clear()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Error encoding response: %v\n", err)
	}
}
