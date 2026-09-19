package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthManager struct {
	Config   *oauth2.Config
	userRepo userRepository
}

type GoogleUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func NewOAuthManager(clientID, clientSecret, redirectURL string, UR userRepository) *OAuthManager {
	return &OAuthManager{
		Config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
		userRepo: UR,
	}
}

func GenerateDRNID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(t), entropy)
	return fmt.Sprintf("drn_%s", id.String())
}

func (o *OAuthManager) HandleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := o.Config.AuthCodeURL("state-token-placeholder", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (o *OAuthManager) HandleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := r.URL.Query().Get("code")

	token, err := o.Config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
		return
	}

	client := o.Config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "Failed to fetch user info", http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var gUser GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&gUser); err != nil {
		http.Error(w, "Failed to parse user profile", http.StatusInternalServerError)
		return
	}

	var drnID string

	if user, _ := o.userRepo.GetByEmail(ctx, gUser.Email); user != nil {
		fmt.Printf("Found user with email %s\n so skipping create", user.Email)
		drnID = user.ID
	} else {
		fmt.Println("creating new user")
		drnID = GenerateDRNID() // Mock existing user lookup
		_, err = o.userRepo.Create(ctx, drnID, gUser.Name, gUser.Email)
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	// Issue Secure Session Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    drnID, // Storing session reference or signed JWT containing drnID
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400 * 7, // 7 days
	})

	// Redirect back to frontend dashboard
	http.Redirect(w, r, "/auth", http.StatusTemporaryRedirect)
}
