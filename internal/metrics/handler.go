package metrics

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler holds dependencies for metrics HTTP handlers.
type Handler struct {
	store *Store
}

// NewHandler creates a new Handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// GetMetrics handles GET /metrics/{service}.
func (h *Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "service")
	if serviceName == "" {
		http.Error(w, "service name is required", http.StatusBadRequest)
		return
	}

	result, err := h.store.GetMetrics(serviceName)
	if err != nil {
		http.Error(w, "could not fetch metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
