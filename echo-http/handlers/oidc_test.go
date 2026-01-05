package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestOIDCDiscoveryHandler(t *testing.T) {
	tests := []struct {
		name               string
		user               string
		pass               string
		host               string
		forwardedProto     string
		expectedScheme     string
		expectedIssuer     string
		expectedAuthzEndpt string
		expectedTokenEndpt string
	}{
		{
			name:               "http request",
			user:               "testuser",
			pass:               "testpass",
			host:               "localhost:8080",
			expectedScheme:     "http",
			expectedIssuer:     "http://localhost:8080/oidc/testuser/testpass",
			expectedAuthzEndpt: "http://localhost:8080/oidc/testuser/testpass/authorize",
			expectedTokenEndpt: "http://localhost:8080/oidc/testuser/testpass/token",
		},
		{
			name:               "https via X-Forwarded-Proto",
			user:               "admin",
			pass:               "secret",
			host:               "example.com",
			forwardedProto:     "https",
			expectedScheme:     "https",
			expectedIssuer:     "https://example.com/oidc/admin/secret",
			expectedAuthzEndpt: "https://example.com/oidc/admin/secret/authorize",
			expectedTokenEndpt: "https://example.com/oidc/admin/secret/token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/.well-known/openid-configuration", OIDCDiscoveryHandler)

			req := httptest.NewRequest(http.MethodGet, "/oidc/"+tt.user+"/"+tt.pass+"/.well-known/openid-configuration", nil)
			req.Host = tt.host
			if tt.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var discovery OIDCDiscoveryResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &discovery); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			// Validate issuer
			if discovery.Issuer != tt.expectedIssuer {
				t.Errorf("expected issuer %s, got %s", tt.expectedIssuer, discovery.Issuer)
			}

			// Validate authorization endpoint
			if discovery.AuthorizationEndpoint != tt.expectedAuthzEndpt {
				t.Errorf("expected authorization_endpoint %s, got %s", tt.expectedAuthzEndpt, discovery.AuthorizationEndpoint)
			}

			// Validate token endpoint
			if discovery.TokenEndpoint != tt.expectedTokenEndpt {
				t.Errorf("expected token_endpoint %s, got %s", tt.expectedTokenEndpt, discovery.TokenEndpoint)
			}

			// Validate supported response types
			if len(discovery.ResponseTypesSupported) == 0 {
				t.Error("expected response_types_supported to be non-empty")
			}
			if discovery.ResponseTypesSupported[0] != "code" {
				t.Errorf("expected response_types_supported to include 'code'")
			}

			// Validate supported subject types
			if len(discovery.SubjectTypesSupported) == 0 {
				t.Error("expected subject_types_supported to be non-empty")
			}
			if discovery.SubjectTypesSupported[0] != "public" {
				t.Errorf("expected subject_types_supported to include 'public'")
			}

			// Validate supported scopes
			expectedScopes := []string{"openid", "profile", "email"}
			if len(discovery.ScopesSupported) != len(expectedScopes) {
				t.Errorf("expected %d scopes, got %d", len(expectedScopes), len(discovery.ScopesSupported))
			}

			// Validate supported grant types
			if len(discovery.GrantTypesSupported) == 0 {
				t.Error("expected grant_types_supported to be non-empty")
			}
			if discovery.GrantTypesSupported[0] != "authorization_code" {
				t.Errorf("expected grant_types_supported to include 'authorization_code'")
			}

			// Validate ID token signing algorithms
			if len(discovery.IDTokenSigningAlgValuesSupported) == 0 {
				t.Error("expected id_token_signing_alg_values_supported to be non-empty")
			}
		})
	}
}

