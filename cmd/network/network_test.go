package network

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

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

func TestListCommandStructure(t *testing.T) {
	cmd := ListCommand()

	if cmd.Name != "list" {
		t.Errorf("expected command name 'list', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags, got %d", len(cmd.Flags))
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

	if len(cmd.Flags) < 3 {
		t.Errorf("expected at least 3 flags, got %d", len(cmd.Flags))
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

func TestNetworkTableOutput(t *testing.T) {
	networks := []map[string]interface{}{
		{"id": "net1", "name": "prod-network", "subnet": "10.0.0.0/24", "vlan_id": 100, "datacenter_id": "dc1"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.PrintNetworkTable(networks)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "SUBNET") {
		t.Error("table output missing headers")
	}
	if !strings.Contains(output, "prod-network") {
		t.Error("table output missing network name")
	}
	if !strings.Contains(output, "10.0.0.0/24") {
		t.Error("table output missing subnet")
	}
}

func TestNetworkJSONOutput(t *testing.T) {
	networks := []map[string]interface{}{
		{"id": "net1", "name": "test-network", "subnet": "192.168.0.0/16"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.PrintJSON(networks)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("JSON output not valid: %v", err)
	}
	if len(parsed) != 1 || parsed[0]["name"] != "test-network" {
		t.Errorf("unexpected JSON output: %s", output)
	}
}

func TestMockNetworkAPIIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/networks" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "net1", "name": "test-network", "subnet": "10.0.0.0/24"},
			})
		case r.URL.Path == "/api/networks/net1" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "net1", "name": "test-network", "subnet": "10.0.0.0/24",
			})
		case r.URL.Path == "/api/networks" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "new-net", "name": "new-network",
			})
		case r.URL.Path == "/api/networks/net1" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		case strings.HasPrefix(r.URL.Path, "/api/networks/net1/pools"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &client.Config{ServerURL: server.URL, Timeout: "5s"}
	c := client.NewClient(cfg)

	// Test list
	resp, err := c.DoRequest("GET", "/api/networks", nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test get
	resp, err = c.DoRequest("GET", "/api/networks/net1", nil)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test create
	resp, err = c.DoRequest("POST", "/api/networks", map[string]string{"name": "new-network", "subnet": "172.16.0.0/16"})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	// Test delete
	resp, err = c.DoRequest("DELETE", "/api/networks/net1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}

	// Test pool list
	resp, err = c.DoRequest("GET", "/api/networks/net1/pools", nil)
	if err != nil {
		t.Fatalf("pool list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
