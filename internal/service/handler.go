package service

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Handler holds dependencies for service HTTP handlers.
type Handler struct {
	store *Store
}

// NewHandler creates a new Handler.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Create handles POST /services.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// Decode the request body into CreateRequest.
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode request", "error", err, "handler", "service.Create")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Basic validation — all fields are required.
	if req.Name == "" || req.RepoURL == "" || req.Owner == "" {
		http.Error(w, "name, repo_url and owner are required", http.StatusBadRequest)
		return
	}

	// Delegate to store — handler does not know about SQL.
	svc, err := h.store.Create(req)
	if err != nil {
		slog.Error("failed to create service", "error", err, "service", req.Name)
		http.Error(w, "could not create service", http.StatusInternalServerError)
		return
	}

	// Respond with the created service.
	slog.Info("service created", "service", svc.Name, "owner", svc.Owner)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201, not 200
	json.NewEncoder(w).Encode(svc)
}

// List handles GET /services.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    services, err := h.store.List()
    if err != nil {
		slog.Error("failed to list services", "error", err)
        http.Error(w, "could not fetch services", http.StatusInternalServerError)
        return
    }

    // If no services exist yet, return an empty array — not null.
    if services == nil {
        services = []Service{}
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(services)
}
