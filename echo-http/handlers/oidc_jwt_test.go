package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestIDTokenJWTFormat verifies that ID token is returned in JWT format (P0-1)
func TestIDTokenJWTFormat(t *testing.T) {
	tests := []struct {
		name     string
		issuer   string
		clientID string
		username string
	}{
		{
			name:     "JWT format with standard values",
			issuer:   "http://localhost:8080/oidc/testuser/testpass",
			clientID: "test-client-id",
			username: "testuser",
		},
		{
			name:     "JWT format with different issuer",
			issuer:   "https://example.com/oidc/admin/secret",
			clientID: "another-client",
			username: "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			idToken := generateMockIDToken(tt.issuer, tt.clientID, tt.username, "")

			// Act: Split JWT into parts
			parts := strings.Split(idToken, ".")

			// Assert: JWT must have exactly 3 parts (header.payload.signature)
			if len(parts) != 3 {
				t.Errorf("expected JWT to have 3 parts, got %d; token: %s", len(parts), idToken)
			}

			// Assert: Header must decode to valid JSON with alg="none" and typ="JWT"
			headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
			if err != nil {
				t.Fatalf("failed to decode JWT header: %v", err)
			}

			var header map[string]interface{}
			if err := json.Unmarshal(headerJSON, &header); err != nil {
				t.Fatalf("failed to parse JWT header JSON: %v", err)
			}

			if header["alg"] != "none" {
				t.Errorf("expected alg=none, got %v", header["alg"])
			}

			if header["typ"] != "JWT" {
				t.Errorf("expected typ=JWT, got %v", header["typ"])
			}

			// Assert: Payload must decode to valid JSON with correct claims
			payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				t.Fatalf("failed to decode JWT payload: %v", err)
			}

			var claims map[string]interface{}
			if err := json.Unmarshal(payloadJSON, &claims); err != nil {
				t.Fatalf("failed to parse JWT payload JSON: %v", err)
			}

			if claims["iss"] != tt.issuer {
				t.Errorf("expected iss=%s, got %v", tt.issuer, claims["iss"])
			}

			if claims["sub"] != tt.username {
				t.Errorf("expected sub=%s, got %v", tt.username, claims["sub"])
			}

			if claims["aud"] != tt.clientID {
				t.Errorf("expected aud=%s, got %v", tt.clientID, claims["aud"])
			}

			// Assert: Signature part must be empty (alg=none)
			if parts[2] != "" {
				t.Errorf("expected empty signature for alg=none, got %s", parts[2])
			}

			// Assert: Standard claims exist
			if _, ok := claims["exp"]; !ok {
				t.Error("expected exp claim to exist")
			}

			if _, ok := claims["iat"]; !ok {
				t.Error("expected iat claim to exist")
			}

			if claims["name"] != tt.username {
				t.Errorf("expected name=%s, got %v", tt.username, claims["name"])
			}

			expectedEmail := tt.username + "@example.com"
			if claims["email"] != expectedEmail {
				t.Errorf("expected email=%s, got %v", expectedEmail, claims["email"])
			}
		})
	}
}

// TestTokenEndpointReturnsJWTIDToken verifies that token endpoint returns JWT format ID token (P0-1)
func TestTokenEndpointReturnsJWTIDToken(t *testing.T) {
	// Arrange
	SetConfig(&Config{
		AuthSupportedScopes: []string{"openid", "profile", "email"},
	})

	authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", "testuser", "openid profile", "", "", "")

	r := chi.NewRouter()
	r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

	formData := url.Values{}
	formData.Add("grant_type", "authorization_code")
	formData.Add("client_id", "test-client")
	formData.Add("code", authCode.Code)
	formData.Add("redirect_uri", "http://localhost/callback")

	req := httptest.NewRequest(http.MethodPost, "/oidc/testuser/testpass/token", strings.NewReader(formData.Encode()))
	req.Host = "localhost:8080"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	// Act
	r.ServeHTTP(rec, req)

	// Assert
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var response TokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if response.IDToken == "" {
		t.Fatal("expected id_token in response")
	}

	// Assert: ID token must be JWT format with 3 parts
	parts := strings.Split(response.IDToken, ".")
	if len(parts) != 3 {
		t.Errorf("expected JWT to have 3 parts, got %d; token: %s", len(parts), response.IDToken)
	}

	// Assert: Decode and verify header
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("failed to decode JWT header: %v", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatalf("failed to parse JWT header JSON: %v", err)
	}

	if header["alg"] != "none" {
		t.Errorf("expected alg=none, got %v", header["alg"])
	}

	// Assert: Decode and verify payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("failed to decode JWT payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		t.Fatalf("failed to parse JWT payload JSON: %v", err)
	}

	if claims["sub"] != "testuser" {
		t.Errorf("expected sub=testuser, got %v", claims["sub"])
	}
}

