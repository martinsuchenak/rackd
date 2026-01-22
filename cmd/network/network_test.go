package network

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
			input:    map[string]interface{}{"name": "test-network"},
			key:      "name",
			expected: "test-network",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{"name": "test"},
			key:      "subnet",
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

	if cmd.Name != "network" {
		t.Errorf("expected command name 'network', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 5 {
		t.Errorf("expected 5 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"list", "get", "add", "delete", "pool"}
	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
	}
}

func TestPoolCommandStructure(t *testing.T) {
	cmd := PoolCommand()

	if cmd.Name != "pool" {
		t.Errorf("expected command name 'pool', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"list", "add"}
	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
	}
}
