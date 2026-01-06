package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// BasicAuthEnvHandler validates Basic Authentication credentials against environment variables.
// Uses AUTH_ALLOWED_USERNAME and AUTH_ALLOWED_PASSWORD from configuration.
// GET /basic-auth - Returns 200 if credentials match, 401 otherwise
func BasicAuthEnvHandler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		writeBasicAuthError(w, r)
		return
	}

	// Validate credentials against environment variables
	if err := validateBasicAuthCredentials(user, pass); err != nil {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		writeBasicAuthError(w, r)
		return
	}

	response := AuthResponse{
		Authenticated: true,
		User:          user,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// writeBasicAuthError writes a 401 response with helpful curl examples.
func writeBasicAuthError(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)

	var username, password string
	if globalConfig != nil && globalConfig.AuthAllowedUsername != "" && globalConfig.AuthAllowedPassword != "" {
		username = globalConfig.AuthAllowedUsername
		password = globalConfig.AuthAllowedPassword
	} else {
		username = "username"
		password = "password"
	}

	message := fmt.Sprintf(`Unauthorized

This endpoint requires Basic Authentication.

Example usage:
  curl -u %s:%s %s

Or with explicit Authorization header:
  curl -H "Authorization: Basic $(echo -n '%s:%s' | base64)" %s

Configure credentials via environment variables:
  AUTH_ALLOWED_USERNAME=%s
  AUTH_ALLOWED_PASSWORD=%s
`,
		username, password, baseURL,
		username, password, baseURL,
		username, password,
	)

	_, _ = w.Write([]byte(message))
}
