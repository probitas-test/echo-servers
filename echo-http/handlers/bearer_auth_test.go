package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBearerAuthEnvHandler(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		authHeader   string
		expectedCode int
		expectJSON   bool
	}{
		{
			name: "valid token",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			// SHA1("testuser:testpass") = 1eac13f1578ef493b9ed5617a5f4a31b271eb667
			authHeader:   "Bearer 1eac13f1578ef493b9ed5617a5f4a31b271eb667",
			expectedCode: http.StatusOK,
			expectJSON:   true,
		},
		{
			name: "case insensitive bearer",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			authHeader:   "bearer 1eac13f1578ef493b9ed5617a5f4a31b271eb667",
			expectedCode: http.StatusOK,
			expectJSON:   true,
		},
		{
			name: "invalid token",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			authHeader:   "Bearer wrongtoken",
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name:         "no auth header",
			config:       &Config{},
			authHeader:   "",
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name:         "credentials not configured",
			config:       &Config{},
			authHeader:   "Bearer sometoken",
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name: "empty token",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			authHeader:   "Bearer ",
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name: "malformed header",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			authHeader:   "Bearer",
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name: "wrong auth type",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			authHeader:   "Basic dGVzdHVzZXI6dGVzdHBhc3M=",
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/bearer-auth", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			// Call handler
			BearerAuthEnvHandler(w, req)

			// Check status code
			if w.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, w.Code)
			}

			// Check WWW-Authenticate header for 401 responses
			if w.Code == http.StatusUnauthorized {
				if w.Header().Get("WWW-Authenticate") == "" {
					t.Error("expected WWW-Authenticate header")
				}
			}

			// Check JSON response for successful auth
			if tt.expectJSON {
				var resp AuthResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if !resp.Authenticated {
					t.Error("expected authenticated=true")
				}
				if resp.Token != "1eac13f1578ef493b9ed5617a5f4a31b271eb667" {
					t.Errorf("expected token to be SHA1 hash, got %s", resp.Token)
				}
			}
		})
	}
}

func TestComputeBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		expected string
	}{
		{
			name:     "testuser:testpass",
			username: "testuser",
			password: "testpass",
			expected: "1eac13f1578ef493b9ed5617a5f4a31b271eb667",
		},
		{
			name:     "admin:secret",
			username: "admin",
			password: "secret",
			expected: "7efaf6701fdf8c6780897f20d5a1a1526dd92029",
		},
		{
			name:     "empty credentials",
			username: "",
			password: "",
			expected: "05a79f06cf3f67f726dae68d18a2290f6c9a50c9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeBearerToken(tt.username, tt.password)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
