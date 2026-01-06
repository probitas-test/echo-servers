package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// OIDCDiscoveryResponse represents the OpenID Connect Discovery metadata
type OIDCDiscoveryResponse struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserInfoEndpoint                 string   `json:"userinfo_endpoint"`
	JwksURI                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
	GrantTypesSupported              []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported,omitempty"`
}

// OIDCDiscoveryHandler provides OpenID Connect Discovery metadata
// GET /oidc/{user}/{pass}/.well-known/openid-configuration
func OIDCDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")
	pass := chi.URLParam(r, "pass")

	// Build base URL from request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check for X-Forwarded-Proto header (proxy support)
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	baseURL := fmt.Sprintf("%s://%s", scheme, host)
	oidcBase := fmt.Sprintf("%s/oidc/%s/%s", baseURL, url.PathEscape(user), url.PathEscape(pass))

	// Get scopes from config, or use defaults if not configured
	supportedScopes := []string{"openid", "profile", "email"}
	if globalConfig != nil && len(globalConfig.OIDCSupportedScopes) > 0 {
		supportedScopes = globalConfig.OIDCSupportedScopes
	}

	discovery := OIDCDiscoveryResponse{
		Issuer:                oidcBase,
		AuthorizationEndpoint: oidcBase + "/authorize",
		TokenEndpoint:         oidcBase + "/token",
		UserInfoEndpoint:      oidcBase + "/userinfo",
		JwksURI:               oidcBase + "/.well-known/jwks.json",
		ResponseTypesSupported: []string{
			"code",
		},
		SubjectTypesSupported: []string{
			"public",
		},
		IDTokenSigningAlgValuesSupported: []string{
			"none", // Mock implementation - no actual JWT signing
		},
		ScopesSupported: supportedScopes,
		GrantTypesSupported: []string{
			"authorization_code",
		},
		CodeChallengeMethodsSupported: []string{
			"plain",
			"S256",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(discovery)
}

// validateScopes validates that all requested scopes are supported.
// Returns error if any requested scope is not in the configured supported scopes.
func validateScopes(requestedScope string) error {
	if requestedScope == "" {
		return nil // Empty scope will use defaults
	}

	requestedScopes := strings.Split(requestedScope, " ")
	supportedScopes := globalConfig.OIDCSupportedScopes

	for _, rs := range requestedScopes {
		rs = strings.TrimSpace(rs)
		if rs == "" {
			continue
		}

		found := false
		for _, ss := range supportedScopes {
			if rs == ss {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("unsupported scope: %s", rs)
		}
	}

	return nil
}

// getDefaultScopes returns default scopes as a space-separated string.
// Returns all configured scopes joined by spaces.
func getDefaultScopes() string {
	return strings.Join(globalConfig.OIDCSupportedScopes, " ")
}

// OIDCAuthorizeHandler handles OIDC authorization requests
// GET /oidc/{user}/{pass}/authorize - Display login form
// POST /oidc/{user}/{pass}/authorize - Process authentication
func OIDCAuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")
	pass := chi.URLParam(r, "pass")

	if r.Method == http.MethodGet {
		// GET: Display login form
		clientID := r.URL.Query().Get("client_id")
		redirectURI := r.URL.Query().Get("redirect_uri")
		scope := r.URL.Query().Get("scope")
		responseType := r.URL.Query().Get("response_type")
		state := r.URL.Query().Get("state") // Client-provided (optional)
		codeChallenge := r.URL.Query().Get("code_challenge")
		codeChallengeMethod := r.URL.Query().Get("code_challenge_method")
		nonce := r.URL.Query().Get("nonce") // OIDC nonce parameter (optional)

		// Validate client_id (REQUIRED per OIDC spec)
		if clientID == "" {
			writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "client_id parameter is required")
			return
		}

		// Validate client_id value if configured
		if globalConfig != nil && globalConfig.OIDCClientID != "" && clientID != globalConfig.OIDCClientID {
			writeAuthorizationError(w, r, ErrorUnauthorizedClient, "unknown client_id", state, redirectURI)
			return
		}

		// Validate required parameters
		if redirectURI == "" {
			writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "redirect_uri parameter is required")
			return
		}

		// Validate redirect_uri if validation is enabled
		if globalConfig != nil && globalConfig.OIDCValidateRedirectURI {
			var allowedPatterns []string
			if globalConfig.OIDCAllowedRedirectURIs != "" {
				// Split comma-separated patterns
				for _, pattern := range strings.Split(globalConfig.OIDCAllowedRedirectURIs, ",") {
					if trimmed := strings.TrimSpace(pattern); trimmed != "" {
						allowedPatterns = append(allowedPatterns, trimmed)
					}
				}
			}

			if err := validateRedirectURI(redirectURI, allowedPatterns); err != nil {
				writeAuthorizationError(w, r, ErrorInvalidRequest, "redirect_uri not in allowlist", state, redirectURI)
				return
			}
		}

		if responseType != "code" {
			writeAuthorizationError(w, r, ErrorUnsupportedResponseType, "only response_type=code is supported", state, redirectURI)
			return
		}

		// Validate and set default scope if not provided
		if scope == "" {
			scope = getDefaultScopes()
		} else {
			if err := validateScopes(scope); err != nil {
				writeAuthorizationError(w, r, ErrorInvalidScope, err.Error(), state, redirectURI)
				return
			}
		}

		// Validate PKCE parameters
		if globalConfig != nil && globalConfig.OIDCRequirePKCE && codeChallenge == "" {
			writeAuthorizationError(w, r, ErrorInvalidRequest, "code_challenge is required", state, redirectURI)
			return
		}

		// If code_challenge is provided, validate method
		if codeChallenge != "" {
			// Default to "plain" if method not specified (per RFC 7636 Section 4.3)
			if codeChallengeMethod == "" {
				codeChallengeMethod = "plain"
			}

			// Validate method is supported
			if codeChallengeMethod != "plain" && codeChallengeMethod != "S256" {
				writeAuthorizationError(w, r, ErrorInvalidRequest, "unsupported code_challenge_method", state, redirectURI)
				return
			}
		}

		// Create a new session with PKCE parameters and nonce
		session, err := DefaultSessionStore.CreateSession(state, redirectURI, scope, codeChallenge, codeChallengeMethod, nonce)
		if err != nil {
			writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to create session")
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "oidc_session",
			Value:    session.ID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		// Render login form
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl := template.Must(template.New("login").Parse(loginFormTemplate))
		data := struct {
			User         string
			Pass         string
			State        string
			RedirectURI  string
			Scope        string
			AuthorizeURL string
		}{
			User:         user,
			Pass:         pass,
			State:        session.State,
			RedirectURI:  redirectURI,
			Scope:        scope,
			AuthorizeURL: fmt.Sprintf("/oidc/%s/%s/authorize", url.PathEscape(user), url.PathEscape(pass)),
		}
		_ = tmpl.Execute(w, data)
		return
	}

	// POST: Process authentication
	expectedUser := user
	expectedPass := pass

	// Get session from cookie
	cookie, err := r.Cookie("oidc_session")
	if err != nil {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "session not found")
		return
	}

	session, ok := DefaultSessionStore.GetSession(cookie.Value)
	if !ok {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "invalid or expired session")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "invalid form data")
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	// Validate required parameters
	if username == "" || password == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "username and password are required")
		return
	}

	// Validate credentials against URL path
	if username != expectedUser || password != expectedPass {
		writeOIDCError(w, http.StatusUnauthorized, ErrorAccessDenied, "invalid username or password")
		return
	}

	// Generate authorization code using session's redirect_uri, PKCE parameters, and nonce
	authCode, err := DefaultSessionStore.CreateAuthCode(session.RedirectURI, username, session.Scope, session.CodeChallenge, session.CodeChallengeMethod, session.Nonce)
	if err != nil {
		writeOIDCError(w, http.StatusInternalServerError, ErrorServerError, "failed to create authorization code")
		return
	}

	// Delete the session as it's been used
	DefaultSessionStore.DeleteSession(session.ID)

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oidc_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Redirect back to the client with the authorization code and state
	redirectURL, _ := url.Parse(session.RedirectURI)
	query := redirectURL.Query()
	query.Set("code", authCode.Code)
	if session.State != "" {
		query.Set("state", session.State) // Only include if client provided it
	}
	redirectURL.RawQuery = query.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// OIDCCallbackHandler handles the callback from the authorization server
