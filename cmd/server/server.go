package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/server"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/paularlott/cli"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Start the HTTP/MCP server",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "data-dir", Usage: "Data directory", DefaultValue: "./data"},
			&cli.StringFlag{Name: "listen-addr", Usage: "Listen address", DefaultValue: ":8080"},
			&cli.StringFlag{Name: "log-level", Usage: "Log level (trace/debug/info/warn/error)", DefaultValue: "info"},
			&cli.StringFlag{Name: "log-format", Usage: "Log format (text/json)", DefaultValue: "text"},
			&cli.StringFlag{Name: "discovery-interval", Usage: "Discovery scan interval", DefaultValue: "24h"},
			&cli.BoolFlag{Name: "dev-mode", Usage: "Development mode (allows missing ENCRYPTION_KEY)"},
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

			if err := cfg.Validate(); err != nil {
				return err
			}

			log.Init(cfg.LogFormat, cfg.LogLevel, os.Stdout)

			store, err := storage.NewExtendedStorage(cfg.DataDir)
			if err != nil {
				return err
			}

			// Get encryption key for credentials (optional in dev-mode)
			devMode := cmd.GetBool("dev-mode")
			encryptionKey, hasKey := getEncryptionKey(devMode)

			// If no encryption key, run basic server without advanced features
			if !hasKey {
				log.Info("Running without credentials/scan profiles/DNS (set ENCRYPTION_KEY or use --dev-mode to enable)")
				return server.Run(cfg, store)
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

			return server.RunWithAdvancedFeatures(cfg, store, credStore, profileStore, scheduledStore, encryptionKey)
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
