package service

import (
	"database/sql"
	"fmt"

	"github.com/generalsanjeet/idp/internal/deploy"
)

// DeploymentStore handles all deployment history DB operations.
type DeploymentStore struct {
	db *sql.DB
}

// NewDeploymentStore creates a new DeploymentStore.
func NewDeploymentStore(db *sql.DB) *DeploymentStore {
	return &DeploymentStore{db: db}
}

// Record inserts a new deployment record.
// Called after every successful deploy.
func (s *DeploymentStore) Record(serviceName, image string) error {
	query := `
		INSERT INTO deployments (service, image)
		VALUES ($1, $2)`

	if _, err := s.db.Exec(query, serviceName, image); err != nil {
		return fmt.Errorf("could not record deployment: %w", err)
	}

	return nil
}

// List returns all deployments for a service ordered by newest first.
func (s *DeploymentStore) List(serviceName string) ([]deploy.DeploymentRecord, error) {
	query := `
		SELECT id, service, image, deployed_at
		FROM deployments
		WHERE service = $1
		ORDER BY deployed_at DESC`

	rows, err := s.db.Query(query, serviceName)
	if err != nil {
		return nil, fmt.Errorf("could not list deployments: %w", err)
	}
	defer rows.Close()

	var records []deploy.DeploymentRecord
	for rows.Next() {
		var d deploy.DeploymentRecord
		if err := rows.Scan(&d.ID, &d.Service, &d.Image, &d.DeployedAt); err != nil {
			return nil, fmt.Errorf("could not scan deployment: %w", err)
		}
		records = append(records, d)
	}

	return records, nil
}

// Previous returns the deployment before the most recent one.
func (s *DeploymentStore) Previous(serviceName string) (deploy.DeploymentRecord, error) {
	query := `
		SELECT id, service, image, deployed_at
		FROM deployments
		WHERE service = $1
		ORDER BY deployed_at DESC
		LIMIT 1 OFFSET 1`

	var d deploy.DeploymentRecord
	err := s.db.QueryRow(query, serviceName).
		Scan(&d.ID, &d.Service, &d.Image, &d.DeployedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return deploy.DeploymentRecord{}, fmt.Errorf("service not found")
		}
		return deploy.DeploymentRecord{}, fmt.Errorf("could not get previous deployment: %w", err)
	}

	return d, nil
}

// Ensure DeploymentStore satisfies deploy.HistoryStore.
var _ deploy.HistoryStore = (*DeploymentStore)(nil)
