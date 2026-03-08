package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type mockDiscoveryScanner struct {
	store storage.ExtendedStorage
}

func (m *mockDiscoveryScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
	scan := &model.DiscoveryScan{
		ID:         uuid.Must(uuid.NewV7()).String(),
		NetworkID:  network.ID,
		Status:     model.ScanStatusRunning,
		ScanType:   scanType,
		TotalHosts: 256,
	}
	if err := m.store.CreateDiscoveryScan(context.Background(), scan); err != nil {
		return nil, err
	}
	return scan, nil
}

func (m *mockDiscoveryScanner) GetScanStatus(ctx context.Context, scanID string) (*model.DiscoveryScan, error) {
	return m.store.GetDiscoveryScan(context.Background(), scanID)
}

func (m *mockDiscoveryScanner) CancelScan(ctx context.Context, scanID string) error {
	return nil
}

func init() {
	// Initialize logger for tests
	log.Init("console", "error", io.Discard)
}

func newTestServer(t *testing.T) (*Server, storage.ExtendedStorage) {
	t.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	scanner := &mockDiscoveryScanner{store: store}
	svc := service.NewServices(store, nil, scanner)
	return NewServer(svc, store, false), store
}

func newTestServerWithAuth(t *testing.T) (*Server, storage.ExtendedStorage) {
	t.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	scanner := &mockDiscoveryScanner{store: store}
	svc := service.NewServices(store, nil, scanner)
	return NewServer(svc, store, true), store
}

func TestNewServer(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	if srv == nil {
		t.Fatal("expected server to be created")
	}
	if srv.Inner() == nil {
		t.Fatal("expected inner MCP server to be created")
	}
}

func TestHandleRequest_NoAuth(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Test tools/list request
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandleRequest_WithAuth_ValidToken(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	// Create a user to associate with the API key
	user := &model.User{
		Username: "testuser",
		Email:    "test@example.com",
		FullName: "Test User",
		IsActive: true,
	}
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create an API key associated with the user
	apiKeySecret := "test-token-12345"
	key := &model.APIKey{
		Name:   "test-key",
		Key:    auth.HashToken(apiKeySecret),
		UserID: user.ID,
	}
	if err := store.CreateAPIKey(context.Background(), key); err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKeySecret)
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestHandleRequest_WithAuth_InvalidToken(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestHandleRequest_WithAuth_LegacyKeyRejected(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	// Create an API key WITHOUT a user association (legacy key)
	apiKeySecret := "legacy-key-no-user"
	key := &model.APIKey{
		Name: "legacy-key",
		Key:  auth.HashToken(apiKeySecret),
		// No UserID — this is the legacy pattern that must be rejected
	}
	if err := store.CreateAPIKey(context.Background(), key); err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKeySecret)
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected legacy key to be rejected with 401, got %d", w.Code)
	}
}

