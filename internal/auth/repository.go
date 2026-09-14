package auth

import (
	"context"

	"github.com/MorningBlossom/auth-service/internal/DB"
)

// This file exists only to support OAuth unit tests; production behavior still
// uses the concrete DB.UserRepository implementation.
type userRepository interface {
	Create(context.Context, string, string, string) (*DB.User, error)
	GetByEmail(context.Context, string) (*DB.User, error)
}
