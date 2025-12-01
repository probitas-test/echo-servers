package main

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Host                  string
	Port                  string
	ReflectionIncludeDeps bool
}

func LoadConfig() *Config {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Host:                  getEnv("HOST", "0.0.0.0"),
		Port:                  getEnv("PORT", "50051"),
		ReflectionIncludeDeps: getEnvBool("REFLECTION_INCLUDE_DEPENDENCIES", false),
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

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	switch value {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		return defaultValue
	}
}
