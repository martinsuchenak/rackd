package scanprofile

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "scan-profile" {
		t.Errorf("expected command name 'scan-profile', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 5 {
		t.Errorf("expected 5 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"list", "get", "create", "update", "delete"}
	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
	}
}

func TestSubcommandRunFunctions(t *testing.T) {
	cmd := Command()
	for _, sub := range cmd.Commands {
		if sub.Run == nil {
			t.Errorf("subcommand %q has nil Run function", sub.Name)
		}
	}
}

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []int
	}{
		{"single port", "80", []int{80}},
		{"multiple ports", "22, 80, 443", []int{22, 80, 443}},
		{"invalid ignored", "22, abc, 443", []int{22, 443}},
		{"empty", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parsePorts(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d ports, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("port %d: expected %d, got %d", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestMockScanProfileAPIIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/scan-profiles" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "sp1", "name": "quick-scan", "scan_type": "quick"},
			})
		case r.URL.Path == "/api/scan-profiles/sp1" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "sp1", "name": "quick-scan", "scan_type": "quick",
				"timeout_sec": 30, "max_workers": 10,
			})
		case r.URL.Path == "/api/scan-profiles" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "sp-new", "name": "new-profile",
			})
		case r.URL.Path == "/api/scan-profiles/sp1" && r.Method == "PUT":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "sp1", "name": "updated-profile",
			})
		case r.URL.Path == "/api/scan-profiles/sp1" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &client.Config{ServerURL: server.URL, Timeout: "5s"}
	c := client.NewClient(cfg)

	// Test list
	resp, err := c.DoRequest("GET", "/api/scan-profiles", nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test get
	resp, err = c.DoRequest("GET", "/api/scan-profiles/sp1", nil)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test create
	resp, err = c.DoRequest("POST", "/api/scan-profiles", map[string]interface{}{
		"name": "new-profile", "scan_type": "full",
	})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	// Test update
	resp, err = c.DoRequest("PUT", "/api/scan-profiles/sp1", map[string]interface{}{
		"name": "updated-profile",
	})
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test delete
	resp, err = c.DoRequest("DELETE", "/api/scan-profiles/sp1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}