// TestIDTokenIssuerAndAudience verifies that iss and aud use actual values (P1-1)
func TestIDTokenIssuerAndAudience(t *testing.T) {
	tests := []struct {
		name             string
		user             string
		pass             string
		host             string
		forwardedProto   string
		clientID         string
		expectedIssuer   string
		expectedAudience string
	}{
		{
			name:             "HTTP request with localhost",
			user:             "testuser",
			pass:             "testpass",
			host:             "localhost:8080",
			forwardedProto:   "",
			clientID:         "test-client-123",
			expectedIssuer:   "http://localhost:8080/oidc/testuser/testpass",
			expectedAudience: "test-client-123",
		},
		{
			name:             "HTTPS request via X-Forwarded-Proto",
			user:             "admin",
			pass:             "secret",
			host:             "example.com",
			forwardedProto:   "https",
			clientID:         "production-client",
			expectedIssuer:   "https://example.com/oidc/admin/secret",
			expectedAudience: "production-client",
		},
		{
			name:             "Different client ID",
			user:             "alice",
			pass:             "pass123",
			host:             "auth.example.com",
			forwardedProto:   "https",
			clientID:         "mobile-app",
			expectedIssuer:   "https://auth.example.com/oidc/alice/pass123",
			expectedAudience: "mobile-app",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			SetConfig(&Config{
				AuthSupportedScopes: []string{"openid", "profile", "email"},
			})

			authCode, _ := DefaultSessionStore.CreateAuthCode("http://localhost/callback", tt.user, "openid profile", "", "", "")

			r := chi.NewRouter()
			r.Post("/oidc/{user}/{pass}/token", OIDCTokenHandler)

			formData := url.Values{}
			formData.Add("grant_type", "authorization_code")
			formData.Add("client_id", tt.clientID)
			formData.Add("code", authCode.Code)
			formData.Add("redirect_uri", "http://localhost/callback")

			req := httptest.NewRequest(http.MethodPost, "/oidc/"+tt.user+"/"+tt.pass+"/token", strings.NewReader(formData.Encode()))
			req.Host = tt.host
			if tt.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()

			// Act
			r.ServeHTTP(rec, req)

			// Assert
			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d; body: %s", http.StatusOK, rec.Code, rec.Body.String())
			}

			var response TokenResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse JSON response: %v", err)
			}

			// Parse JWT to verify claims
			parts := strings.Split(response.IDToken, ".")
			if len(parts) != 3 {
				t.Fatalf("expected JWT to have 3 parts, got %d", len(parts))
			}

			payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
			if err != nil {
				t.Fatalf("failed to decode JWT payload: %v", err)
			}

			var claims map[string]interface{}
			if err := json.Unmarshal(payloadJSON, &claims); err != nil {
				t.Fatalf("failed to parse JWT payload JSON: %v", err)
			}

			// Assert: iss must match expected issuer
			if claims["iss"] != tt.expectedIssuer {
				t.Errorf("expected iss=%s, got %v", tt.expectedIssuer, claims["iss"])
			}

			// Assert: aud must match client_id
			if claims["aud"] != tt.expectedAudience {
				t.Errorf("expected aud=%s, got %v", tt.expectedAudience, claims["aud"])
			}
		})
	}
}

