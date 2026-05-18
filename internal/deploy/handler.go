package deploy

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler holds dependencies for deploy HTTP handlers.
type Handler struct {
	store *Store
}

// NewHandler creates a new Handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// deployRequest is what the caller sends.
type deployRequest struct {
	Image string `json:"image"`
}

// deployResponse is what we send back.
type deployResponse struct {
	Service string `json:"service"`
	Image   string `json:"image"`
	Status  string `json:"status"`
}

// Deploy handles POST /deploy/{service}.
func (h *Handler) Deploy(w http.ResponseWriter, r *http.Request) {
	// chi extracts {service} from the URL cleanly.
	serviceName := chi.URLParam(r, "service")
	if serviceName == "" {
		http.Error(w, "service name is required", http.StatusBadRequest)
		return
	}

	var req deployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Image == "" {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}

	if err := h.store.Deploy(context.Background(), serviceName, req.Image); err != nil {
		http.Error(w, "could not deploy service", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(deployResponse{
		Service: serviceName,
		Image:   req.Image,
		Status:  "deployed",
	})
}
