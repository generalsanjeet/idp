package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/generalsanjeet/idp/internal/health"
	"github.com/generalsanjeet/idp/internal/deploy"
	"github.com/generalsanjeet/idp/internal/db"
	idplogs "github.com/generalsanjeet/idp/internal/logs"
	"github.com/generalsanjeet/idp/internal/metrics"
	"github.com/generalsanjeet/idp/internal/service"
)

func main() {
	// Read DSN from environment. Never hardcode credentials.
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to Postgres. If this fails, we stop immediately.
	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	fmt.Println("connected to database")

	// Run migrations before starting the server.
	if err := db.Migrate(database); err != nil {
		log.Fatalf("could not run migrations: %v", err)
	}
	fmt.Println("migrations complete")

	// Read kubeconfig path from env, default to standard location.
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	deployStore, err := deploy.NewStore(kubeconfig)
	if err != nil {
		log.Fatalf("could not create k8s client: %v", err)
	}
	fmt.Println("connected to kubernetes")

	// Read Loki URL from env, default to local port-forward address.
	lokiURL := os.Getenv("LOKI_URL")
	if lokiURL == "" {
		lokiURL = "http://localhost:3100"
	}

	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL == "" {
		prometheusURL = "http://localhost:9091"
	}

	// Wire up service feature.
    serviceStore := service.NewStore(database)
    serviceHandler := service.NewHandler(serviceStore)
	deployHandler := deploy.NewHandler(deployStore)
	logsStore := idplogs.NewStore(lokiURL)
	logsHandler := idplogs.NewHandler(logsStore)
	metricsStore := metrics.NewStore(prometheusURL)
	metricsHandler := metrics.NewHandler(metricsStore)

	mux := http.NewServeMux()

	// Register routes here. Each route maps a URL path to a handler function.
	mux.HandleFunc("/health", health.Handler)
	mux.HandleFunc("/services", serviceHandler.Route)
	mux.HandleFunc("/deploy/", deployHandler.Deploy) // trailing slash catches /deploy/{anything}
	mux.HandleFunc("/logs/", logsHandler.GetLogs)
	mux.HandleFunc("/metrics/", metricsHandler.GetMetrics)

	addr := ":8080"
	fmt.Printf("IDP server starting on %s\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
