package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Host string
	Port string

	// OIDC Configuration
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

func LoadConfig() *Config {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Host: getEnv("HOST", "0.0.0.0"),
		Port: getEnv("PORT", "80"),

		// OIDC settings
		OIDCClientID:            getEnv("OIDC_CLIENT_ID", ""),
		OIDCClientSecret:        getEnv("OIDC_CLIENT_SECRET", ""),
		OIDCSupportedScopes:     parseScopes(getEnv("OIDC_SUPPORTED_SCOPES", "openid,profile,email")),
		OIDCRequirePKCE:         getBoolEnv("OIDC_REQUIRE_PKCE", false),
		OIDCSessionTTL:          getIntEnv("OIDC_SESSION_TTL", 300),
		OIDCTokenExpiry:         getIntEnv("OIDC_TOKEN_EXPIRY", 3600),
		OIDCValidateRedirectURI: getBoolEnv("OIDC_VALIDATE_REDIRECT_URI", true),
		OIDCAllowedRedirectURIs: getEnv("OIDC_ALLOWED_REDIRECT_URIS", ""),
		OIDCEnableJWTSigning:    getBoolEnv("OIDC_ENABLE_JWT_SIGNING", false),
	}
}

func (c *Config) Addr() string {
	return c.Host + ":" + c.Port
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseScopes parses comma-separated scopes into a slice of strings.
// Empty values and surrounding whitespace are trimmed.
func parseScopes(s string) []string {
	scopes := strings.Split(s, ",")
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if trimmed := strings.TrimSpace(scope); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// getBoolEnv retrieves a boolean value from environment variables.
// Returns true if the value is "true" or "1", false otherwise.
// If the environment variable is not set or empty, returns defaultValue.
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}

// getIntEnv retrieves an integer value from environment variables.
// If the environment variable is not set, empty, or cannot be parsed, returns defaultValue.
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
