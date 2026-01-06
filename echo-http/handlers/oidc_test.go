package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
		clientID       string
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
			clientID:       "test-client",
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
			clientID:       "test-client",
			redirectURI:    "http://localhost/callback",
			responseType:   "code",
			expectedStatus: http.StatusOK,
			checkHTML:      true,
		},
		{
			name:           "missing redirect_uri",
			user:           "testuser",
			pass:           "testpass",
			clientID:       "test-client",
			responseType:   "code",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unsupported response_type",
			user:           "testuser",
			pass:           "testpass",
			clientID:       "test-client",
			redirectURI:    "http://localhost/callback",
			responseType:   "token",
			expectedStatus: http.StatusFound, // Now redirects with error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize config for each test
			SetConfig(&Config{
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)

			queryParams := url.Values{}
			if tt.clientID != "" {
				queryParams.Add("client_id", tt.clientID)
			}
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
				session, _ := DefaultSessionStore.CreateSession(tt.state, tt.redirectURI, "openid profile", "", "", "")
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
	authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", "testuser", "openid profile", "", "", "")

	tests := []struct {
		name           string
		user           string
		pass           string
		clientID       string
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
			clientID:       "test-client",
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
			clientID:       "test-client",
			code:           authCode.Code,
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid grant_type",
			user:           "testuser",
			pass:           "testpass",
			clientID:       "test-client",
			grantType:      "implicit",
			code:           authCode.Code,
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing code",
			user:           "testuser",
			pass:           "testpass",
			clientID:       "test-client",
			grantType:      "authorization_code",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing redirect_uri",
			user:           "testuser",
			pass:           "testpass",
			clientID:       "test-client",
			grantType:      "authorization_code",
			code:           authCode.Code,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid code",
			user:           "testuser",
			pass:           "testpass",
			clientID:       "test-client",
			grantType:      "authorization_code",
			code:           "invalid-code",
			redirectURI:    "http://localhost/callback",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize config for each test
			SetConfig(&Config{
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			r := chi.NewRouter()
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			formData := url.Values{}
			if tt.grantType != "" {
				formData.Add("grant_type", tt.grantType)
			}
			if tt.clientID != "" {
				formData.Add("client_id", tt.clientID)
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

func TestOIDCAuthorizeHandler_ClientIDValidation(t *testing.T) {
	tests := []struct {
		name            string
		configClientID  string
		requestClientID string
		expectedStatus  int
		expectedError   string
	}{
		{
			name:            "client_id parameter missing",
			configClientID:  "",
			requestClientID: "",
			expectedStatus:  http.StatusBadRequest,
			expectedError:   ErrorInvalidRequest,
		},
		{
			name:            "any client_id accepted when config is empty",
			configClientID:  "",
			requestClientID: "any-client",
			expectedStatus:  http.StatusOK,
			expectedError:   "",
		},
		{
			name:            "correct client_id when configured",
			configClientID:  "test-client",
			requestClientID: "test-client",
			expectedStatus:  http.StatusOK,
			expectedError:   "",
		},
		{
			name:            "incorrect client_id when configured",
			configClientID:  "test-client",
			requestClientID: "wrong-client",
			expectedStatus:  http.StatusFound, // Redirect with error
			expectedError:   ErrorUnauthorizedClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			SetConfig(&Config{
				AuthAllowedClientID: tt.configClientID,
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)

			// Build query parameters
			queryParams := url.Values{}
			if tt.requestClientID != "" {
				queryParams.Add("client_id", tt.requestClientID)
			}
			queryParams.Add("redirect_uri", "http://localhost/callback")
			queryParams.Add("response_type", "code")
			queryParams.Add("state", "test-state")

			req := httptest.NewRequest(http.MethodGet, "/oidc/testuser/testpass/authorize?"+queryParams.Encode(), nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			// Validate status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Validate error response
			if tt.expectedError != "" {
				if rec.Code == http.StatusFound {
					// Error should be in redirect URL
					location := rec.Header().Get("Location")
					if !strings.Contains(location, "error="+tt.expectedError) {
						t.Errorf("expected error=%s in redirect URL, got: %s", tt.expectedError, location)
					}
				} else {
					// Error should be in JSON response
					var errResp OIDCError
					if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
						t.Fatalf("failed to parse error response: %v", err)
					}
					if errResp.Error != tt.expectedError {
						t.Errorf("expected error=%s, got %s", tt.expectedError, errResp.Error)
					}
				}
			}
		})
	}
}

func TestOIDCTokenHandler_ClientValidation(t *testing.T) {
	tests := []struct {
		name             string
		configClientID   string
		configClientSec  string
		requestClientID  string
		requestClientSec string
		expectedStatus   int
		expectedError    string
	}{
		{
			name:            "client_id parameter missing",
			configClientID:  "",
			configClientSec: "",
			requestClientID: "",
			expectedStatus:  http.StatusBadRequest,
			expectedError:   ErrorInvalidRequest,
		},
		{
			name:            "any client_id accepted when config is empty",
			configClientID:  "",
			configClientSec: "",
			requestClientID: "any-client",
			expectedStatus:  http.StatusOK,
			expectedError:   "",
		},
		{
			name:            "correct client_id when configured",
			configClientID:  "test-client",
			configClientSec: "",
			requestClientID: "test-client",
			expectedStatus:  http.StatusOK,
			expectedError:   "",
		},
		{
			name:            "incorrect client_id when configured",
			configClientID:  "test-client",
			configClientSec: "",
			requestClientID: "wrong-client",
			expectedStatus:  http.StatusUnauthorized,
			expectedError:   ErrorInvalidClient,
		},
		{
			name:             "client_secret required and correct",
			configClientID:   "test-client",
			configClientSec:  "test-secret",
			requestClientID:  "test-client",
			requestClientSec: "test-secret",
			expectedStatus:   http.StatusOK,
			expectedError:    "",
		},
		{
			name:             "client_secret required but missing",
			configClientID:   "test-client",
			configClientSec:  "test-secret",
			requestClientID:  "test-client",
			requestClientSec: "",
			expectedStatus:   http.StatusUnauthorized,
			expectedError:    ErrorInvalidClient,
		},
		{
			name:             "client_secret required but incorrect",
			configClientID:   "test-client",
			configClientSec:  "test-secret",
			requestClientID:  "test-client",
			requestClientSec: "wrong-secret",
			expectedStatus:   http.StatusUnauthorized,
			expectedError:    ErrorInvalidClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			SetConfig(&Config{
				AuthAllowedClientID:     tt.configClientID,
				AuthAllowedClientSecret: tt.configClientSec,
				AuthSupportedScopes:     []string{"openid", "profile", "email"},
			})

			// Create a valid auth code
			authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", "testuser", "openid profile", "", "", "")

			r := chi.NewRouter()
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			// Build form data
			formData := url.Values{}
			formData.Add("grant_type", "authorization_code")
			formData.Add("code", authCode.Code)
			formData.Add("redirect_uri", "http://localhost/callback")
			if tt.requestClientID != "" {
				formData.Add("client_id", tt.requestClientID)
			}
			if tt.requestClientSec != "" {
				formData.Add("client_secret", tt.requestClientSec)
			}

			req := httptest.NewRequest(http.MethodPost, "/oidc/testuser/testpass/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			// Validate status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Validate error response
			if tt.expectedError != "" {
				var errResp OIDCError
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to parse error response: %v", err)
				}
				if errResp.Error != tt.expectedError {
					t.Errorf("expected error=%s, got %s", tt.expectedError, errResp.Error)
				}
			}
		})
	}
}

func TestOIDCTokenHandler_PKCEVerification(t *testing.T) {
	tests := []struct {
		name                string
		codeChallenge       string
		codeChallengeMethod string
		codeVerifier        string
		expectedStatus      int
		expectedError       string
	}{
		{
			name:                "plain method - verification success",
			codeChallenge:       "test-challenge-plain-verifier-with-43-chars",
			codeChallengeMethod: "plain",
			codeVerifier:        "test-challenge-plain-verifier-with-43-chars",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
		{
			name:                "plain method - verification failure",
			codeChallenge:       "test-challenge-plain-verifier-with-43-chars",
			codeChallengeMethod: "plain",
			codeVerifier:        "wrong-verifier-that-is-long-enough-43-chars",
			expectedStatus:      http.StatusBadRequest,
			expectedError:       ErrorInvalidGrant,
		},
		{
			name:                "S256 method - verification success",
			codeChallenge:       "hCgqGmRwPjKdkmihOZdKwKgGirlXOSc6edj1J7fg3YQ",
			codeChallengeMethod: "S256",
			codeVerifier:        "test-verifier-that-is-long-enough-for-pkce-",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
		{
			name:                "S256 method - verification failure",
			codeChallenge:       "hCgqGmRwPjKdkmihOZdKwKgGirlXOSc6edj1J7fg3YQ",
			codeChallengeMethod: "S256",
			codeVerifier:        "wrong-verifier-that-is-long-enough-43-chars",
			expectedStatus:      http.StatusBadRequest,
			expectedError:       ErrorInvalidGrant,
		},
		{
			name:                "missing code_verifier when code_challenge was provided",
			codeChallenge:       "test-challenge-plain-verifier-with-43-chars",
			codeChallengeMethod: "plain",
			codeVerifier:        "",
			expectedStatus:      http.StatusBadRequest,
			expectedError:       ErrorInvalidGrant,
		},
		{
			name:                "no verification when PKCE not used",
			codeChallenge:       "",
			codeChallengeMethod: "",
			codeVerifier:        "",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
		{
			name:                "code_verifier provided when PKCE not used - should be ignored",
			codeChallenge:       "",
			codeChallengeMethod: "",
			codeVerifier:        "some-verifier",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			SetConfig(&Config{
				AuthAllowedClientID: "",
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			// Create an auth code with PKCE parameters
			authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", "testuser", "openid profile", tt.codeChallenge, tt.codeChallengeMethod, "")

			r := chi.NewRouter()
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			// Build form data
			formData := url.Values{}
			formData.Add("grant_type", "authorization_code")
			formData.Add("client_id", "test-client")
			formData.Add("code", authCode.Code)
			formData.Add("redirect_uri", "http://localhost/callback")
			if tt.codeVerifier != "" {
				formData.Add("code_verifier", tt.codeVerifier)
			}

			req := httptest.NewRequest(http.MethodPost, "/oidc/testuser/testpass/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			// Validate status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			// Validate error response
			if tt.expectedError != "" {
				var errResp OIDCError
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to parse error response: %v", err)
				}
				if errResp.Error != tt.expectedError {
					t.Errorf("expected error=%s, got %s", tt.expectedError, errResp.Error)
				}
			} else {
				// Validate successful token response
				var tokenResp TokenResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &tokenResp); err != nil {
					t.Fatalf("failed to parse token response: %v", err)
				}
				if tokenResp.AccessToken == "" {
					t.Error("expected access_token in response")
				}
			}
		})
	}
}

func TestOIDCFullFlow(t *testing.T) {
	// Initialize config for test
	SetConfig(&Config{
		AuthSupportedScopes: []string{"openid", "profile", "email"},
	})

	r := chi.NewRouter()
	r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
	r.Post("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
	r.Get("/oidc/{user}/{pass}/callback", OIDCCallbackHandler)
	r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

	user := "testuser"
	pass := "testpass"

	// Step 1: Get login form (with client-provided state)
	clientState := "client-random-state-123"
	req1 := httptest.NewRequest(http.MethodGet, "/oidc/"+user+"/"+pass+"/authorize?client_id=test-client&redirect_uri=http://localhost/callback&response_type=code&state="+clientState, nil)
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
	tokenFormData.Add("client_id", "test-client")
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

func TestValidateScopes(t *testing.T) {
	SetConfig(&Config{
		AuthSupportedScopes: []string{"openid", "profile", "email"},
	})

	tests := []struct {
		name           string
		requestedScope string
		expectError    bool
		errorMessage   string
	}{
		{
			name:           "valid single scope",
			requestedScope: "openid",
			expectError:    false,
		},
		{
			name:           "valid multiple scopes",
			requestedScope: "openid profile email",
			expectError:    false,
		},
		{
			name:           "valid subset of scopes",
			requestedScope: "openid profile",
			expectError:    false,
		},
		{
			name:           "invalid scope",
			requestedScope: "openid invalid",
			expectError:    true,
			errorMessage:   "unsupported scope: invalid",
		},
		{
			name:           "all invalid scopes",
			requestedScope: "admin superuser",
			expectError:    true,
			errorMessage:   "unsupported scope: admin",
		},
		{
			name:           "empty scope",
			requestedScope: "",
			expectError:    false,
		},
		{
			name:           "scope with extra whitespace",
			requestedScope: "openid  profile   email",
			expectError:    false,
		},
		{
			name:           "scope with leading and trailing whitespace",
			requestedScope: "  openid profile email  ",
			expectError:    false,
		},
		{
			name:           "scope with empty strings in the middle",
			requestedScope: "openid  profile",
			expectError:    false,
		},
		{
			name:           "duplicate scopes should be allowed",
			requestedScope: "openid openid profile",
			expectError:    false,
		},
		{
			name:           "case sensitive - invalid uppercase",
			requestedScope: "OPENID",
			expectError:    true,
			errorMessage:   "unsupported scope: OPENID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScopes(tt.requestedScope)
			if (err != nil) != tt.expectError {
				t.Errorf("expected error=%v, got error=%v", tt.expectError, err)
			}
			if tt.expectError && err != nil && tt.errorMessage != "" {
				if err.Error() != tt.errorMessage {
					t.Errorf("expected error message %q, got %q", tt.errorMessage, err.Error())
				}
			}
		})
	}
}

func TestGetDefaultScopes(t *testing.T) {
	tests := []struct {
		name            string
		configScopes    []string
		expectedDefault string
	}{
		{
			name:            "default scopes from config",
			configScopes:    []string{"openid", "profile", "email"},
			expectedDefault: "openid profile email",
		},
		{
			name:            "single scope",
			configScopes:    []string{"openid"},
			expectedDefault: "openid",
		},
		{
			name:            "custom scopes",
			configScopes:    []string{"openid", "custom", "api"},
			expectedDefault: "openid custom api",
		},
		{
			name:            "empty config scopes",
			configScopes:    []string{},
			expectedDefault: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetConfig(&Config{
				AuthSupportedScopes: tt.configScopes,
			})

			result := getDefaultScopes()
			if result != tt.expectedDefault {
				t.Errorf("expected default scopes %q, got %q", tt.expectedDefault, result)
			}
		})
	}
}

func TestOIDCAuthorizeHandler_ScopeValidation(t *testing.T) {
	tests := []struct {
		name           string
		configScopes   []string
		requestScope   string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid scopes accepted",
			configScopes:   []string{"openid", "profile", "email"},
			requestScope:   "openid profile",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "all valid scopes accepted",
			configScopes:   []string{"openid", "profile", "email"},
			requestScope:   "openid profile email",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "invalid scope rejected",
			configScopes:   []string{"openid", "profile", "email"},
			requestScope:   "openid admin",
			expectedStatus: http.StatusFound,
			expectedError:  ErrorInvalidScope,
		},
		{
			name:           "empty scope uses defaults",
			configScopes:   []string{"openid", "profile", "email"},
			requestScope:   "",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "scope with extra whitespace accepted",
			configScopes:   []string{"openid", "profile", "email"},
			requestScope:   "openid  profile   email",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			SetConfig(&Config{
				AuthAllowedClientID: "",
				AuthSupportedScopes: tt.configScopes,
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)

			// Build query parameters
			queryParams := url.Values{}
			queryParams.Add("client_id", "test-client")
			queryParams.Add("redirect_uri", "http://localhost/callback")
			queryParams.Add("response_type", "code")
			queryParams.Add("state", "test-state")
			if tt.requestScope != "" {
				queryParams.Add("scope", tt.requestScope)
			}

			req := httptest.NewRequest(http.MethodGet, "/oidc/testuser/testpass/authorize?"+queryParams.Encode(), nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			// Validate status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Validate error response
			if tt.expectedError != "" {
				if rec.Code == http.StatusFound {
					// Error should be in redirect URL
					location := rec.Header().Get("Location")
					if !strings.Contains(location, "error="+tt.expectedError) {
						t.Errorf("expected error=%s in redirect URL, got: %s", tt.expectedError, location)
					}
				} else {
					// Error should be in JSON response
					var errResp OIDCError
					if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
						t.Fatalf("failed to parse error response: %v", err)
					}
					if errResp.Error != tt.expectedError {
						t.Errorf("expected error=%s, got %s", tt.expectedError, errResp.Error)
					}
				}
			}
		})
	}
}

func TestOIDCDiscoveryHandler_DynamicScopes(t *testing.T) {
	tests := []struct {
		name           string
		configScopes   []string
		expectedScopes []string
	}{
		{
			name:           "default scopes",
			configScopes:   []string{"openid", "profile", "email"},
			expectedScopes: []string{"openid", "profile", "email"},
		},
		{
			name:           "custom scopes",
			configScopes:   []string{"openid", "custom", "api"},
			expectedScopes: []string{"openid", "custom", "api"},
		},
		{
			name:           "single scope",
			configScopes:   []string{"openid"},
			expectedScopes: []string{"openid"},
		},
		{
			name:           "many scopes",
			configScopes:   []string{"openid", "profile", "email", "address", "phone", "offline_access"},
			expectedScopes: []string{"openid", "profile", "email", "address", "phone", "offline_access"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			SetConfig(&Config{
				AuthSupportedScopes: tt.configScopes,
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/.well-known/openid-configuration", OIDCDiscoveryHandler)

			req := httptest.NewRequest(http.MethodGet, "/oidc/testuser/testpass/.well-known/openid-configuration", nil)
			req.Host = "localhost:8080"
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
			}

			var discovery OIDCDiscoveryResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &discovery); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			// Validate scopes_supported matches config
			if len(discovery.ScopesSupported) != len(tt.expectedScopes) {
				t.Errorf("expected %d scopes, got %d", len(tt.expectedScopes), len(discovery.ScopesSupported))
			}

			for i, expectedScope := range tt.expectedScopes {
				if i >= len(discovery.ScopesSupported) {
					t.Errorf("missing scope at index %d: %s", i, expectedScope)
					continue
				}
				if discovery.ScopesSupported[i] != expectedScope {
					t.Errorf("expected scope at index %d to be %s, got %s", i, expectedScope, discovery.ScopesSupported[i])
				}
			}
		})
	}
}

func TestVerifyCodeChallenge(t *testing.T) {
	tests := []struct {
		name                string
		codeChallenge       string
		codeChallengeMethod string
		codeVerifier        string
		expectedResult      bool
	}{
		{
			name:                "plain method - verification success",
			codeChallenge:       "test-challenge-plain",
			codeChallengeMethod: "plain",
			codeVerifier:        "test-challenge-plain",
			expectedResult:      true,
		},
		{
			name:                "plain method - verification failure",
			codeChallenge:       "test-challenge-plain",
			codeChallengeMethod: "plain",
			codeVerifier:        "wrong-verifier",
			expectedResult:      false,
		},
		{
			name:                "S256 method - verification success",
			codeChallenge:       "JBbiqONGWPaAmwXk_8bT6UnlPfrn65D32eZlJS-zGG0",
			codeChallengeMethod: "S256",
			codeVerifier:        "test-verifier",
			expectedResult:      true,
		},
		{
			name:                "S256 method - verification failure",
			codeChallenge:       "JBbiqONGWPaAmwXk_8bT6UnlPfrn65D32eZlJS-zGG0",
			codeChallengeMethod: "S256",
			codeVerifier:        "wrong-verifier",
			expectedResult:      false,
		},
		{
			name:                "invalid method returns false",
			codeChallenge:       "test-challenge",
			codeChallengeMethod: "SHA512",
			codeVerifier:        "test-verifier",
			expectedResult:      false,
		},
		{
			name:                "empty method returns false",
			codeChallenge:       "test-challenge",
			codeChallengeMethod: "",
			codeVerifier:        "test-verifier",
			expectedResult:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyCodeChallenge(tt.codeChallenge, tt.codeChallengeMethod, tt.codeVerifier)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestOIDCAuthorizeHandler_PKCEValidation(t *testing.T) {
	tests := []struct {
		name                string
		configRequirePKCE   bool
		codeChallenge       string
		codeChallengeMethod string
		expectedStatus      int
		expectedError       string
	}{
		{
			name:                "PKCE parameters accepted and stored - plain method",
			configRequirePKCE:   false,
			codeChallenge:       "test-challenge-plain",
			codeChallengeMethod: "plain",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
		{
			name:                "PKCE parameters accepted and stored - S256 method",
			configRequirePKCE:   false,
			codeChallenge:       "JBbiqONGWPaAmwXk_8bT6UnlPfrn65D32eZlJS-zGG0",
			codeChallengeMethod: "S256",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
		{
			name:                "PKCE parameters accepted - default to plain when method not specified",
			configRequirePKCE:   false,
			codeChallenge:       "test-challenge-plain",
			codeChallengeMethod: "",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
		{
			name:                "invalid code_challenge_method rejected",
			configRequirePKCE:   false,
			codeChallenge:       "test-challenge",
			codeChallengeMethod: "SHA512",
			expectedStatus:      http.StatusFound, // Redirect with error
			expectedError:       ErrorInvalidRequest,
		},
		{
			name:                "PKCE required when configured",
			configRequirePKCE:   true,
			codeChallenge:       "",
			codeChallengeMethod: "",
			expectedStatus:      http.StatusFound, // Redirect with error
			expectedError:       ErrorInvalidRequest,
		},
		{
			name:                "PKCE optional when not required",
			configRequirePKCE:   false,
			codeChallenge:       "",
			codeChallengeMethod: "",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
		{
			name:                "PKCE required and provided - success",
			configRequirePKCE:   true,
			codeChallenge:       "test-challenge-plain",
			codeChallengeMethod: "plain",
			expectedStatus:      http.StatusOK,
			expectedError:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			SetConfig(&Config{
				AuthAllowedClientID: "",
				AuthSupportedScopes: []string{"openid", "profile", "email"},
				AuthCodeRequirePKCE: tt.configRequirePKCE,
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)

			// Build query parameters
			queryParams := url.Values{}
			queryParams.Add("client_id", "test-client")
			queryParams.Add("redirect_uri", "http://localhost/callback")
			queryParams.Add("response_type", "code")
			queryParams.Add("state", "test-state")
			if tt.codeChallenge != "" {
				queryParams.Add("code_challenge", tt.codeChallenge)
			}
			if tt.codeChallengeMethod != "" {
				queryParams.Add("code_challenge_method", tt.codeChallengeMethod)
			}

			req := httptest.NewRequest(http.MethodGet, "/oidc/testuser/testpass/authorize?"+queryParams.Encode(), nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			// Validate status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			// Validate error response
			if tt.expectedError != "" {
				if rec.Code == http.StatusFound {
					// Error should be in redirect URL
					location := rec.Header().Get("Location")
					if !strings.Contains(location, "error="+tt.expectedError) {
						t.Errorf("expected error=%s in redirect URL, got: %s", tt.expectedError, location)
					}
				} else {
					// Error should be in JSON response
					var errResp OIDCError
					if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
						t.Fatalf("failed to parse error response: %v", err)
					}
					if errResp.Error != tt.expectedError {
						t.Errorf("expected error=%s, got %s", tt.expectedError, errResp.Error)
					}
				}
			}
		})
	}
}

func TestOIDC_FullFlow_WithPKCE(t *testing.T) {
	tests := []struct {
		name                string
		codeChallengeMethod string
		codeVerifier        string
	}{
		{
			name:                "complete authorization flow with PKCE plain method",
			codeChallengeMethod: "plain",
			codeVerifier:        "test-challenge-plain-verifier-with-43-chars",
		},
		{
			name:                "complete authorization flow with PKCE S256 method",
			codeChallengeMethod: "S256",
			codeVerifier:        "test-verifier-for-s256-with-at-least-43-chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize config for test
			SetConfig(&Config{
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
			r.Post("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			user := "testuser"
			pass := "testpass"

			// Generate PKCE parameters
			var codeChallenge string
			if tt.codeChallengeMethod == "plain" {
				codeChallenge = tt.codeVerifier
			} else {
				// S256: BASE64URL(SHA256(verifier))
				h := sha256.Sum256([]byte(tt.codeVerifier))
				codeChallenge = base64.RawURLEncoding.EncodeToString(h[:])
			}

			// Step 1: Get login form with PKCE parameters
			clientState := "client-random-state-123"
			authURL := fmt.Sprintf("/oidc/%s/%s/authorize?client_id=test-client&redirect_uri=%s&response_type=code&scope=openid+profile&state=%s&code_challenge=%s&code_challenge_method=%s",
				user, pass,
				url.QueryEscape("http://localhost/callback"),
				clientState,
				url.QueryEscape(codeChallenge),
				tt.codeChallengeMethod)

			req1 := httptest.NewRequest(http.MethodGet, authURL, nil)
			rec1 := httptest.NewRecorder()
			r.ServeHTTP(rec1, req1)

			if rec1.Code != http.StatusOK {
				t.Fatalf("login form request failed: %d; body: %s", rec1.Code, rec1.Body.String())
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

			// Step 2: Submit login form
			formData := url.Values{}
			formData.Add("username", user)
			formData.Add("password", pass)

			req2 := httptest.NewRequest(http.MethodPost, "/oidc/"+user+"/"+pass+"/authorize", strings.NewReader(formData.Encode()))
			req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req2.AddCookie(sessionCookie)
			rec2 := httptest.NewRecorder()
			r.ServeHTTP(rec2, req2)

			if rec2.Code != http.StatusFound {
				t.Fatalf("authorize request failed: %d; body: %s", rec2.Code, rec2.Body.String())
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

			// Step 3: Exchange code for tokens with code_verifier
			tokenFormData := url.Values{}
			tokenFormData.Add("grant_type", "authorization_code")
			tokenFormData.Add("client_id", "test-client")
			tokenFormData.Add("code", code)
			tokenFormData.Add("redirect_uri", "http://localhost/callback")
			tokenFormData.Add("code_verifier", tt.codeVerifier)

			req3 := httptest.NewRequest(http.MethodPost, "/oidc/"+user+"/"+pass+"/token", strings.NewReader(tokenFormData.Encode()))
			req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec3 := httptest.NewRecorder()
			r.ServeHTTP(rec3, req3)

			if rec3.Code != http.StatusOK {
				t.Fatalf("token request failed: %d; body: %s", rec3.Code, rec3.Body.String())
			}

			// Verify tokens returned
			var tokenResponse TokenResponse
			if err := json.Unmarshal(rec3.Body.Bytes(), &tokenResponse); err != nil {
				t.Fatalf("failed to parse token response: %v", err)
			}

			if tokenResponse.AccessToken == "" {
				t.Error("expected access_token in final response")
			}
			if tokenResponse.IDToken == "" {
				t.Error("expected id_token in final response")
			}
			if tokenResponse.RefreshToken == "" {
				t.Error("expected refresh_token in final response")
			}
		})
	}
}

// TestOIDCNonceSupport tests nonce parameter support per OIDC Core specification
func TestOIDCNonceSupport(t *testing.T) {
	tests := []struct {
		name        string
		nonce       string
		expectNonce bool
	}{
		{
			name:        "nonce parameter provided in authorization request",
			nonce:       "test-nonce-123",
			expectNonce: true,
		},
		{
			name:        "nonce parameter not provided",
			nonce:       "",
			expectNonce: false,
		},
		{
			name:        "nonce parameter with special characters",
			nonce:       "nonce-with-special_chars.123",
			expectNonce: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Initialize config
			SetConfig(&Config{
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
			r.Post("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			user := "testuser"
			pass := "testpass"
			clientState := "state-123"

			// Step 1: Authorization request with nonce parameter
			queryParams := url.Values{}
			queryParams.Add("client_id", "test-client")
			queryParams.Add("redirect_uri", "http://localhost/callback")
			queryParams.Add("response_type", "code")
			queryParams.Add("scope", "openid profile")
			queryParams.Add("state", clientState)
			if tt.nonce != "" {
				queryParams.Add("nonce", tt.nonce)
			}

			req1 := httptest.NewRequest(http.MethodGet, "/oidc/"+user+"/"+pass+"/authorize?"+queryParams.Encode(), nil)
			rec1 := httptest.NewRecorder()
			r.ServeHTTP(rec1, req1)

			if rec1.Code != http.StatusOK {
				t.Fatalf("authorization request failed: %d; body: %s", rec1.Code, rec1.Body.String())
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

			// Step 2: Submit credentials
			formData := url.Values{}
			formData.Add("username", user)
			formData.Add("password", pass)

			req2 := httptest.NewRequest(http.MethodPost, "/oidc/"+user+"/"+pass+"/authorize", strings.NewReader(formData.Encode()))
			req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req2.AddCookie(sessionCookie)
			rec2 := httptest.NewRecorder()
			r.ServeHTTP(rec2, req2)

			if rec2.Code != http.StatusFound {
				t.Fatalf("authentication failed: %d; body: %s", rec2.Code, rec2.Body.String())
			}

			// Extract authorization code
			location := rec2.Header().Get("Location")
			locationURL, _ := url.Parse(location)
			code := locationURL.Query().Get("code")
			if code == "" {
				t.Fatalf("no code in redirect URL: %s", location)
			}

			// Step 3: Exchange code for tokens
			tokenFormData := url.Values{}
			tokenFormData.Add("grant_type", "authorization_code")
			tokenFormData.Add("client_id", "test-client")
			tokenFormData.Add("code", code)
			tokenFormData.Add("redirect_uri", "http://localhost/callback")

			req3 := httptest.NewRequest(http.MethodPost, "/oidc/"+user+"/"+pass+"/token", strings.NewReader(tokenFormData.Encode()))
			req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec3 := httptest.NewRecorder()
			r.ServeHTTP(rec3, req3)

			if rec3.Code != http.StatusOK {
				t.Fatalf("token request failed: %d; body: %s", rec3.Code, rec3.Body.String())
			}

			// Verify ID token contains or omits nonce
			var tokenResponse TokenResponse
			if err := json.Unmarshal(rec3.Body.Bytes(), &tokenResponse); err != nil {
				t.Fatalf("failed to parse token response: %v", err)
			}

			if tokenResponse.IDToken == "" {
				t.Fatal("expected id_token in response")
			}

			// Parse ID token (JWT format: header.payload.signature)
			parts := strings.Split(tokenResponse.IDToken, ".")
			if len(parts) != 3 {
				t.Fatalf("invalid JWT format: expected 3 parts, got %d", len(parts))
			}

			// Decode payload
			payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				t.Fatalf("failed to decode JWT payload: %v", err)
			}

			var claims map[string]interface{}
			if err := json.Unmarshal(payloadJSON, &claims); err != nil {
				t.Fatalf("failed to parse JWT claims: %v", err)
			}

			// Verify nonce claim
			nonceValue, hasNonce := claims["nonce"]
			if tt.expectNonce {
				if !hasNonce {
					t.Error("expected nonce claim in ID token, but it was not present")
				} else if nonceValue != tt.nonce {
					t.Errorf("expected nonce=%q, got %q", tt.nonce, nonceValue)
				}
			} else {
				if hasNonce {
					t.Errorf("expected no nonce claim in ID token, but found: %v", nonceValue)
				}
			}
		})
	}
}

// TestCodeVerifierLengthValidation tests RFC 7636 Section 4.1 length requirements
func TestCodeVerifierLengthValidation(t *testing.T) {
	tests := []struct {
		name           string
		codeVerifier   string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid code_verifier length - 43 characters (minimum)",
			codeVerifier:   "1234567890123456789012345678901234567890123",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "valid code_verifier length - 128 characters (maximum)",
			codeVerifier:   "12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "valid code_verifier length - 64 characters (middle range)",
			codeVerifier:   "1234567890123456789012345678901234567890123456789012345678901234",
			expectedStatus: http.StatusOK,
			expectedError:  "",
		},
		{
			name:           "invalid code_verifier length - 42 characters (too short)",
			codeVerifier:   "123456789012345678901234567890123456789012",
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrorInvalidGrant,
		},
		{
			name:           "invalid code_verifier length - 129 characters (too long)",
			codeVerifier:   "123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789",
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrorInvalidGrant,
		},
		{
			name:           "invalid code_verifier length - 1 character (way too short)",
			codeVerifier:   "a",
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrorInvalidGrant,
		},
		{
			name:           "invalid code_verifier length - 200 characters (way too long)",
			codeVerifier:   "12345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890",
			expectedStatus: http.StatusBadRequest,
			expectedError:  ErrorInvalidGrant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup config
			SetConfig(&Config{
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			// Create auth code with PKCE challenge (use verifier as challenge for plain method)
			codeChallenge := tt.codeVerifier
			authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", "testuser", "openid profile", codeChallenge, "plain", "")

			r := chi.NewRouter()
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			// Build form data
			formData := url.Values{}
			formData.Add("grant_type", "authorization_code")
			formData.Add("client_id", "test-client")
			formData.Add("code", authCode.Code)
			formData.Add("redirect_uri", "http://localhost/callback")
			formData.Add("code_verifier", tt.codeVerifier)

			req := httptest.NewRequest(http.MethodPost, "/oidc/testuser/testpass/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			// Validate status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d; body: %s", tt.expectedStatus, rec.Code, rec.Body.String())
			}

			// Validate error response
			if tt.expectedError != "" {
				var errResp OIDCError
				if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
					t.Fatalf("failed to parse error response: %v", err)
				}
				if errResp.Error != tt.expectedError {
					t.Errorf("expected error=%s, got %s", tt.expectedError, errResp.Error)
				}
			} else {
				// Validate successful token response
				var tokenResp TokenResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &tokenResp); err != nil {
					t.Fatalf("failed to parse token response: %v", err)
				}
				if tokenResp.AccessToken == "" {
					t.Error("expected access_token in response")
				}
			}
		})
	}
}

// TestCodeVerifierLengthValidationWithoutPKCE verifies length is not checked when PKCE is not used
func TestCodeVerifierLengthValidationWithoutPKCE(t *testing.T) {
	// Setup config
	SetConfig(&Config{
		AuthSupportedScopes: []string{"openid", "profile", "email"},
	})

	// Create auth code WITHOUT PKCE challenge
	authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", "testuser", "openid profile", "", "", "")

	r := chi.NewRouter()
	r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

	// Build form data with short code_verifier (would be invalid with PKCE)
	formData := url.Values{}
	formData.Add("grant_type", "authorization_code")
	formData.Add("client_id", "test-client")
	formData.Add("code", authCode.Code)
	formData.Add("redirect_uri", "http://localhost/callback")
	formData.Add("code_verifier", "short") // Only 5 characters, would fail PKCE validation

	req := httptest.NewRequest(http.MethodPost, "/oidc/testuser/testpass/token", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	// Should succeed because PKCE was not used
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("failed to parse token response: %v", err)
	}
	if tokenResp.AccessToken == "" {
		t.Error("expected access_token in response")
	}
}

// TestOIDCUserInfoHandler tests the UserInfo endpoint
func TestOIDCUserInfoHandler(t *testing.T) {
	tests := []struct {
		name           string
		user           string
		pass           string
		accessToken    string
		expectedStatus int
		expectJSON     bool
	}{
		{
			name:           "valid access token returns user info",
			user:           "testuser",
			pass:           "testpass",
			accessToken:    "valid-token-123",
			expectedStatus: http.StatusOK,
			expectJSON:     true,
		},
		{
			name:           "missing authorization header returns 401",
			user:           "testuser",
			pass:           "testpass",
			accessToken:    "",
			expectedStatus: http.StatusUnauthorized,
			expectJSON:     false,
		},
		{
			name:           "invalid authorization scheme returns 401",
			user:           "testuser",
			pass:           "testpass",
			accessToken:    "Basic invalid",
			expectedStatus: http.StatusUnauthorized,
			expectJSON:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/userinfo", OIDCUserInfoHandler)

			req := httptest.NewRequest(http.MethodGet, "/oidc/"+tt.user+"/"+tt.pass+"/userinfo", nil)
			if tt.accessToken != "" {
				if strings.HasPrefix(tt.accessToken, "Basic") {
					req.Header.Set("Authorization", tt.accessToken)
				} else {
					req.Header.Set("Authorization", "Bearer "+tt.accessToken)
				}
			}
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectJSON && rec.Code == http.StatusOK {
				var userInfo map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &userInfo); err != nil {
					t.Fatalf("failed to parse JSON response: %v", err)
				}

				// Verify required claims
				if userInfo["sub"] != tt.user {
					t.Errorf("expected sub=%s, got %v", tt.user, userInfo["sub"])
				}

				if userInfo["name"] != tt.user {
					t.Errorf("expected name=%s, got %v", tt.user, userInfo["name"])
				}

				expectedEmail := tt.user + "@example.com"
				if userInfo["email"] != expectedEmail {
					t.Errorf("expected email=%s, got %v", expectedEmail, userInfo["email"])
				}
			}
		})
	}
}

// TestOIDCJWKSHandler tests the JWKS endpoint
func TestOIDCJWKSHandler(t *testing.T) {
	tests := []struct {
		name           string
		user           string
		pass           string
		expectedStatus int
	}{
		{
			name:           "returns empty JWKS",
			user:           "testuser",
			pass:           "testpass",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "works with different user",
			user:           "alice",
			pass:           "secret123",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/.well-known/jwks.json", OIDCJWKSHandler)

			req := httptest.NewRequest(http.MethodGet, "/oidc/"+tt.user+"/"+tt.pass+"/.well-known/jwks.json", nil)
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if rec.Code == http.StatusOK {
				var jwks map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &jwks); err != nil {
					t.Fatalf("failed to parse JSON response: %v", err)
				}

				// Verify keys array exists and is empty
				keys, ok := jwks["keys"]
				if !ok {
					t.Error("expected 'keys' field in JWKS response")
				}

				keysArray, ok := keys.([]interface{})
				if !ok {
					t.Error("expected 'keys' to be an array")
				}

				if len(keysArray) != 0 {
					t.Errorf("expected empty keys array, got %d keys", len(keysArray))
				}
			}
		})
	}
}
