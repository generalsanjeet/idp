package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/generalsanjeet/idp/internal/config"
	"github.com/generalsanjeet/idp/internal/db"
	"github.com/generalsanjeet/idp/internal/deploy"
	"github.com/generalsanjeet/idp/internal/health"
	idplogs "github.com/generalsanjeet/idp/internal/logs"
	"github.com/generalsanjeet/idp/internal/metrics"
	"github.com/generalsanjeet/idp/internal/service"
)

func main() {
	// Set up JSON logger writing to stdout.
	// Every log line will be a JSON object — machine readable.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Set as the default logger so slog.Info() etc work globally.
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded")

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		slog.Error("could not connect to database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to database")

	if err := db.Migrate(database, cfg.MigrationsPath); err != nil {
		slog.Error("could not run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations complete")

	// GitOps deploy store — no k8s client needed anymore.
	deployStore, err := deploy.NewStore(
		cfg.GitOpsRepoURL,
		cfg.GitOpsLocalPath,
		cfg.GitHubToken,
		cfg.KubeconfigPath, // ← add this
	)
	if err != nil {
		slog.Error("could not create deploy store", "error", err)
		os.Exit(1)
	}
	slog.Info("gitops deploy store ready", "repo", cfg.GitOpsRepoURL)

	// Wire up service feature.
	serviceStore := service.NewStore(database)
	deploymentStore := service.NewDeploymentStore(database)
	serviceHandler := service.NewHandler(serviceStore, deployStore)
	deployHandler := deploy.NewHandler(deployStore, deploymentStore)
	rollbackHandler := deploy.NewRollbackHandler(deployStore, deploymentStore)
	logsStore := idplogs.NewStore(cfg.LokiURL)
	logsHandler := idplogs.NewHandler(logsStore)
	metricsStore := metrics.NewStore(cfg.PrometheusURL)
	metricsHandler := metrics.NewHandler(metricsStore)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", health.Handler)
	r.Post("/services", serviceHandler.Create)
	r.Get("/services", serviceHandler.List)
	r.Post("/deploy/{service}", deployHandler.Deploy)
	r.Get("/deployments/{service}", rollbackHandler.ListDeployments)
	r.Post("/rollback/{service}", rollbackHandler.Rollback)
	r.Get("/logs/{service}", logsHandler.GetLogs)
	r.Get("/metrics/{service}", metricsHandler.GetMetrics)

	//r.Handle("/*", http.StripPrefix("/", http.FileServer(http.Dir("./ui"))))
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./ui/index.html")
	})

	slog.Info("IDP server starting", "addr", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
