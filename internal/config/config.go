package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/paularlott/cli/env"
)

type Config struct {
	DataDir                 string
	PreMigrationBackup      bool
	ListenAddr              string
	RequestTimeout          time.Duration
	LogFormat               string
	LogLevel                string
	DiscoveryInterval       time.Duration
	DiscoveryCleanupDays    int
	DiscoveryScanOnStartup  bool
	DiscoverySNMPv2cEnabled bool
	RateLimitEnabled        bool
	RateLimitRequests       int
	RateLimitWindow         time.Duration
	AuditEnabled            bool
	AuditRetentionDays      int
	SessionTTL              time.Duration
	SessionMaxLifetime      time.Duration
	SessionStoreType        string
	ValkeyURL               string
	LoginRateLimitRequests  int
	LoginRateLimitWindow    time.Duration
	LoginAccountMaxFailures int
	LoginAccountWindow      time.Duration
	CookieSecure            bool
	TrustProxy              bool
	TrustedProxies          string
	InitialAdminUsername    string
	InitialAdminPassword    string
	InitialAdminEmail       string
	InitialAdminFullName    string

	// OAuth 2.1 for MCP
	MCPOAuthEnabled         bool
	MCPOAuthIssuerURL       string
	MCPOAuthAccessTokenTTL  time.Duration
	MCPOAuthRefreshTokenTTL time.Duration

	// Utilization snapshots
	SnapshotInterval      time.Duration
	SnapshotRetentionDays int
	// PoolUtilizationAlertPercent is the pool utilization at which
	// pool.utilization_high is published; 0 turns it off.
	PoolUtilizationAlertPercent float64

	// DNS sync
	DNSSyncInterval time.Duration

	// SSRF policy for outbound integrations (webhooks, DNS providers).
	// When false (default), loopback, link-local, multicast and private
	// ranges (RFC1918/CGNAT/ULA) are rejected. Intentionally internal
	// integrations can be enabled explicitly.
	SSRFAllowPrivateTargets bool
	// SSRFAllowedTargets lists private IPs or CIDR ranges outbound requests
	// may reach without allowing every private range (comma-separated).
	SSRFAllowedTargets string
}

var cfg Config

func Load() *Config {
	_ = env.Load()

	cfg = Config{
		DataDir:                 getEnv("DATA_DIR", "./data"),
		PreMigrationBackup:      getBoolEnv("PRE_MIGRATION_BACKUP", true),
		ListenAddr:              getEnv("LISTEN_ADDR", ":8080"),
		RequestTimeout:          getDurationEnv("REQUEST_TIMEOUT", 30*time.Second),
		LogFormat:               getEnv("LOG_FORMAT", "text"),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		DiscoveryInterval:       getDurationEnv("DISCOVERY_INTERVAL", 24*time.Hour),
		DiscoveryCleanupDays:    getIntEnv("DISCOVERY_CLEANUP_DAYS", 30),
		DiscoveryScanOnStartup:  getBoolEnv("DISCOVERY_SCAN_ON_STARTUP", false),
		DiscoverySNMPv2cEnabled: getBoolEnv("DISCOVERY_SNMPV2C_ENABLED", false),
		RateLimitEnabled:        getBoolEnv("RATE_LIMIT_ENABLED", true),
		RateLimitRequests:       getIntEnv("RATE_LIMIT_REQUESTS", 100),
		RateLimitWindow:         getDurationEnv("RATE_LIMIT_WINDOW", 1*time.Minute),
		AuditEnabled:            getBoolEnv("AUDIT_ENABLED", false),
		AuditRetentionDays:      getIntEnv("AUDIT_RETENTION_DAYS", 90),
		SessionTTL:              getDurationEnv("SESSION_TTL", 24*time.Hour),
		SessionMaxLifetime:      getDurationEnv("SESSION_MAX_LIFETIME", 7*24*time.Hour),
		SessionStoreType:        getEnv("SESSION_STORE_TYPE", "sqlite"),
		ValkeyURL:               getEnv("VALKEY_URL", "redis://localhost:6379/0"),
		LoginRateLimitRequests:  getIntEnv("LOGIN_RATE_LIMIT_REQUESTS", 5),
		LoginRateLimitWindow:    getDurationEnv("LOGIN_RATE_LIMIT_WINDOW", 1*time.Minute),
		LoginAccountMaxFailures: getIntEnv("LOGIN_ACCOUNT_MAX_FAILURES", 10),
		LoginAccountWindow:      getDurationEnv("LOGIN_ACCOUNT_WINDOW", 15*time.Minute),
		CookieSecure:            getBoolEnv("COOKIE_SECURE", true),
		TrustProxy:              getBoolEnv("TRUST_PROXY", false),
		TrustedProxies:          getEnv("TRUSTED_PROXIES", ""),
		InitialAdminUsername:    getEnv("INITIAL_ADMIN_USERNAME", ""),
		InitialAdminPassword:    getEnv("INITIAL_ADMIN_PASSWORD", ""),
		InitialAdminEmail:       getEnv("INITIAL_ADMIN_EMAIL", "admin@localhost"),
		InitialAdminFullName:    getEnv("INITIAL_ADMIN_FULL_NAME", "System Administrator"),

		MCPOAuthEnabled:         getBoolEnv("MCP_OAUTH_ENABLED", false),
		MCPOAuthIssuerURL:       getEnv("MCP_OAUTH_ISSUER_URL", ""),
		MCPOAuthAccessTokenTTL:  getDurationEnv("MCP_OAUTH_ACCESS_TOKEN_TTL", 1*time.Hour),
		MCPOAuthRefreshTokenTTL: getDurationEnv("MCP_OAUTH_REFRESH_TOKEN_TTL", 30*24*time.Hour),

		SnapshotInterval:            getDurationEnv("SNAPSHOT_INTERVAL", 1*time.Hour),
		SnapshotRetentionDays:       getIntEnv("SNAPSHOT_RETENTION_DAYS", 90),
		PoolUtilizationAlertPercent: float64(getIntEnv("POOL_UTILIZATION_ALERT_PERCENT", 90)),

		DNSSyncInterval: getDurationEnv("DNS_SYNC_INTERVAL", 1*time.Hour),

		SSRFAllowPrivateTargets: getBoolEnv("SSRF_ALLOW_PRIVATE_TARGETS", false),
		SSRFAllowedTargets:      getEnv("SSRF_ALLOWED_TARGETS", ""),
	}

	return &cfg
}