// TestDemoPageClientIDParameter verifies that demo page includes client_id in token exchange (P0-2)
func TestDemoPageClientIDParameter(t *testing.T) {
	// This test verifies that the demo page template includes client_id extraction and usage
	// We'll test the full flow to ensure client_id is passed through
	t.Run("demo flow includes client_id in token exchange", func(t *testing.T) {
		// Arrange
		SetConfig(&Config{
			AuthSupportedScopes: []string{"openid", "profile", "email"},
		})

		r := chi.NewRouter()
		r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
		r.Post("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)
		r.Get("/oidc/{user}/{pass}/demo", OIDCDemoHandler)

		user := "testuser"
		pass := "testpass"

		// Step 1: Access demo page (should redirect to authorize)
		req1 := httptest.NewRequest(http.MethodGet, "/oidc/"+user+"/"+pass+"/demo", nil)
		req1.Host = "localhost:8080"
		rec1 := httptest.NewRecorder()
		r.ServeHTTP(rec1, req1)

		if rec1.Code != http.StatusFound {
			t.Fatalf("expected redirect status %d, got %d", http.StatusFound, rec1.Code)
		}

		// Extract demo state cookie for later verification
		var demoStateCookie *http.Cookie
		for _, c := range rec1.Result().Cookies() {
			if c.Name == "demo_state" {
				demoStateCookie = c
				break
			}
		}

		// Step 2: Simulate returning from authorize with code
		// The demo page should render with client_id available
		req2 := httptest.NewRequest(http.MethodGet, "/oidc/"+user+"/"+pass+"/demo?code=test-code&state="+demoStateCookie.Value, nil)
		req2.Host = "localhost:8080"
		req2.AddCookie(demoStateCookie)
		rec2 := httptest.NewRecorder()
		r.ServeHTTP(rec2, req2)

		if rec2.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec2.Code)
		}

		// Assert: HTML should contain client_id in the template data or JavaScript
		body := rec2.Body.String()
		if !strings.Contains(body, "client_id") {
			t.Error("expected demo page to reference client_id parameter")
		}
	})
}

// TestCallbackPageClientIDParameter verifies that callback page includes client_id in token exchange (P0-2)
func TestCallbackPageClientIDParameter(t *testing.T) {
	t.Run("callback page should include mechanism to pass client_id", func(t *testing.T) {
		// Arrange
		r := chi.NewRouter()
		r.Get("/oidc/{user}/{pass}/callback", OIDCCallbackHandler)

		user := "testuser"
		pass := "testpass"

		req := httptest.NewRequest(http.MethodGet, "/oidc/"+user+"/"+pass+"/callback?code=test-code&state=test-state", nil)
		rec := httptest.NewRecorder()

		// Act
		r.ServeHTTP(rec, req)

		// Assert
		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
		}

		body := rec.Body.String()
		// The callback page should include client_id in the form submission
		if !strings.Contains(body, "client_id") {
			t.Error("expected callback page to reference client_id parameter")
		}
	})
}

// TestValidateRedirectURI verifies redirect URI validation with patterns (P1-2)
func TestValidateRedirectURI(t *testing.T) {
	tests := []struct {
		name            string
		redirectURI     string
		allowedPatterns []string
		expectError     bool
	}{
		{
			name:            "exact match",
			redirectURI:     "http://localhost:8080/callback",
			allowedPatterns: []string{"http://localhost:8080/callback"},
			expectError:     false,
		},
		{
			name:            "wildcard port match",
			redirectURI:     "http://localhost:8080/callback",
			allowedPatterns: []string{"http://localhost:*/callback"},
			expectError:     false,
		},
		{
			name:            "wildcard port different port",
			redirectURI:     "http://localhost:3000/callback",
			allowedPatterns: []string{"http://localhost:*/callback"},
			expectError:     false,
		},
		{
			name:            "wildcard path match",
			redirectURI:     "http://localhost:8080/any/path/here",
			allowedPatterns: []string{"http://localhost:8080/*"},
			expectError:     false,
		},
		{
			name:            "wildcard path with subpath",
			redirectURI:     "http://localhost:8080/callback/success",
			allowedPatterns: []string{"http://localhost:8080/*"},
			expectError:     false,
		},
		{
			name:            "multiple patterns - first matches",
			redirectURI:     "http://localhost:8080/callback",
			allowedPatterns: []string{"http://localhost:8080/callback", "http://example.com/*"},
			expectError:     false,
		},
		{
			name:            "multiple patterns - second matches",
			redirectURI:     "http://example.com/auth/callback",
			allowedPatterns: []string{"http://localhost:8080/callback", "http://example.com/*"},
			expectError:     false,
		},
		{
			name:            "no match",
			redirectURI:     "http://evil.com/callback",
			allowedPatterns: []string{"http://localhost:8080/callback"},
			expectError:     true,
		},
		{
			name:            "port mismatch without wildcard",
			redirectURI:     "http://localhost:3000/callback",
			allowedPatterns: []string{"http://localhost:8080/callback"},
			expectError:     true,
		},
		{
			name:            "path mismatch",
			redirectURI:     "http://localhost:8080/different",
			allowedPatterns: []string{"http://localhost:8080/callback"},
			expectError:     true,
		},
		{
			name:            "empty patterns - allow all",
			redirectURI:     "http://any.domain.com/anywhere",
			allowedPatterns: []string{},
			expectError:     false,
		},
		{
			name:            "nil patterns - allow all",
			redirectURI:     "http://any.domain.com/anywhere",
			allowedPatterns: nil,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := validateRedirectURI(tt.redirectURI, tt.allowedPatterns)

			// Assert
			if (err != nil) != tt.expectError {
				t.Errorf("expected error=%v, got error=%v", tt.expectError, err)
			}
		})
	}
}