func TestHandleRequest_WithAuth_MissingToken(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestHandleRequest_WithAuth_NoBearerPrefix(t *testing.T) {
	srv, store := newTestServerWithAuth(t)
	defer store.Close()

	// Create an API key
	apiKeySecret := "test-token-12345"
	key := &model.APIKey{
		Name: "test-key",
		Key:  auth.HashToken(apiKeySecret),
	}
	if err := store.CreateAPIKey(context.Background(), key); err != nil {
		t.Fatalf("failed to create API key: %v", err)
	}

	reqBody := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", apiKeySecret) // Purposefully missing "Bearer "
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestToolsRegistered(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	tools := srv.Inner().ListTools()

	// Only native (non-discoverable) tools appear in ListTools().
	// Discoverable tools are registered but hidden until discovered via keywords.
	expectedTools := []string{
		"search",
		"device_save",
		"device_get",
		"device_list",
		"device_delete",
		"datacenter_list",
		"datacenter_get",
		"network_list",
		"network_get",
		"pool_get_next_ip",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("expected tool %q to be registered", expected)
		}
	}
}

// Helper to call MCP tool
func callTool(t *testing.T, srv *Server, toolName string, args map[string]interface{}) map[string]interface{} {
	t.Helper()

	params := map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	}
	paramsJSON, _ := json.Marshal(params)

	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  json.RawMessage(paramsJSON),
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	srv.HandleRequest(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	return resp
}

func TestDeviceSave_Create(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	resp := callTool(t, srv, "device_save", map[string]interface{}{
		"name":        "test-device",
		"description": "A test device",
		"tags":        []string{"test", "dev"},
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result := resp["result"].(map[string]interface{})
	content := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatal("expected content in response")
	}
}

func TestDeviceList(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Create a device first
	callTool(t, srv, "device_save", map[string]interface{}{
		"name": "list-test-device",
	})

	resp := callTool(t, srv, "device_list", map[string]interface{}{})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestDatacenterSave_Create(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	resp := callTool(t, srv, "datacenter_save", map[string]interface{}{
		"name":     "test-dc",
		"location": "US-East",
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestDatacenterList(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Create a datacenter first
	callTool(t, srv, "datacenter_save", map[string]interface{}{
		"name": "list-test-dc",
	})

	resp := callTool(t, srv, "datacenter_list", map[string]interface{}{})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestNetworkSave_Create(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	resp := callTool(t, srv, "network_save", map[string]interface{}{
		"name":   "test-network",
		"subnet": "192.168.1.0/24",
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestNetworkList(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Create a network first
	callTool(t, srv, "network_save", map[string]interface{}{
		"name":   "list-test-network",
		"subnet": "10.0.0.0/24",
	})

	resp := callTool(t, srv, "network_list", map[string]interface{}{})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestDiscoveryScan(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Create a network first
	callTool(t, srv, "network_save", map[string]interface{}{
		"name":   "scan-test-network",
		"subnet": "172.16.0.0/24",
	})

	networks, _ := store.ListNetworks(context.Background(), nil)
	if len(networks) == 0 {
		t.Fatal("expected network to be created")
	}

	resp := callTool(t, srv, "discovery_scan", map[string]interface{}{
		"network_id": networks[0].ID,
		"scan_type":  "quick",
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestDiscoveryList(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	resp := callTool(t, srv, "discovery_list", map[string]interface{}{})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestAddRelationship_InvalidType(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Create two devices
	callTool(t, srv, "device_save", map[string]interface{}{"name": "parent"})
	callTool(t, srv, "device_save", map[string]interface{}{"name": "child"})

	devices, _ := store.ListDevices(context.Background(), nil)
	if len(devices) < 2 {
		t.Fatal("expected 2 devices")
	}

	resp := callTool(t, srv, "device_add_relationship", map[string]interface{}{
		"parent_id": devices[0].ID,
		"child_id":  devices[1].ID,
		"type":      "invalid_type",
	})

	if resp["error"] == nil {
		t.Fatal("expected error for invalid relationship type")
	}
}

func TestAddRelationship_ValidType(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Create two devices
	callTool(t, srv, "device_save", map[string]interface{}{"name": "parent"})
	callTool(t, srv, "device_save", map[string]interface{}{"name": "child"})

	devices, _ := store.ListDevices(context.Background(), nil)
	if len(devices) < 2 {
		t.Fatal("expected 2 devices")
	}

	resp := callTool(t, srv, "device_add_relationship", map[string]interface{}{
		"parent_id": devices[0].ID,
		"child_id":  devices[1].ID,
		"type":      "contains",
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestGetRelationships(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	// Create a device
	callTool(t, srv, "device_save", map[string]interface{}{"name": "test"})
	devices, _ := store.ListDevices(context.Background(), nil)
	if len(devices) == 0 {
		t.Fatal("expected device")
	}

	resp := callTool(t, srv, "device_get_relationships", map[string]interface{}{
		"id": devices[0].ID,
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

func TestInner(t *testing.T) {
	srv, store := newTestServer(t)
	defer store.Close()

	inner := srv.Inner()
	if inner == nil {
		t.Fatal("Inner() should return the underlying MCP server")
	}
}
