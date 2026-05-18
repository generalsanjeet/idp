package main

import (
	"fmt"
	"log"
	"net/http"
//	"os"
	"github.com/generalsanjeet/idp/internal/config"
	"github.com/generalsanjeet/idp/internal/health"
	"github.com/generalsanjeet/idp/internal/deploy"
	"github.com/generalsanjeet/idp/internal/db"
	idplogs "github.com/generalsanjeet/idp/internal/logs"
	"github.com/generalsanjeet/idp/internal/metrics"
	"github.com/generalsanjeet/idp/internal/service"
)

func main() {
	// Load all config first. If anything is wrong, stop immediately.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	fmt.Println("config loaded")

	// Connect to Postgres. If this fails, we stop immediately.
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	fmt.Println("connected to database")

	// Run migrations before starting the server.
	if err := db.Migrate(database); err != nil {
		log.Fatalf("could not run migrations: %v", err)
	}
	fmt.Println("migrations complete")

	deployStore, err := deploy.NewStore(cfg.KubeconfigPath)
	if err != nil {
		log.Fatalf("could not create k8s client: %v", err)
	}
	fmt.Println("connected to kubernetes")



	// Wire up service feature.
    serviceStore := service.NewStore(database)
    serviceHandler := service.NewHandler(serviceStore)
	deployHandler := deploy.NewHandler(deployStore)
	logsStore := idplogs.NewStore(cfg.LokiURL)
	logsHandler := idplogs.NewHandler(logsStore)
	metricsStore := metrics.NewStore(cfg.PrometheusURL)
	metricsHandler := metrics.NewHandler(metricsStore)

	mux := http.NewServeMux()

	// Register routes here. Each route maps a URL path to a handler function.
	mux.HandleFunc("/health", health.Handler)
	mux.HandleFunc("/services", serviceHandler.Route)
	mux.HandleFunc("/deploy/", deployHandler.Deploy) // trailing slash catches /deploy/{anything}
	mux.HandleFunc("/logs/", logsHandler.GetLogs)
	mux.HandleFunc("/metrics/", metricsHandler.GetMetrics)

	//addr := ":8080"
	fmt.Printf("IDP server starting on %s\n", cfg.ServerAddr)

	if err := http.ListenAndServe(cfg.ServerAddr, mux); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
