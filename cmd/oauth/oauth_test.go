package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "oauth" {
		t.Errorf("expected command name 'oauth', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 2 {
		t.Errorf("expected 2 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"list", "delete"}
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

func TestMockOAuthAPIIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/oauth/clients" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"client_id":       "client-1",
					"client_name":     "test-app",
					"grant_types":     []string{"authorization_code"},
					"is_confidential": true,
					"created_at":      "2026-01-01T00:00:00Z",
				},
			})
		case r.URL.Path == "/api/oauth/clients/client-1" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &client.Config{ServerURL: server.URL, Timeout: "5s"}
	c := client.NewClient(cfg)

	// Test list
	resp, err := c.DoRequest("GET", "/api/oauth/clients", nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var clients []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&clients)
	resp.Body.Close()
	if len(clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(clients))
	}
	if clients[0]["client_name"] != "test-app" {
		t.Errorf("expected client_name 'test-app', got %v", clients[0]["client_name"])
	}

	// Test delete
	resp, err = c.DoRequest("DELETE", "/api/oauth/clients/client-1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.StatusCode)
	}
}
