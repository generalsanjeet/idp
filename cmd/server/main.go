package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/generalsanjeet/idp/internal/health"
	 "github.com/generalsanjeet/idp/internal/db"
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

	mux := http.NewServeMux()

	// Register routes here. Each route maps a URL path to a handler function.
	mux.HandleFunc("/health", health.Handler)

	addr := ":8080"
	fmt.Printf("IDP server starting on %s\n", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
