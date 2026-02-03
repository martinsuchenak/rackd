package server

import (
	"testing"
)

func TestCommand(t *testing.T) {
	cmd := Command()

	if cmd.Name != "server" {
		t.Errorf("expected name 'server', got %s", cmd.Name)
	}

	if cmd.Run == nil {
		t.Error("expected Run function to be set")
	}

	if len(cmd.Flags) != 6 {
		t.Errorf("expected 6 flags, got %d", len(cmd.Flags))
	}
}
