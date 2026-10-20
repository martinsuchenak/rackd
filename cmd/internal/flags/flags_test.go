package flags

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paularlott/cli"
)

// An unset --data-dir leaves DATA_DIR in charge: the flag used to default to
// ./data, which always overrode it.
func TestDataDirFallsBackToConfig(t *testing.T) {
	t.Setenv("DATA_DIR", "/srv/rackd-from-env")
	run := func(args ...string) string {
		var got string
		cmd := &cli.Command{
			Name:  "rackd",
			Flags: []cli.Flag{DataDir()},
			Run: func(_ context.Context, cmd *cli.Command) error {
				got = ResolveDataDir(cmd)
				return nil
			},
		}
		saved := os.Args
		os.Args = append([]string{"rackd"}, args...)
		defer func() { os.Args = saved }()
		if err := cmd.Execute(context.Background()); err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := run(); got != "/srv/rackd-from-env" {
		t.Errorf("no flag: %q, want DATA_DIR", got)
	}
	if got := run("--data-dir", "/flag/wins"); got != "/flag/wins" {
		t.Errorf("with the flag: %q, want the flag", got)
	}
}

// A DATA_DIR that was ignored until now must not quietly start an empty
// inventory beside the real database in ./data.
func TestCheckDataDirMove(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)
	if err := os.MkdirAll("data", 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := &cli.Command{Name: "rackd", Flags: []cli.Flag{DataDir()}}
	target := filepath.Join(work, "moved")

	t.Setenv("DATA_DIR", target)
	if err := CheckDataDirMove(cmd, target); err != nil {
		t.Errorf("no database anywhere (a new install): %v", err)
	}
	if err := os.WriteFile(filepath.Join("data", "rackd.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckDataDirMove(cmd, target); err == nil || !strings.Contains(err.Error(), "ignored DATA_DIR") {
		t.Errorf("database only in ./data: %v, want a refusal", err)
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "rackd.db"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CheckDataDirMove(cmd, target); err != nil {
		t.Errorf("database moved: %v", err)
	}
	t.Setenv("DATA_DIR", "./data")
	if err := CheckDataDirMove(cmd, "./data"); err != nil {
		t.Errorf("DATA_DIR=./data: %v", err)
	}
}
