package role

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "role" {
		t.Fatalf("expected command name 'role', got %q", cmd.Name)
	}

	expectedSubcommands := []string{"list", "permissions", "create", "delete", "assign", "revoke"}
	if len(cmd.Commands) != len(expectedSubcommands) {
		t.Fatalf("expected %d subcommands, got %d", len(expectedSubcommands), len(cmd.Commands))
	}

	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
		if cmd.Commands[i].Run == nil {
			t.Errorf("subcommand %q has nil Run function", cmd.Commands[i].Name)
		}
	}
}

func TestMockRoleAPIIntegration(t *testing.T) {
	var bodies []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.ContentLength != 0 {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			bodies = append(bodies, body)
		}

		switch {
		case r.URL.Path == "/api/roles" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "role-1", "name": "operator"},
			})
		case r.URL.Path == "/api/permissions" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "perm-1", "resource": "devices", "action": "read"},
			})
		case r.URL.Path == "/api/roles" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "role-2", "name": "dns-operator",
			})
		case r.URL.Path == "/api/roles/role-2" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/users/grant-role" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == "/api/users/revoke-role" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := client.NewClient(&client.Config{ServerURL: server.URL, Timeout: "5s"})

	resp, err := c.DoRequest("GET", "/api/roles", nil)
	if err != nil {
		t.Fatalf("list roles request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = c.DoRequest("GET", "/api/permissions?resource=devices&action=read", nil)
	if err != nil {
		t.Fatalf("list permissions request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = c.DoRequest("POST", "/api/roles", map[string]any{
		"name":        "dns-operator",
		"description": "DNS operator",
	})
	if err != nil {
		t.Fatalf("create role request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = c.DoRequest("POST", "/api/users/grant-role", map[string]any{
		"user_id": "user-1",
		"role_id": "role-2",
	})
	if err != nil {
		t.Fatalf("grant role request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = c.DoRequest("POST", "/api/users/revoke-role", map[string]any{
		"user_id": "user-1",
		"role_id": "role-2",
	})
	if err != nil {
		t.Fatalf("revoke role request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = c.DoRequest("DELETE", "/api/roles/role-2", nil)
	if err != nil {
		t.Fatalf("delete role request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	if len(bodies) < 3 {
		t.Fatalf("expected request bodies for create/assign/revoke, got %d", len(bodies))
	}
}