// GET /oidc/{user}/{pass}/callback?code={code}&state={state}
func OIDCCallbackHandler(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")
	pass := chi.URLParam(r, "pass")
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Build token endpoint URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	tokenEndpoint := fmt.Sprintf("%s://%s/oidc/%s/%s/token", scheme, host, url.PathEscape(user), url.PathEscape(pass))

	// Render callback page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("callback").Parse(callbackTemplate))
	data := struct {
		Code          string
		State         string
		Error         string
		TokenEndpoint string
	}{
		Code:          code,
		State:         state,
		TokenEndpoint: tokenEndpoint,
	}

	if code == "" {
		data.Error = "No authorization code received"
	}

	_ = tmpl.Execute(w, data)
}

// TokenResponse represents the response from the token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OIDCTokenHandler exchanges an authorization code for tokens
// POST /oidc/{user}/{pass}/token
func OIDCTokenHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "invalid form data")
		return
	}

	grantType := r.PostForm.Get("grant_type")
	code := r.PostForm.Get("code")
	redirectURI := r.PostForm.Get("redirect_uri")
	clientID := r.PostForm.Get("client_id")
	clientSecret := r.PostForm.Get("client_secret")
	codeVerifier := r.PostForm.Get("code_verifier")

	// Validate grant_type
	if grantType != "authorization_code" {
		writeOIDCError(w, http.StatusBadRequest, ErrorUnsupportedGrantType, "only grant_type=authorization_code is supported")
		return
	}

	// Validate client_id (REQUIRED per OIDC spec)
	if clientID == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "client_id parameter is required")
		return
	}

	// Validate client_id value if configured
	if globalConfig != nil && globalConfig.OIDCClientID != "" && clientID != globalConfig.OIDCClientID {
		writeOIDCError(w, http.StatusUnauthorized, ErrorInvalidClient, "unknown client_id")
		return
	}

	// Validate client_secret if configured (confidential client)
	if globalConfig != nil && globalConfig.OIDCClientSecret != "" {
		if clientSecret != globalConfig.OIDCClientSecret {
			writeOIDCError(w, http.StatusUnauthorized, ErrorInvalidClient, "invalid client_secret")
			return
		}
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
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidGrant, "invalid or expired authorization code")
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
		if !verifyCodeChallenge(authCode.CodeChallenge, authCode.CodeChallengeMethod, codeVerifier) {
			writeOIDCError(w, http.StatusBadRequest, ErrorInvalidGrant, "invalid code_verifier")
			return
		}
	}

	// Delete the authorization code (single-use)
	DefaultSessionStore.DeleteAuthCode(code)

	// Generate mock tokens
	accessToken, _ := generateRandomString(32)
	refreshToken, _ := generateRandomString(32)

	// Build issuer URL for ID token
	user := chi.URLParam(r, "user")
	pass := chi.URLParam(r, "pass")
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	issuer := fmt.Sprintf("%s://%s/oidc/%s/%s", scheme, host, url.PathEscape(user), url.PathEscape(pass))

	// Create ID token in JWT format with actual issuer, client_id, and nonce
	idToken := generateMockIDToken(issuer, clientID, authCode.Username, authCode.Nonce)

	response := TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: refreshToken,
		IDToken:      idToken,
		Scope:        authCode.Scope,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// generateMockIDToken creates a mock ID token in JWT format with algorithm "none".
