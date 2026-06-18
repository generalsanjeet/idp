package deploy

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// DeploymentRecord is a single deployment entry.
// Defined here independently to avoid import cycles.
type DeploymentRecord struct {
	ID         int       `json:"id"`
	Service    string    `json:"service"`
	Image      string    `json:"image"`
	DeployedAt time.Time `json:"deployed_at"`
}

// HistoryStore is the interface for deployment history operations.
type HistoryStore interface {
	Record(serviceName, image string) error
	List(serviceName string) ([]DeploymentRecord, error)
	Previous(serviceName string) (DeploymentRecord, error)
}

// RollbackHandler handles deployment history and rollback operations.
type RollbackHandler struct {
	store    *Store
	history HistoryStore
}

// NewRollbackHandler creates a new RollbackHandler.
func NewRollbackHandler(store *Store, history HistoryStore) *RollbackHandler {
	return &RollbackHandler{store: store, history: history}
}

// ListDeployments handles GET /deployments/{service}.
// Returns the full deployment history for a service.
func (h *RollbackHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "service")
	if serviceName == "" {
		http.Error(w, "service name is required", http.StatusBadRequest)
		return
	}

	records, err := h.history.List(serviceName)
	if err != nil {
		slog.Error("failed to list deployments", "error", err, "service", serviceName)
		http.Error(w, "could not list deployments", http.StatusInternalServerError)
		return
	}

	if records == nil {
		records = []DeploymentRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(records)
}

// Rollback handles POST /rollback/{service}.
// Finds the previous deployment and redeploys it.
func (h *RollbackHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	serviceName := chi.URLParam(r, "service")
	if serviceName == "" {
		http.Error(w, "service name is required", http.StatusBadRequest)
		return
	}

	previous, err := h.history.Previous(serviceName)
	if err != nil {
		slog.Error("failed to get previous deployment",
			"error", err,
			"service", serviceName,
		)
		http.Error(w, "no previous deployment found", http.StatusNotFound)
		return
	}

	slog.Info("rolling back service",
		"service", serviceName,
		"to_image", previous.Image,
	)

	// Deploy the previous image — same flow as a normal deploy.
	if err := h.store.Deploy(serviceName, previous.Image); err != nil {
		slog.Error("failed to rollback service", "error", err, "service", serviceName)
		http.Error(w, "could not rollback service", http.StatusInternalServerError)
		return
	}

	// Record the rollback as a new deployment.
	if err := h.history.Record(serviceName, previous.Image); err != nil {
		slog.Error("failed to record rollback", "error", err, "service", serviceName)
	}

	slog.Info("service rolled back successfully",
		"service", serviceName,
		"image", previous.Image,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(deployResponse{
		Service: serviceName,
		Image:   previous.Image,
		Status:  "rolled back",
	})
}
