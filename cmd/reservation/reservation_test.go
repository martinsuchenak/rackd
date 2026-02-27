package reservation

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
			input:    map[string]interface{}{"id": "reservation-1"},
			key:      "id",
			expected: "reservation-1",
		},
		{
			name:     "missing key",
			input:    map[string]interface{}{"status": "active"},
			key:      "ip_address",
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

	if cmd.Name != "reservation" {
		t.Errorf("expected command name 'reservation', got %q", cmd.Name)
	}

	if len(cmd.Commands) != 6 {
		t.Errorf("expected 6 subcommands, got %d", len(cmd.Commands))
	}

	expectedSubcommands := []string{"list", "get", "create", "update", "delete", "release"}
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

	if len(cmd.Flags) < 4 {
		t.Errorf("expected at least 4 flags, got %d", len(cmd.Flags))
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

func TestCreateCommandStructure(t *testing.T) {
	cmd := CreateCommand()

	if cmd.Name != "create" {
		t.Errorf("expected command name 'create', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 6 {
		t.Errorf("expected at least 6 flags (pool, ip, hostname, purpose, expires, notes), got %d", len(cmd.Flags))
	}
}

func TestUpdateCommandStructure(t *testing.T) {
	cmd := UpdateCommand()

	if cmd.Name != "update" {
		t.Errorf("expected command name 'update', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 5 {
		t.Errorf("expected at least 5 flags (id, hostname, purpose, expires, notes), got %d", len(cmd.Flags))
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

func TestReleaseCommandStructure(t *testing.T) {
	cmd := ReleaseCommand()

	if cmd.Name != "release" {
		t.Errorf("expected command name 'release', got %q", cmd.Name)
	}

	if len(cmd.Flags) < 1 {
		t.Errorf("expected at least 1 flag (id), got %d", len(cmd.Flags))
	}
}

func TestPrintReservationTable(t *testing.T) {
	reservations := []map[string]interface{}{
		{"id": "res-1", "ip_address": "192.168.1.100", "pool_id": "pool-1", "hostname": "server1", "status": "active", "reserved_by": "admin"},
		{"id": "res-2", "ip_address": "192.168.1.101", "pool_id": "pool-1", "hostname": "server2", "status": "expired", "reserved_by": "user1"},
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printReservationTable(reservations)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "ID") || !strings.Contains(output, "IP ADDRESS") || !strings.Contains(output, "STATUS") {
		t.Error("table output missing headers")
	}
	if !strings.Contains(output, "192.168.1.100") {
		t.Error("table output missing IP address")
	}
	if !strings.Contains(output, "active") {
		t.Error("table output missing status")
	}
}

func TestPrintReservationDetail(t *testing.T) {
	reservation := map[string]interface{}{
		"id":          "res-1",
		"pool_id":     "pool-1",
		"ip_address":  "192.168.1.100",
		"hostname":    "server1.example.com",
		"purpose":     "Web server",
		"status":      "active",
		"reserved_by": "admin",
		"reserved_at": "2024-01-15T10:30:00Z",
		"notes":       "Production server",
	}

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printReservationDetail(reservation)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "res-1") {
		t.Error("output missing reservation id")
	}
	if !strings.Contains(output, "192.168.1.100") {
		t.Error("output missing ip address")
	}
	if !strings.Contains(output, "server1.example.com") {
		t.Error("output missing hostname")
	}
	if !strings.Contains(output, "Web server") {
		t.Error("output missing purpose")
	}
}

func TestMockReservationAPIIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/reservations" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{"id": "res-1", "ip_address": "192.168.1.100", "status": "active"},
			})
		case r.URL.Path == "/api/reservations/res-1" && r.Method == "GET":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "res-1", "ip_address": "192.168.1.100", "status": "active",
			})
		case r.URL.Path == "/api/reservations" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "res-new", "ip_address": "192.168.1.102", "status": "active",
			})
		case r.URL.Path == "/api/reservations/res-1" && r.Method == "PUT":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "res-1", "ip_address": "192.168.1.100", "status": "active", "hostname": "updated",
			})
		case r.URL.Path == "/api/reservations/res-1/release" && r.Method == "POST":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "Reservation released successfully"})
		case r.URL.Path == "/api/reservations/res-1" && r.Method == "DELETE":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"message": "Reservation deleted successfully"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &client.Config{ServerURL: server.URL, Timeout: "5s"}
	c := client.NewClient(cfg)

	// Test list
	resp, err := c.DoRequest("GET", "/api/reservations", nil)
	if err != nil {
		t.Fatalf("list request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test get
	resp, err = c.DoRequest("GET", "/api/reservations/res-1", nil)
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test create
	resp, err = c.DoRequest("POST", "/api/reservations", map[string]string{"pool_id": "pool-1"})
	if err != nil {
		t.Fatalf("create request failed: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	// Test update
	resp, err = c.DoRequest("PUT", "/api/reservations/res-1", map[string]string{"hostname": "updated"})
	if err != nil {
		t.Fatalf("update request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test release
	resp, err = c.DoRequest("POST", "/api/reservations/res-1/release", nil)
	if err != nil {
		t.Fatalf("release request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	// Test delete
	resp, err = c.DoRequest("DELETE", "/api/reservations/res-1", nil)
	if err != nil {
		t.Fatalf("delete request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
