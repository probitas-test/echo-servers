package handlers

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// validateClientCredentials validates client_id and client_secret against configured values.
// Returns error with appropriate message if validation fails.
// If requireSecret is true, validates both client_id and client_secret.
// If requireSecret is false, only validates client_id.
func validateClientCredentials(clientID, clientSecret string, requireSecret bool) error {
	if clientID == "" {
		return errors.New("client_id is required")
	}

	// If no client_id is configured, accept any client (permissive mode for testing)
	if globalConfig == nil || globalConfig.AuthAllowedClientID == "" {
		return nil
	}

	// Validate client_id
	if clientID != globalConfig.AuthAllowedClientID {
		return errors.New("unknown client_id")
	}

	// Validate client_secret if required (confidential client)
	if requireSecret || globalConfig.AuthAllowedClientSecret != "" {
		if globalConfig.AuthAllowedClientSecret == "" {
			return errors.New("client_secret is required but not configured")
		}
		if !constantTimeCompare(clientSecret, globalConfig.AuthAllowedClientSecret) {
			return errors.New("invalid client_secret")
		}
	}

	return nil
}

// validateGrantType checks if the requested grant type is in the allowed list.
// Returns error if not supported.
func validateGrantType(grantType string, allowedTypes []string) error {
	if grantType == "" {
		return errors.New("grant_type is required")
	}

	if !isGrantTypeAllowed(grantType, allowedTypes) {
		return fmt.Errorf("unsupported grant_type: %s", grantType)
	}

	return nil
}

// validateBasicAuthCredentials validates username and password against configured values.
// Uses constant-time comparison to prevent timing attacks.
// Returns error if credentials don't match or are not configured.
func validateBasicAuthCredentials(username, password string) error {
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}

	// Check if credentials are configured
	if globalConfig == nil || globalConfig.AuthAllowedUsername == "" || globalConfig.AuthAllowedPassword == "" {
		return errors.New("authentication credentials not configured")
	}

	// Validate using constant-time comparison to prevent timing attacks
	usernameMatch := constantTimeCompare(username, globalConfig.AuthAllowedUsername)
	passwordMatch := constantTimeCompare(password, globalConfig.AuthAllowedPassword)

	if !usernameMatch || !passwordMatch {
		return errors.New("invalid username or password")
	}

	return nil
}

// isGrantTypeAllowed checks if a grant type is in the allowed list.
func isGrantTypeAllowed(grantType string, allowedTypes []string) bool {
	for _, allowed := range allowedTypes {
		if grantType == allowed {
			return true
		}
	}
	return false
}

// buildBaseURL constructs the base URL from the request, respecting X-Forwarded-Proto.
func buildBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Respect X-Forwarded-Proto header (common in reverse proxy setups)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	return fmt.Sprintf("%s://%s", scheme, host)
}

// buildIssuerURL constructs the issuer URL based on the request.
// For deprecated endpoints: includes /oidc/{user}/{pass}
// For new endpoints: uses base URL only
func buildIssuerURL(r *http.Request, user, pass string, deprecated bool) string {
	baseURL := buildBaseURL(r)
	if deprecated && user != "" && pass != "" {
		return fmt.Sprintf("%s/oidc/%s/%s", baseURL, user, pass)
	}
	return baseURL
}

// constantTimeCompare performs constant-time string comparison to prevent timing attacks.
// Returns true if strings are equal, false otherwise.
func constantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// getAllowedGrantTypes returns the list of allowed grant types from global config.
// If not configured, returns default grant types.
func getAllowedGrantTypes() []string {
	if globalConfig != nil && len(globalConfig.AuthAllowedGrantTypes) > 0 {
		return globalConfig.AuthAllowedGrantTypes
	}
	// Default grant types
	return []string{"authorization_code", "client_credentials"}
}

// joinScopes joins a slice of scopes into a space-separated string.
func joinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}

// splitScopes splits a space-separated scope string into a slice of scopes.
func splitScopes(scope string) []string {
	if scope == "" {
		return []string{}
	}
	scopes := strings.Split(scope, " ")
	result := make([]string, 0, len(scopes))
	for _, s := range scopes {
		if trimmed := strings.TrimSpace(s); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// sliceContains checks if a string slice contains a specific string
func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