func TestOIDCAuthorizeHandler_GET(t *testing.T) {
	tests := []struct {
		name           string
		user           string
		pass           string
		redirectURI    string
		scope          string
		responseType   string
		expectedStatus int
		checkHTML      bool
	}{
		{
			name:           "valid request with all parameters",
			user:           "testuser",
			pass:           "testpass",
			redirectURI:    "http://localhost/callback",
			scope:          "openid profile email",
			responseType:   "code",
			expectedStatus: http.StatusOK,
			checkHTML:      true,
		},
		{
			name:           "valid request with default scope",
			user:           "testuser",
			pass:           "testpass",
			redirectURI:    "http://localhost/callback",
			responseType:   "code",
			expectedStatus: http.StatusOK,
			checkHTML:      true,
		},
		{
			name:           "missing redirect_uri",
			user:           "testuser",
			pass:           "testpass",
			responseType:   "code",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unsupported response_type",
			user:           "testuser",
			pass:           "testpass",
			redirectURI:    "http://localhost/callback",
			responseType:   "token",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)

			queryParams := url.Values{}
			if tt.redirectURI != "" {
				queryParams.Add("redirect_uri", tt.redirectURI)
			}
			if tt.scope != "" {
				queryParams.Add("scope", tt.scope)
			}
			if tt.responseType != "" {
				queryParams.Add("response_type", tt.responseType)
			}

			req := httptest.NewRequest(http.MethodGet, "/oidc/"+tt.user+"/"+tt.pass+"/authorize?"+queryParams.Encode(), nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkHTML && rec.Code == http.StatusOK {
				body := rec.Body.String()
				if !strings.Contains(body, "OIDC Login") {
					t.Errorf("expected HTML login form, got: %s", body)
				}
				if !strings.Contains(body, "username") {
					t.Errorf("expected username field in form")
				}
				if !strings.Contains(body, "password") {
					t.Errorf("expected password field in form")
				}
			}
		})
	}
}

func TestOIDCAuthorizeHandler_POST(t *testing.T) {
	tests := []struct {
		name           string
		urlUser        string
		urlPass        string
		username       string
		password       string
		state          string
		redirectURI    string
		expectedStatus int
		checkRedirect  bool
		needsSession   bool
	}{
		{
			name:           "valid authorization",
			urlUser:        "testuser",
			urlPass:        "testpass",
			username:       "testuser",
			password:       "testpass",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusFound,
			checkRedirect:  true,
			needsSession:   true,
		},
		{
			name:           "invalid credentials - username mismatch",
			urlUser:        "testuser",
			urlPass:        "testpass",
			username:       "wronguser",
			password:       "testpass",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusUnauthorized,
			needsSession:   true,
		},
		{
			name:           "invalid credentials - password mismatch",
			urlUser:        "testuser",
			urlPass:        "testpass",
			username:       "testuser",
			password:       "wrongpass",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusUnauthorized,
			needsSession:   true,
		},
		{
			name:           "missing username",
			urlUser:        "testuser",
			urlPass:        "testpass",
			password:       "testpass",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
			needsSession:   true,
		},
		{
			name:           "missing password",
			urlUser:        "testuser",
			urlPass:        "testpass",
			username:       "testuser",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
			needsSession:   true,
		},
		{
			name:           "missing state",
			urlUser:        "testuser",
			urlPass:        "testpass",
			username:       "testuser",
			password:       "testpass",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid state",
			urlUser:        "testuser",
			urlPass:        "testpass",
			username:       "testuser",
			password:       "testpass",
			state:          "invalid-state",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)

			// Create a session and set cookie if needed
			var sessionCookie *http.Cookie
			if tt.needsSession {
				session, _ := DefaultSessionStore.CreateSession(tt.state, tt.redirectURI, "openid profile")
				sessionCookie = &http.Cookie{
					Name:  "oidc_session",
					Value: session.ID,
				}
			}

			formData := url.Values{}
			if tt.username != "" {
				formData.Add("username", tt.username)
			}
			if tt.password != "" {
				formData.Add("password", tt.password)
			}

			req := httptest.NewRequest(http.MethodPost, "/oidc/"+tt.urlUser+"/"+tt.urlPass+"/authorize", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			if sessionCookie != nil {
				req.AddCookie(sessionCookie)
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkRedirect && rec.Code == http.StatusFound {
				location := rec.Header().Get("Location")
				if location == "" {
					t.Errorf("expected Location header in redirect response")
				}
				if !strings.Contains(location, "code=") {
					t.Errorf("expected code parameter in redirect URL")
				}
				// state is only included if client provided it
				if tt.state != "" && !strings.Contains(location, "state=") {
					t.Errorf("expected state parameter in redirect URL when client provided it")
				}
			}
		})
	}
}

