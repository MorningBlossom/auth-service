package graphql

// THIS CODE WILL BE UPDATED WITH SCHEMA CHANGES. PREVIOUS IMPLEMENTATION FOR SCHEMA CHANGES WILL BE KEPT IN THE COMMENT SECTION. IMPLEMENTATION FOR UNCHANGED SCHEMA WILL BE KEPT.

import (
	"context"
	"errors"
	"fmt"

	"github.com/MorningBlossom/auth-service/internal/DB"
	"github.com/MorningBlossom/auth-service/internal/middleware"
)

type Resolver struct {
	UserRepo *DB.UserRepository
}

// Me is the resolver for the me field.
func (r *queryResolver) Me(ctx context.Context) (*User, error) {
	userID, ok := ctx.Value(middleware.UserContextKey).(string)
	if !ok || userID == "" {
		return nil, errors.New("unauthorized: missing or invalid session")
	}

	// 2. Fetch user details from the database via the repository
	dbUser, err := r.UserRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %w", err)
	}

	// 3. Map internal repository user model to the generated GraphQL User type
	return &User{
		ID:    dbUser.ID,
		Name:  dbUser.Name,
		Email: dbUser.Email,
	}, nil
}

// User is the resolver for the User field.
func (r *queryResolver) User(ctx context.Context, id string) (*User, error) {
	dbUser, err := r.UserRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	return &User{
		ID:    dbUser.ID,
		Name:  dbUser.Name,
		Email: dbUser.Email,
	}, nil
}

// Query returns QueryResolver implementation.
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

type queryResolver struct{ *Resolver }
