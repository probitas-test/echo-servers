package handlers

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuth2TokenHandler_ClientCredentials(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		formData      map[string]string
		expectedCode  int
		expectError   bool
		errorType     string
		checkResponse func(*testing.T, *TokenResponse)
	}{
		{
			name: "valid client credentials",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
				AuthSupportedScopes:     []string{"openid", "profile"},
				AuthTokenExpiry:         7200,
				AuthAllowedGrantTypes:   []string{"client_credentials"},
			},
			formData: map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     "test-client",
				"client_secret": "test-secret",
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.AccessToken == "" {
					t.Error("expected access_token")
				}
				if resp.TokenType != "Bearer" {
					t.Errorf("expected token_type Bearer, got %s", resp.TokenType)
				}
				if resp.ExpiresIn != 7200 {
					t.Errorf("expected expires_in 7200, got %d", resp.ExpiresIn)
				}
				// Client Credentials should NOT include id_token
				if resp.IDToken != "" {
					t.Error("client_credentials should not return id_token")
				}
				// Should NOT include refresh_token in basic implementation
				if resp.RefreshToken != "" {
					t.Error("client_credentials should not return refresh_token")
				}
			},
		},
		{
			name: "with custom scope",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
				AuthSupportedScopes:     []string{"openid", "profile", "email"},
				AuthAllowedGrantTypes:   []string{"client_credentials"},
			},
			formData: map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     "test-client",
				"client_secret": "test-secret",
				"scope":         "openid profile",
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.Scope != "openid profile" {
					t.Errorf("expected scope 'openid profile', got %s", resp.Scope)
				}
			},
		},
		{
			name: "unsupported scope",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
				AuthSupportedScopes:     []string{"openid", "profile"},
				AuthAllowedGrantTypes:   []string{"client_credentials"},
			},
			formData: map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     "test-client",
				"client_secret": "test-secret",
				"scope":         "invalid_scope",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidScope,
		},
		{
			name: "missing client_id",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
				AuthAllowedGrantTypes:   []string{"client_credentials"},
			},
			formData: map[string]string{
				"grant_type":    "client_credentials",
				"client_secret": "test-secret",
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
			errorType:    ErrorInvalidClient,
		},
		{
			name: "invalid client_secret",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
				AuthAllowedGrantTypes:   []string{"client_credentials"},
			},
			formData: map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     "test-client",
				"client_secret": "wrong-secret",
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
			errorType:    ErrorInvalidClient,
		},
		{
			name: "grant_type not allowed",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
				AuthAllowedGrantTypes:   []string{"authorization_code"}, // client_credentials not allowed
			},
			formData: map[string]string{
				"grant_type":    "client_credentials",
				"client_id":     "test-client",
				"client_secret": "test-secret",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorUnsupportedGrantType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			// Create request
			formData := url.Values{}
			for k, v := range tt.formData {
				formData.Set(k, v)
			}

			req := httptest.NewRequest(http.MethodPost, "/oauth2/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// Call handler
			OAuth2TokenHandler(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectError {
				// Check error response
				var errResp OIDCError
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error != tt.errorType {
					t.Errorf("expected error %s, got %s", tt.errorType, errResp.Error)
				}
			} else if tt.checkResponse != nil {
				// Check success response
				var resp TokenResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestOAuth2TokenHandler_AuthorizationCode(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		setupAuthCode func() string // Returns authorization code
		formData      map[string]string
		expectedCode  int
		expectError   bool
		errorType     string
		checkResponse func(*testing.T, *TokenResponse)
	}{
		{
			name: "valid authorization code",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "",
				AuthSupportedScopes:     []string{"openid", "profile"},
				AuthTokenExpiry:         7200,
				AuthAllowedGrantTypes:   []string{"authorization_code"},
			},
			setupAuthCode: func() string {
				code, _ := DefaultSessionStore.CreateAuthCode(
					"http://localhost/callback",
					"testuser",
					"openid profile",
					"",
					"",
					"test-nonce",
				)
				return code.Code
			},
			formData: map[string]string{
				"grant_type":   "authorization_code",
				"client_id":    "test-client",
				"redirect_uri": "http://localhost/callback",
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.AccessToken == "" {
					t.Error("expected access_token")
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh_token")
				}
				// Authorization Code with OIDC SHOULD include id_token
				if resp.IDToken == "" {
					t.Error("authorization_code should return id_token")
				}
				if resp.Scope != "openid profile" {
					t.Errorf("expected scope 'openid profile', got %s", resp.Scope)
				}
			},
		},
		{
			name: "with PKCE S256",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedGrantTypes: []string{"authorization_code"},
			},
			setupAuthCode: func() string {
				// Generate code_verifier
				verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
				// Compute S256 challenge
				h := sha256.Sum256([]byte(verifier))
				challenge := base64.RawURLEncoding.EncodeToString(h[:])

				code, _ := DefaultSessionStore.CreateAuthCode(
					"http://localhost/callback",
					"testuser",
					"openid",
					challenge,
					"S256",
					"",
				)
				return code.Code
			},
			formData: map[string]string{
				"grant_type":    "authorization_code",
				"client_id":     "test-client",
				"redirect_uri":  "http://localhost/callback",
				"code_verifier": "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "invalid code_verifier",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedGrantTypes: []string{"authorization_code"},
			},
			setupAuthCode: func() string {
				verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
				h := sha256.Sum256([]byte(verifier))
				challenge := base64.RawURLEncoding.EncodeToString(h[:])

				code, _ := DefaultSessionStore.CreateAuthCode(
					"http://localhost/callback",
					"testuser",
					"openid",
					challenge,
					"S256",
					"",
				)
				return code.Code
			},
			formData: map[string]string{
				"grant_type":    "authorization_code",
				"client_id":     "test-client",
				"redirect_uri":  "http://localhost/callback",
				"code_verifier": "wrong-verifier-but-valid-length-aaaaaaaaaaa",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
		{
			name: "missing code_verifier when required",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedGrantTypes: []string{"authorization_code"},
			},
			setupAuthCode: func() string {
				code, _ := DefaultSessionStore.CreateAuthCode(
					"http://localhost/callback",
					"testuser",
					"openid",
					"test-challenge",
					"plain",
					"",
				)
				return code.Code
			},
			formData: map[string]string{
				"grant_type":   "authorization_code",
				"client_id":    "test-client",
				"redirect_uri": "http://localhost/callback",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
		{
			name: "invalid authorization code",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedGrantTypes: []string{"authorization_code"},
			},
			setupAuthCode: func() string {
				return "invalid-code"
			},
			formData: map[string]string{
				"grant_type":   "authorization_code",
				"client_id":    "test-client",
				"code":         "invalid-code",
				"redirect_uri": "http://localhost/callback",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
		{
			name: "redirect_uri mismatch",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedGrantTypes: []string{"authorization_code"},
			},
			setupAuthCode: func() string {
				code, _ := DefaultSessionStore.CreateAuthCode(
					"http://localhost/callback",
					"testuser",
					"openid",
					"",
					"",
					"",
				)
				return code.Code
			},
			formData: map[string]string{
				"grant_type":   "authorization_code",
				"client_id":    "test-client",
				"redirect_uri": "http://wrong/callback",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			// Setup authorization code
			code := tt.setupAuthCode()

			// Create request
			formData := url.Values{}
			for k, v := range tt.formData {
				formData.Set(k, v)
			}
			// Add code if not already in formData
			if _, exists := tt.formData["code"]; !exists {
				formData.Set("code", code)
			}

			req := httptest.NewRequest(http.MethodPost, "/oauth2/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// Call handler
			OAuth2TokenHandler(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectError {
				// Check error response
				var errResp OIDCError
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error != tt.errorType {
					t.Errorf("expected error %s, got %s", tt.errorType, errResp.Error)
				}
			} else if tt.checkResponse != nil {
				// Check success response
				var resp TokenResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestOAuth2TokenHandler_Password(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		formData      map[string]string
		expectedCode  int
		expectError   bool
		errorType     string
		checkResponse func(*testing.T, *TokenResponse)
	}{
		{
			name: "valid credentials",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "",
				AuthAllowedUsername:     "testuser",
				AuthAllowedPassword:     "testpass",
				AuthSupportedScopes:     []string{"openid", "profile", "email"},
				AuthTokenExpiry:         3600,
				AuthAllowedGrantTypes:   []string{"password"},
			},
			formData: map[string]string{
				"grant_type": "password",
				"username":   "testuser",
				"password":   "testpass",
				"client_id":  "test-client",
				"scope":      "openid profile",
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.AccessToken == "" {
					t.Error("expected access_token")
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh_token")
				}
				if resp.IDToken == "" {
					t.Error("expected id_token with openid scope")
				}
				if resp.TokenType != "Bearer" {
					t.Errorf("expected token_type Bearer, got %s", resp.TokenType)
				}
				if resp.Scope != "openid profile" {
					t.Errorf("expected scope 'openid profile', got %s", resp.Scope)
				}
			},
		},
		{
			name: "without openid scope - no id_token",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedUsername:   "testuser",
				AuthAllowedPassword:   "testpass",
				AuthSupportedScopes:   []string{"profile", "email"},
				AuthAllowedGrantTypes: []string{"password"},
			},
			formData: map[string]string{
				"grant_type": "password",
				"username":   "testuser",
				"password":   "testpass",
				"client_id":  "test-client",
				"scope":      "profile email",
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.IDToken != "" {
					t.Error("should not return id_token without openid scope")
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh_token")
				}
			},
		},
		{
			name: "invalid username",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedUsername:   "testuser",
				AuthAllowedPassword:   "testpass",
				AuthAllowedGrantTypes: []string{"password"},
			},
			formData: map[string]string{
				"grant_type": "password",
				"username":   "wronguser",
				"password":   "testpass",
				"client_id":  "test-client",
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
		{
			name: "invalid password",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedUsername:   "testuser",
				AuthAllowedPassword:   "testpass",
				AuthAllowedGrantTypes: []string{"password"},
			},
			formData: map[string]string{
				"grant_type": "password",
				"username":   "testuser",
				"password":   "wrongpass",
				"client_id":  "test-client",
			},
			expectedCode: http.StatusUnauthorized,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
		{
			name: "unsupported scope",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedUsername:   "testuser",
				AuthAllowedPassword:   "testpass",
				AuthSupportedScopes:   []string{"openid", "profile"},
				AuthAllowedGrantTypes: []string{"password"},
			},
			formData: map[string]string{
				"grant_type": "password",
				"username":   "testuser",
				"password":   "testpass",
				"client_id":  "test-client",
				"scope":      "invalid_scope",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidScope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			// Create request
			formData := url.Values{}
			for k, v := range tt.formData {
				formData.Set(k, v)
			}

			req := httptest.NewRequest(http.MethodPost, "/oauth2/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// Call handler
			OAuth2TokenHandler(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectError {
				// Check error response
				var errResp OIDCError
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error != tt.errorType {
					t.Errorf("expected error %s, got %s", tt.errorType, errResp.Error)
				}
			} else if tt.checkResponse != nil {
				// Check success response
				var resp TokenResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestOAuth2TokenHandler_RefreshToken(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		setupToken    func() string // Returns refresh token
		formData      map[string]string
		expectedCode  int
		expectError   bool
		errorType     string
		checkResponse func(*testing.T, *TokenResponse)
	}{
		{
			name: "valid refresh token",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthSupportedScopes:   []string{"openid", "profile", "email"},
				AuthTokenExpiry:       3600,
				AuthAllowedGrantTypes: []string{"refresh_token"},
			},
			setupToken: func() string {
				token, _ := DefaultSessionStore.CreateRefreshToken(
					"testuser",
					"test-client",
					"openid profile email",
					"test-nonce",
				)
				return token.Token
			},
			formData: map[string]string{
				"grant_type": "refresh_token",
				"client_id":  "test-client",
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.AccessToken == "" {
					t.Error("expected access_token")
				}
				if resp.RefreshToken == "" {
					t.Error("expected refresh_token")
				}
				if resp.IDToken == "" {
					t.Error("expected id_token with openid scope")
				}
				if resp.Scope != "openid profile email" {
					t.Errorf("expected scope 'openid profile email', got %s", resp.Scope)
				}
			},
		},
		{
			name: "scope narrowing - valid",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthSupportedScopes:   []string{"openid", "profile", "email"},
				AuthAllowedGrantTypes: []string{"refresh_token"},
			},
			setupToken: func() string {
				token, _ := DefaultSessionStore.CreateRefreshToken(
					"testuser",
					"test-client",
					"openid profile email",
					"",
				)
				return token.Token
			},
			formData: map[string]string{
				"grant_type": "refresh_token",
				"client_id":  "test-client",
				"scope":      "openid profile", // Narrower than original
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.Scope != "openid profile" {
					t.Errorf("expected scope 'openid profile', got %s", resp.Scope)
				}
			},
		},
		{
			name: "scope expansion - invalid",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthSupportedScopes:   []string{"openid", "profile", "email"},
				AuthAllowedGrantTypes: []string{"refresh_token"},
			},
			setupToken: func() string {
				token, _ := DefaultSessionStore.CreateRefreshToken(
					"testuser",
					"test-client",
					"openid profile",
					"",
				)
				return token.Token
			},
			formData: map[string]string{
				"grant_type": "refresh_token",
				"client_id":  "test-client",
				"scope":      "openid profile email", // Broader than original
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidScope,
		},
		{
			name: "invalid refresh token",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedGrantTypes: []string{"refresh_token"},
			},
			setupToken: func() string {
				return "invalid-token"
			},
			formData: map[string]string{
				"grant_type":    "refresh_token",
				"client_id":     "test-client",
				"refresh_token": "invalid-token",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
		{
			name: "client_id mismatch",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthAllowedGrantTypes: []string{"refresh_token"},
			},
			setupToken: func() string {
				token, _ := DefaultSessionStore.CreateRefreshToken(
					"testuser",
					"other-client",
					"openid",
					"",
				)
				return token.Token
			},
			formData: map[string]string{
				"grant_type": "refresh_token",
				"client_id":  "test-client",
			},
			expectedCode: http.StatusBadRequest,
			expectError:  true,
			errorType:    ErrorInvalidGrant,
		},
		{
			name: "without openid scope - no id_token",
			config: &Config{
				AuthAllowedClientID:   "test-client",
				AuthSupportedScopes:   []string{"profile", "email"},
				AuthAllowedGrantTypes: []string{"refresh_token"},
			},
			setupToken: func() string {
				token, _ := DefaultSessionStore.CreateRefreshToken(
					"testuser",
					"test-client",
					"profile email",
					"",
				)
				return token.Token
			},
			formData: map[string]string{
				"grant_type": "refresh_token",
				"client_id":  "test-client",
			},
			expectedCode: http.StatusOK,
			checkResponse: func(t *testing.T, resp *TokenResponse) {
				if resp.IDToken != "" {
					t.Error("should not return id_token without openid scope")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			// Setup refresh token
			refreshToken := tt.setupToken()

			// Create request
			formData := url.Values{}
			for k, v := range tt.formData {
				formData.Set(k, v)
			}
			// Add refresh_token if not already in formData
			if _, exists := tt.formData["refresh_token"]; !exists {
				formData.Set("refresh_token", refreshToken)
			}

			req := httptest.NewRequest(http.MethodPost, "/oauth2/token", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// Call handler
			OAuth2TokenHandler(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.expectError {
				// Check error response
				var errResp OIDCError
				if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}
				if errResp.Error != tt.errorType {
					t.Errorf("expected error %s, got %s", tt.errorType, errResp.Error)
				}
			} else if tt.checkResponse != nil {
				// Check success response
				var resp TokenResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.checkResponse(t, &resp)
			}
		})
	}
}

func TestVerifyPKCECodeChallenge(t *testing.T) {
	tests := []struct {
		name      string
		challenge string
		method    string
		verifier  string
		expected  bool
	}{
		{
			name:      "plain method - valid",
			challenge: "test-challenge",
			method:    "plain",
			verifier:  "test-challenge",
			expected:  true,
		},
		{
			name:      "plain method - invalid",
			challenge: "test-challenge",
			method:    "plain",
			verifier:  "wrong-verifier",
			expected:  false,
		},
		{
			name:      "S256 method - valid",
			challenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			method:    "S256",
			verifier:  "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
			expected:  true,
		},
		{
			name:      "S256 method - invalid",
			challenge: "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM",
			method:    "S256",
			verifier:  "wrong-verifier",
			expected:  false,
		},
		{
			name:      "unknown method",
			challenge: "test-challenge",
			method:    "unknown",
			verifier:  "test-challenge",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyPKCECodeChallenge(tt.challenge, tt.method, tt.verifier)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