// Returns a JWT in the format: header.payload.signature (where signature is empty for alg=none).
func generateMockIDToken(issuer, clientID, username, nonce string) string {
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
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
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

// verifyCodeChallenge verifies PKCE code_verifier against code_challenge.
// Supports "plain" and "S256" methods per RFC 7636.
func verifyCodeChallenge(challenge, method, verifier string) bool {
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

// validateRedirectURI validates that redirectURI matches one of the allowed patterns.
// Returns nil if validation passes, error otherwise.
// Empty or nil allowedPatterns means no restrictions (allow all).
// Supports wildcards: * for any port or path segment.
func validateRedirectURI(redirectURI string, allowedPatterns []string) error {
	if len(allowedPatterns) == 0 {
		return nil // No restrictions
	}

	for _, pattern := range allowedPatterns {
		if matchRedirectPattern(redirectURI, pattern) {
			return nil
		}
	}

	return fmt.Errorf("redirect_uri not in allowlist")
}

// matchRedirectPattern checks if uri matches pattern.
// Supports wildcards:
// - "http://localhost:*/callback" matches any port
// - "http://localhost:8080/*" matches any path
func matchRedirectPattern(uri, pattern string) bool {
	// Exact match
	if uri == pattern {
		return true
	}

	// Parse URI
	uriParsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// Handle pattern specially to support wildcard port
	// Replace :* with a valid port temporarily for parsing
	patternForParsing := strings.Replace(pattern, ":*", ":9999", 1)
	hasWildcardPort := patternForParsing != pattern

	patternParsed, err := url.Parse(patternForParsing)
	if err != nil {
		return false
	}

	// Scheme must match exactly
	if uriParsed.Scheme != patternParsed.Scheme {
		return false
	}

	// Host must match exactly
	uriHost := uriParsed.Hostname()
	patternHost := patternParsed.Hostname()
	if uriHost != patternHost {
		return false
	}

	// Port matching: support wildcard *
	if !hasWildcardPort {
		// Ports must match exactly (including both being empty for default ports)
		uriPort := uriParsed.Port()
		patternPort := patternParsed.Port()
		if uriPort != patternPort {
			return false
		}
	}
	// If hasWildcardPort, accept any port

	// Path matching: support wildcard *
	if patternParsed.Path == "/*" {
		return true // Any path allowed
	}

	if uriParsed.Path != patternParsed.Path {
		return false
	}

	return true
}

const loginFormTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>OIDC Login</title>
</head>
<body>
    <h1>OIDC Login</h1>
    <form method="POST" action="{{.AuthorizeURL}}">
        <p>
            <label>Username: <input type="text" name="username" required autofocus></label>
        </p>
        <p>
            <label>Password: <input type="password" name="password" required></label>
        </p>
        <p>
            <button type="submit">Login</button>
        </p>
    </form>
    <hr>
    <p>Expected: {{.User}}</p>
    <p>Scope: {{.Scope}}</p>
    <p>Redirect: {{.RedirectURI}}</p>
</body>
</html>`

const callbackTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>OIDC Callback</title>
    <style>
        .code-box { font-family: monospace; word-break: break-all; }
        #tokenResult { display: none; }
    </style>
    <script>
        function exchangeToken() {
            const code = document.getElementById('authCode').value;
            const clientId = document.getElementById('clientId').value || 'demo-client';
            const redirectUri = window.location.origin + window.location.pathname;
            const tokenEndpoint = '{{.TokenEndpoint}}';

            const formData = new URLSearchParams();
            formData.append('grant_type', 'authorization_code');
            formData.append('client_id', clientId);
            formData.append('code', code);
            formData.append('redirect_uri', redirectUri);

            fetch(tokenEndpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: formData
            })
            .then(response => {
                if (!response.ok) {
                    return response.text().then(text => { throw new Error(text); });
                }
                return response.json();
            })
            .then(data => {
                document.getElementById('tokenResult').style.display = 'block';
                document.getElementById('tokenData').textContent = JSON.stringify(data, null, 2);
            })
            .catch(error => {
                document.getElementById('tokenResult').style.display = 'block';
                document.getElementById('tokenData').textContent = 'Error: ' + error.message;
            });
        }
    </script>
</head>
<body>
    <h1>OIDC Callback</h1>
    {{if .Error}}
        <p><strong>Error:</strong> {{.Error}}</p>
    {{else}}
        <p>Authorization successful</p>
        <h2>Authorization Code</h2>
        <div class="code-box" id="authCode">{{.Code}}</div>
        <h2>State</h2>
        <div class="code-box">{{.State}}</div>
        <h2>Token Exchange</h2>
        <p>
            <label>Client ID: <input type="text" id="clientId" value="demo-client" style="width: 300px;"></label>
        </p>
        <button onclick="exchangeToken()">Exchange Code for Tokens</button>
        <div id="tokenResult">
            <h3>Tokens</h3>
            <pre id="tokenData"></pre>
        </div>
    {{end}}
</body>
</html>`

// OIDCDemoHandler provides an interactive demo of the OIDC flow
// GET /oidc/{user}/{pass}/demo
func OIDCDemoHandler(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")
	pass := chi.URLParam(r, "pass")

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	// If no code and no error, initiate OIDC flow
	if code == "" && errorParam == "" {
		// Generate state for CSRF protection (demo client generates its own state)
		demoState, err := generateRandomString(16)
		if err != nil {
			http.Error(w, "failed to generate state", http.StatusInternalServerError)
			return
		}

		// Store state in cookie for later verification
		http.SetCookie(w, &http.Cookie{
			Name:     "demo_state",
			Value:    demoState,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		// Build redirect URI for this demo page
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
			scheme = proto
		}

		host := r.Host
		redirectURI := fmt.Sprintf("%s://%s/oidc/%s/%s/demo", scheme, host, url.PathEscape(user), url.PathEscape(pass))
		authorizeURL := fmt.Sprintf("/oidc/%s/%s/authorize?redirect_uri=%s&response_type=code&scope=openid%%20profile%%20email&state=%s",
			url.PathEscape(user), url.PathEscape(pass), url.QueryEscape(redirectURI), demoState)

		http.Redirect(w, r, authorizeURL, http.StatusFound)
		return
	}

	// Verify state matches (CSRF protection)
	cookie, err := r.Cookie("demo_state")
	if err == nil && cookie.Value != state {
		errorParam = "state_mismatch"
	}

	// Clear demo state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "demo_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Render demo page with code/error
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("demo").Parse(demoPageTemplate))

	// Build token endpoint URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	tokenEndpoint := fmt.Sprintf("%s://%s/oidc/%s/%s/token", scheme, host, url.PathEscape(user), url.PathEscape(pass))
	redirectURI := fmt.Sprintf("%s://%s/oidc/%s/%s/demo", scheme, host, url.PathEscape(user), url.PathEscape(pass))

	data := struct {
		User          string
		Pass          string
		Code          string
		State         string
		Error         string
		TokenEndpoint string
		RedirectURI   string
	}{
		User:          user,
		Pass:          pass,
		Code:          code,
		State:         state,
		Error:         errorParam,
		TokenEndpoint: tokenEndpoint,
		RedirectURI:   redirectURI,
	}

	_ = tmpl.Execute(w, data)
}

// OIDCUserInfoHandler returns user information based on the access token
// GET /oidc/{user}/{pass}/userinfo
func OIDCUserInfoHandler(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")

	// Extract Bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		writeOIDCError(w, http.StatusUnauthorized, ErrorInvalidRequest, "missing authorization header")
		return
	}

	// Validate Bearer scheme
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		writeOIDCError(w, http.StatusUnauthorized, ErrorInvalidRequest, "invalid authorization scheme")
		return
	}

	// In a real implementation, we would validate the access token
	// For this mock server, we accept any non-empty Bearer token
	accessToken := parts[1]
	if accessToken == "" {
		writeOIDCError(w, http.StatusUnauthorized, ErrorInvalidRequest, "empty access token")
		return
	}

	// Return user information based on the user from URL path
	userInfo := map[string]interface{}{
		"sub":   user,
		"name":  user,
		"email": fmt.Sprintf("%s@example.com", user),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userInfo)
}