func TestOIDCCallbackHandler(t *testing.T) {
	tests := []struct {
		name      string
		user      string
		pass      string
		code      string
		state     string
		checkHTML bool
	}{
		{
			name:      "valid callback with code",
			user:      "testuser",
			pass:      "testpass",
			code:      "test-auth-code",
			state:     "test-state",
			checkHTML: true,
		},
		{
			name:      "callback without code",
			user:      "testuser",
			pass:      "testpass",
			state:     "test-state",
			checkHTML: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/callback", OIDCCallbackHandler)

			queryParams := url.Values{}
			if tt.code != "" {
				queryParams.Add("code", tt.code)
			}
			if tt.state != "" {
				queryParams.Add("state", tt.state)
			}

			req := httptest.NewRequest(http.MethodGet, "/oidc/"+tt.user+"/"+tt.pass+"/callback?"+queryParams.Encode(), nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			if tt.checkHTML {
				body := rec.Body.String()
				if !strings.Contains(body, "OIDC Callback") {
					t.Errorf("expected HTML callback page, got: %s", body)
				}
			}
		})
	}
}

func TestOIDCTokenHandler(t *testing.T) {
	// Create a valid auth code first
	authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", "testuser", "openid profile")

	tests := []struct {
		name           string
		user           string
		pass           string
		grantType      string
		code           string
		redirectURI    string
		expectedStatus int
		checkJSON      bool
	}{
		{
			name:           "valid token exchange",
			user:           "testuser",
			pass:           "testpass",
			grantType:      "authorization_code",
			code:           authCode.Code,
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
		},
		{
			name:           "missing grant_type",
			user:           "testuser",
			pass:           "testpass",
			code:           authCode.Code,
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid grant_type",
			user:           "testuser",
			pass:           "testpass",
			grantType:      "implicit",
			code:           authCode.Code,
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing code",
			user:           "testuser",
			pass:           "testpass",
			grantType:      "authorization_code",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing redirect_uri",
			user:           "testuser",
			pass:           "testpass",
			grantType:      "authorization_code",
			code:           authCode.Code,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid code",
			user:           "testuser",
			pass:           "testpass",
			grantType:      "authorization_code",
			code:           "invalid-code",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			formData := url.Values{}
			if tt.grantType != "" {
				formData.Add("grant_type", tt.grantType)
			}
			if tt.code != "" {
				formData.Add("code", tt.code)
			}
			if tt.redirectURI != "" {
				formData.Add("redirect_uri", tt.redirectURI)
			}

			req := httptest.NewRequest(http.MethodPost, "/oidc/"+tt.user+"/"+tt.pass+"/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkJSON && rec.Code == http.StatusOK {
				var response TokenResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
					t.Errorf("failed to parse JSON response: %v", err)
				}

				if response.AccessToken == "" {
					t.Errorf("expected access_token in response")
				}
				if response.TokenType != "Bearer" {
					t.Errorf("expected token_type=Bearer, got %s", response.TokenType)
				}
				if response.ExpiresIn <= 0 {
					t.Errorf("expected positive expires_in value")
				}
				if response.IDToken == "" {
					t.Errorf("expected id_token in response")
				}
			}
		})
	}
}

