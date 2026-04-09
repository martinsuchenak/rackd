package user

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "user" {
		t.Fatalf("expected command name 'user', got %q", cmd.Name)
	}

	expectedSubcommands := []string{"list", "create", "update", "delete", "password"}
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

func TestUpdateCommandFlags(t *testing.T) {
	cmd := UpdateCommand()
	if cmd.Name != "update" {
		t.Fatalf("expected update command, got %q", cmd.Name)
	}
	if len(cmd.Flags) < 10 {
		t.Fatalf("expected update command to expose role and status flags, got %d flags", len(cmd.Flags))
	}
}

func TestUserRoleHelpersAndUserAPIIntegration(t *testing.T) {
	var seenPaths []string
	var seenMethods []string
	var seenBodies []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)
		seenMethods = append(seenMethods, r.Method)

		if r.Body != nil && r.ContentLength != 0 {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			seenBodies = append(seenBodies, body)
		}

		switch {
		case r.URL.Path == "/api/users" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "u1", "username": "alice", "email": "alice@example.com"},
			})
		case r.URL.Path == "/api/users/u1" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "u1", "username": "alice", "email": "alice@example.com",
			})
		case r.URL.Path == "/api/users/u1" && r.Method == http.MethodPut:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "u1", "username": "alice-renamed", "email": "alice@example.com",
			})
		case r.URL.Path == "/api/users/u1" && r.Method == http.MethodDelete:
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

	resp, err := c.DoRequest("GET", "/api/users", nil)
	if err != nil {
		t.Fatalf("list users request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from list users, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err = c.DoRequest("PUT", "/api/users/u1", map[string]any{
		"username":  "alice-renamed",
		"full_name": "Alice Renamed",
	})
	if err != nil {
		t.Fatalf("update user request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from update user, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	if err := assignRole(c, "u1", "role-1"); err != nil {
		t.Fatalf("assignRole returned error: %v", err)
	}

	if err := revokeRole(c, "u1", "role-1"); err != nil {
		t.Fatalf("revokeRole returned error: %v", err)
	}

	resp, err = c.DoRequest("DELETE", "/api/users/u1", nil)
	if err != nil {
		t.Fatalf("delete user request failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 from delete user, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	if len(seenPaths) < 5 {
		t.Fatalf("expected multiple user API calls, saw %d", len(seenPaths))
	}

	foundGrant := false
	foundRevoke := false
	for i, path := range seenPaths {
		if path == "/api/users/grant-role" && seenMethods[i] == http.MethodPost {
			foundGrant = true
		}
		if path == "/api/users/revoke-role" && seenMethods[i] == http.MethodPost {
			foundRevoke = true
		}
	}
	if !foundGrant {
		t.Fatal("expected grant-role request to be issued")
	}
	if !foundRevoke {
		t.Fatal("expected revoke-role request to be issued")
	}

	if len(seenBodies) < 3 {
		t.Fatalf("expected captured JSON request bodies, got %d", len(seenBodies))
	}
}
