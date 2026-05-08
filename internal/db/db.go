package db

import (
    "database/sql"
    "fmt"

    _ "github.com/lib/pq" // registers the postgres driver, we don't use it directly
)

// Connect opens a connection to Postgres and verifies it is alive.
// It returns the *sql.DB handle that the rest of the app will use.
// dsn is the full connection string e.g:
// "postgres://user:password@localhost:5432/idp?sslmode=disable"
func Connect(dsn string) (*sql.DB, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to open db: %w", err)
    }

    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping db: %w", err)
    }

    return db, nil
}
