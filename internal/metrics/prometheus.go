package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// prometheusResponse mirrors what Prometheus returns.
// We only decode the fields we need.
type prometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// ServiceMetrics is what we return to the caller.
type ServiceMetrics struct {
	Service       string `json:"service"`
	ReadyReplicas string `json:"ready_replicas"`
	Status        string `json:"status"` // "healthy" or "degraded"
}

// Store handles all Prometheus query operations.
type Store struct {
	prometheusURL string // e.g. http://localhost:9091
}

// NewStore creates a new Prometheus store.
func NewStore(prometheusURL string) *Store {
	return &Store{prometheusURL: prometheusURL}
}

// GetMetrics fetches the ready replica count for a service.
func (s *Store) GetMetrics(serviceName string) (ServiceMetrics, error) {
	// PromQL: how many pods of this deployment are ready?
	promql := fmt.Sprintf(
		`kube_deployment_status_replicas_ready{deployment="%s"}`,
		serviceName,
	)

	params := url.Values{}
	params.Set("query", promql)

	fullURL := fmt.Sprintf("%s/api/v1/query?%s", s.prometheusURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return ServiceMetrics{}, fmt.Errorf("failed to query prometheus: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ServiceMetrics{}, fmt.Errorf("prometheus returned %d: %s", resp.StatusCode, string(body))
	}

	var promResp prometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return ServiceMetrics{}, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	result := ServiceMetrics{
		Service:       serviceName,
		ReadyReplicas: "0",
		Status:        "degraded",
	}

	// If prometheus returned at least one result, read the value.
	// Value is a [timestamp, valueString] pair — we want index 1.
	if len(promResp.Data.Result) > 0 {
		value := promResp.Data.Result[0].Value
		if len(value) == 2 {
			result.ReadyReplicas = fmt.Sprintf("%v", value[1])
		}
	}

	// Mark healthy if at least one replica is ready.
	if result.ReadyReplicas != "0" && result.ReadyReplicas != "" {
		result.Status = "healthy"
	}

	return result, nil
}
