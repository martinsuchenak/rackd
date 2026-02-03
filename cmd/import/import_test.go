package importcmd

import (
	"testing"

	"github.com/paularlott/cli"
)

func TestCommand(t *testing.T) {
	cmd := Command()
	if cmd == nil {
		t.Fatal("Command() returned nil")
	}
	if cmd.Name != "import" {
		t.Errorf("Name = %v, want import", cmd.Name)
	}
	if len(cmd.Commands) != 3 {
		t.Errorf("expected 3 subcommands, got %d", len(cmd.Commands))
	}
}

func TestDevicesCommand(t *testing.T) {
	cmd := DevicesCommand()
	if cmd == nil {
		t.Fatal("DevicesCommand() returned nil")
	}
	if cmd.Name != "devices" {
		t.Errorf("Name = %v, want devices", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
	if len(cmd.Flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(cmd.Flags))
	}

	hasFile := false
	for _, flag := range cmd.Flags {
		if sf, ok := flag.(*cli.StringFlag); ok && sf.Name == "file" {
			hasFile = true
			if !sf.Required {
				t.Error("file flag should be required")
			}
		}
	}
	if !hasFile {
		t.Error("expected file flag")
	}
}

func TestNetworksCommand(t *testing.T) {
	cmd := NetworksCommand()
	if cmd == nil {
		t.Fatal("NetworksCommand() returned nil")
	}
	if cmd.Name != "networks" {
		t.Errorf("Name = %v, want networks", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
	if len(cmd.Flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(cmd.Flags))
	}
}

func TestDatacentersCommand(t *testing.T) {
	cmd := DatacentersCommand()
	if cmd == nil {
		t.Fatal("DatacentersCommand() returned nil")
	}
	if cmd.Name != "datacenters" {
		t.Errorf("Name = %v, want datacenters", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
	if len(cmd.Flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(cmd.Flags))
	}
}
