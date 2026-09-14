package DB

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRepository struct {
	db userDB
}

type userDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewUserRepository(conn *pgxpool.Pool) (*UserRepository, error) {
	return &UserRepository{
		db: conn,
	}, nil
}

func (r *UserRepository) Create(ctx context.Context, id, name, email string) (*User, error) {
	query := `
INSERT INTO users (id, name, email,created_at,updated_at) 
VALUES ($1, $2, $3,NOW(),NOW())
ON CONFLICT(email) DO UPDATE SET
                      name = EXCLUDED.name,
                      updated_at = NOW()
                      RETURNING id, name, email, created_at, updated_at;
`
	var user User
	err := r.db.QueryRow(ctx, query, id, name, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, name, email FROM users WHERE email = $1;`

	var user User

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	query := `SELECT id, name, email FROM users WHERE id = $1;`

	var user User

	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
