package deploy

import (
	"encoding/json"
	"net/http"
	"log/slog"

	"github.com/go-chi/chi/v5"
)

// Recorder is the interface for recording deployment history.
// Defined here to avoid import cycles between deploy and service packages.
type Recorder interface {
	Record(serviceName, image string) error
}

// Handler holds dependencies for deploy HTTP handlers.
type Handler struct {
	store *Store
	recorder Recorder
}

// NewHandler creates a new Handler.
func NewHandler(store *Store, recorder Recorder) *Handler {
	return &Handler{store: store, recorder: recorder}
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
		slog.Error("failed to decode request", "error", err, "handler", "deploy.Deploy")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Image == "" {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}

	if err := h.store.Deploy(serviceName, req.Image); err != nil {
		slog.Error("failed to deploy service", "error", err, "service", serviceName)
		http.Error(w, "could not deploy service", http.StatusInternalServerError)
		return
	}

	// Record the deployment in history.
	// We log but don't fail the request if recording fails —
	// the deploy already happened, recording is secondary.
	if err := h.recorder.Record(serviceName, req.Image); err != nil {
		slog.Error("failed to record deployment", "error", err, "service", serviceName)
	}

	slog.Info("service deployed", "service", serviceName, "image", req.Image)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(deployResponse{
		Service: serviceName,
		Image:   req.Image,
		Status:  "deployed",
	})
}