func (c *Config) Validate() error {
	validLogLevels := map[string]bool{
		"trace": true,
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid LOG_LEVEL: %s (must be trace, debug, info, warn, or error)", c.LogLevel)
	}

	if c.LogFormat != "text" && c.LogFormat != "json" {
		return fmt.Errorf("invalid LOG_FORMAT: %s (must be text or json)", c.LogFormat)
	}

	if c.DiscoveryInterval <= 0 {
		return fmt.Errorf("DISCOVERY_INTERVAL must be positive, got %v", c.DiscoveryInterval)
	}

	if c.DiscoveryCleanupDays <= 0 {
		return fmt.Errorf("DISCOVERY_CLEANUP_DAYS must be positive, got %d", c.DiscoveryCleanupDays)
	}

	if c.RateLimitEnabled {
		if c.RateLimitRequests <= 0 {
			return fmt.Errorf("RATE_LIMIT_REQUESTS must be positive, got %d", c.RateLimitRequests)
		}
		if c.RateLimitWindow <= 0 {
			return fmt.Errorf("RATE_LIMIT_WINDOW must be positive, got %v", c.RateLimitWindow)
		}
	}

	if c.AuditRetentionDays < 0 {
		return fmt.Errorf("AUDIT_RETENTION_DAYS must be 0 (keep everything) or positive, got %d", c.AuditRetentionDays)
	}

	if c.MCPOAuthEnabled && c.MCPOAuthIssuerURL == "" {
		return fmt.Errorf("MCP_OAUTH_ISSUER_URL is required when MCP_OAUTH_ENABLED is true")
	}

	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf("Config{DataDir:%s, ListenAddr:%s, LogFormat:%s, LogLevel:%s, DiscoveryInterval:%v, DiscoveryCleanupDays:%d, DiscoveryScanOnStartup:%v, DiscoverySNMPv2cEnabled:%v, RateLimitEnabled:%v, RateLimitRequests:%d, RateLimitWindow:%v}",
		c.DataDir,
		c.ListenAddr,
		c.LogFormat,
		c.LogLevel,
		c.DiscoveryInterval,
		c.DiscoveryCleanupDays,
		c.DiscoveryScanOnStartup,
		c.DiscoverySNMPv2cEnabled,
		c.RateLimitEnabled,
		c.RateLimitRequests,
		c.RateLimitWindow,
	)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if result, err := strconv.ParseBool(value); err == nil {
			return result
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if result, err := time.ParseDuration(value); err == nil {
			return result
		}
	}
	return defaultValue
}
