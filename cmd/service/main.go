package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/MorningBlossom/auth-service/graphql"
	"github.com/MorningBlossom/auth-service/internal/DB"
	"github.com/MorningBlossom/auth-service/internal/auth"
	"github.com/MorningBlossom/auth-service/internal/middleware"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := godotenv.Load(); err != nil {
		log.Println("Note: No .env file found, falling back to system environment")
	}

	db, err := DB.ConnectDB(ctx)
	if err != nil {
		log.Fatal(err)
	}

	userRepo, err := DB.NewUserRepository(db)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize Google Oauth Manager
	authManager := auth.NewOAuthManager(
		os.Getenv("AUTH_CLIENT_ID"),
		os.Getenv("AUTH_CLIENT_SECRET"),
		os.Getenv("AUTH_REDIRECT_URL"),
		userRepo,
	)

	mux := http.NewServeMux()

	mux.HandleFunc("/auth/google", authManager.HandleGoogleLogin)
	mux.HandleFunc("/auth/google/callback", authManager.HandleGoogleCallback)

	resolver := &graphql.Resolver{
		UserRepo: userRepo,
	}

	srv := handler.New(graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})
	srv.Use(extension.Introspection{})

	mux.Handle("/", playground.Handler("GraphQL Playground", "/query"))
	mux.Handle("/query", middleware.CookieAuthMiddleware(srv))

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
