// Package flags holds command-line flags shared by rackd's commands.
package flags

import (
	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/paularlott/cli"
)

// DataDir is the --data-dir flag. It has no default of its own: unset, the
// directory comes from configuration (DATA_DIR, then ./data), which a
// flag default would always override.
func DataDir() cli.Flag {
	return &cli.StringFlag{Name: "data-dir", Usage: "Data directory containing rackd.db (default: DATA_DIR, or ./data)"}
}

// ResolveDataDir returns --data-dir when it was given, and the configured
// data directory otherwise.
func ResolveDataDir(cmd *cli.Command) string {
	if cmd.HasFlag("data-dir") {
		return cmd.GetString("data-dir")
	}
	return config.Load().DataDir
}
