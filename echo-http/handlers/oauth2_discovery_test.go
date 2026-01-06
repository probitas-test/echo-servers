package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOAuth2MetadataHandler(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		check  func(*testing.T, *OAuth2MetadataResponse)
	}{
		{
			name: "default configuration",
			config: &Config{
				AuthSupportedScopes:   []string{"openid", "profile", "email"},
				AuthAllowedGrantTypes: []string{"authorization_code", "client_credentials"},
			},
			check: func(t *testing.T, resp *OAuth2MetadataResponse) {
				if resp.Issuer != "http://example.com" {
					t.Errorf("expected issuer http://example.com, got %s", resp.Issuer)
				}
				if resp.TokenEndpoint != "http://example.com/oauth2/token" {
					t.Errorf("unexpected token_endpoint: %s", resp.TokenEndpoint)
				}
				if len(resp.GrantTypesSupported) != 2 {
					t.Errorf("expected 2 grant types, got %d", len(resp.GrantTypesSupported))
				}
				// Should include authorization endpoint when authorization_code is allowed
				if resp.AuthorizationEndpoint != "http://example.com/oauth2/authorize" {
					t.Errorf("expected authorization_endpoint, got %s", resp.AuthorizationEndpoint)
				}
			},
		},
		{
			name: "only client_credentials allowed",
			config: &Config{
				AuthSupportedScopes:   []string{"openid"},
				AuthAllowedGrantTypes: []string{"client_credentials"},
			},
			check: func(t *testing.T, resp *OAuth2MetadataResponse) {
				// Should NOT include authorization endpoint when authorization_code is not allowed
				if resp.AuthorizationEndpoint != "" {
					t.Errorf("expected no authorization_endpoint for client_credentials only, got %s", resp.AuthorizationEndpoint)
				}
				if len(resp.ResponseTypesSupported) != 0 {
					t.Error("expected no response_types_supported for client_credentials only")
				}
			},
		},
		{
			name: "PKCE support included",
			config: &Config{
				AuthAllowedGrantTypes: []string{"authorization_code"},
			},
			check: func(t *testing.T, resp *OAuth2MetadataResponse) {
				if len(resp.CodeChallengeMethodsSupported) != 2 {
					t.Errorf("expected 2 code_challenge_methods, got %d", len(resp.CodeChallengeMethodsSupported))
				}
				found := make(map[string]bool)
				for _, method := range resp.CodeChallengeMethodsSupported {
					found[method] = true
				}
				if !found["plain"] || !found["S256"] {
					t.Error("expected plain and S256 methods")
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

			// Create request
			req := httptest.NewRequest(http.MethodGet, "http://example.com/.well-known/oauth-authorization-server", nil)
			w := httptest.NewRecorder()

			// Call handler
			OAuth2MetadataHandler(w, req)

			// Check status code
			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			// Decode response
			var resp OAuth2MetadataResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Run custom checks
			if tt.check != nil {
				tt.check(t, &resp)
			}
		})
	}
}

func TestOIDCDiscoveryRootHandler(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		check  func(*testing.T, *OIDCDiscoveryResponse)
	}{
		{
			name: "default configuration",
			config: &Config{
				AuthSupportedScopes:   []string{"openid", "profile"},
				AuthAllowedGrantTypes: []string{"authorization_code"},
			},
			check: func(t *testing.T, resp *OIDCDiscoveryResponse) {
				if resp.Issuer != "http://example.com" {
					t.Errorf("expected issuer http://example.com, got %s", resp.Issuer)
				}
				if resp.TokenEndpoint != "http://example.com/oauth2/token" {
					t.Errorf("unexpected token_endpoint: %s", resp.TokenEndpoint)
				}
				if resp.UserInfoEndpoint != "http://example.com/oauth2/userinfo" {
					t.Errorf("unexpected userinfo_endpoint: %s", resp.UserInfoEndpoint)
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

			// Create request
			req := httptest.NewRequest(http.MethodGet, "http://example.com/.well-known/openid-configuration", nil)
			w := httptest.NewRecorder()

			// Call handler
			OIDCDiscoveryRootHandler(w, req)

			// Check status code
			if w.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", w.Code)
			}

			// Decode response
			var resp OIDCDiscoveryResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Run custom checks
			if tt.check != nil {
				tt.check(t, &resp)
			}
		})
	}
}

func TestOAuth2JWKSHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/.well-known/jwks.json", nil)
	w := httptest.NewRecorder()

	OAuth2JWKSHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp JWKSResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should return empty keys array (alg=none)
	if len(resp.Keys) != 0 {
		t.Errorf("expected empty keys array, got %d keys", len(resp.Keys))
	}
}
