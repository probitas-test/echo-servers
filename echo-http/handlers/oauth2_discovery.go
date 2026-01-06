package handlers

import (
	"encoding/json"
	"net/http"
)

// OAuth2MetadataResponse represents the OAuth 2.0 Authorization Server Metadata.
// Spec: RFC 8414 - OAuth 2.0 Authorization Server Metadata
type OAuth2MetadataResponse struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	JwksURI                           string   `json:"jwks_uri,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported,omitempty"`
	GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
	SubjectTypesSupported             []string `json:"subject_types_supported,omitempty"`
	IDTokenSigningAlgValuesSupported  []string `json:"id_token_signing_alg_values_supported,omitempty"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
	UserInfoEndpoint                  string   `json:"userinfo_endpoint,omitempty"`
}

// OAuth2MetadataHandler provides OAuth 2.0 Authorization Server Metadata.
// GET /.well-known/oauth-authorization-server
// Spec: RFC 8414
func OAuth2MetadataHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := buildBaseURL(r)

	// Get allowed grant types from config
	allowedGrantTypes := getAllowedGrantTypes()

	// Determine which endpoints to include based on allowed grant types
	var authorizationEndpoint string
	var responseTypesSupported []string
	var codeChallengeMethodsSupported []string

	// Include authorization endpoint only if authorization_code is allowed
	for _, gt := range allowedGrantTypes {
		if gt == "authorization_code" {
			authorizationEndpoint = baseURL + "/oauth2/authorize"
			responseTypesSupported = []string{"code"}
			codeChallengeMethodsSupported = []string{"plain", "S256"}
			break
		}
	}

	// Get scopes from config, or use defaults if not configured
	supportedScopes := []string{"openid", "profile", "email"}
	if globalConfig != nil && len(globalConfig.AuthSupportedScopes) > 0 {
		supportedScopes = globalConfig.AuthSupportedScopes
	}

	metadata := OAuth2MetadataResponse{
		Issuer:                 baseURL,
		AuthorizationEndpoint:  authorizationEndpoint,
		TokenEndpoint:          baseURL + "/oauth2/token",
		JwksURI:                baseURL + "/.well-known/jwks.json",
		ResponseTypesSupported: responseTypesSupported,
		GrantTypesSupported:    allowedGrantTypes,
		SubjectTypesSupported: []string{
			"public",
		},
		IDTokenSigningAlgValuesSupported: []string{
			"none", // Mock implementation - no actual JWT signing
		},
		ScopesSupported: supportedScopes,
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_post",
			"client_secret_basic",
		},
		CodeChallengeMethodsSupported: codeChallengeMethodsSupported,
		UserInfoEndpoint:              baseURL + "/oauth2/userinfo",
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(metadata)
}

// OIDCDiscoveryRootHandler provides OpenID Connect Discovery metadata for root path.
// GET /.well-known/openid-configuration
// Spec: OpenID Connect Discovery 1.0
func OIDCDiscoveryRootHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := buildBaseURL(r)

	// Get allowed grant types from config
	allowedGrantTypes := getAllowedGrantTypes()

	// Determine which endpoints to include based on allowed grant types
	var authorizationEndpoint string
	var responseTypesSupported []string
	var codeChallengeMethodsSupported []string

	// Include authorization endpoint only if authorization_code is allowed
	for _, gt := range allowedGrantTypes {
		if gt == "authorization_code" {
			authorizationEndpoint = baseURL + "/oauth2/authorize"
			responseTypesSupported = []string{"code"}
			codeChallengeMethodsSupported = []string{"plain", "S256"}
			break
		}
	}

	// Get scopes from config, or use defaults if not configured
	supportedScopes := []string{"openid", "profile", "email"}
	if globalConfig != nil && len(globalConfig.AuthSupportedScopes) > 0 {
		supportedScopes = globalConfig.AuthSupportedScopes
	}

	// OIDC Discovery uses the same structure as OAuth2 metadata
	// but is specifically for OIDC-compliant endpoints
	discovery := OIDCDiscoveryResponse{
		Issuer:                 baseURL,
		AuthorizationEndpoint:  authorizationEndpoint,
		TokenEndpoint:          baseURL + "/oauth2/token",
		UserInfoEndpoint:       baseURL + "/oauth2/userinfo",
		JwksURI:                baseURL + "/.well-known/jwks.json",
		ResponseTypesSupported: responseTypesSupported,
		SubjectTypesSupported: []string{
			"public",
		},
		IDTokenSigningAlgValuesSupported: []string{
			"none", // Mock implementation - no actual JWT signing
		},
		ScopesSupported:               supportedScopes,
		GrantTypesSupported:           allowedGrantTypes,
		CodeChallengeMethodsSupported: codeChallengeMethodsSupported,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(discovery)
}

// OAuth2JWKSHandler returns an empty JWKS (JSON Web Key Set) for root path.
// GET /.well-known/jwks.json
// Used by both OAuth2 and OIDC discovery endpoints.
func OAuth2JWKSHandler(w http.ResponseWriter, r *http.Request) {
	// Return empty JWKS since we use alg="none" (no signature)
	jwks := JWKSResponse{
		Keys: []interface{}{},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jwks)
}
