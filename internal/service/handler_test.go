package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeStore is our test double — implements Storer interface
// but returns whatever we tell it to, no DB involved.
type fakeStore struct {
	createFn func(req CreateRequest) (Service, error)
	listFn   func() ([]Service, error)
}

func (f *fakeStore) Create(req CreateRequest) (Service, error) {
	return f.createFn(req)
}

func (f *fakeStore) List() ([]Service, error) {
	return f.listFn()
}

// TestHandlerCreate_Success verifies that a valid request
// returns 201 and the created service JSON.
func TestHandlerCreate_Success(t *testing.T) {
	// Wire the handler with a fake store that always succeeds.
	store := &fakeStore{
		createFn: func(req CreateRequest) (Service, error) {
			return Service{
				ID:        1,
				Name:      req.Name,
				RepoURL:   req.RepoURL,
				Owner:     req.Owner,
				CreatedAt: time.Now(),
			}, nil
		},
	}
	handler := NewHandler(store)

	// Build a fake HTTP request — no real network involved.
	body := `{"name":"payments","repo_url":"https://github.com/org/payments","owner":"team-alpha"}`
	r := httptest.NewRequest(http.MethodPost, "/services", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")

	// httptest.NewRecorder captures the response without a real server.
	w := httptest.NewRecorder()

	handler.Create(w, r)

	// Assert status code.
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}

	// Assert response body contains the service.
	var svc Service
	if err := json.NewDecoder(w.Body).Decode(&svc); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if svc.Name != "payments" {
		t.Errorf("expected name 'payments', got '%s'", svc.Name)
	}
}

// TestHandlerCreate_Duplicate verifies that ErrDuplicate from the store
// results in a 409 response — not 500.
func TestHandlerCreate_Duplicate(t *testing.T) {
	store := &fakeStore{
		createFn: func(req CreateRequest) (Service, error) {
			return Service{}, ErrDuplicate
		},
	}
	handler := NewHandler(store)

	body := `{"name":"payments","repo_url":"https://github.com/org/payments","owner":"team-alpha"}`
	r := httptest.NewRequest(http.MethodPost, "/services", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

// TestHandlerCreate_MissingFields verifies that missing required
// fields return 400 — before even hitting the store.
func TestHandlerCreate_MissingFields(t *testing.T) {
	// Store should never be called for invalid input.
	// If it is called, the test fails — that's a bug.
	store := &fakeStore{
		createFn: func(req CreateRequest) (Service, error) {
			t.Error("store.Create should not be called for invalid input")
			return Service{}, nil
		},
	}
	handler := NewHandler(store)

	body := `{"name":"payments"}` // missing repo_url and owner
	r := httptest.NewRequest(http.MethodPost, "/services", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestHandlerList_Success verifies that List returns 200
// and a JSON array of services.
func TestHandlerList_Success(t *testing.T) {
	store := &fakeStore{
		listFn: func() ([]Service, error) {
			return []Service{
				{ID: 1, Name: "payments", RepoURL: "https://github.com/org/payments", Owner: "team-alpha"},
				{ID: 2, Name: "billing", RepoURL: "https://github.com/org/billing", Owner: "team-finance"},
			}, nil
		},
	}
	handler := NewHandler(store)

	r := httptest.NewRequest(http.MethodGet, "/services", nil)
	w := httptest.NewRecorder()

	handler.List(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var services []Service
	if err := json.NewDecoder(w.Body).Decode(&services); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d", len(services))
	}
}

// TestHandlerList_Empty verifies that an empty DB returns []
// not null in JSON.
func TestHandlerList_Empty(t *testing.T) {
	store := &fakeStore{
		listFn: func() ([]Service, error) {
			return nil, nil // store returns nil for empty
		},
	}
	handler := NewHandler(store)

	r := httptest.NewRequest(http.MethodGet, "/services", nil)
	w := httptest.NewRecorder()

	handler.List(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Raw body should be [] not null.
	body := w.Body.String()
	if body != "[]\n" {
		t.Errorf("expected '[]', got '%s'", body)
	}
}
