package handlers

// globalConfig holds the global OAuth2/OIDC configuration.
// It is used by authentication handlers.
var globalConfig *Config

// Config holds the OAuth2/OIDC configuration for handlers.
type Config struct {
	// OAuth2 Configuration (shared across all flows)
	AuthAllowedClientID     string
	AuthAllowedClientSecret string
	AuthSupportedScopes     []string
	AuthTokenExpiry         int

	// Authorization Code Flow Configuration
	AuthCodeRequirePKCE         bool
	AuthCodeSessionTTL          int
	AuthCodeValidateRedirectURI bool
	AuthCodeAllowedRedirectURIs string

	// OIDC Configuration (id_token specific)
	OIDCEnableJWTSigning bool
}

// SetConfig sets the global configuration for handlers.
func SetConfig(cfg *Config) {
	globalConfig = cfg
}

// GetConfig returns the global configuration for handlers.
// This function will be used by OIDC handlers in Milestone 2 and beyond.
func GetConfig() *Config {
	return globalConfig
}
