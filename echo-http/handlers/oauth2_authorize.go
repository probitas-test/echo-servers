package handlers

import (
	"fmt"
	"html/template"
	"net/http"
)

// OAuth2AuthorizeHandler handles OAuth2/OIDC authorization requests with environment-based authentication.
// Uses AUTH_ALLOWED_USERNAME and AUTH_ALLOWED_PASSWORD from configuration.
// GET /oauth2/authorize - Display login form
// POST /oauth2/authorize - Process authentication
func OAuth2AuthorizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		handleOAuth2AuthorizeGET(w, r)
		return
	}
	if r.Method == http.MethodPost {
		handleOAuth2AuthorizePOST(w, r)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleOAuth2AuthorizeGET(w http.ResponseWriter, r *http.Request) {
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
	if globalConfig != nil && globalConfig.AuthAllowedClientID != "" && clientID != globalConfig.AuthAllowedClientID {
		writeAuthorizationError(w, r, ErrorUnauthorizedClient, "unknown client_id", state, redirectURI)
		return
	}

	// Validate required parameters
	if redirectURI == "" {
		writeOIDCError(w, http.StatusBadRequest, ErrorInvalidRequest, "redirect_uri parameter is required")
		return
	}

	// Validate redirect_uri if validation is enabled
	if globalConfig != nil && globalConfig.AuthCodeValidateRedirectURI {
		var allowedPatterns []string
		if globalConfig.AuthCodeAllowedRedirectURIs != "" {
			// Split comma-separated patterns
			for _, pattern := range splitScopes(globalConfig.AuthCodeAllowedRedirectURIs) {
				if trimmed := pattern; trimmed != "" {
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
		scope = joinScopes(globalConfig.AuthSupportedScopes)
	} else {
		// Validate scopes
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
				writeAuthorizationError(w, r, ErrorInvalidScope, fmt.Sprintf("unsupported scope: %s", rs), state, redirectURI)
				return
			}
		}
	}

	// Validate PKCE parameters
	if globalConfig != nil && globalConfig.AuthCodeRequirePKCE && codeChallenge == "" {
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
		Name:     "oauth2_session",
		Value:    session.ID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Render login form
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl := template.Must(template.New("login").Parse(oauth2LoginFormTemplate))
	data := struct {
		State        string
		RedirectURI  string
		Scope        string
		AuthorizeURL string
	}{
		State:        session.State,
		RedirectURI:  redirectURI,
		Scope:        scope,
		AuthorizeURL: "/oauth2/authorize",
	}
	_ = tmpl.Execute(w, data)
}

func handleOAuth2AuthorizePOST(w http.ResponseWriter, r *http.Request) {
	// Get session from cookie
	cookie, err := r.Cookie("oauth2_session")
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

	// Validate credentials against environment variables
	if err := validateBasicAuthCredentials(username, password); err != nil {
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
		Name:   "oauth2_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	// Redirect back to the client with the authorization code and state
	redirectURL := session.RedirectURI + "?code=" + authCode.Code
	if session.State != "" {
		redirectURL += "&state=" + session.State
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

const oauth2LoginFormTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>OAuth2 Login</title>
</head>
<body>
    <h1>OAuth2 Login</h1>
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
    <p>Scope: {{.Scope}}</p>
    <p>Redirect: {{.RedirectURI}}</p>
</body>
</html>`
