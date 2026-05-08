package health

import (
	"encoding/json"
	"net/http"
)

// response is what we send back to the caller.
type response struct {
	Status string `json:"status"`
}

// Handler handles GET /health.
// It writes {"status":"ok"} and a 200 status code.
func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response{Status: "ok"})
}
