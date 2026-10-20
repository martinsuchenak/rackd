package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/martinsuchenak/rackd/cmd/internal/flags"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/server"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/paularlott/cli"
)

// Command returns the server command with no extensions.
func Command() *cli.Command {
	return NewCommand(nil, "", nil)
}

// NewCommand returns the server command, running features as extensions,
// reporting edition in /api/v1/config (empty means "oss") and serving ui as the
// web UI (nil means the built-in one).
func NewCommand(features []extension.Feature, edition string, ui fs.FS) *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Start the HTTP/MCP server",
		Flags: []cli.Flag{
			// No flag has a default of its own: an unset flag leaves the
			// configuration (environment, .env, then built-in defaults) in
			// charge, which a flag default would always override.
			flags.DataDir(),
			&cli.StringFlag{Name: "listen-addr", Usage: "Listen address (default: LISTEN_ADDR, or :8080)"},
			&cli.StringFlag{Name: "log-level", Usage: "Log level: trace, debug, info, warn or error (default: LOG_LEVEL, or info)"},
			&cli.StringFlag{Name: "log-format", Usage: "Log format: text or json (default: LOG_FORMAT, or text)"},
			&cli.StringFlag{Name: "discovery-interval", Usage: "Discovery scan interval (default: DISCOVERY_INTERVAL, or 24h)"},
			&cli.BoolFlag{Name: "dev-mode", Usage: "Development mode (relaxes security: no TLS cookies, no rate limiting, allows missing ENCRYPTION_KEY)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.Load()

			// Flags that were given override the configuration.
			if cmd.HasFlag("data-dir") {
				cfg.DataDir = cmd.GetString("data-dir")
			}
			if cmd.HasFlag("listen-addr") {
				cfg.ListenAddr = cmd.GetString("listen-addr")
			}
			if cmd.HasFlag("log-level") {
				cfg.LogLevel = cmd.GetString("log-level")
			}
			if cmd.HasFlag("log-format") {
				cfg.LogFormat = cmd.GetString("log-format")
			}
			if cmd.HasFlag("discovery-interval") {
				d, err := time.ParseDuration(cmd.GetString("discovery-interval"))
				if err != nil {
					return fmt.Errorf("--discovery-interval: %w", err)
				}
				cfg.DiscoveryInterval = d
			}

			// Dev mode: relax security defaults for local development
			devMode := cmd.GetBool("dev-mode")
			if devMode {
				cfg.CookieSecure = false
				cfg.RateLimitEnabled = false
				log.Init(cfg.LogFormat, cfg.LogLevel, os.Stdout)
				log.Warn("Development mode enabled: cookie secure=false, rate limiting disabled")
			}

			if err := cfg.Validate(); err != nil {
				return err
			}
			if err := flags.CheckDataDirMove(cmd, cfg.DataDir); err != nil {
				return err
			}

			// Configure the shared SSRF guard before any outbound
			// integration request can be issued.
			webhook.SetPrivateRangePolicy(cfg.SSRFAllowPrivateTargets)
			allowedTargets, err := webhook.ParseAllowedTargets(strings.Split(cfg.SSRFAllowedTargets, ","))
			if err != nil {
				return fmt.Errorf("SSRF_ALLOWED_TARGETS: %w", err)
			}
			webhook.SetAllowedPrivateTargets(allowedTargets)

			if !devMode {
				log.Init(cfg.LogFormat, cfg.LogLevel, os.Stdout)
			}

			store, err := storage.NewExtendedStorageWithOptions(cfg.DataDir, storage.StorageOptions{SkipPreMigrationBackup: !cfg.PreMigrationBackup})
			if err != nil {
				return err
			}
			// Closing the store after the server stops flushes queued audit entries.
			defer func() {
				if err := store.Close(); err != nil {
					log.Error("Failed to close storage", "error", err)
				}
			}()

			ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			opts := server.Options{Config: cfg, Store: store, Features: features, Edition: edition, UI: ui}
			encryptionKey, err := getEncryptionKey(devMode)
			if err != nil {
				return err
			}

			// If no encryption key, run basic server without advanced features
			if encryptionKey == nil {
				log.Warn("ENCRYPTION_KEY is not set: credentials, scan profiles and DNS are disabled, and webhook secrets are stored unencrypted")
				return server.Run(ctx, opts)
			}

			// Initialize credentials storage
			credStore, err := credentials.NewSQLiteStorage(store.DB(), encryptionKey)
			if err != nil {
				return fmt.Errorf("failed to initialize credentials storage: %w", err)
			}

			// Initialize profiles storage
			profileStore, err := storage.NewSQLiteProfileStorage(store.DB())
			if err != nil {
				return fmt.Errorf("failed to initialize profiles storage: %w", err)
			}

			// Initialize scheduled scans storage
			scheduledStore, err := storage.NewSQLiteScheduledScanStorage(store.DB())
			if err != nil {
				return fmt.Errorf("failed to initialize scheduled scans storage: %w", err)
			}

			opts.Credentials = credStore
			opts.Profiles = profileStore
			opts.ScheduledScans = scheduledStore
			opts.EncryptionKey = encryptionKey
			return server.Run(ctx, opts)
		},
	}
}

// getEncryptionKey returns the key from ENCRYPTION_KEY, a random one in dev
// mode when it is unset, or nil when it is unset otherwise. A key that is set
// but malformed is an error: running without encryption the operator asked
// for would store secrets in plaintext.
func getEncryptionKey(devMode bool) ([]byte, error) {
	keyHex := os.Getenv("ENCRYPTION_KEY")
	if keyHex == "" {
		if !devMode {
			return nil, nil
		}
		fmt.Fprintln(os.Stderr, "Warning: ENCRYPTION_KEY not set - generating random key (credentials will not persist across restarts)")
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		return key, nil
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("ENCRYPTION_KEY must be 64 hex characters (32 bytes), such as the output of: openssl rand -hex 32")
	}
	return key, nil
}
