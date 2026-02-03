package export

import (
	"testing"

	"github.com/paularlott/cli"
)

func TestCommand(t *testing.T) {
	cmd := Command()
	if cmd == nil {
		t.Fatal("Command() returned nil")
	}
	if cmd.Name != "export" {
		t.Errorf("Name = %v, want export", cmd.Name)
	}
	if len(cmd.Commands) != 4 {
		t.Errorf("expected 4 subcommands, got %d", len(cmd.Commands))
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
	if len(cmd.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(cmd.Flags))
	}

	hasFormat := false
	for _, flag := range cmd.Flags {
		if sf, ok := flag.(*cli.StringFlag); ok && sf.Name == "format" {
			hasFormat = true
			if sf.DefaultValue != "json" {
				t.Errorf("format default = %v, want json", sf.DefaultValue)
			}
		}
	}
	if !hasFormat {
		t.Error("expected format flag")
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
	if len(cmd.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(cmd.Flags))
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
	if len(cmd.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(cmd.Flags))
	}
}

func TestAllCommand(t *testing.T) {
	cmd := AllCommand()
	if cmd == nil {
		t.Fatal("AllCommand() returned nil")
	}
	if cmd.Name != "all" {
		t.Errorf("Name = %v, want all", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
	if len(cmd.Flags) != 2 {
		t.Errorf("expected 2 flags, got %d", len(cmd.Flags))
	}
}
