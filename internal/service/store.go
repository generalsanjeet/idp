package service

import (
    "database/sql"
	"errors"
    "fmt"

	"github.com/lib/pq"
)

// Sentinel errors — the handler checks against these.
// Using errors.Is() to compare, never string matching.
var (
	ErrDuplicate = errors.New("service already exists")
	ErrNotFound  = errors.New("service not found")
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
		// Check if Postgres returned a unique constraint violation.
		// pq.Error is the Postgres-specific error type from lib/pq.
		// Code "23505" is the Postgres error code for unique_violation.
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return Service{}, ErrDuplicate
		}
        return Service{}, fmt.Errorf("could not create service: %w", err)
    }

    return svc, nil
}

// List returns all registered services ordered by newest first.
func (s *Store) List() ([]Service, error) {
    query := `
        SELECT id, name, repo_url, owner, created_at
        FROM services
        ORDER BY created_at DESC`

    rows, err := s.db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("could not list services: %w", err)
    }
    defer rows.Close()

    var services []Service
    for rows.Next() {
        var svc Service
        if err := rows.Scan(&svc.ID, &svc.Name, &svc.RepoURL, &svc.Owner, &svc.CreatedAt); err != nil {
            return nil, fmt.Errorf("could not scan service: %w", err)
        }
        services = append(services, svc)
    }

    return services, nil
}

// GetByName fetches a single service by name.
// Returns ErrNotFound if no service with that name exists.
func (s *Store) GetByName(name string) (Service, error) {
	query := `
		SELECT id, name, repo_url, owner, created_at
		FROM services
		WHERE name = $1`

	var svc Service
	err := s.db.QueryRow(query, name).
		Scan(&svc.ID, &svc.Name, &svc.RepoURL, &svc.Owner, &svc.CreatedAt)
	if err != nil {
		// sql.ErrNoRows is what Go returns when QueryRow finds nothing.
		if errors.Is(err, sql.ErrNoRows) {
			return Service{}, ErrNotFound
		}
		return Service{}, fmt.Errorf("could not get service: %w", err)
	}

	return svc, nil
}

