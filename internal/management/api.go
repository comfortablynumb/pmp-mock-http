package management

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/comfortablynumb/pmp-mock-http/internal/observability"
	"go.uber.org/zap"
)

// APIHandler handles management API requests
type APIHandler struct {
	manager *Manager
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(manager *Manager) *APIHandler {
	return &APIHandler{
		manager: manager,
	}
}

// RegisterRoutes registers management API routes
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	// Mock CRUD operations
	mux.HandleFunc("/api/v1/mocks", h.handleMocks)
	mux.HandleFunc("/api/v1/mocks/", h.handleMockByID)

	// Version management
	mux.HandleFunc("/api/v1/mocks/{id}/versions", h.handleVersions)
	mux.HandleFunc("/api/v1/mocks/{id}/versions/{version}", h.handleVersion)
	mux.HandleFunc("/api/v1/mocks/{id}/rollback", h.handleRollback)

	// Templates
	mux.HandleFunc("/api/v1/templates", h.handleTemplates)
	mux.HandleFunc("/api/v1/templates/", h.handleTemplateByID)
	mux.HandleFunc("/api/v1/templates/instantiate", h.handleInstantiateTemplate)

	// Import/Export
	mux.HandleFunc("/api/v1/import", h.handleImport)
	mux.HandleFunc("/api/v1/export", h.handleExport)

	// Stats
	mux.HandleFunc("/api/v1/stats", h.handleStats)
}

// handleMocks handles listing and creating mocks
func (h *APIHandler) handleMocks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listMocks(w, r)
	case http.MethodPost:
		h.createMock(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleMockByID handles operations on individual mocks
func (h *APIHandler) handleMockByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/mocks/"):]

	switch r.Method {
	case http.MethodGet:
		h.getMock(w, r, id)
	case http.MethodPut:
		h.updateMock(w, r, id)
	case http.MethodDelete:
		h.deleteMock(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listMocks lists all mocks with optional filtering
func (h *APIHandler) listMocks(w http.ResponseWriter, r *http.Request) {
	var filter MockFilter

	// Parse query parameters
	if tags := r.URL.Query()["tags"]; len(tags) > 0 {
		filter.Tags = tags
	}
	if source := r.URL.Query().Get("source"); source != "" {
		filter.Source = source
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}

	mocks, err := h.manager.ListMocks(&filter)
	if err != nil {
		observability.Error("Failed to list mocks", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mocks)
}

// createMock creates a new mock
func (h *APIHandler) createMock(w http.ResponseWriter, r *http.Request) {
	var req CreateMockRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mock, err := h.manager.CreateMock(req)
	if err != nil {
		observability.Error("Failed to create mock", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mock)
}

// getMock retrieves a mock by ID
func (h *APIHandler) getMock(w http.ResponseWriter, r *http.Request, id string) {
	mock, err := h.manager.GetMock(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mock)
}

// updateMock updates a mock
func (h *APIHandler) updateMock(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateMockRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mock, err := h.manager.UpdateMock(id, req)
	if err != nil {
		observability.Error("Failed to update mock", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mock)
}

// deleteMock deletes a mock
func (h *APIHandler) deleteMock(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.manager.DeleteMock(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleVersions handles version history requests
func (h *APIHandler) handleVersions(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/mocks/"):]
	id = id[:len(id)-len("/versions")]

	versions, err := h.manager.GetVersionHistory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(versions)
}

// handleVersion handles individual version requests
func (h *APIHandler) handleVersion(w http.ResponseWriter, r *http.Request) {
	// Parse URL to extract ID and version
	// This is simplified - in production, use a router like gorilla/mux
	parts := parseVersionURL(r.URL.Path)
	if len(parts) < 2 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	id := parts[0]
	version, err := strconv.Atoi(parts[1])
	if err != nil {
		http.Error(w, "Invalid version number", http.StatusBadRequest)
		return
	}

	mockVersion, err := h.manager.GetVersion(id, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mockVersion)
}

// handleRollback handles rollback requests
func (h *APIHandler) handleRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/api/v1/mocks/"):]
	id = id[:len(id)-len("/rollback")]

	var req struct {
		Version int    `json:"version"`
		Author  string `json:"author"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mock, err := h.manager.RollbackToVersion(id, req.Version, req.Author)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mock)
}

// handleTemplates handles template listing and creation
func (h *APIHandler) handleTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		category := r.URL.Query().Get("category")
		templates, err := h.manager.ListTemplates(category)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(templates)

	case http.MethodPost:
		var req CreateTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		template, err := h.manager.CreateTemplate(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(template)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTemplateByID handles operations on individual templates
func (h *APIHandler) handleTemplateByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/v1/templates/"):]

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	template, err := h.manager.GetTemplate(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(template)
}

// handleInstantiateTemplate handles template instantiation
func (h *APIHandler) handleInstantiateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req InstantiateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mock, err := h.manager.InstantiateTemplate(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mock)
}

// handleImport handles mock import
func (h *APIHandler) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	count, err := h.manager.Import(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"imported": count,
	})
}

// handleExport handles mock export
func (h *APIHandler) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	data, err := h.manager.Export(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set content type based on format
	switch req.Format {
	case ExportFormatYAML:
		w.Header().Set("Content-Type", "application/x-yaml")
	case ExportFormatJSON:
		w.Header().Set("Content-Type", "application/json")
	default:
		w.Header().Set("Content-Type", "text/plain")
	}

	_, _ = w.Write([]byte(data))
}

// handleStats handles statistics requests
func (h *APIHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.manager.GetStats()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// Helper function to parse version URL
func parseVersionURL(path string) []string {
	// Extract ID and version from path like /api/v1/mocks/{id}/versions/{version}
	// This is a simplified implementation
	var parts []string
	// In production, use a proper router
	return parts
}
