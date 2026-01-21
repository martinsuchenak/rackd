package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	if cfg.DataDir != "./data" {
		t.Errorf("Expected default DataDir ./data, got %s", cfg.DataDir)
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("Expected default ListenAddr :8080, got %s", cfg.ListenAddr)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("Expected default LogFormat text, got %s", cfg.LogFormat)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel info, got %s", cfg.LogLevel)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("DATA_DIR", "/test/data")
	os.Setenv("LISTEN_ADDR", ":9999")
	os.Setenv("LOG_FORMAT", "json")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DISCOVERY_ENABLED", "false")
	os.Setenv("DISCOVERY_INTERVAL", "1h")
	os.Setenv("DISCOVERY_MAX_CONCURRENT", "5")

	cfg := Load()

	if cfg.DataDir != "/test/data" {
		t.Errorf("Expected DataDir /test/data, got %s", cfg.DataDir)
	}
	if cfg.ListenAddr != ":9999" {
		t.Errorf("Expected ListenAddr :9999, got %s", cfg.ListenAddr)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("Expected LogFormat json, got %s", cfg.LogFormat)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel debug, got %s", cfg.LogLevel)
	}
	if cfg.DiscoveryEnabled != false {
		t.Errorf("Expected DiscoveryEnabled false, got %v", cfg.DiscoveryEnabled)
	}
	if cfg.DiscoveryInterval != time.Hour {
		t.Errorf("Expected DiscoveryInterval 1h, got %v", cfg.DiscoveryInterval)
	}
	if cfg.DiscoveryMaxConcurrent != 5 {
		t.Errorf("Expected DiscoveryMaxConcurrent 5, got %d", cfg.DiscoveryMaxConcurrent)
	}

	os.Unsetenv("DATA_DIR")
	os.Unsetenv("LISTEN_ADDR")
	os.Unsetenv("LOG_FORMAT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("DISCOVERY_ENABLED")
	os.Unsetenv("DISCOVERY_INTERVAL")
	os.Unsetenv("DISCOVERY_MAX_CONCURRENT")
}

func TestGetIntEnv(t *testing.T) {
	os.Clearenv()
	if result := getIntEnv("NONEXISTENT", 42); result != 42 {
		t.Errorf("Expected default value 42, got %d", result)
	}

	os.Setenv("TEST_INT", "100")
	if result := getIntEnv("TEST_INT", 42); result != 100 {
		t.Errorf("Expected 100, got %d", result)
	}

	os.Setenv("TEST_INT_INVALID", "notanumber")
	if result := getIntEnv("TEST_INT_INVALID", 42); result != 42 {
		t.Errorf("Expected default 42 for invalid input, got %d", result)
	}
	os.Unsetenv("TEST_INT")
	os.Unsetenv("TEST_INT_INVALID")
}

func TestGetBoolEnv(t *testing.T) {
	os.Clearenv()
	if result := getBoolEnv("NONEXISTENT", true); result != true {
		t.Errorf("Expected default true, got %v", result)
	}

	os.Setenv("TEST_BOOL", "true")
	if result := getBoolEnv("TEST_BOOL", false); result != true {
		t.Errorf("Expected true, got %v", result)
	}

	os.Setenv("TEST_BOOL", "false")
	if result := getBoolEnv("TEST_BOOL", true); result != false {
		t.Errorf("Expected false, got %v", result)
	}

	os.Setenv("TEST_BOOL_INVALID", "notabool")
	if result := getBoolEnv("TEST_BOOL_INVALID", true); result != true {
		t.Errorf("Expected default true for invalid input, got %v", result)
	}
	os.Unsetenv("TEST_BOOL")
	os.Unsetenv("TEST_BOOL_INVALID")
}

func TestGetDurationEnv(t *testing.T) {
	os.Clearenv()
	defaultDuration := 30 * time.Minute
	if result := getDurationEnv("NONEXISTENT", defaultDuration); result != defaultDuration {
		t.Errorf("Expected default %v, got %v", defaultDuration, result)
	}

	os.Setenv("TEST_DURATION", "5s")
	if result := getDurationEnv("TEST_DURATION", defaultDuration); result != 5*time.Second {
		t.Errorf("Expected 5s, got %v", result)
	}

	os.Setenv("TEST_DURATION_INVALID", "notaduration")
	if result := getDurationEnv("TEST_DURATION_INVALID", defaultDuration); result != defaultDuration {
		t.Errorf("Expected default for invalid input, got %v", result)
	}
	os.Unsetenv("TEST_DURATION")
	os.Unsetenv("TEST_DURATION_INVALID")
}

func TestValidate(t *testing.T) {
	os.Clearenv()
	os.Setenv("LOG_LEVEL", "invalid")
	cfg := Load()

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid log level, got nil")
	}
	if !strings.Contains(err.Error(), "invalid LOG_LEVEL") {
		t.Errorf("Expected error message to mention invalid log level, got: %v", err)
	}
	os.Unsetenv("LOG_LEVEL")

	os.Clearenv()
	os.Setenv("LOG_FORMAT", "invalid")
	cfg = Load()

	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for invalid log format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid LOG_FORMAT") {
		t.Errorf("Expected error message to mention invalid log format, got: %v", err)
	}
	os.Unsetenv("LOG_FORMAT")

	os.Clearenv()
	os.Setenv("DISCOVERY_INTERVAL", "-1h")
	cfg = Load()

	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for negative interval, got nil")
	}
	if !strings.Contains(err.Error(), "DISCOVERY_INTERVAL") {
		t.Errorf("Expected error message to mention interval, got: %v", err)
	}
	os.Unsetenv("DISCOVERY_INTERVAL")

	os.Clearenv()
	os.Setenv("DISCOVERY_MAX_CONCURRENT", "0")
	cfg = Load()

	err = cfg.Validate()
	if err == nil {
		t.Error("Expected error for zero max concurrent, got nil")
	}
	if !strings.Contains(err.Error(), "DISCOVERY_MAX_CONCURRENT") {
		t.Errorf("Expected error message to mention max concurrent, got: %v", err)
	}
	os.Unsetenv("DISCOVERY_MAX_CONCURRENT")

	os.Clearenv()
	cfg = Load()

	err = cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}
}

func TestConfigStringRedaction(t *testing.T) {
	os.Clearenv()
	cfg := Load()

	str := cfg.String()

	if strings.Contains(str, "***REDACTED***") {
		t.Error("String() should not contain redaction markers for empty secrets")
	}
	if !strings.Contains(str, "(empty)") {
		t.Error("String() should contain (empty) for empty secrets")
	}

	os.Clearenv()
	os.Setenv("API_AUTH_TOKEN", "secret123")
	os.Setenv("MCP_AUTH_TOKEN", "mcp-secret")

	cfg = Load()
	str = cfg.String()

	if strings.Contains(str, "secret123") {
		t.Error("String() should not contain actual API token")
	}
	if strings.Contains(str, "mcp-secret") {
		t.Error("String() should not contain actual MCP token")
	}
	if !strings.Contains(str, "***REDACTED***") {
		t.Error("String() should contain redaction markers for secrets")
	}

	os.Unsetenv("API_AUTH_TOKEN")
	os.Unsetenv("MCP_AUTH_TOKEN")
}
