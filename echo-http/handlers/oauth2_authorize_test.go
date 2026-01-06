package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestOAuth2AuthorizeHandler_GET(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		queryParams  map[string]string
		expectedCode int
		expectCookie bool
	}{
		{
			name: "valid request",
			config: &Config{
				AuthAllowedClientID: "test-client",
				AuthSupportedScopes: []string{"openid", "profile"},
			},
			queryParams: map[string]string{
				"client_id":     "test-client",
				"redirect_uri":  "http://localhost/callback",
				"response_type": "code",
				"scope":         "openid",
				"state":         "test-state",
			},
			expectedCode: http.StatusOK,
			expectCookie: true,
		},
		{
			name:   "missing client_id",
			config: &Config{},
			queryParams: map[string]string{
				"redirect_uri":  "http://localhost/callback",
				"response_type": "code",
			},
			expectedCode: http.StatusBadRequest,
			expectCookie: false,
		},
		{
			name: "missing redirect_uri",
			config: &Config{
				AuthAllowedClientID: "test-client",
			},
			queryParams: map[string]string{
				"client_id":     "test-client",
				"response_type": "code",
			},
			expectedCode: http.StatusBadRequest,
			expectCookie: false,
		},
		{
			name: "invalid response_type",
			config: &Config{
				AuthAllowedClientID: "test-client",
			},
			queryParams: map[string]string{
				"client_id":     "test-client",
				"redirect_uri":  "http://localhost/callback",
				"response_type": "token",
			},
			expectedCode: http.StatusFound, // Redirect with error
			expectCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			// Build query string
			query := url.Values{}
			for k, v := range tt.queryParams {
				query.Set(k, v)
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/oauth2/authorize?"+query.Encode(), nil)
			w := httptest.NewRecorder()

			// Call handler
			OAuth2AuthorizeHandler(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			// Check cookie
			cookies := w.Result().Cookies()
			hasCookie := false
			for _, c := range cookies {
				if c.Name == "oauth2_session" {
					hasCookie = true
					break
				}
			}
			if tt.expectCookie && !hasCookie {
				t.Error("expected oauth2_session cookie")
			}
		})
	}
}

func TestOAuth2AuthorizeHandler_POST(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		setupSession func() string // Returns session ID
		formData     map[string]string
		expectedCode int
	}{
		{
			name: "valid credentials",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
				AuthSupportedScopes: []string{"openid"},
			},
			setupSession: func() string {
				session, _ := DefaultSessionStore.CreateSession(
					"test-state",
					"http://localhost/callback",
					"openid",
					"",
					"",
					"",
				)
				return session.ID
			},
			formData: map[string]string{
				"username": "testuser",
				"password": "testpass",
			},
			expectedCode: http.StatusFound, // Redirect with code
		},
		{
			name: "invalid credentials",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			setupSession: func() string {
				session, _ := DefaultSessionStore.CreateSession(
					"test-state",
					"http://localhost/callback",
					"openid",
					"",
					"",
					"",
				)
				return session.ID
			},
			formData: map[string]string{
				"username": "testuser",
				"password": "wrongpass",
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "missing session",
			config: &Config{},
			setupSession: func() string {
				return "invalid-session-id"
			},
			formData: map[string]string{
				"username": "testuser",
				"password": "testpass",
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			// Setup session
			sessionID := tt.setupSession()

			// Build form data
			formData := url.Values{}
			for k, v := range tt.formData {
				formData.Set(k, v)
			}

			// Create request with session cookie
			req := httptest.NewRequest(http.MethodPost, "/oauth2/authorize", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{
				Name:  "oauth2_session",
				Value: sessionID,
			})
			w := httptest.NewRecorder()

			// Call handler
			OAuth2AuthorizeHandler(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}
