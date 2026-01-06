package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuth2CallbackHandler(t *testing.T) {
	tests := []struct {
		name         string
		queryParams  string
		expectedCode int
		checkBody    bool
	}{
		{
			name:         "with code and state",
			queryParams:  "?code=test-code&state=test-state",
			expectedCode: http.StatusOK,
			checkBody:    true,
		},
		{
			name:         "without code",
			queryParams:  "?state=test-state",
			expectedCode: http.StatusOK,
			checkBody:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/oauth2/callback"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			OAuth2CallbackHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.checkBody {
				body := w.Body.String()
				if body == "" {
					t.Error("expected HTML body")
				}
			}
		})
	}
}

func TestOAuth2UserInfoHandler(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		authHeader   string
		expectedCode int
		checkJSON    bool
	}{
		{
			name: "valid bearer token",
			config: &Config{
				AuthAllowedUsername: "testuser",
			},
			authHeader:   "Bearer test-token",
			expectedCode: http.StatusOK,
			checkJSON:    true,
		},
		{
			name:         "missing authorization header",
			config:       &Config{},
			authHeader:   "",
			expectedCode: http.StatusUnauthorized,
			checkJSON:    false,
		},
		{
			name:         "invalid scheme",
			config:       &Config{},
			authHeader:   "Basic dGVzdDp0ZXN0",
			expectedCode: http.StatusUnauthorized,
			checkJSON:    false,
		},
		{
			name:         "empty token",
			config:       &Config{},
			authHeader:   "Bearer ",
			expectedCode: http.StatusUnauthorized,
			checkJSON:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			req := httptest.NewRequest(http.MethodGet, "/oauth2/userinfo", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			OAuth2UserInfoHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			if tt.checkJSON {
				var userInfo map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&userInfo); err != nil {
					t.Fatalf("failed to decode JSON: %v", err)
				}
				if _, ok := userInfo["sub"]; !ok {
					t.Error("expected 'sub' field in userinfo")
				}
			}
		})
	}
}

func TestOAuth2DemoHandler(t *testing.T) {
	tests := []struct {
		name         string
		queryParams  string
		expectedCode int
	}{
		{
			name:         "initiate flow",
			queryParams:  "",
			expectedCode: http.StatusFound, // Redirect to authorize
		},
		{
			name:         "callback with code",
			queryParams:  "?code=test-code&state=test-state",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = &Config{
				AuthSupportedScopes: []string{"openid", "profile"},
			}
			defer func() { globalConfig = originalConfig }()

			req := httptest.NewRequest(http.MethodGet, "/oauth2/demo"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			OAuth2DemoHandler(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}
		})
	}
}
