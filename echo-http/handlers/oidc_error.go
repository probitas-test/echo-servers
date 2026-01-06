package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
)

// OIDCError represents an OAuth 2.0/OIDC error response.
// Spec: RFC 6749 Section 5.2, OIDC Core Section 3.1.2.6
type OIDCError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResp := OIDCError{
		Error:            errorCode,
		ErrorDescription: description,
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
