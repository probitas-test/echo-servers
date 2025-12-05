package main

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Host                     string
	Port                     string
	DisableConnectRPC        bool
	DisableGRPC              bool
	DisableGRPCWeb           bool
	ReflectionIncludeDeps    bool
	DisableReflectionV1      bool
	DisableReflectionV1Alpha bool
}

func LoadConfig() *Config {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Host:                     getEnv("HOST", "0.0.0.0"),
		Port:                     getEnv("PORT", "8080"),
		DisableConnectRPC:        getEnvBool("DISABLE_CONNECTRPC", false),
		DisableGRPC:              getEnvBool("DISABLE_GRPC", false),
		DisableGRPCWeb:           getEnvBool("DISABLE_GRPC_WEB", false),
		ReflectionIncludeDeps:    getEnvBool("REFLECTION_INCLUDE_DEPENDENCIES", false),
		DisableReflectionV1:      getEnvBool("DISABLE_REFLECTION_V1", false),
		DisableReflectionV1Alpha: getEnvBool("DISABLE_REFLECTION_V1ALPHA", false),
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
