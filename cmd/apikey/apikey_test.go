package apikey

import (
	"testing"

	"github.com/paularlott/cli"
)

func TestCommand(t *testing.T) {
	cmd := Command()
	if cmd == nil {
		t.Fatal("Command() returned nil")
	}
	if cmd.Name != "apikey" {
		t.Errorf("Name = %v, want apikey", cmd.Name)
	}
	if len(cmd.Commands) != 4 {
		t.Errorf("expected 4 subcommands, got %d", len(cmd.Commands))
	}
}

func TestListCommand(t *testing.T) {
	cmd := ListCommand()
	if cmd == nil {
		t.Fatal("ListCommand() returned nil")
	}
	if cmd.Name != "list" {
		t.Errorf("Name = %v, want list", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
}

func TestCreateCommand(t *testing.T) {
	cmd := CreateCommand()
	if cmd == nil {
		t.Fatal("CreateCommand() returned nil")
	}
	if cmd.Name != "create" {
		t.Errorf("Name = %v, want create", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
	if len(cmd.Flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(cmd.Flags))
	}

	hasName := false
	for _, flag := range cmd.Flags {
		if sf, ok := flag.(*cli.StringFlag); ok && sf.Name == "name" {
			hasName = true
			if !sf.Required {
				t.Error("name flag should be required")
			}
		}
	}
	if !hasName {
		t.Error("expected name flag")
	}
}

func TestDeleteCommand(t *testing.T) {
	cmd := DeleteCommand()
	if cmd == nil {
		t.Fatal("DeleteCommand() returned nil")
	}
	if cmd.Name != "delete" {
		t.Errorf("Name = %v, want delete", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
	if len(cmd.Flags) != 1 {
		t.Errorf("expected 1 flag, got %d", len(cmd.Flags))
	}
}

func TestGenerateCommand(t *testing.T) {
	cmd := GenerateCommand()
	if cmd == nil {
		t.Fatal("GenerateCommand() returned nil")
	}
	if cmd.Name != "generate" {
		t.Errorf("Name = %v, want generate", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
}
