package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/MorningBlossom/auth-service/internal/DB"
	"golang.org/x/oauth2"
)

type fakeUserRepository struct {
	user      *DB.User
	lookupErr error
	created   *DB.User
	createErr error
}

func (repo *fakeUserRepository) GetByEmail(context.Context, string) (*DB.User, error) {
	return repo.user, repo.lookupErr
}

func (repo *fakeUserRepository) Create(_ context.Context, id, name, email string) (*DB.User, error) {
	if repo.createErr != nil {
		return nil, repo.createErr
	}
	repo.created = &DB.User{ID: id, Name: name, Email: email}
	return repo.created, nil
}

func TestGenerateDRNID(t *testing.T) {
	id := GenerateDRNID()
	if !strings.HasPrefix(id, "drn_") {
		t.Fatalf("GenerateDRNID() = %q, want drn_ prefix", id)
	}
	if len(id) != len("drn_")+26 {
		t.Fatalf("GenerateDRNID() = %q, want a ULID after the prefix", id)
	}
}

func TestHandleGoogleLogin(t *testing.T) {
	manager := NewOAuthManager("client-id", "client-secret", "http://localhost/callback", &fakeUserRepository{})
	recorder := httptest.NewRecorder()
	manager.HandleGoogleLogin(recorder, httptest.NewRequest(http.MethodGet, "/auth/google", nil))

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("HandleGoogleLogin status = %d, want %d", recorder.Code, http.StatusTemporaryRedirect)
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	if location.Query().Get("client_id") != "client-id" || location.Query().Get("state") != "state-token-placeholder" {
		t.Fatalf("unexpected OAuth redirect: %s", location)
	}
}

func TestHandleGoogleCallbackExistingUser(t *testing.T) {
	repo := &fakeUserRepository{user: &DB.User{ID: "drn_existing", Email: "ada@example.com"}}
	manager, server := newOAuthTestManager(t, repo, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, map[string]string{"access_token": "token", "token_type": "Bearer"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, GoogleUser{ID: "google-id", Email: "ada@example.com", Name: "Ada"})
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	manager.HandleGoogleCallback(recorder, oauthRequest(server))

	if recorder.Code != http.StatusTemporaryRedirect || recorder.Header().Get("Location") != "/" {
		t.Fatalf("callback response = %d %q, want redirect to /", recorder.Code, recorder.Header().Get("Location"))
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session_token" || cookies[0].Value != "drn_existing" {
		t.Fatalf("session cookies = %v, want session_token=drn_existing", cookies)
	}
	if repo.created != nil {
		t.Fatal("existing user should not be created again")
	}
}

func TestHandleGoogleCallbackCreatesUser(t *testing.T) {
	repo := &fakeUserRepository{}
	manager, server := newOAuthTestManager(t, repo, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, map[string]string{"access_token": "token", "token_type": "Bearer"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, GoogleUser{ID: "google-id", Email: "ada@example.com", Name: "Ada"})
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	manager.HandleGoogleCallback(recorder, oauthRequest(server))

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("callback status = %d, want %d", recorder.Code, http.StatusTemporaryRedirect)
	}
	if repo.created == nil || repo.created.Name != "Ada" || repo.created.Email != "ada@example.com" {
		t.Fatalf("created user = %+v, want name and email from Google profile", repo.created)
	}
}

func TestHandleGoogleCallbackExchangeError(t *testing.T) {
	manager, server := newOAuthTestManager(t, &fakeUserRepository{}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "exchange failed", http.StatusBadRequest)
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	manager.HandleGoogleCallback(recorder, oauthRequest(server, "invalid"))

	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "Failed to exchange code") {
		t.Fatalf("exchange error response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestHandleGoogleCallbackCreateError(t *testing.T) {
	repo := &fakeUserRepository{createErr: errors.New("create failed")}
	manager, server := newOAuthTestManager(t, repo, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, map[string]string{"access_token": "token", "token_type": "Bearer"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, GoogleUser{Email: "ada@example.com", Name: "Ada"})
	}))
	defer server.Close()

	recorder := httptest.NewRecorder()
	manager.HandleGoogleCallback(recorder, oauthRequest(server))

	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "Failed to create user") {
		t.Fatalf("create error response = %d %q", recorder.Code, recorder.Body.String())
	}
}

func newOAuthTestManager(t *testing.T, repo userRepository, handler http.Handler) (*OAuthManager, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	manager := NewOAuthManager("client-id", "client-secret", server.URL+"/callback", repo)
	manager.Config.Endpoint = oauth2.Endpoint{AuthURL: server.URL + "/authorize", TokenURL: server.URL + "/token"}
	return manager, server
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode JSON response: %v", err)
	}
}

func oauthRequest(server *httptest.Server, code ...string) *http.Request {
	value := "valid"
	if len(code) > 0 {
		value = code[0]
	}
	client := &http.Client{Transport: routeGoogleProfileTransport{serverURL: server.URL, base: http.DefaultTransport}}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, client)
	return httptest.NewRequest(http.MethodGet, "/auth/google/callback?code="+value, nil).WithContext(ctx)
}

type routeGoogleProfileTransport struct {
	serverURL string
	base      http.RoundTripper
}

func (transport routeGoogleProfileTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host == "www.googleapis.com" && request.URL.Path == "/oauth2/v2/userinfo" {
		clone := request.Clone(request.Context())
		clone.URL, _ = url.Parse(transport.serverURL + "/userinfo")
		request = clone
	}
	return transport.base.RoundTrip(request)
}
