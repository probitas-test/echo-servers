package handlers

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// BearerAuthEnvHandler validates Bearer token authentication against environment variables.
// The expected token is SHA1(username:password) where username and password are from
// AUTH_ALLOWED_USERNAME and AUTH_ALLOWED_PASSWORD configuration.
// GET /bearer-auth - Returns 200 if token matches, 401 otherwise
func BearerAuthEnvHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		writeBearerAuthError(w, r)
		return
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		writeBearerAuthError(w, r)
		return
	}

	token := parts[1]
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		writeBearerAuthError(w, r)
		return
	}

	// Check if credentials are configured
	if globalConfig == nil || globalConfig.AuthAllowedUsername == "" || globalConfig.AuthAllowedPassword == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		writeBearerAuthError(w, r)
		return
	}

	// Compute expected token as SHA1(username:password)
	expectedToken := computeBearerToken(globalConfig.AuthAllowedUsername, globalConfig.AuthAllowedPassword)

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		writeBearerAuthError(w, r)
		return
	}

	response := AuthResponse{
		Authenticated: true,
		Token:         token,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// writeBearerAuthError writes a 401 response with helpful curl examples.
func writeBearerAuthError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)

	var username, password, token string
	if globalConfig != nil && globalConfig.AuthAllowedUsername != "" && globalConfig.AuthAllowedPassword != "" {
		username = globalConfig.AuthAllowedUsername
		password = globalConfig.AuthAllowedPassword
		token = computeBearerToken(username, password)
	} else {
		username = "username"
		password = "password"
		token = computeBearerToken(username, password)
	}

	message := fmt.Sprintf(`Unauthorized

This endpoint requires Bearer token authentication.
The token is SHA1(username:password).

Example usage:
  curl -H "Authorization: Bearer %s" %s

Generate token:
  echo -n "%s:%s" | shasum -a 1 | cut -d' ' -f1

Configure credentials via environment variables:
  AUTH_ALLOWED_USERNAME=%s
  AUTH_ALLOWED_PASSWORD=%s
`,
		token, baseURL,
		username, password,
		username, password,
	)

	_, _ = w.Write([]byte(message))
}

// computeBearerToken computes SHA1 hash of "username:password"
func computeBearerToken(username, password string) string {
	data := fmt.Sprintf("%s:%s", username, password)
	hash := sha1.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}
