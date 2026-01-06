package handlers

import (
	"fmt"
	"net/url"
	"strings"
)

// OIDCDiscoveryResponse represents the OpenID Connect Discovery metadata
type OIDCDiscoveryResponse struct {
	Issuer                           string   `json:"issuer"`
	AuthorizationEndpoint            string   `json:"authorization_endpoint"`
	TokenEndpoint                    string   `json:"token_endpoint"`
	UserInfoEndpoint                 string   `json:"userinfo_endpoint"`
	JwksURI                          string   `json:"jwks_uri"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
	GrantTypesSupported              []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported    []string `json:"code_challenge_methods_supported,omitempty"`
}

// TokenResponse represents the response from the token endpoint
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// JWKSResponse represents a JSON Web Key Set response
type JWKSResponse struct {
	Keys []interface{} `json:"keys"`
}

// validateRedirectURI validates that redirectURI matches one of the allowed patterns.
// Returns nil if validation passes, error otherwise.
// Empty or nil allowedPatterns means no restrictions (allow all).
// Supports wildcards: * for any port or path segment.
func validateRedirectURI(redirectURI string, allowedPatterns []string) error {
	if len(allowedPatterns) == 0 {
		return nil // No restrictions
	}

	for _, pattern := range allowedPatterns {
		if matchRedirectPattern(redirectURI, pattern) {
			return nil
		}
	}

	return fmt.Errorf("redirect_uri not in allowlist")
}

// matchRedirectPattern checks if uri matches pattern.
// Supports wildcards:
// - "http://localhost:*/callback" matches any port
// - "http://localhost:8080/*" matches any path
func matchRedirectPattern(uri, pattern string) bool {
	// Exact match
	if uri == pattern {
		return true
	}

	// Parse URI
	uriParsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// Handle pattern specially to support wildcard port
	// Replace :* with a valid port temporarily for parsing
	patternForParsing := strings.Replace(pattern, ":*", ":9999", 1)
	hasWildcardPort := patternForParsing != pattern

	patternParsed, err := url.Parse(patternForParsing)
	if err != nil {
		return false
	}

	// Scheme must match exactly
	if uriParsed.Scheme != patternParsed.Scheme {
		return false
	}

	// Host must match exactly
	uriHost := uriParsed.Hostname()
	patternHost := patternParsed.Hostname()
	if uriHost != patternHost {
		return false
	}

	// Port matching: support wildcard *
	if !hasWildcardPort {
		// Ports must match exactly (including both being empty for default ports)
		uriPort := uriParsed.Port()
		patternPort := patternParsed.Port()
		if uriPort != patternPort {
			return false
		}
	}
	// If hasWildcardPort, accept any port

	// Path matching: support wildcard *
	if patternParsed.Path == "/*" {
		return true // Any path allowed
	}

	if uriParsed.Path != patternParsed.Path {
		return false
	}

	return true
}
