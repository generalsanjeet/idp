package logs

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler holds dependencies for log HTTP handlers.
type Handler struct {
	store *Store
}

// NewHandler creates a new Handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// GetLogs handles GET /logs/{service}.
func (h *Handler) GetLogs(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "service")
	if serviceName == "" {
		http.Error(w, "service name is required", http.StatusBadRequest)
		return
	}

	lines, err := h.store.GetLogs(serviceName, 100)
	if err != nil {
		http.Error(w, "could not fetch logs", http.StatusInternalServerError)
		return
	}

	// Return empty array not null when no logs exist yet.
	if lines == nil {
		lines = []LogLine{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(lines)
}
