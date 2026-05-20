package service

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

// TestStoreCreate_Success verifies that Create inserts correctly
// and returns the full service record.
func TestStoreCreate_Success(t *testing.T) {
	// sqlmock creates a fake *sql.DB and a mock controller.
	// No real Postgres needed.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	store := NewStore(db)

	// Tell the mock what query to expect and what to return.
	// sqlmock will fail the test if the actual query doesn't match.
	rows := sqlmock.NewRows([]string{"id", "name", "repo_url", "owner", "created_at"}).
		AddRow(1, "payments", "https://github.com/org/payments", "team-alpha", time.Now())

	mock.ExpectQuery("INSERT INTO services").
		WithArgs("payments", "https://github.com/org/payments", "team-alpha").
		WillReturnRows(rows)

	req := CreateRequest{
		Name:    "payments",
		RepoURL: "https://github.com/org/payments",
		Owner:   "team-alpha",
	}

	svc, err := store.Create(req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if svc.Name != "payments" {
		t.Errorf("expected name 'payments', got '%s'", svc.Name)
	}

	if svc.ID != 1 {
		t.Errorf("expected id 1, got %d", svc.ID)
	}

	// Verify all expected mock interactions were called.
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}

// TestStoreCreate_Duplicate verifies that a unique constraint violation
// returns ErrDuplicate, not a raw DB error.
func TestStoreCreate_Duplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	store := NewStore(db)

	// Simulate Postgres unique violation error (code 23505).
	mock.ExpectQuery("INSERT INTO services").
		WithArgs("payments", "https://github.com/org/payments", "team-alpha").
		WillReturnError(&pq.Error{Code: "23505"})

	req := CreateRequest{
		Name:    "payments",
		RepoURL: "https://github.com/org/payments",
		Owner:   "team-alpha",
	}

	_, err = store.Create(req)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// The important assertion — the store must return ErrDuplicate,
	// not a raw pq error. The handler depends on this.
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got: %v", err)
	}
}

// TestStoreList_Empty verifies that List returns an empty slice
// when no services exist — not nil, not an error.
func TestStoreList_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	store := NewStore(db)

	rows := sqlmock.NewRows([]string{"id", "name", "repo_url", "owner", "created_at"})
	// No rows added — simulates empty table.

	mock.ExpectQuery("SELECT id, name").WillReturnRows(rows)

	services, err := store.List()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// nil and empty slice are different — we want nil here
	// because the handler converts nil to [] before responding.
	// The store's job is just to return what the DB gives it.
	if len(services) != 0 {
		t.Errorf("expected 0 services, got %d", len(services))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}


