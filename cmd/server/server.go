package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

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
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory", DefaultValue: "./data"},
			&cli.StringFlag{Name: "listen-addr", Usage: "Listen address", DefaultValue: ":8080"},
			&cli.StringFlag{Name: "log-level", Usage: "Log level (trace/debug/info/warn/error)", DefaultValue: "info"},
			&cli.StringFlag{Name: "log-format", Usage: "Log format (text/json)", DefaultValue: "text"},
			&cli.StringFlag{Name: "discovery-interval", Usage: "Discovery scan interval", DefaultValue: "24h"},
			&cli.BoolFlag{Name: "dev-mode", Usage: "Development mode (relaxes security: no TLS cookies, no rate limiting, allows missing ENCRYPTION_KEY)"},
		},
		Run: func(ctx context.Context, cmd *cli.Command) error {
			cfg := config.Load()

			// Override config with CLI flags
			if v := cmd.GetString("data-dir"); v != "" {
				cfg.DataDir = v
			}
			if v := cmd.GetString("listen-addr"); v != "" {
				cfg.ListenAddr = v
			}
			if v := cmd.GetString("log-level"); v != "" {
				cfg.LogLevel = v
			}
			if v := cmd.GetString("log-format"); v != "" {
				cfg.LogFormat = v
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

			// Configure the shared SSRF guard before any outbound
			// integration request can be issued.
			webhook.SetPrivateRangePolicy(cfg.SSRFAllowPrivateTargets)

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
			encryptionKey, hasKey := getEncryptionKey(devMode)

			// If no encryption key, run basic server without advanced features
			if !hasKey {
				log.Info("Running without credentials/scan profiles/DNS (set ENCRYPTION_KEY or use --dev-mode to enable)")
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

func getEncryptionKey(devMode bool) ([]byte, bool) {
	keyHex := os.Getenv("ENCRYPTION_KEY")
	if keyHex == "" {
		if !devMode {
			return nil, false
		}
		fmt.Fprintln(os.Stderr, "Warning: ENCRYPTION_KEY not set - generating random key (credentials will not persist across restarts)")
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, false
		}
		return key, true
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: invalid ENCRYPTION_KEY (must be hex-encoded): %v\n", err)
		return nil, false
	}
	if len(key) != 32 {
		fmt.Fprintln(os.Stderr, "Warning: invalid ENCRYPTION_KEY (must be 32 bytes / 64 hex chars)")
		return nil, false
	}
	return key, true
}
