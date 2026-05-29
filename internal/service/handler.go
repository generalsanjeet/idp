package service

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// Bootstrapper is the interface for bootstrapping a new service
// in the GitOps repo. Defined here to avoid import cycles —
// the service package doesn't import the deploy package directly.
type Bootstrapper interface {
	Bootstrap(serviceName string) error
}

// Storer is the interface the handler depends on for DB operations.
type Storer interface {
	Create(req CreateRequest) (Service, error)
	List() ([]Service, error)
}

// Handler holds dependencies for service HTTP handlers.
type Handler struct {
	store       Storer
	bootstrapper Bootstrapper
}

// NewHandler creates a new Handler.
func NewHandler(store Storer, bootstrapper Bootstrapper) *Handler {
	return &Handler{store: store, bootstrapper: bootstrapper}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("failed to decode request", "error", err, "handler", "service.Create")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.RepoURL == "" || req.Owner == "" {
		http.Error(w, "name, repo_url and owner are required", http.StatusBadRequest)
		return
	}

	// Register in the database first.
	svc, err := h.store.Create(req)
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			http.Error(w, "service already exists", http.StatusConflict)
			return
		}
		slog.Error("failed to create service", "error", err, "service", req.Name)
		http.Error(w, "could not create service", http.StatusInternalServerError)
		return
	}

	// Bootstrap the GitOps structure for the new service.
	// We pass the gitops repo URL (not the service's repo URL)
	// so ArgoCD knows where to find the Helm chart.
	if err := h.bootstrapper.Bootstrap(svc.Name); err != nil {
		// Log the error but don't fail the request —
		// the service is registered in the DB, bootstrap can be retried.
		slog.Error("failed to bootstrap service", "error", err, "service", svc.Name)
	} else {
		slog.Info("service bootstrapped in gitops repo", "service", svc.Name)
	}

	slog.Info("service created", "service", svc.Name, "owner", svc.Owner)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(svc)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	services, err := h.store.List()
	if err != nil {
		slog.Error("failed to list services", "error", err)
		http.Error(w, "could not fetch services", http.StatusInternalServerError)
		return
	}

	if services == nil {
		services = []Service{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(services)
}
