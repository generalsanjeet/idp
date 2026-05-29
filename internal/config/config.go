package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the IDP server.
// Every env var the application needs is represented as a field here.
// Nothing else in the app should ever call os.Getenv directly.
type Config struct {
	DatabaseURL string
	LokiURL string
	PrometheusURL string
	ServerAddr string

	GitOpsRepoURL   string // e.g. https://github.com/yourname/gitops-repo
	GitOpsLocalPath string // local path to clone the repo into
	GitHubToken     string // personal access token for pushing
	MigrationsPath  string
}

func Load() (Config, error) {
	cfg := Config{}

	// Required — no default possible.
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
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

		// GitOps config.
	cfg.GitOpsRepoURL = os.Getenv("GITOPS_REPO_URL")
	if cfg.GitOpsRepoURL == "" {
		return Config{}, fmt.Errorf("GITOPS_REPO_URL is required")
	}

	cfg.GitOpsLocalPath = os.Getenv("GITOPS_LOCAL_PATH")
	if cfg.GitOpsLocalPath == "" {
		cfg.GitOpsLocalPath = "/tmp/gitops-repo"
	}

	cfg.GitHubToken = os.Getenv("GITHUB_TOKEN")
	if cfg.GitHubToken == "" {
		return Config{}, fmt.Errorf("GITHUB_TOKEN is required")
	}

	cfg.MigrationsPath = os.Getenv("MIGRATIONS_PATH")
	if cfg.MigrationsPath == "" {
		cfg.MigrationsPath = "./migrations"
	}

	return cfg, nil
}
