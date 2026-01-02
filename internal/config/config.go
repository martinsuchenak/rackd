package config

import (
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	DataDir      string
	ListenAddr   string
	BearerToken  string
	StorageFormat string // "json" or "toml"
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	cfg := &Config{
		DataDir:       getEnv("DM_DATA_DIR", "./data"),
		ListenAddr:    getEnv("DM_LISTEN_ADDR", ":8080"),
		BearerToken:   getEnv("DM_BEARER_TOKEN", ""),
		StorageFormat: getEnv("DM_STORAGE_FORMAT", "json"),
	}

	// Validate storage format
	if cfg.StorageFormat != "json" && cfg.StorageFormat != "toml" {
		cfg.StorageFormat = "json"
	}

	return cfg
}

// IsMCPEnabled checks if MCP authentication is configured
func (c *Config) IsMCPEnabled() bool {
	return c.BearerToken != ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
