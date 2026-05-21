package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/generalsanjeet/idp/internal/config"
	"github.com/generalsanjeet/idp/internal/health"
	"github.com/generalsanjeet/idp/internal/deploy"
	"github.com/generalsanjeet/idp/internal/db"
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

	deployStore, err := deploy.NewStore(cfg.KubeconfigPath)
	if err != nil {
		slog.Error("could not create k8s client", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to kubernetes")

	// Wire up service feature.
    serviceStore := service.NewStore(database)
    serviceHandler := service.NewHandler(serviceStore)
	deployHandler := deploy.NewHandler(deployStore)
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
	r.Get("/logs/{service}", logsHandler.GetLogs)
	r.Get("/metrics/{service}", metricsHandler.GetMetrics)

	slog.Info("IDP server starting", "addr", cfg.ServerAddr)
	if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