// TestAuthorizeHandlerRedirectURIValidation verifies authorization endpoint validates redirect_uri (P1-2)
func TestAuthorizeHandlerRedirectURIValidation(t *testing.T) {
	tests := []struct {
		name                  string
		validateRedirectURI   bool
		allowedRedirectURIs   string
		requestRedirectURI    string
		expectedStatus        int
		expectErrorInRedirect bool
	}{
		{
			name:                  "validation disabled - any URI accepted",
			validateRedirectURI:   false,
			allowedRedirectURIs:   "http://localhost:8080/callback",
			requestRedirectURI:    "http://evil.com/callback",
			expectedStatus:        http.StatusOK,
			expectErrorInRedirect: false,
		},
		{
			name:                  "validation enabled - exact match allowed",
			validateRedirectURI:   true,
			allowedRedirectURIs:   "http://localhost:8080/callback",
			requestRedirectURI:    "http://localhost:8080/callback",
			expectedStatus:        http.StatusOK,
			expectErrorInRedirect: false,
		},
		{
			name:                  "validation enabled - wildcard port allowed",
			validateRedirectURI:   true,
			allowedRedirectURIs:   "http://localhost:*/callback",
			requestRedirectURI:    "http://localhost:3000/callback",
			expectedStatus:        http.StatusOK,
			expectErrorInRedirect: false,
		},
		{
			name:                  "validation enabled - wildcard path allowed",
			validateRedirectURI:   true,
			allowedRedirectURIs:   "http://localhost:8080/*",
			requestRedirectURI:    "http://localhost:8080/auth/callback",
			expectedStatus:        http.StatusOK,
			expectErrorInRedirect: false,
		},
		{
			name:                  "validation enabled - multiple patterns (comma-separated)",
			validateRedirectURI:   true,
			allowedRedirectURIs:   "http://localhost:8080/callback,http://localhost:3000/*",
			requestRedirectURI:    "http://localhost:3000/auth/cb",
			expectedStatus:        http.StatusOK,
			expectErrorInRedirect: false,
		},
		{
			name:                  "validation enabled - not in allowlist",
			validateRedirectURI:   true,
			allowedRedirectURIs:   "http://localhost:8080/callback",
			requestRedirectURI:    "http://evil.com/callback",
			expectedStatus:        http.StatusFound,
			expectErrorInRedirect: true,
		},
		{
			name:                  "validation enabled - empty allowlist allows all",
			validateRedirectURI:   true,
			allowedRedirectURIs:   "",
			requestRedirectURI:    "http://any.domain.com/callback",
			expectedStatus:        http.StatusOK,
			expectErrorInRedirect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			SetConfig(&Config{
				AuthSupportedScopes:         []string{"openid", "profile", "email"},
				AuthCodeValidateRedirectURI: tt.validateRedirectURI,
				AuthCodeAllowedRedirectURIs: tt.allowedRedirectURIs,
			})

			r := chi.NewRouter()
			r.Get("/oidc/{user}/{pass}/authorize", OIDCAuthorizeHandler)

			queryParams := url.Values{}
			queryParams.Add("client_id", "test-client")
			queryParams.Add("redirect_uri", tt.requestRedirectURI)
			queryParams.Add("response_type", "code")
			queryParams.Add("state", "test-state")

			req := httptest.NewRequest(http.MethodGet, "/oidc/testuser/testpass/authorize?"+queryParams.Encode(), nil)
			rec := httptest.NewRecorder()

			// Act
			r.ServeHTTP(rec, req)

			// Assert
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectErrorInRedirect && rec.Code == http.StatusFound {
				location := rec.Header().Get("Location")
				if !strings.Contains(location, "error=") {
					t.Errorf("expected error in redirect URL, got: %s", location)
				}
			}
		})
	}
}
