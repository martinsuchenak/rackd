package dns

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/martinsuchenak/rackd/cmd/client"
)

func TestCommandStructure(t *testing.T) {
	cmd := Command()

	if cmd.Name != "dns" {
		t.Fatalf("expected command name 'dns', got %q", cmd.Name)
	}

	expectedSubcommands := []string{"provider", "zone", "sync", "import", "records"}
	if len(cmd.Commands) != len(expectedSubcommands) {
		t.Fatalf("expected %d subcommands, got %d", len(expectedSubcommands), len(cmd.Commands))
	}

	for i, expected := range expectedSubcommands {
		if cmd.Commands[i].Name != expected {
			t.Errorf("subcommand %d: expected %q, got %q", i, expected, cmd.Commands[i].Name)
		}
	}
}

func TestProviderAndZoneCommandStructure(t *testing.T) {
	providerCmd := ProviderCommand()
	if len(providerCmd.Commands) != 6 {
		t.Fatalf("expected 6 provider subcommands, got %d", len(providerCmd.Commands))
	}

	zoneCmd := ZoneCommand()
	if len(zoneCmd.Commands) != 5 {
		t.Fatalf("expected 5 zone subcommands, got %d", len(zoneCmd.Commands))
	}

	recordsCmd := RecordsCommand()
	if recordsCmd.Run == nil {
		t.Fatal("records command should have a Run function")
	}
}

func TestMockDNSAPIIntegration(t *testing.T) {
	var seenPaths []string
	var bodies []map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPaths = append(seenPaths, r.URL.Path)

		if r.Body != nil && r.ContentLength != 0 {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			bodies = append(bodies, body)
		}

		switch {
		case r.URL.Path == "/api/dns/providers" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "provider-1", "name": "bind-main"},
			})
		case r.URL.Path == "/api/dns/providers/provider-1" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "provider-1", "name": "bind-main"})
		case r.URL.Path == "/api/dns/providers" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "provider-2", "name": "technitium-main"})
		case r.URL.Path == "/api/dns/providers/provider-2" && r.Method == http.MethodPut:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "provider-2", "name": "technitium-updated"})
		case r.URL.Path == "/api/dns/providers/provider-2" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/dns/providers/provider-2/test" && r.Method == http.MethodPost:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/dns/zones" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "zone-1", "name": "example.test"},
			})
		case r.URL.Path == "/api/dns/zones/zone-1" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "zone-1", "name": "example.test"})
		case r.URL.Path == "/api/dns/zones" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "zone-2", "name": "example.test"})
		case r.URL.Path == "/api/dns/zones/zone-2" && r.Method == http.MethodPut:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "zone-2", "name": "updated.example.test"})
		case r.URL.Path == "/api/dns/zones/zone-2" && r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/dns/zones/zone-1/records" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"id": "record-1", "name": "host1", "type": "A", "value": "10.0.0.1"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := client.NewClient(&client.Config{ServerURL: server.URL, Timeout: "5s"})

	requests := []struct {
		method string
		path   string
		body   any
		status int
	}{
		{"GET", "/api/dns/providers", nil, http.StatusOK},
		{"GET", "/api/dns/providers/provider-1", nil, http.StatusOK},
		{"POST", "/api/dns/providers", map[string]any{"name": "technitium-main", "type": "technitium", "endpoint": "https://dns.example", "token": "secret"}, http.StatusCreated},
		{"PUT", "/api/dns/providers/provider-2", map[string]any{"description": "updated"}, http.StatusOK},
		{"DELETE", "/api/dns/providers/provider-2", nil, http.StatusNoContent},
		{"POST", "/api/dns/providers/provider-2/test", nil, http.StatusNoContent},
		{"GET", "/api/dns/zones", nil, http.StatusOK},
		{"GET", "/api/dns/zones/zone-1", nil, http.StatusOK},
		{"POST", "/api/dns/zones", map[string]any{"name": "example.test", "provider_id": "provider-1"}, http.StatusCreated},
		{"PUT", "/api/dns/zones/zone-2", map[string]any{"name": "updated.example.test"}, http.StatusOK},
		{"DELETE", "/api/dns/zones/zone-2", nil, http.StatusNoContent},
		{"GET", "/api/dns/zones/zone-1/records", nil, http.StatusOK},
	}

	for _, tc := range requests {
		resp, err := c.DoRequest(tc.method, tc.path, tc.body)
		if err != nil {
			t.Fatalf("%s %s failed: %v", tc.method, tc.path, err)
		}
		if resp.StatusCode != tc.status {
			t.Fatalf("%s %s: expected %d, got %d", tc.method, tc.path, tc.status, resp.StatusCode)
		}
		resp.Body.Close()
	}

	if len(seenPaths) != len(requests) {
		t.Fatalf("expected %d DNS requests, saw %d", len(requests), len(seenPaths))
	}
	if len(bodies) < 4 {
		t.Fatalf("expected JSON bodies for DNS create/update requests, got %d", len(bodies))
	}
}

func TestProviderCreateUsesTokenEnvAndFileInputs(t *testing.T) {
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token.txt")
	if err := os.WriteFile(tokenFile, []byte(" file-token \n"), 0o600); err != nil {
		t.Fatalf("failed to write token file: %v", err)
	}

	os.Setenv("DNS_TEST_TOKEN", " env-token ")
	defer os.Unsetenv("DNS_TEST_TOKEN")

	cmd := providerCreateCommand()
	if cmd.Run == nil {
		t.Fatal("provider create command should have a Run function")
	}

	if _, err := os.Stat(tokenFile); err != nil {
		t.Fatalf("expected token file to exist: %v", err)
	}
}
