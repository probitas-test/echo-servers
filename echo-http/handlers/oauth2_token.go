package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OAuth2TokenHandler is the unified token endpoint for OAuth2/OIDC flows.
// Supports both authorization_code and client_credentials grant types.
// POST /oauth2/token
func OAuth2TokenHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "invalid form data")
		return
	}

	grantType := r.PostForm.Get("grant_type")

	// Validate grant_type is provided
	if err := validateGrantType(grantType, getAllowedGrantTypes()); err != nil {
		writeOIDCError(w, http.StatusBadRequest, ErrorUnsupportedGrantType, err.Error())
		return
	}

	// Route to appropriate grant handler
	switch grantType {
	case "authorization_code":
		handleAuthorizationCodeGrant(w, r)
	case "client_credentials":
		handleClientCredentialsGrant(w, r)
	case "password":
		handlePasswordGrant(w, r)
	case "refresh_token":
		handleRefreshTokenGrant(w, r)
	default:
		// This should never happen after validateGrantType, but handle defensively
		writeOIDCError(w, http.StatusBadRequest, ErrorUnsupportedGrantType, fmt.Sprintf("unsupported grant_type: %s", grantType))
	}
}

// handleClientCredentialsGrant handles the OAuth2 Client Credentials flow.
// Returns only access_token (no id_token, as there is no user context).
// RFC 6749 Section 4.4
func handleClientCredentialsGrant(w http.ResponseWriter, r *http.Request) {
	clientID := r.PostForm.Get("client_id")
	clientSecret := r.PostForm.Get("client_secret")
	scope := r.PostForm.Get("scope")

	// Validate client credentials (client_secret is required for confidential clients)
	if err := validateClientCredentials(clientID, clientSecret, true); err != nil {
		hint := buildClientCredentialsHint(r)
		writeOIDCErrorWithHint(w, http.StatusUnauthorized, ErrorInvalidClient, err.Error(), hint)
		return
	}

	// Validate and set default scope if not provided
	if scope == "" {
		scope = joinScopes(globalConfig.AuthSupportedScopes)
	} else {
		// Split and validate requested scopes
		requestedScopes := splitScopes(scope)
		for _, rs := range requestedScopes {
			found := false
			for _, ss := range globalConfig.AuthSupportedScopes {
				if rs == ss {
					found = true
					break
				}
			}
			if !found {
				writeOIDCError(w, http.StatusBadRequest, ErrorInvalidScope, fmt.Sprintf("unsupported scope: %s", rs))
				return
			}
		}
	}

	// Generate access token
	accessToken, err := generateRandomString(32)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to generate access token")
		return
	}

	// Get token expiry from config
	expiresIn := 3600 // Default 1 hour
	if globalConfig != nil && globalConfig.AuthTokenExpiry > 0 {
		expiresIn = globalConfig.AuthTokenExpiry
	}

	// Client Credentials flow does NOT include id_token or refresh_token
	response := TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		Scope:       scope,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleAuthorizationCodeGrant handles the OAuth2 Authorization Code flow with OIDC extension.
