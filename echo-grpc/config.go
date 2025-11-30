package main

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Host string
	Port string
}

func LoadConfig() *Config {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	return &Config{
		Host: getEnv("HOST", "0.0.0.0"),
		Port: getEnv("PORT", "50051"),
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
