package service

import (
	"encoding/json"
	"net/http"
	"log"
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
	// Only allow POST.
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode the request body into CreateRequest.
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
		http.Error(w, "could not create service", http.StatusInternalServerError)
		log.Println(err)
		return
	}

	// Respond with the created service.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201, not 200
	json.NewEncoder(w).Encode(svc)
}