// Returns access_token, refresh_token, and id_token (OIDC).
// RFC 6749 Section 4.1 + OpenID Connect Core 1.0
func handleAuthorizationCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.PostForm.Get("code")
	redirectURI := r.PostForm.Get("redirect_uri")
	clientID := r.PostForm.Get("client_id")
	clientSecret := r.PostForm.Get("client_secret")
	codeVerifier := r.PostForm.Get("code_verifier")

	// Validate client_id (REQUIRED per OIDC spec)
	if clientID == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "client_id parameter is required")
		return
	}

	// Determine if client_secret is required based on configuration
	requireSecret := globalConfig != nil && globalConfig.AuthAllowedClientSecret != ""

	// Validate client credentials
	if err := validateClientCredentials(clientID, clientSecret, requireSecret); err != nil {
		writeOIDCError(w, http.StatusUnauthorized, ErrorInvalidClient, err.Error())
		return
	}

	// Validate required parameters
	if code == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "code parameter is required")
		return
	}

	if redirectURI == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "redirect_uri parameter is required")
		return
	}

	// Validate authorization code
	authCode, ok := DefaultSessionStore.GetAuthCode(code)
	if !ok {
		hint := buildAuthorizationCodeHint(r)
		writeOIDCErrorWithHint(w, http.StatusBadRequest, ErrorInvalidGrant, "invalid or expired authorization code", hint)
		return
	}

	// Validate redirect URI matches
	if authCode.RedirectURI != redirectURI {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidGrant, "redirect_uri mismatch")
		return
	}

	// Validate PKCE if code_challenge was provided during authorization
	if authCode.CodeChallenge != "" {
		// code_verifier is required when code_challenge was used
		if codeVerifier == "" {
			writeOIDCError(w, http.StatusBadRequest, ErrorInvalidGrant, "code_verifier is required")
			return
		}

		// RFC 7636 Section 4.1: code_verifier length must be 43-128 characters
		if len(codeVerifier) < 43 || len(codeVerifier) > 128 {
			writeOIDCError(w, http.StatusBadRequest, ErrorInvalidGrant, "code_verifier length must be between 43 and 128 characters (RFC 7636)")
			return
		}

		// Verify code_verifier against code_challenge
		if !verifyPKCECodeChallenge(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier) {
			writeOIDCError(w, http.StatusBadRequest, ErrorInvalidGrant, "invalid code_verifier")
			return
		}
	}

	// Delete the authorization code (single-use)
	DefaultSessionStore.DeleteAuthCode(code)

	// Generate access token
	accessToken, err := generateRandomString(32)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to generate access token")
		return
	}

	// Create refresh token and store it
	refreshTokenObj, err := DefaultSessionStore.CreateRefreshToken(authCode.Username, clientID, authCode.Scope, authCode.Nonce)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to generate refresh token")
		return
	}

	// Build issuer URL for ID token (base URL only for new endpoint)
	issuer := buildBaseURL(r)

	// Get token expiry from config
	expiresIn := 3600 // Default 1 hour
	if globalConfig != nil && globalConfig.AuthTokenExpiry > 0 {
		expiresIn = globalConfig.AuthTokenExpiry
	}

	// Create ID token in JWT format with actual issuer, client_id, and nonce
	idToken := generateOAuth2IDToken(issuer, clientID, authCode.Username, authCode.Nonce, expiresIn)

	response := TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: refreshTokenObj.Token,
		IDToken:      idToken,
		Scope:        authCode.Scope,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// generateOAuth2IDToken creates a mock ID token in JWT format with algorithm "none".
