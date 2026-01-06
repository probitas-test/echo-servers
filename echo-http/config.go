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

	// OAuth2 Configuration (shared across all flows)
	AuthAllowedClientID     string
	AuthAllowedClientSecret string
	AuthSupportedScopes     []string
	AuthTokenExpiry         int
	AuthAllowedGrantTypes   []string

	// Resource Owner Password Credentials / Basic Auth
	AuthAllowedUsername string
	AuthAllowedPassword string

	// Authorization Code Flow Configuration
	AuthCodeRequirePKCE         bool
	AuthCodeSessionTTL          int
	AuthCodeValidateRedirectURI bool
	AuthCodeAllowedRedirectURIs string
}

func LoadConfig() *Config {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Host: getEnv("HOST", "0.0.0.0"),
		Port: getEnv("PORT", "80"),

		// OAuth2 settings (shared across all flows)
		AuthAllowedClientID:     getEnv("AUTH_ALLOWED_CLIENT_ID", ""),
		AuthAllowedClientSecret: getEnv("AUTH_ALLOWED_CLIENT_SECRET", ""),
		AuthSupportedScopes:     parseScopes(getEnv("AUTH_SUPPORTED_SCOPES", "openid,profile,email")),
		AuthTokenExpiry:         getIntEnv("AUTH_TOKEN_EXPIRY", 3600),
		AuthAllowedGrantTypes:   parseGrantTypes(getEnv("AUTH_ALLOWED_GRANT_TYPES", "authorization_code,client_credentials,password,refresh_token")),

		// Resource Owner Password Credentials / Basic Auth settings
		AuthAllowedUsername: getEnv("AUTH_ALLOWED_USERNAME", "testuser"),
		AuthAllowedPassword: getEnv("AUTH_ALLOWED_PASSWORD", "testpass"),

		// Authorization Code Flow settings
		AuthCodeRequirePKCE:         getBoolEnv("AUTH_CODE_REQUIRE_PKCE", false),
		AuthCodeSessionTTL:          getIntEnv("AUTH_CODE_SESSION_TTL", 300),
		AuthCodeValidateRedirectURI: getBoolEnv("AUTH_CODE_VALIDATE_REDIRECT_URI", false),
		AuthCodeAllowedRedirectURIs: getEnv("AUTH_CODE_ALLOWED_REDIRECT_URIS", ""),
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

// parseGrantTypes parses comma-separated grant types into a slice of strings.
// Empty values and surrounding whitespace are trimmed.
func parseGrantTypes(s string) []string {
	grantTypes := strings.Split(s, ",")
	result := make([]string, 0, len(grantTypes))
	for _, grantType := range grantTypes {
		if trimmed := strings.TrimSpace(grantType); trimmed != "" {
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