// JWKSResponse represents a JSON Web Key Set response
type JWKSResponse struct {
	Keys []interface{} `json:"keys"`
}

// OIDCJWKSHandler returns an empty JWKS (JSON Web Key Set)
// GET /oidc/{user}/{pass}/.well-known/jwks.json
func OIDCJWKSHandler(w http.ResponseWriter, r *http.Request) {
	// Return empty JWKS since we use alg="none" (no signature)
	jwks := JWKSResponse{
		Keys: []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jwks)
}

const demoPageTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>OIDC Demo</title>
    <style>
        .code-box { font-family: monospace; word-break: break-all; }
        #tokenResult { display: none; }
    </style>
    <script>
        function exchangeToken() {
            const code = '{{.Code}}';
            const redirectURI = '{{.RedirectURI}}';
            const tokenEndpoint = '{{.TokenEndpoint}}';
            const clientId = document.getElementById('clientId').value || 'demo-client';

            const formData = new URLSearchParams();
            formData.append('grant_type', 'authorization_code');
            formData.append('client_id', clientId);
            formData.append('code', code);
            formData.append('redirect_uri', redirectURI);

            fetch(tokenEndpoint, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: formData
            })
            .then(response => {
                if (!response.ok) {
                    return response.text().then(text => { throw new Error(text); });
                }
                return response.json();
            })
            .then(data => {
                document.getElementById('tokenResult').style.display = 'block';
                document.getElementById('tokenData').textContent = JSON.stringify(data, null, 2);
            })
            .catch(error => {
                document.getElementById('tokenResult').style.display = 'block';
                document.getElementById('tokenData').textContent = 'Error: ' + error.message;
            });
        }
    </script>
</head>
<body>
    <h1>OIDC Demo</h1>
    {{if .Error}}
        <p><strong>Error:</strong> {{.Error}}</p>
    {{else}}
        <p>Authorization successful</p>
        <h2>Authorization Code</h2>
        <div class="code-box">{{.Code}}</div>
        <h2>State</h2>
        <div class="code-box">{{.State}}</div>
        <h2>Token Exchange</h2>
        <p>
            <label>Client ID: <input type="text" id="clientId" value="demo-client" style="width: 300px;"></label>
        </p>
        <button onclick="exchangeToken()">Exchange Code for Tokens</button>
        <div id="tokenResult">
            <h3>Tokens</h3>
            <pre id="tokenData"></pre>
        </div>
    {{end}}
</body>
</html>`
