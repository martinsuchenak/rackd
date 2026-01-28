package server

import (
	"context"
	"os"

	"github.com/martinsuchenak/rackd/internal/config"
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
			&cli.StringFlag{Name: "api-auth-token", Usage: "API authentication token"},
			&cli.StringFlag{Name: "mcp-auth-token", Usage: "MCP authentication token"},
			&cli.StringFlag{Name: "log-level", Usage: "Log level (trace/debug/info/warn/error)", DefaultValue: "info"},
			&cli.StringFlag{Name: "log-format", Usage: "Log format (text/json)", DefaultValue: "text"},
			&cli.StringFlag{Name: "discovery-interval", Usage: "Discovery scan interval", DefaultValue: "24h"},
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
			if v := cmd.GetString("api-auth-token"); v != "" {
				cfg.APIAuthToken = v
			}
			if v := cmd.GetString("mcp-auth-token"); v != "" {
				cfg.MCPAuthToken = v
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

			return server.Run(cfg, store)
		},
	}
}
