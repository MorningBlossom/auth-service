package DB

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fakeUserDB struct {
	query string
	args  []any
	row   pgx.Row
}

func (db *fakeUserDB) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	db.query = query
	db.args = args
	return db.row
}

type fakeRow struct {
	values []any
	err    error
}

func (row fakeRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	for index, value := range row.values {
		switch destination := dest[index].(type) {
		case *string:
			*destination = value.(string)
		case *time.Time:
			*destination = value.(time.Time)
		default:
			return errors.New("unsupported scan destination")
		}
	}
	return nil
}

func TestUserRepositoryCreate(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	db := &fakeUserDB{row: fakeRow{values: []any{"drn_123", "Ada", "ada@example.com", createdAt, updatedAt}}}

	user, err := (&UserRepository{db: db}).Create(context.Background(), "drn_123", "Ada", "ada@example.com")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if user.ID != "drn_123" || user.Name != "Ada" || user.Email != "ada@example.com" {
		t.Fatalf("Create returned unexpected user: %+v", user)
	}
	if !user.CreatedAt.Equal(createdAt) || !user.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("Create returned unexpected timestamps: %+v", user)
	}
	if db.args[0] != "drn_123" || db.args[1] != "Ada" || db.args[2] != "ada@example.com" {
		t.Fatalf("Create passed unexpected arguments: %v", db.args)
	}
}

func TestUserRepositoryCreateReturnsQueryError(t *testing.T) {
	db := &fakeUserDB{row: fakeRow{err: errors.New("insert failed")}}

	user, err := (&UserRepository{db: db}).Create(context.Background(), "drn_123", "Ada", "ada@example.com")
	if user != nil {
		t.Fatalf("Create returned user on error: %+v", user)
	}
	if err == nil || err.Error() != "insert failed" {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
}

func TestUserRepositoryGetByEmail(t *testing.T) {
	db := &fakeUserDB{row: fakeRow{values: []any{"drn_123", "Ada", "ada@example.com"}}}

	user, err := (&UserRepository{db: db}).GetByEmail(context.Background(), "ada@example.com")
	if err != nil {
		t.Fatalf("GetByEmail returned error: %v", err)
	}
	if user.ID != "drn_123" || user.Name != "Ada" || user.Email != "ada@example.com" {
		t.Fatalf("GetByEmail returned unexpected user: %+v", user)
	}
}

func TestUserRepositoryGetByID(t *testing.T) {
	db := &fakeUserDB{row: fakeRow{values: []any{"drn_123", "Ada", "ada@example.com"}}}

	user, err := (&UserRepository{db: db}).GetByID(context.Background(), "drn_123")
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if user.ID != "drn_123" || user.Name != "Ada" || user.Email != "ada@example.com" {
		t.Fatalf("GetByID returned unexpected user: %+v", user)
	}
	if db.args[0] != "drn_123" {
		t.Fatalf("GetByID passed unexpected arguments: %v", db.args)
	}
}

func TestUserRepositoryGetByIDReturnsQueryError(t *testing.T) {
	db := &fakeUserDB{row: fakeRow{err: errors.New("lookup failed")}}

	user, err := (&UserRepository{db: db}).GetByID(context.Background(), "drn_123")
	if user != nil {
		t.Fatalf("GetByID returned user on error: %+v", user)
	}
	if err == nil || err.Error() != "lookup failed" {
		t.Fatalf("GetByID returned unexpected error: %v", err)
	}
}
