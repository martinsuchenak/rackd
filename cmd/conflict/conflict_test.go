package conflict

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
			input:    map[string]interface{}{"id": "conflict-1"},
			key:      "id",
			expected: "conflict-1",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{"type": "duplicate_ip"},
			key:      "status",
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

	if cmd.Name != "conflict" {
		t.Errorf("expected command name 'conflict', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 5 {
		t.Errorf("expected 5 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"list", "get", "detect", "resolve", "delete"}
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

	if len(cmd.Flags) < 3 {
		t.Errorf("expected at least 3 flags, got %d", len(cmd.Flags))
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

func TestDetectCommandStructure(t *testing.T) {
	cmd := DetectCommand()

	if cmd.Name != "detect" {
		t.Errorf("expected command name 'detect', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags, got %d", len(cmd.Flags))
	}
}

func TestResolveCommandStructure(t *testing.T) {
	cmd := ResolveCommand()

	if cmd.Name != "resolve" {
		t.Errorf("expected command name 'resolve', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 4 {
		t.Errorf("expected at least 4 flags (id, keep-device-id, keep-network-id, notes), got %d", len(cmd.Flags))
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

func TestConflictTableOutput(t *testing.T) {
	conflicts := []map[string]interface{}{
		{"id": "conflict-1", "type": "duplicate_ip", "status": "active", "description": "IP 10.0.0.1 assigned to 2 devices"},
		{"id": "conflict-2", "type": "overlapping_subnet", "status": "resolved", "description": "Subnets 10.0.0.0/24 and 10.0.0.0/16 overlap"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.PrintConflictTable(conflicts)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "ID") || !strings.Contains(output, "TYPE") || !strings.Contains(output, "STATUS") {
		t.Error("table output missing headers")
	}
	if !strings.Contains(output, "conflict-1") {
		t.Error("table output missing conflict id")
	}
	if !strings.Contains(output, "duplicate_ip") {
		t.Error("table output missing conflict type")
	}
}

func TestConflictJSONOutput(t *testing.T) {
	conflicts := []map[string]interface{}{
		{"id": "conflict-1", "type": "duplicate_ip", "status": "active"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	client.PrintJSON(conflicts)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("JSON output not valid: %v", err)
	}
	if len(parsed) != 1 || parsed[0]["id"] != "conflict-1" {
		t.Errorf("unexpected JSON output: %s", output)
	}
}

func TestMockConflictAPIIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/conflicts" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "conflict-1", "type": "duplicate_ip", "status": "active", "description": "IP 10.0.0.1 conflict"},
			})
		case r.URL.Path == "/api/conflicts/conflict-1" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "conflict-1", "type": "duplicate_ip", "status": "active", "description": "IP 10.0.0.1 conflict",
			})
		case r.URL.Path == "/api/conflicts/detect" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"conflicts": []map[string]interface{}{
					{"id": "detected-1", "type": "duplicate_ip", "status": "active"},
				},
			})
		case r.URL.Path == "/api/conflicts/resolve" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
		case r.URL.Path == "/api/conflicts/conflict-1" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &client.Config{ServerURL: server.URL, Timeout: "5s"}
	c := client.NewClient(cfg)

	// Test list
	resp, err := c.DoRequest("GET", "/api/conflicts", nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test get
	resp, err = c.DoRequest("GET", "/api/conflicts/conflict-1", nil)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test detect
	resp, err = c.DoRequest("POST", "/api/conflicts/detect", nil)
	if err != nil {
		t.Fatalf("detect request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test resolve
	resp, err = c.DoRequest("POST", "/api/conflicts/resolve", map[string]string{"conflict_id": "conflict-1", "notes": "resolved"})
	if err != nil {
		t.Fatalf("resolve request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test delete
	resp, err = c.DoRequest("DELETE", "/api/conflicts/conflict-1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestPrintConflictDetail(t *testing.T) {
	conflict := map[string]interface{}{
		"id":          "conflict-1",
		"type":        "duplicate_ip",
		"status":      "active",
		"description": "IP 10.0.0.1 assigned to multiple devices",
		"ip_address": "10.0.0.1",
		"device_ids":  []interface{}{"device-1", "device-2"},
		"device_names": []interface{}{"server1", "server2"},
		"detected_at": "2024-01-15T10:30:00Z",
		"resolved_at": nil,
		"resolved_by": nil,
		"notes":       nil,
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printConflictDetail(conflict)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "conflict-1") {
		t.Error("output missing conflict id")
	}
	if !strings.Contains(output, "duplicate_ip") {
		t.Error("output missing conflict type")
	}
	if !strings.Contains(output, "10.0.0.1") {
		t.Error("output missing ip address")
	}
	if !strings.Contains(output, "server1") {
		t.Error("output missing device name")
	}
}
