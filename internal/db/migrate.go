package db

import (
    "database/sql"
    "fmt"
)

// Migrate runs all schema creation statements.
// It is safe to call on every startup — IF NOT EXISTS means it won't
// recreate or wipe tables that already exist.
func Migrate(db *sql.DB) error {
    query := `
    CREATE TABLE IF NOT EXISTS services (
        id         SERIAL PRIMARY KEY,
        name       TEXT NOT NULL UNIQUE,
        repo_url   TEXT NOT NULL,
        owner      TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT NOW()
    );`

    if _, err := db.Exec(query); err != nil {
        return fmt.Errorf("failed to run migrations: %w", err)
    }

    return nil
}
