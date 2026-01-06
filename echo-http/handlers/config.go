package handlers

// globalConfig holds the global OIDC configuration.
// It will be used by OIDC handlers in Milestone 2 and beyond.
var globalConfig *Config

// Config holds the OIDC configuration for handlers.
type Config struct {
	OIDCClientID            string
	OIDCClientSecret        string
	OIDCSupportedScopes     []string
	OIDCRequirePKCE         bool
	OIDCSessionTTL          int
	OIDCTokenExpiry         int
	OIDCValidateRedirectURI bool
	OIDCAllowedRedirectURIs string
	OIDCEnableJWTSigning    bool
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
