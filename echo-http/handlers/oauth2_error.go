package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// OIDCError represents an OAuth 2.0/OIDC error response.
// Spec: RFC 6749 Section 5.2, OIDC Core Section 3.1.2.6
type OIDCError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
	Hint             string `json:"hint,omitempty"` // Non-standard but helpful for developers
}

// Standard OAuth 2.0 error codes
const (
	ErrorInvalidRequest          = "invalid_request"
	ErrorUnauthorizedClient      = "unauthorized_client"
	ErrorAccessDenied            = "access_denied"
	ErrorUnsupportedResponseType = "unsupported_response_type"
	ErrorInvalidScope            = "invalid_scope"
	ErrorServerError             = "server_error"
	ErrorTemporarilyUnavailable  = "temporarily_unavailable"
	ErrorInvalidClient           = "invalid_client"
	ErrorInvalidGrant            = "invalid_grant"
	ErrorUnsupportedGrantType    = "unsupported_grant_type"
)

// writeOIDCError writes an OAuth 2.0/OIDC compliant error response.
func writeOIDCError(w http.ResponseWriter, statusCode int, errorCode, description string) {
	writeOIDCErrorWithHint(w, statusCode, errorCode, description, "")
}

// writeOIDCErrorWithHint writes an OAuth 2.0/OIDC error response with an optional hint.
// The hint field contains helpful curl examples for developers.
func writeOIDCErrorWithHint(w http.ResponseWriter, statusCode int, errorCode, description, hint string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResp := OIDCError{
		Error:            errorCode,
		ErrorDescription: description,
		Hint:             hint,
	}

	_ = json.NewEncoder(w).Encode(errResp)
}

// writeAuthorizationError writes an error for authorization endpoint.
// Per OIDC spec, these errors should redirect to redirect_uri with error in query.
func writeAuthorizationError(w http.ResponseWriter, r *http.Request, errorCode, description, state, redirectURI string) {
	if redirectURI == "" {
		// No redirect_uri, return JSON error
		writeOIDCError(w, http.StatusBadRequest, errorCode, description)
		return
	}

	// Build error redirect
	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "invalid redirect_uri")
		return
	}

	query := redirectURL.Query()
	query.Set("error", errorCode)
	if description != "" {
		query.Set("error_description", description)
	}
	if state != "" {
		query.Set("state", state)
	}
	redirectURL.RawQuery = query.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// buildClientCredentialsHint builds a hint for client_credentials grant errors.
func buildClientCredentialsHint(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	tokenURL := fmt.Sprintf("%s://%s/oauth2/token", scheme, r.Host)

	var clientID, clientSecret string
	if globalConfig != nil && globalConfig.AuthAllowedClientID != "" {
		clientID = globalConfig.AuthAllowedClientID
		clientSecret = globalConfig.AuthAllowedClientSecret
		if clientSecret == "" {
			clientSecret = "<not-required>"
		}
	} else {
		clientID = "your-client-id"
		clientSecret = "your-client-secret"
	}

	return fmt.Sprintf(`Example usage:
  curl -X POST %s \
    -d "grant_type=client_credentials" \
    -d "client_id=%s" \
    -d "client_secret=%s"

Configure via environment variables:
  AUTH_ALLOWED_CLIENT_ID=%s
  AUTH_ALLOWED_CLIENT_SECRET=%s
  AUTH_ALLOWED_GRANT_TYPES=client_credentials`, tokenURL, clientID, clientSecret, clientID, clientSecret)
}

// buildPasswordGrantHint builds a hint for password grant errors.
func buildPasswordGrantHint(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	tokenURL := fmt.Sprintf("%s://%s/oauth2/token", scheme, r.Host)

	var clientID, username, password string
	if globalConfig != nil && globalConfig.AuthAllowedClientID != "" {
		clientID = globalConfig.AuthAllowedClientID
		username = globalConfig.AuthAllowedUsername
		password = globalConfig.AuthAllowedPassword
		if username == "" {
			username = "username"
		}
		if password == "" {
			password = "password"
		}
	} else {
		clientID = "your-client-id"
		username = "username"
		password = "password"
	}

	return fmt.Sprintf(`Example usage:
  curl -X POST %s \
    -d "grant_type=password" \
    -d "client_id=%s" \
    -d "username=%s" \
    -d "password=%s" \
    -d "scope=openid profile"

Configure via environment variables:
  AUTH_ALLOWED_CLIENT_ID=%s
  AUTH_ALLOWED_USERNAME=%s
  AUTH_ALLOWED_PASSWORD=%s
  AUTH_ALLOWED_GRANT_TYPES=password`, tokenURL, clientID, username, password, clientID, username, password)
}

// buildAuthorizationCodeHint builds a hint for authorization_code grant errors.
func buildAuthorizationCodeHint(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	var clientID string
	if globalConfig != nil && globalConfig.AuthAllowedClientID != "" {
		clientID = globalConfig.AuthAllowedClientID
	} else {
		clientID = "your-client-id"
	}

	return fmt.Sprintf(`The authorization code is invalid or expired. Start the flow again:

1. Get authorization code:
   %s/oauth2/authorize?client_id=%s&redirect_uri=http://localhost/callback&response_type=code&scope=openid

2. Exchange code for token:
   curl -X POST %s/oauth2/token \
     -d "grant_type=authorization_code" \
     -d "client_id=%s" \
     -d "code=YOUR_AUTH_CODE" \
     -d "redirect_uri=http://localhost/callback"

Or try the demo page:
   %s/oauth2/demo`, baseURL, clientID, baseURL, clientID, baseURL)
}

// buildRefreshTokenHint builds a hint for refresh_token grant errors.
func buildRefreshTokenHint(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	var clientID string
	if globalConfig != nil && globalConfig.AuthAllowedClientID != "" {
		clientID = globalConfig.AuthAllowedClientID
	} else {
		clientID = "your-client-id"
	}

	return fmt.Sprintf(`The refresh token is invalid or expired. Re-authenticate to get a new refresh token:

Start authorization flow:
  %s/oauth2/authorize?client_id=%s&redirect_uri=http://localhost/callback&response_type=code&scope=openid

Or use password grant (if enabled):
  curl -X POST %s/oauth2/token \
    -d "grant_type=password" \
    -d "client_id=%s" \
    -d "username=..." \
    -d "password=..."`, baseURL, clientID, baseURL, clientID)
}

// buildUserInfoHint builds a hint for UserInfo endpoint errors.
func buildUserInfoHint(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	return fmt.Sprintf(`This endpoint requires a valid Bearer token.

1. Get a token first:
   curl -X POST %s/oauth2/token \
     -d "grant_type=client_credentials" \
     -d "client_id=your-client-id" \
     -d "client_secret=your-client-secret"

2. Use the token:
   curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" %s/oauth2/userinfo

Or use the demo page to complete the flow:
   %s/oauth2/demo`, baseURL, baseURL, baseURL)
}
