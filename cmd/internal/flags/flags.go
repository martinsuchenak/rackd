// Package flags holds command-line flags shared by rackd's commands.
package flags

import (
	"fmt"
	"os"
	"path/filepath"

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

// legacyDataDir is where rackd before v0.6.0 kept its database whenever
// --data-dir wasn't given, whatever DATA_DIR said.
const legacyDataDir = "./data"

// CheckDataDirMove refuses to start on an empty data directory chosen by
// DATA_DIR when ./data holds a database: before v0.6.0 that variable was
// ignored, so the database there is most likely the real one, and starting
// would create an empty inventory instead.
func CheckDataDirMove(cmd *cli.Command, dataDir string) error {
	if cmd.HasFlag("data-dir") || os.Getenv("DATA_DIR") == "" {
		return nil
	}
	want, err1 := filepath.Abs(dataDir)
	legacy, err2 := filepath.Abs(legacyDataDir)
	if err1 != nil || err2 != nil || want == legacy {
		return nil
	}
	if _, err := os.Stat(filepath.Join(want, "rackd.db")); err == nil {
		return nil
	}
	if _, err := os.Stat(filepath.Join(legacy, "rackd.db")); err != nil {
		return nil
	}
	return fmt.Errorf("DATA_DIR is %s, which has no database, but %s does: rackd before v0.6.0 ignored DATA_DIR and used ./data. "+
		"Move %s to %s, or set DATA_DIR=./data (or pass --data-dir ./data) to keep using it",
		want, filepath.Join(legacy, "rackd.db"), legacy, want)
}
