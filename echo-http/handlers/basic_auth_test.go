package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBasicAuthEnvHandler(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		username     string
		password     string
		setAuth      bool
		expectedCode int
		expectJSON   bool
	}{
		{
			name: "valid credentials",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			username:     "testuser",
			password:     "testpass",
			setAuth:      true,
			expectedCode: http.StatusOK,
			expectJSON:   true,
		},
		{
			name: "invalid username",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			username:     "wronguser",
			password:     "testpass",
			setAuth:      true,
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name: "invalid password",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			username:     "testuser",
			password:     "wrongpass",
			setAuth:      true,
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name:         "no auth header",
			config:       &Config{},
			setAuth:      false,
			expectedCode: http.StatusUnauthorized,
			expectJSON:   false,
		},
		{
			name:         "credentials not configured",
			config:       &Config{},
			username:     "testuser",
			password:     "testpass",
			setAuth:      true,
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
			req := httptest.NewRequest(http.MethodGet, "/basic-auth", nil)
			if tt.setAuth {
				req.SetBasicAuth(tt.username, tt.password)
			}
			w := httptest.NewRecorder()

			// Call handler
			BasicAuthEnvHandler(w, req)

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
				if resp.User != tt.username {
					t.Errorf("expected user %s, got %s", tt.username, resp.User)
				}
			}
		})
	}
}
