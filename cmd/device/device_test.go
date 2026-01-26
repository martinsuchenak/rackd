package device

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
	"github.com/martinsuchenak/rackd/internal/model"
)

func intPtr(i int) *int { return &i }

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
				{IP: "192.168.1.1", Port: intPtr(22), Type: "ipv4"},
			},
		},
		{
			name:  "IP with port and type",
			input: "192.168.1.1:22:ipv4",
			expected: []model.Address{
				{IP: "192.168.1.1", Port: intPtr(22), Type: "ipv4"},
			},
		},
		{
			name:  "multiple addresses",
			input: "192.168.1.1:22:ipv4,10.0.0.1:80:ipv4",
			expected: []model.Address{
				{IP: "192.168.1.1", Port: intPtr(22), Type: "ipv4"},
				{IP: "10.0.0.1", Port: intPtr(80), Type: "ipv4"},
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
				expectedPort := tt.expected[i].Port
				if (addr.Port == nil) != (expectedPort == nil) {
					t.Errorf("address %d: expected Port nil=%v, got nil=%v", i, expectedPort == nil, addr.Port == nil)
				} else if addr.Port != nil && *addr.Port != *expectedPort {
					t.Errorf("address %d: expected Port %d, got %d", i, *expectedPort, *addr.Port)
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

func TestListCommandStructure(t *testing.T) {
	cmd := ListCommand()

	if cmd.Name != "list" {
		t.Errorf("expected command name 'list', got %q", cmd.Name)
	}

	// Verify flags exist (count check)
	if len(cmd.Flags) < 5 {
		t.Errorf("expected at least 5 flags, got %d", len(cmd.Flags))
	}
}

func TestGetCommandStructure(t *testing.T) {
	cmd := GetCommand()

	if cmd.Name != "get" {
		t.Errorf("expected command name 'get', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags (id, output), got %d", len(cmd.Flags))
	}
}

func TestAddCommandStructure(t *testing.T) {
	cmd := AddCommand()

	if cmd.Name != "add" {
		t.Errorf("expected command name 'add', got %q", cmd.Name)
	}

	// Should have multiple flags for device fields
	if len(cmd.Flags) < 5 {
		t.Errorf("expected at least 5 flags, got %d", len(cmd.Flags))
	}
}

func TestUpdateCommandStructure(t *testing.T) {
	cmd := UpdateCommand()

	if cmd.Name != "update" {
		t.Errorf("expected command name 'update', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags, got %d", len(cmd.Flags))
	}
}

func TestDeleteCommandStructure(t *testing.T) {
	cmd := DeleteCommand()

	if cmd.Name != "delete" {
		t.Errorf("expected command name 'delete', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags (id, force), got %d", len(cmd.Flags))
	}
}

func TestOutputFormats_JSON(t *testing.T) {
	devices := []map[string]interface{}{
		{"id": "1", "name": "server1", "make_model": "Dell", "os": "Ubuntu", "datacenter_id": "dc1"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.PrintJSON(devices)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("JSON output not valid: %v", err)
	}
	if len(parsed) != 1 || parsed[0]["name"] != "server1" {
		t.Errorf("unexpected JSON output: %s", output)
	}
}

func TestOutputFormats_Table(t *testing.T) {
	devices := []map[string]interface{}{
		{"id": "abc123", "name": "web-server", "make_model": "Dell R640", "os": "Ubuntu", "datacenter_id": "dc1"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.PrintDeviceTable(devices)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") {
		t.Error("table output missing headers")
	}
	if !strings.Contains(output, "web-server") {
		t.Error("table output missing device name")
	}
}

func TestMockAPIIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/devices" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "1", "name": "test-device"},
			})
		case r.URL.Path == "/api/devices/1" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "1", "name": "test-device", "make_model": "Dell",
			})
		case r.URL.Path == "/api/devices" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "new-id", "name": "new-device",
			})
		case r.URL.Path == "/api/devices/1" && r.Method == "PUT":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "1", "name": "updated-device",
			})
		case r.URL.Path == "/api/devices/1" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &client.Config{ServerURL: server.URL, Timeout: "5s"}
	c := client.NewClient(cfg)

	// Test list
	resp, err := c.DoRequest("GET", "/api/devices", nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test get
	resp, err = c.DoRequest("GET", "/api/devices/1", nil)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test create
	resp, err = c.DoRequest("POST", "/api/devices", map[string]string{"name": "new-device"})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	// Test update
	resp, err = c.DoRequest("PUT", "/api/devices/1", map[string]string{"name": "updated-device"})
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test delete
	resp, err = c.DoRequest("DELETE", "/api/devices/1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}
