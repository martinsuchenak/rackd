package backup

import "testing"

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "backup" {
		t.Errorf("expected command name 'backup', got %q", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
	if len(cmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags (data-dir, output), got %d", len(cmd.Flags))
	}

	subs := map[string]bool{}
	for _, c := range cmd.Commands {
		subs[c.Name] = c.Run != nil
	}
	for _, name := range []string{"verify", "restore"} {
		if !subs[name] {
			t.Errorf("missing runnable subcommand %q", name)
		}
	}
}
