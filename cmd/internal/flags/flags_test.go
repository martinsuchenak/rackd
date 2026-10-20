package flags

import (
	"context"
	"os"
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