// Returns a JWT in the format: header.payload.signature (where signature is empty for alg=none).
// Used by the new OAuth2 endpoint (non-deprecated).
func generateOAuth2IDToken(issuer, clientID, username, nonce string, expiresIn int) string {
	// Header for JWT with alg="none"
	header := map[string]string{
		"alg": "none",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Payload (claims)
	claims := map[string]interface{}{
		"iss":   issuer,
		"sub":   username,
		"aud":   clientID,
		"exp":   time.Now().Add(time.Duration(expiresIn) * time.Second).Unix(),
		"iat":   time.Now().Unix(),
		"name":  username,
		"email": fmt.Sprintf("%s@example.com", username),
	}
	// Include nonce claim only if provided
	if nonce != "" {
		claims["nonce"] = nonce
	}
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// JWT format: header.payload.signature (empty signature for "none")
	return headerB64 + "." + claimsB64 + "."
}

// handlePasswordGrant handles the OAuth2 Resource Owner Password Credentials flow.
// Returns access_token, refresh_token, and optionally id_token (if openid scope requested).
// RFC 6749 Section 4.3 (deprecated in OAuth 2.1, but useful for testing)
func handlePasswordGrant(w http.ResponseWriter, r *http.Request) {
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")
	clientID := r.PostForm.Get("client_id")
	clientSecret := r.PostForm.Get("client_secret")
	scope := r.PostForm.Get("scope")

	// Validate client_id (REQUIRED)
	if clientID == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "client_id parameter is required")
		return
	}

	// Determine if client_secret is required based on configuration
	requireSecret := globalConfig != nil && globalConfig.AuthAllowedClientSecret != ""

	// Validate client credentials
	if err := validateClientCredentials(clientID, clientSecret, requireSecret); err != nil {
		hint := buildPasswordGrantHint(r)
		writeOIDCErrorWithHint(w, http.StatusUnauthorized, ErrorInvalidClient, err.Error(), hint)
		return
	}

	// Validate username and password against configured credentials
	if err := validateBasicAuthCredentials(username, password); err != nil {
		hint := buildPasswordGrantHint(r)
		writeOIDCErrorWithHint(w, http.StatusUnauthorized, ErrorInvalidGrant, "invalid username or password", hint)
		return
	}

	// Validate and set default scope if not provided
	if scope == "" {
		scope = joinScopes(globalConfig.AuthSupportedScopes)
	} else {
		// Split and validate requested scopes
		requestedScopes := splitScopes(scope)
		for _, rs := range requestedScopes {
			found := false
			for _, ss := range globalConfig.AuthSupportedScopes {
				if rs == ss {
					found = true
					break
				}
			}
			if !found {
				writeOIDCError(w, http.StatusBadRequest, ErrorInvalidScope, fmt.Sprintf("unsupported scope: %s", rs))
				return
			}
		}
	}

	// Generate access token
	accessToken, err := generateRandomString(32)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to generate access token")
		return
	}

	// Get token expiry from config
	expiresIn := 3600 // Default 1 hour
	if globalConfig != nil && globalConfig.AuthTokenExpiry > 0 {
		expiresIn = globalConfig.AuthTokenExpiry
	}

	// Create refresh token and store it
	refreshTokenObj, err := DefaultSessionStore.CreateRefreshToken(username, clientID, scope, "")
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to generate refresh token")
		return
	}

	// Build response
	response := TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: refreshTokenObj.Token,
		Scope:        scope,
	}

	// Include id_token only if openid scope is requested
	if sliceContains(splitScopes(scope), "openid") {
		issuer := buildBaseURL(r)
		response.IDToken = generateOAuth2IDToken(issuer, clientID, username, "", expiresIn)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// handleRefreshTokenGrant handles the OAuth2 Refresh Token flow.
// Returns new access_token, optionally new refresh_token, and optionally id_token.
// RFC 6749 Section 6
func handleRefreshTokenGrant(w http.ResponseWriter, r *http.Request) {
	refreshToken := r.PostForm.Get("refresh_token")
	clientID := r.PostForm.Get("client_id")
	clientSecret := r.PostForm.Get("client_secret")
	scope := r.PostForm.Get("scope")

	// Validate client_id (REQUIRED)
	if clientID == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "client_id parameter is required")
		return
	}

	// Determine if client_secret is required based on configuration
	requireSecret := globalConfig != nil && globalConfig.AuthAllowedClientSecret != ""

	// Validate client credentials
	if err := validateClientCredentials(clientID, clientSecret, requireSecret); err != nil {
		writeOIDCError(w, http.StatusUnauthorized, ErrorInvalidClient, err.Error())
		return
	}

	// Validate refresh_token parameter
	if refreshToken == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "refresh_token parameter is required")
		return
	}

	// Validate refresh token exists and is not expired
	storedToken, ok := DefaultSessionStore.GetRefreshToken(refreshToken)
	if !ok {
		hint := buildRefreshTokenHint(r)
		writeOIDCErrorWithHint(w, http.StatusBadRequest, ErrorInvalidGrant, "invalid or expired refresh token", hint)
		return
	}

	// Validate client_id matches the one that originally obtained the refresh token
	if storedToken.ClientID != clientID {
		hint := buildRefreshTokenHint(r)
		writeOIDCErrorWithHint(w, http.StatusBadRequest, ErrorInvalidGrant, "client_id mismatch", hint)
		return
	}

	// Handle scope parameter
	// If scope is provided, it must not exceed the original scope
	finalScope := storedToken.Scope
	if scope != "" {
		requestedScopes := splitScopes(scope)
		originalScopes := splitScopes(storedToken.Scope)

		// Verify all requested scopes were in the original grant
		for _, rs := range requestedScopes {
			if !sliceContains(originalScopes, rs) {
				writeOIDCError(w, http.StatusBadRequest, ErrorInvalidScope, fmt.Sprintf("scope exceeds original grant: %s", rs))
				return
			}
		}
		finalScope = scope
	}

	// Generate new access token
	accessToken, err := generateRandomString(32)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to generate access token")
		return
	}

	// Get token expiry from config
	expiresIn := 3600 // Default 1 hour
	if globalConfig != nil && globalConfig.AuthTokenExpiry > 0 {
		expiresIn = globalConfig.AuthTokenExpiry
	}

	// Optionally issue a new refresh token (rotation)
	// For simplicity, we'll reuse the same refresh token
	// In production, you might want to implement refresh token rotation

	// Build response
	response := TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    expiresIn,
		RefreshToken: refreshToken, // Reuse same refresh token
		Scope:        finalScope,
	}

	// Include id_token only if openid scope is in the final scope
	if sliceContains(splitScopes(finalScope), "openid") {
		issuer := buildBaseURL(r)
		response.IDToken = generateOAuth2IDToken(issuer, clientID, storedToken.Username, storedToken.Nonce, expiresIn)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// verifyPKCECodeChallenge verifies PKCE code_verifier against code_challenge.
// Supports "plain" and "S256" methods per RFC 7636.
func verifyPKCECodeChallenge(challenge, method, verifier string) bool {
	switch method {
	case "plain":
		// Plain method: challenge == verifier
		return challenge == verifier

	case "S256":
		// S256 method: challenge == BASE64URL(SHA256(ASCII(verifier)))
		h := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(h[:])
		return challenge == computed

	default:
		// Unknown or empty method
		return false
	}
}
