package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type AuthResponse struct {
	Authenticated bool   `json:"authenticated"`
	User          string `json:"user,omitempty"`
	Token         string `json:"token,omitempty"`
}

// BasicAuthHandler validates Basic Authentication credentials.
// GET /basic-auth/{user}/{pass} - Returns 200 if credentials match, 401 otherwise
func BasicAuthHandler(w http.ResponseWriter, r *http.Request) {
	expectedUser := chi.URLParam(r, "user")
	expectedPass := chi.URLParam(r, "pass")

	user, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Use constant-time comparison to prevent timing attacks
	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(expectedUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) == 1

	if !userMatch || !passMatch {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := AuthResponse{
		Authenticated: true,
		User:          user,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// HiddenBasicAuthHandler is similar to BasicAuthHandler but doesn't prompt for credentials.
// GET /hidden-basic-auth/{user}/{pass} - Returns 404 instead of 401 if not authenticated
func HiddenBasicAuthHandler(w http.ResponseWriter, r *http.Request) {
	expectedUser := chi.URLParam(r, "user")
	expectedPass := chi.URLParam(r, "pass")

	user, pass, ok := r.BasicAuth()
	if !ok {
		http.NotFound(w, r)
		return
	}

	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(expectedUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) == 1

	if !userMatch || !passMatch {
		http.NotFound(w, r)
		return
	}

	response := AuthResponse{
		Authenticated: true,
		User:          user,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// BearerHandler validates Bearer token authentication.
// GET /bearer - Returns 200 if valid Bearer token present, 401 otherwise
func BearerHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := parts[1]
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := AuthResponse{
		Authenticated: true,
		Token:         token,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
