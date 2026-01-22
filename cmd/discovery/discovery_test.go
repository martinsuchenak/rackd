package discovery

import "testing"

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "discovery" {
		t.Errorf("expected command name 'discovery', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 3 {
		t.Errorf("expected 3 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"scan", "list", "promote"}
	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
	}
}

func TestScanCommandFlags(t *testing.T) {
	cmd := ScanCommand()

	if cmd.Name != "scan" {
		t.Errorf("expected command name 'scan', got %q", cmd.Name)
	}

	if len(cmd.Flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(cmd.Flags))
	}
}

func TestListCommandFlags(t *testing.T) {
	cmd := ListCommand()

	if cmd.Name != "list" {
		t.Errorf("expected command name 'list', got %q", cmd.Name)
	}

	if len(cmd.Flags) != 4 {
		t.Errorf("expected 4 flags, got %d", len(cmd.Flags))
	}
}

func TestPromoteCommandFlags(t *testing.T) {
	cmd := PromoteCommand()

	if cmd.Name != "promote" {
		t.Errorf("expected command name 'promote', got %q", cmd.Name)
	}

	if len(cmd.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(cmd.Flags))
	}
}
