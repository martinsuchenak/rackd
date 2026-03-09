package scheduledscan

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "scheduled-scan" {
		t.Errorf("expected command name 'scheduled-scan', got %q", cmd.Name)
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

func TestMockScheduledScanAPIIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/scheduled-scans" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id": "ss1", "name": "nightly-scan", "network_id": "net1",
					"profile_id": "sp1", "cron_expression": "0 2 * * *", "enabled": true,
				},
			})
		case r.URL.Path == "/api/scheduled-scans/ss1" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "ss1", "name": "nightly-scan", "network_id": "net1",
				"profile_id": "sp1", "cron_expression": "0 2 * * *", "enabled": true,
			})
		case r.URL.Path == "/api/scheduled-scans" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "ss-new", "name": "new-scan",
			})
		case r.URL.Path == "/api/scheduled-scans/ss1" && r.Method == "PUT":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "ss1", "name": "updated-scan",
			})
		case r.URL.Path == "/api/scheduled-scans/ss1" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &client.Config{ServerURL: server.URL, Timeout: "5s"}
	c := client.NewClient(cfg)

	// Test list
	resp, err := c.DoRequest("GET", "/api/scheduled-scans", nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test list with network filter
	resp, err = c.DoRequest("GET", "/api/scheduled-scans?network_id=net1", nil)
	if err != nil {
		t.Fatalf("filtered list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test get
	resp, err = c.DoRequest("GET", "/api/scheduled-scans/ss1", nil)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test create
	resp, err = c.DoRequest("POST", "/api/scheduled-scans", map[string]interface{}{
		"name": "new-scan", "network_id": "net1", "profile_id": "sp1",
		"cron_expression": "0 3 * * *", "enabled": true,
	})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	// Test update
	resp, err = c.DoRequest("PUT", "/api/scheduled-scans/ss1", map[string]interface{}{
		"name": "updated-scan",
	})
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test delete
	resp, err = c.DoRequest("DELETE", "/api/scheduled-scans/ss1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}

func TestListCommandFlags(t *testing.T) {
	cmd := Command()
	listCmd := cmd.Commands[0]
	if listCmd.Name != "list" {
		t.Fatalf("expected first subcommand to be 'list', got %q", listCmd.Name)
	}

	// Should have network filter and output flags
	if len(listCmd.Flags) < 2 {
		t.Errorf("expected at least 2 flags (network, output), got %d", len(listCmd.Flags))
	}
}

func TestCreateCommandFlags(t *testing.T) {
	cmd := Command()
	createCmd := cmd.Commands[2]
	if createCmd.Name != "create" {
		t.Fatalf("expected third subcommand to be 'create', got %q", createCmd.Name)
	}

	// Should have name, network, profile, cron, enabled, description
	if len(createCmd.Flags) < 4 {
		t.Errorf("expected at least 4 flags, got %d", len(createCmd.Flags))
	}
}
