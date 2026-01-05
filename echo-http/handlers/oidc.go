package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
)

// OIDCDiscoveryResponse represents the OpenID Connect Discovery metadata
type OIDCDiscoveryResponse struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
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

	discovery := OIDCDiscoveryResponse{
		Issuer:                oidcBase,
		AuthorizationEndpoint: oidcBase + "/authorize",
		TokenEndpoint:         oidcBase + "/token",
		ResponseTypesSupported: []string{
			"code",
		},
		SubjectTypesSupported: []string{
			"public",
		},
		IDTokenSigningAlgValuesSupported: []string{
			"none", // Mock implementation - no actual JWT signing
		},
		ScopesSupported: []string{
			"openid",
			"profile",
			"email",
		},
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

// OIDCAuthorizeHandler handles OIDC authorization requests
// GET /oidc/{user}/{pass}/authorize - Display login form
// POST /oidc/{user}/{pass}/authorize - Process authentication
func OIDCAuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	user := chi.URLParam(r, "user")
	pass := chi.URLParam(r, "pass")

	if r.Method == http.MethodGet {
		// GET: Display login form
		redirectURI := r.URL.Query().Get("redirect_uri")
		scope := r.URL.Query().Get("scope")
		responseType := r.URL.Query().Get("response_type")
		state := r.URL.Query().Get("state") // Client-provided (optional)

		// Validate required parameters
		if redirectURI == "" {
			http.Error(w, "redirect_uri parameter is required", http.StatusBadRequest)
			return
		}

		if responseType != "code" {
			http.Error(w, "only response_type=code is supported", http.StatusBadRequest)
			return
		}

		// Set default scope if not provided
		if scope == "" {
			scope = "openid profile email"
		}

		// Create a new session
		session, err := DefaultSessionStore.CreateSession(state, redirectURI, scope)
		if err != nil {
			http.Error(w, "failed to create session", http.StatusInternalServerError)
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
		http.Error(w, "session not found", http.StatusBadRequest)
		return
	}

	session, ok := DefaultSessionStore.GetSession(cookie.Value)
	if !ok {
		http.Error(w, "invalid or expired session", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	// Validate required parameters
	if username == "" || password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	// Validate credentials against URL path
	if username != expectedUser || password != expectedPass {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate authorization code using session's redirect_uri
	authCode, err := DefaultSessionStore.CreateAuthCode(session.RedirectURI, username, session.Scope)
	if err != nil {
		http.Error(w, "failed to create authorization code", http.StatusInternalServerError)
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
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Render callback page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("callback").Parse(callbackTemplate))
	data := struct {
		Code  string
		State string
		Error string
	}{
		Code:  code,
		State: state,
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
		http.Error(w, "invalid form data", http.StatusBadRequest)
		return
	}

	grantType := r.PostForm.Get("grant_type")
	code := r.PostForm.Get("code")
	redirectURI := r.PostForm.Get("redirect_uri")

	// Validate grant_type
	if grantType != "authorization_code" {
		http.Error(w, "only grant_type=authorization_code is supported", http.StatusBadRequest)
		return
	}

	// Validate required parameters
	if code == "" {
		http.Error(w, "code parameter is required", http.StatusBadRequest)
		return
	}

	if redirectURI == "" {
		http.Error(w, "redirect_uri parameter is required", http.StatusBadRequest)
		return
	}

	// Validate authorization code
	authCode, ok := DefaultSessionStore.GetAuthCode(code)
	if !ok {
		http.Error(w, "invalid or expired authorization code", http.StatusBadRequest)
		return
	}

	// Validate redirect URI matches
	if authCode.RedirectURI != redirectURI {
		http.Error(w, "redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	// Delete the authorization code (single-use)
	DefaultSessionStore.DeleteAuthCode(code)

	// Generate mock tokens
	accessToken, _ := generateRandomString(32)
	refreshToken, _ := generateRandomString(32)

	// Create a simple mock ID token (not a real JWT, just for demonstration)
	idToken := generateMockIDToken(authCode.Username)

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

// generateMockIDToken creates a mock ID token payload (not a real JWT)
func generateMockIDToken(username string) string {
	claims := map[string]interface{}{
		"iss":   "http://localhost/oidc",
		"sub":   username,
		"aud":   "mock-client-id",
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"name":  username,
		"email": fmt.Sprintf("%s@example.com", username),
	}

	// In a real implementation, this would be a properly signed JWT
	// For mock purposes, we just return a JSON string
	jsonBytes, _ := json.Marshal(claims)
	return string(jsonBytes)
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
            const redirectUri = 'http://localhost/oidc/callback';
            const formData = new URLSearchParams();
            formData.append('grant_type', 'authorization_code');
            formData.append('code', code);
            formData.append('redirect_uri', redirectUri);
            fetch('/oidc/token', {
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

            const formData = new URLSearchParams();
            formData.append('grant_type', 'authorization_code');
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
        <button onclick="exchangeToken()">Exchange Code for Tokens</button>
        <div id="tokenResult">
            <h3>Tokens</h3>
            <pre id="tokenData"></pre>
        </div>
    {{end}}
</body>
</html>`
