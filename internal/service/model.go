package service

import "time"

// Service represents a registered service in the IDP.
// This is the single source of truth for what a service looks like.
type Service struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    RepoURL   string    `json:"repo_url"`
    Owner     string    `json:"owner"`
    CreatedAt time.Time `json:"created_at"`
}

// CreateRequest is what the caller must send in the request body.
// It is separate from Service because the caller never sets id or created_at.
type CreateRequest struct {
    Name    string `json:"name"`
    RepoURL string `json:"repo_url"`
    Owner   string `json:"owner"`
}
