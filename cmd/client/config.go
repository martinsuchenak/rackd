package client

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	Timeout   string `json:"timeout"`
	Output    string `json:"output"`
	VerifySSL bool   `json:"verify_ssl"`
}

var defaultConfig = Config{
	ServerURL: "http://localhost:8080",
	Timeout:   "30s",
	Output:    "table",
	VerifySSL: true,
}

func LoadConfig() *Config {
	cfg := defaultConfig
	configPath := filepath.Join(getConfigDir(), "config.json")

	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &cfg)
	}

	if url := os.Getenv("RACKD_SERVER_URL"); url != "" {
		cfg.ServerURL = url
	}
	if token := os.Getenv("RACKD_TOKEN"); token != "" {
		cfg.Token = token
	}
	if v := os.Getenv("RACKD_VERIFY_SSL"); v == "false" || v == "0" {
		cfg.VerifySSL = false
	}

	return &cfg
}

func getConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "rackd")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "rackd")
}

func (c *Config) GetTimeout() time.Duration {
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}
