package service

import (
    "database/sql"
    "fmt"
)

// Store handles all database operations for services.
type Store struct {
    db *sql.DB
}

// NewStore creates a new Store.
func NewStore(db *sql.DB) *Store {
    return &Store{db: db}
}

// Create inserts a new service and returns the full created record.
func (s *Store) Create(req CreateRequest) (Service, error) {
    query := `
        INSERT INTO services (name, repo_url, owner)
        VALUES ($1, $2, $3)
        RETURNING id, name, repo_url, owner, created_at`

    var svc Service
    err := s.db.QueryRow(query, req.Name, req.RepoURL, req.Owner).
        Scan(&svc.ID, &svc.Name, &svc.RepoURL, &svc.Owner, &svc.CreatedAt)
    if err != nil {
        return Service{}, fmt.Errorf("could not create service: %w", err)
    }

    return svc, nil
}
