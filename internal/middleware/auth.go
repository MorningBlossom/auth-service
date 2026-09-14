package middleware

import (
	"context"
	"net/http"
)

// Define a private type for context keys to avoid collisions across packages
type contextKey string

// UserContextKey is the exported key used to store and retrieve the user ID in request contexts
const UserContextKey contextKey = "drn_user_id"

func CookieAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Inject DRN ID/session claims into context
		ctx := context.WithValue(r.Context(), UserContextKey, cookie.Value)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
