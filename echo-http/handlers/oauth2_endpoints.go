package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
)

// OAuth2CallbackHandler handles the callback from the authorization server.
// GET /oauth2/callback?code={code}&state={state}
func OAuth2CallbackHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	// Build token endpoint URL
	baseURL := buildBaseURL(r)
	tokenEndpoint := baseURL + "/oauth2/token"

	// Render callback page
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("callback").Parse(oauth2CallbackTemplate))
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

// OAuth2UserInfoHandler returns user information based on the access token.
// GET /oauth2/userinfo
// Requires Bearer token in Authorization header.
func OAuth2UserInfoHandler(w http.ResponseWriter, r *http.Request) {
	// Extract Bearer token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		hint := buildUserInfoHint(r)
		writeOIDCErrorWithHint(w, http.StatusUnauthorized, ErrorInvalidRequest, "missing authorization header", hint)
		return
	}

	// Validate Bearer scheme
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		hint := buildUserInfoHint(r)
		writeOIDCErrorWithHint(w, http.StatusUnauthorized, ErrorInvalidRequest, "invalid authorization scheme", hint)
		return
	}

	// In a real implementation, we would validate the access token
	// For this mock server, we accept any non-empty Bearer token
	accessToken := parts[1]
	if accessToken == "" {
		hint := buildUserInfoHint(r)
		writeOIDCErrorWithHint(w, http.StatusUnauthorized, ErrorInvalidRequest, "empty access token", hint)
		return
	}

	// Return generic user information (mock implementation)
	// In a real implementation, we would look up the user associated with the token
	username := "mockuser"
	if globalConfig != nil && globalConfig.AuthAllowedUsername != "" {
		username = globalConfig.AuthAllowedUsername
	}

	userInfo := map[string]interface{}{
		"sub":   username,
		"name":  username,
		"email": fmt.Sprintf("%s@example.com", username),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(userInfo)
}

// OAuth2DemoHandler provides an interactive demo of the OAuth2/OIDC flow.
// GET /oauth2/demo
func OAuth2DemoHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	// If no code and no error, initiate OAuth2 flow
	if code == "" && errorParam == "" {
		// Generate state for CSRF protection (demo client generates its own state)
		demoState, err := generateRandomString(16)
		if err != nil {
			http.Error(w, "failed to generate state", http.StatusInternalServerError)
			return
		}

		// Store state in cookie for later verification
		http.SetCookie(w, &http.Cookie{
			Name:     "oauth2_demo_state",
			Value:    demoState,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		// Build redirect URI for this demo page
		baseURL := buildBaseURL(r)
		redirectURI := baseURL + "/oauth2/demo"
		authorizeURL := fmt.Sprintf("/oauth2/authorize?client_id=demo-client&redirect_uri=%s&response_type=code&scope=openid%%20profile%%20email&state=%s",
			url.QueryEscape(redirectURI), demoState)

		http.Redirect(w, r, authorizeURL, http.StatusFound)
		return
	}

	// Verify state matches (CSRF protection)
	cookie, err := r.Cookie("oauth2_demo_state")
	if err == nil && cookie.Value != state {
		errorParam = "state_mismatch"
	}

	// Clear demo state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oauth2_demo_state",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Render demo page with code/error
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("demo").Parse(oauth2DemoPageTemplate))

	// Build URLs
	baseURL := buildBaseURL(r)
	tokenEndpoint := baseURL + "/oauth2/token"
	redirectURI := baseURL + "/oauth2/demo"

	data := struct {
		Code          string
		State         string
		Error         string
		TokenEndpoint string
		RedirectURI   string
	}{
		Code:          code,
		State:         state,
		Error:         errorParam,
		TokenEndpoint: tokenEndpoint,
		RedirectURI:   redirectURI,
	}

	_ = tmpl.Execute(w, data)
}

const oauth2CallbackTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>OAuth2 Callback</title>
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
    <h1>OAuth2 Callback</h1>
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

const oauth2DemoPageTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>OAuth2 Demo</title>
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
    <h1>OAuth2 Demo</h1>
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
