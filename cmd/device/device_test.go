package device

import (
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestParseDeviceFlags(t *testing.T) {
	// Test parseAddresses function
	tests := []struct {
		name     string
		input    string
		expected []model.Address
	}{
		{
			name:  "single IP",
			input: "192.168.1.1",
			expected: []model.Address{
				{IP: "192.168.1.1", Type: "ipv4"},
			},
		},
		{
			name:  "IP with port",
			input: "192.168.1.1:22",
			expected: []model.Address{
				{IP: "192.168.1.1", Port: 22, Type: "ipv4"},
			},
		},
		{
			name:  "IP with port and type",
			input: "192.168.1.1:22:ipv4",
			expected: []model.Address{
				{IP: "192.168.1.1", Port: 22, Type: "ipv4"},
			},
		},
		{
			name:  "multiple addresses",
			input: "192.168.1.1:22:ipv4,10.0.0.1:80:ipv4",
			expected: []model.Address{
				{IP: "192.168.1.1", Port: 22, Type: "ipv4"},
				{IP: "10.0.0.1", Port: 80, Type: "ipv4"},
			},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseAddresses(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d addresses, got %d", len(tt.expected), len(result))
				return
			}
			for i, addr := range result {
				if addr.IP != tt.expected[i].IP {
					t.Errorf("address %d: expected IP %s, got %s", i, tt.expected[i].IP, addr.IP)
				}
				if addr.Port != tt.expected[i].Port {
					t.Errorf("address %d: expected Port %d, got %d", i, tt.expected[i].Port, addr.Port)
				}
				if addr.Type != tt.expected[i].Type {
					t.Errorf("address %d: expected Type %s, got %s", i, tt.expected[i].Type, addr.Type)
				}
			}
		})
	}
}

func TestGetString(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		key      string
		expected string
	}{
		{
			name:     "existing string key",
			input:    map[string]interface{}{"name": "test"},
			key:      "name",
			expected: "test",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{"name": "test"},
			key:      "other",
			expected: "",
		},
		{
			name:     "non-string value",
			input:    map[string]interface{}{"count": 42},
			key:      "count",
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

	if cmd.Name != "device" {
		t.Errorf("expected command name 'device', got %q", cmd.Name)
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
