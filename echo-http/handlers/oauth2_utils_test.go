package handlers

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateClientCredentials(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		clientID      string
		clientSecret  string
		requireSecret bool
		expectError   bool
		errorContains string
	}{
		{
			name:        "missing client_id",
			config:      &Config{},
			clientID:    "",
			expectError: true,
		},
		{
			name:        "no config - accept any client",
			config:      nil,
			clientID:    "any-client",
			expectError: false,
		},
		{
			name: "empty allowed client_id - accept any client",
			config: &Config{
				AuthAllowedClientID: "",
			},
			clientID:    "any-client",
			expectError: false,
		},
		{
			name: "valid client_id - no secret required",
			config: &Config{
				AuthAllowedClientID: "test-client",
			},
			clientID:      "test-client",
			requireSecret: false,
			expectError:   false,
		},
		{
			name: "invalid client_id",
			config: &Config{
				AuthAllowedClientID: "test-client",
			},
			clientID:      "wrong-client",
			expectError:   true,
			errorContains: "unknown client_id",
		},
		{
			name: "valid client_id and client_secret",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
			},
			clientID:      "test-client",
			clientSecret:  "test-secret",
			requireSecret: true,
			expectError:   false,
		},
		{
			name: "valid client_id but invalid client_secret",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
			},
			clientID:      "test-client",
			clientSecret:  "wrong-secret",
			requireSecret: true,
			expectError:   true,
			errorContains: "invalid client_secret",
		},
		{
			name: "secret configured but not provided",
			config: &Config{
				AuthAllowedClientID:     "test-client",
				AuthAllowedClientSecret: "test-secret",
			},
			clientID:      "test-client",
			clientSecret:  "",
			requireSecret: false,
			expectError:   true,
			errorContains: "invalid client_secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config for this test
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			err := validateClientCredentials(tt.clientID, tt.clientSecret, tt.requireSecret)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.errorContains != "" && err != nil {
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

func TestValidateGrantType(t *testing.T) {
	tests := []struct {
		name          string
		grantType     string
		allowedTypes  []string
		expectError   bool
		errorContains string
	}{
		{
			name:        "empty grant_type",
			grantType:   "",
			expectError: true,
		},
		{
			name:         "supported grant_type",
			grantType:    "authorization_code",
			allowedTypes: []string{"authorization_code", "client_credentials"},
			expectError:  false,
		},
		{
			name:          "unsupported grant_type",
			grantType:     "password",
			allowedTypes:  []string{"authorization_code", "client_credentials"},
			expectError:   true,
			errorContains: "unsupported grant_type",
		},
		{
			name:         "client_credentials supported",
			grantType:    "client_credentials",
			allowedTypes: []string{"client_credentials"},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGrantType(tt.grantType, tt.allowedTypes)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.errorContains != "" && err != nil {
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

func TestValidateBasicAuthCredentials(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		username      string
		password      string
		expectError   bool
		errorContains string
	}{
		{
			name:        "empty username",
			config:      &Config{},
			username:    "",
			password:    "pass",
			expectError: true,
		},
		{
			name:        "empty password",
			config:      &Config{},
			username:    "user",
			password:    "",
			expectError: true,
		},
		{
			name:          "credentials not configured",
			config:        &Config{},
			username:      "user",
			password:      "pass",
			expectError:   true,
			errorContains: "not configured",
		},
		{
			name: "valid credentials",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			username:    "testuser",
			password:    "testpass",
			expectError: false,
		},
		{
			name: "invalid username",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			username:      "wronguser",
			password:      "testpass",
			expectError:   true,
			errorContains: "invalid username or password",
		},
		{
			name: "invalid password",
			config: &Config{
				AuthAllowedUsername: "testuser",
				AuthAllowedPassword: "testpass",
			},
			username:      "testuser",
			password:      "wrongpass",
			expectError:   true,
			errorContains: "invalid username or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config for this test
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			err := validateBasicAuthCredentials(tt.username, tt.password)

			if tt.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
			if tt.errorContains != "" && err != nil {
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain %q, got: %v", tt.errorContains, err)
				}
			}
		})
	}
}

func TestIsGrantTypeAllowed(t *testing.T) {
	tests := []struct {
		name         string
		grantType    string
		allowedTypes []string
		expected     bool
	}{
		{
			name:         "grant type is allowed",
			grantType:    "authorization_code",
			allowedTypes: []string{"authorization_code", "client_credentials"},
			expected:     true,
		},
		{
			name:         "grant type is not allowed",
			grantType:    "password",
			allowedTypes: []string{"authorization_code", "client_credentials"},
			expected:     false,
		},
		{
			name:         "empty allowed list",
			grantType:    "authorization_code",
			allowedTypes: []string{},
			expected:     false,
		},
		{
			name:         "single allowed type matches",
			grantType:    "client_credentials",
			allowedTypes: []string{"client_credentials"},
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGrantTypeAllowed(tt.grantType, tt.allowedTypes)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBuildBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		tls      bool
		proto    string
		expected string
	}{
		{
			name:     "http without TLS",
			host:     "localhost:8080",
			tls:      false,
			expected: "http://localhost:8080",
		},
		{
			name:     "https with TLS",
			host:     "example.com",
			tls:      true,
			expected: "https://example.com",
		},
		{
			name:     "X-Forwarded-Proto overrides",
			host:     "localhost:8080",
			tls:      false,
			proto:    "https",
			expected: "https://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host
			if tt.tls {
				req.TLS = &tls.ConnectionState{} // Non-nil TLS indicates HTTPS
			}
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}

			result := buildBaseURL(req)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildIssuerURL(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		user       string
		pass       string
		deprecated bool
		expected   string
	}{
		{
			name:       "new endpoint - no user/pass in URL",
			host:       "localhost:8080",
			user:       "testuser",
			pass:       "testpass",
			deprecated: false,
			expected:   "http://localhost:8080",
		},
		{
			name:       "deprecated endpoint - includes user/pass",
			host:       "localhost:8080",
			user:       "testuser",
			pass:       "testpass",
			deprecated: true,
			expected:   "http://localhost:8080/oidc/testuser/testpass",
		},
		{
			name:       "deprecated with empty user/pass - base URL only",
			host:       "localhost:8080",
			user:       "",
			pass:       "",
			deprecated: true,
			expected:   "http://localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host

			result := buildIssuerURL(req, tt.user, tt.pass, tt.deprecated)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetAllowedGrantTypes(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected []string
	}{
		{
			name:     "no config - return defaults",
			config:   nil,
			expected: []string{"authorization_code", "client_credentials"},
		},
		{
			name: "configured grant types",
			config: &Config{
				AuthAllowedGrantTypes: []string{"client_credentials"},
			},
			expected: []string{"client_credentials"},
		},
		{
			name: "empty grant types - return defaults",
			config: &Config{
				AuthAllowedGrantTypes: []string{},
			},
			expected: []string{"authorization_code", "client_credentials"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global config for this test
			originalConfig := globalConfig
			globalConfig = tt.config
			defer func() { globalConfig = originalConfig }()

			result := getAllowedGrantTypes()
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d grant types, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("expected grant type %q at index %d, got %q", expected, i, result[i])
				}
			}
		})
	}
}

func TestJoinScopes(t *testing.T) {
	tests := []struct {
		name     string
		scopes   []string
		expected string
	}{
		{
			name:     "single scope",
			scopes:   []string{"openid"},
			expected: "openid",
		},
		{
			name:     "multiple scopes",
			scopes:   []string{"openid", "profile", "email"},
			expected: "openid profile email",
		},
		{
			name:     "empty slice",
			scopes:   []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinScopes(tt.scopes)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestSplitScopes(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		expected []string
	}{
		{
			name:     "single scope",
			scope:    "openid",
			expected: []string{"openid"},
		},
		{
			name:     "multiple scopes",
			scope:    "openid profile email",
			expected: []string{"openid", "profile", "email"},
		},
		{
			name:     "empty string",
			scope:    "",
			expected: []string{},
		},
		{
			name:     "scopes with extra spaces",
			scope:    "openid  profile   email",
			expected: []string{"openid", "profile", "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitScopes(tt.scope)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d scopes, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("expected scope %q at index %d, got %q", expected, i, result[i])
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