func TestOIDCFullFlow(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
	r.Post("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
	r.Get("/oidc/{user}/{pass}/callback", OIDCCallbackHandler)
	r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

	user := "testuser"
	pass := "testpass"

	// Step 1: Get login form (with client-provided state)
	clientState := "client-random-state-123"
	req1 := httptest.NewRequest(http.MethodGet, "/oidc/"+user+"/"+pass+"/authorize?redirect_uri=http://localhost/callback&response_type=code&state="+clientState, nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("login form request failed: %d", rec1.Code)
	}

	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, c := range rec1.Result().Cookies() {
		if c.Name == "oidc_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("no session cookie returned")
	}

	// Step 2: Submit login form (redirect_uri comes from session)
	formData := url.Values{}
	formData.Add("username", user)
	formData.Add("password", pass)

	req2 := httptest.NewRequest(http.MethodPost, "/oidc/"+user+"/"+pass+"/authorize", strings.NewReader(formData.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.AddCookie(sessionCookie)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusFound {
		t.Fatalf("authorize request failed: %d", rec2.Code)
	}

	// Extract code and state from redirect URL
	location := rec2.Header().Get("Location")
	locationURL, _ := url.Parse(location)
	code := locationURL.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in redirect URL: %s", location)
	}
	returnedState := locationURL.Query().Get("state")
	if returnedState != clientState {
		t.Errorf("expected state %s, got %s", clientState, returnedState)
	}

	// Step 3: Exchange code for tokens
	tokenFormData := url.Values{}
	tokenFormData.Add("grant_type", "authorization_code")
	tokenFormData.Add("code", code)
	tokenFormData.Add("redirect_uri", "http://localhost/callback")

	req3 := httptest.NewRequest(http.MethodPost, "/oidc/"+user+"/"+pass+"/token", strings.NewReader(tokenFormData.Encode()))
	req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Fatalf("token request failed: %d", rec3.Code)
	}

	var tokenResponse TokenResponse
	if err := json.Unmarshal(rec3.Body.Bytes(), &tokenResponse); err != nil {
		t.Fatalf("failed to parse token response: %v", err)
	}

	if tokenResponse.AccessToken == "" {
		t.Errorf("expected access_token in final response")
	}
	if tokenResponse.IDToken == "" {
		t.Errorf("expected id_token in final response")
	}
}

func TestOIDCDemoHandler(t *testing.T) {
	tests := []struct {
		name           string
		user           string
		pass           string
		queryParams    string
		expectedStatus int
		expectRedirect bool
		checkHTML      bool
	}{
		{
			name:           "initial access redirects to login",
			user:           "testuser",
			pass:           "testpass",
			queryParams:    "",
			expectedStatus: http.StatusFound,
			expectRedirect: true,
			checkHTML:      false,
		},
		{
			name:           "with code shows demo page",
			user:           "demouser",
			pass:           "demopass",
			queryParams:    "?code=abc123&state=xyz789",
			expectedStatus: http.StatusOK,
			expectRedirect: false,
			checkHTML:      true,
		},
		{
			name:           "with error shows error page",
			user:           "testuser",
			pass:           "testpass",
			queryParams:    "?error=access_denied",
			expectedStatus: http.StatusOK,
			expectRedirect: false,
			checkHTML:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/demo", OIDCDemoHandler)

			req := httptest.NewRequest(http.MethodGet, "/oidc/"+tt.user+"/"+tt.pass+"/demo"+tt.queryParams, nil)
			req.Host = "localhost:8080"
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectRedirect {
				location := rec.Header().Get("Location")
				if location == "" {
					t.Errorf("expected Location header for redirect")
				}
				if !strings.Contains(location, "/authorize") {
					t.Errorf("expected redirect to authorize, got %s", location)
				}
				if !strings.Contains(location, "redirect_uri=") {
					t.Errorf("expected redirect_uri parameter in authorize URL")
				}
			}

			if tt.checkHTML {
				body := rec.Body.String()
				if !strings.Contains(body, "OIDC Interactive Demo") && !strings.Contains(body, "OIDC Demo") {
					t.Errorf("expected demo page HTML title")
				}

				// Check for authorization code if present
				if strings.Contains(tt.queryParams, "code=") {
					if !strings.Contains(body, "Authorization Code") {
						t.Errorf("expected 'Authorization Code' section in HTML")
					}
				}

				// Check for error if present
				if strings.Contains(tt.queryParams, "error=") {
					if !strings.Contains(body, "Error") {
						t.Errorf("expected 'Error' section in HTML")
					}
				}
			}
		})
	}
}
