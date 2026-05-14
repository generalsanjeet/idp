package logs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// lokiResponse mirrors the structure Loki returns.
// We only decode what we need.
type lokiResponse struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Values [][]string `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// LogLine is a single log entry we return to the caller.
type LogLine struct {
	Timestamp string `json:"timestamp"`
	Line      string `json:"line"`
}

// Store handles all Loki query operations.
type Store struct {
	lokiURL string // e.g. http://localhost:3100
}

// NewStore creates a new Loki store.
func NewStore(lokiURL string) *Store {
	return &Store{lokiURL: lokiURL}
}

// GetLogs fetches the last `limit` log lines for a service.
func (s *Store) GetLogs(serviceName string, limit int) ([]LogLine, error) {
	// Build the LogQL query — select logs where app label matches service name.
	logql := fmt.Sprintf(`{app="%s"}`, serviceName)

	// Query last 1 hour of logs.
	end := time.Now()
	start := end.Add(-1 * time.Hour)

	// Build the full Loki query URL with parameters.
	params := url.Values{}
	params.Set("query", logql)
	params.Set("start", strconv.FormatInt(start.UnixNano(), 10))
	params.Set("end", strconv.FormatInt(end.UnixNano(), 10))
	params.Set("limit", strconv.Itoa(limit))

	fullURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", s.lokiURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("loki returned %d: %s", resp.StatusCode, string(body))
	}

	var lokiResp lokiResponse
	if err := json.NewDecoder(resp.Body).Decode(&lokiResp); err != nil {
		return nil, fmt.Errorf("failed to decode loki response: %w", err)
	}

	// Flatten all log lines from all streams into a single slice.
	var lines []LogLine
	for _, result := range lokiResp.Data.Result {
		for _, value := range result.Values {
			// Loki returns [unixNanoTimestamp, logLine] pairs.
			if len(value) != 2 {
				continue
			}

			// Convert nanosecond timestamp to readable format.
			nanos, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				continue
			}
			ts := time.Unix(0, nanos).UTC().Format(time.RFC3339)

			lines = append(lines, LogLine{
				Timestamp: ts,
				Line:      value[1],
			})
		}
	}

	return lines, nil
}
