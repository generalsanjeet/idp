package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the IDP server.
// Every env var the application needs is represented as a field here.
// Nothing else in the app should ever call os.Getenv directly.
type Config struct {
	// DatabaseURL is the full Postgres connection string.
	// Required — no default.
	DatabaseURL string

	// KubeconfigPath is the path to the kubeconfig file.
	// Defaults to ~/.kube/config if not set.
	KubeconfigPath string

	// LokiURL is the base URL of the Loki instance.
	// Defaults to http://localhost:3100.
	LokiURL string

	// PrometheusURL is the base URL of the Prometheus instance.
	// Defaults to http://localhost:9091.
	PrometheusURL string

	// ServerAddr is the address the HTTP server listens on.
	// Defaults to :8080.
	ServerAddr string
}

// Load reads all config from environment variables.
// It returns an error if any required variable is missing.
// Call this once at startup — never call os.Getenv anywhere else.
func Load() (Config, error) {
	cfg := Config{}

	// Required — no default possible.
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	// Optional with defaults.
	cfg.KubeconfigPath = os.Getenv("KUBECONFIG")
	if cfg.KubeconfigPath == "" {
		cfg.KubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}

	cfg.LokiURL = os.Getenv("LOKI_URL")
	if cfg.LokiURL == "" {
		cfg.LokiURL = "http://localhost:3100"
	}

	cfg.PrometheusURL = os.Getenv("PROMETHEUS_URL")
	if cfg.PrometheusURL == "" {
		cfg.PrometheusURL = "http://localhost:9091"
	}

	cfg.ServerAddr = os.Getenv("SERVER_ADDR")
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = ":8080"
	}

	return cfg, nil
}
