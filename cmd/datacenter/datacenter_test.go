package datacenter

import "testing"

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string key",
			input:    map[string]interface{}{"name": "DC1"},
			key:      "name",
			expected: "DC1",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{"name": "DC1"},
			key:      "location",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getString(tt.input, tt.key)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "datacenter" {
		t.Errorf("expected command name 'datacenter', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 5 {
		t.Errorf("expected 5 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"list", "get", "add", "update", "delete"}
	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
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

func TestGetCommand(t *testing.T) {
	cmd := GetCommand()
	if cmd == nil {
		t.Fatal("GetCommand() returned nil")
	}
	if cmd.Name != "get" {
		t.Errorf("Name = %v, want get", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
}

func TestAddCommand(t *testing.T) {
	cmd := AddCommand()
	if cmd == nil {
		t.Fatal("AddCommand() returned nil")
	}
	if cmd.Name != "add" {
		t.Errorf("Name = %v, want add", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
	}
}

func TestUpdateCommand(t *testing.T) {
	cmd := UpdateCommand()
	if cmd == nil {
		t.Fatal("UpdateCommand() returned nil")
	}
	if cmd.Name != "update" {
		t.Errorf("Name = %v, want update", cmd.Name)
	}
	if cmd.Run == nil {
		t.Error("Run function should not be nil")
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
}
