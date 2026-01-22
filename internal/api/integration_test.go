//go:build !short

package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func init() {
	log.Init("console", "error", io.Discard)
}

// setupIntegrationServer creates a full server with auth middleware
func setupIntegrationServer(t *testing.T, authToken string) (*httptest.Server, storage.ExtendedStorage) {
	t.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	h := NewHandler(store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Register UI config endpoint
	uiBuilder := NewUIConfigBuilder()
	mux.HandleFunc("GET /api/config", uiBuilder.Handler())

	// Wrap with security headers
	handler := SecurityHeaders(mux)

	// If auth token provided, wrap routes that need auth
	if authToken != "" {
		authMux := http.NewServeMux()
		authMux.HandleFunc("/api/", AuthMiddleware(authToken, func(w http.ResponseWriter, r *http.Request) {
			mux.ServeHTTP(w, r)
		}))
		handler = SecurityHeaders(authMux)
	}

	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
		store.Close()
	})

	return server, store
}

func TestFullDeviceWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server, _ := setupIntegrationServer(t, "")

	// 1. Create datacenter
	dcBody := `{"name":"Integration DC","location":"Test Location"}`
	resp, err := http.Post(server.URL+"/api/datacenters", "application/json", bytes.NewBufferString(dcBody))
	if err != nil {
		t.Fatalf("create datacenter request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var dc model.Datacenter
	json.NewDecoder(resp.Body).Decode(&dc)

	// 2. Create network
	netBody := `{"name":"Integration Network","subnet":"10.0.0.0/24","datacenter_id":"` + dc.ID + `"}`
	resp, err = http.Post(server.URL+"/api/networks", "application/json", bytes.NewBufferString(netBody))
	if err != nil {
		t.Fatalf("create network request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var network model.Network
	json.NewDecoder(resp.Body).Decode(&network)

	// 3. Create device with address
	deviceBody := `{"name":"integration-server","make_model":"Dell R640","datacenter_id":"` + dc.ID + `","addresses":[{"ip":"10.0.0.10","type":"ipv4","network_id":"` + network.ID + `"}],"tags":["web","prod"]}`
	resp, err = http.Post(server.URL+"/api/devices", "application/json", bytes.NewBufferString(deviceBody))
	if err != nil {
		t.Fatalf("create device request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var device model.Device
	json.NewDecoder(resp.Body).Decode(&device)
	if device.ID == "" {
		t.Fatal("device ID should be set")
	}

	// 4. Get device
	resp, err = http.Get(server.URL + "/api/devices/" + device.ID)
	if err != nil {
		t.Fatalf("get device request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var retrieved model.Device
	json.NewDecoder(resp.Body).Decode(&retrieved)
	if retrieved.Name != "integration-server" {
		t.Errorf("expected name 'integration-server', got '%s'", retrieved.Name)
	}

	// 5. Update device
	updateBody := `{"name":"updated-server","make_model":"Dell R640","tags":["updated"]}`
	req, _ := http.NewRequest("PUT", server.URL+"/api/devices/"+device.ID, bytes.NewBufferString(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update device request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	// 6. List devices with filter
	resp, err = http.Get(server.URL + "/api/devices?tags=updated")
	if err != nil {
		t.Fatalf("list devices request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var devices []model.Device
	json.NewDecoder(resp.Body).Decode(&devices)
	if len(devices) != 1 {
		t.Errorf("expected 1 device with tag 'updated', got %d", len(devices))
	}

	// 7. Search devices
	resp, err = http.Get(server.URL + "/api/devices/search?q=updated")
	if err != nil {
		t.Fatalf("search devices request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// 8. Delete device
	req, _ = http.NewRequest("DELETE", server.URL+"/api/devices/"+device.ID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete device request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}

	// 9. Verify deletion
	resp, err = http.Get(server.URL + "/api/devices/" + device.ID)
	if err != nil {
		t.Fatalf("get deleted device request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for deleted device, got %d", resp.StatusCode)
	}
}

func TestAuthMiddlewareIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	authToken := "test-secret-token"
	server, _ := setupIntegrationServer(t, authToken)

	// Request without auth should fail
	resp, err := http.Get(server.URL + "/api/devices")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
	}

	// Request with wrong token should fail
	req, _ := http.NewRequest("GET", server.URL+"/api/devices", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with wrong token, got %d", resp.StatusCode)
	}

	// Request with correct token should succeed
	req, _ = http.NewRequest("GET", server.URL+"/api/devices", nil)
	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with correct token, got %d", resp.StatusCode)
	}
}

func TestSecurityHeadersIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server, _ := setupIntegrationServer(t, "")

	resp, err := http.Get(server.URL + "/api/devices")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		if got := resp.Header.Get(header); got != expected {
			t.Errorf("expected %s: %s, got: %s", header, expected, got)
		}
	}

	// CSP should be present
	if csp := resp.Header.Get("Content-Security-Policy"); csp == "" {
		t.Error("Content-Security-Policy header should be set")
	}
}

func TestDiscoveryWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server, _ := setupIntegrationServer(t, "")

	// 1. Create network
	netBody := `{"name":"Discovery Network","subnet":"192.168.1.0/24"}`
	resp, err := http.Post(server.URL+"/api/networks", "application/json", bytes.NewBufferString(netBody))
	if err != nil {
		t.Fatalf("create network failed: %v", err)
	}
	defer resp.Body.Close()

	var network model.Network
	json.NewDecoder(resp.Body).Decode(&network)

	// 2. Create discovery rule
	ruleBody := `{"network_id":"` + network.ID + `","enabled":true,"scan_type":"quick","interval_hours":24}`
	resp, err = http.Post(server.URL+"/api/discovery/rules", "application/json", bytes.NewBufferString(ruleBody))
	if err != nil {
		t.Fatalf("create rule failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	// 3. List discovery rules
	resp, err = http.Get(server.URL + "/api/discovery/rules")
	if err != nil {
		t.Fatalf("list rules failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var rules []model.DiscoveryRule
	json.NewDecoder(resp.Body).Decode(&rules)
	if len(rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(rules))
	}

	// 4. Start scan
	scanBody := `{"scan_type":"quick"}`
	resp, err = http.Post(server.URL+"/api/discovery/networks/"+network.ID+"/scan", "application/json", bytes.NewBufferString(scanBody))
	if err != nil {
		t.Fatalf("start scan failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 202, got %d: %s", resp.StatusCode, body)
	}

	var scan model.DiscoveryScan
	json.NewDecoder(resp.Body).Decode(&scan)

	// 5. Get scan status
	resp, err = http.Get(server.URL + "/api/discovery/scans/" + scan.ID)
	if err != nil {
		t.Fatalf("get scan failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRelationshipWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server, _ := setupIntegrationServer(t, "")

	// Create parent device
	parentBody := `{"name":"rack-01","make_model":"42U Rack"}`
	resp, err := http.Post(server.URL+"/api/devices", "application/json", bytes.NewBufferString(parentBody))
	if err != nil {
		t.Fatalf("create parent failed: %v", err)
	}
	defer resp.Body.Close()

	var parent model.Device
	json.NewDecoder(resp.Body).Decode(&parent)

	// Create child device
	childBody := `{"name":"server-01","make_model":"Dell R640"}`
	resp, err = http.Post(server.URL+"/api/devices", "application/json", bytes.NewBufferString(childBody))
	if err != nil {
		t.Fatalf("create child failed: %v", err)
	}
	defer resp.Body.Close()

	var child model.Device
	json.NewDecoder(resp.Body).Decode(&child)

	// Add relationship
	relBody := `{"child_id":"` + child.ID + `","type":"contains"}`
	resp, err = http.Post(server.URL+"/api/devices/"+parent.ID+"/relationships", "application/json", bytes.NewBufferString(relBody))
	if err != nil {
		t.Fatalf("add relationship failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	// Get relationships
	resp, err = http.Get(server.URL + "/api/devices/" + parent.ID + "/relationships")
	if err != nil {
		t.Fatalf("get relationships failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var rels []model.DeviceRelationship
	json.NewDecoder(resp.Body).Decode(&rels)
	if len(rels) != 1 {
		t.Errorf("expected 1 relationship, got %d", len(rels))
	}

	// Get related devices
	resp, err = http.Get(server.URL + "/api/devices/" + parent.ID + "/related?type=contains")
	if err != nil {
		t.Fatalf("get related failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var related []model.Device
	json.NewDecoder(resp.Body).Decode(&related)
	if len(related) != 1 || related[0].ID != child.ID {
		t.Errorf("expected child device in related, got %v", related)
	}

	// Delete relationship
	req, _ := http.NewRequest("DELETE", server.URL+"/api/devices/"+parent.ID+"/relationships/"+child.ID+"/contains", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete relationship failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestNetworkPoolWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server, _ := setupIntegrationServer(t, "")

	// Create network
	netBody := `{"name":"Pool Network","subnet":"172.16.0.0/24"}`
	resp, err := http.Post(server.URL+"/api/networks", "application/json", bytes.NewBufferString(netBody))
	if err != nil {
		t.Fatalf("create network failed: %v", err)
	}
	defer resp.Body.Close()

	var network model.Network
	json.NewDecoder(resp.Body).Decode(&network)

	// Create pool
	poolBody := `{"name":"DHCP Pool","network_id":"` + network.ID + `","start_ip":"172.16.0.100","end_ip":"172.16.0.200"}`
	resp, err = http.Post(server.URL+"/api/networks/"+network.ID+"/pools", "application/json", bytes.NewBufferString(poolBody))
	if err != nil {
		t.Fatalf("create pool failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 201, got %d: %s", resp.StatusCode, body)
	}

	var pool model.NetworkPool
	json.NewDecoder(resp.Body).Decode(&pool)

	// Get next available IP
	resp, err = http.Get(server.URL + "/api/pools/" + pool.ID + "/next-ip")
	if err != nil {
		t.Fatalf("get next IP failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var ipResp map[string]string
	json.NewDecoder(resp.Body).Decode(&ipResp)
	if ipResp["ip"] != "172.16.0.100" {
		t.Errorf("expected first IP '172.16.0.100', got '%s'", ipResp["ip"])
	}

	// Get pool heatmap
	resp, err = http.Get(server.URL + "/api/pools/" + pool.ID + "/heatmap")
	if err != nil {
		t.Fatalf("get heatmap failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Get network utilization
	resp, err = http.Get(server.URL + "/api/networks/" + network.ID + "/utilization")
	if err != nil {
		t.Fatalf("get utilization failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestUIConfigEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	server, _ := setupIntegrationServer(t, "")

	resp, err := http.Get(server.URL + "/api/config")
	if err != nil {
		t.Fatalf("get config failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var config UIConfig
	json.NewDecoder(resp.Body).Decode(&config)
	if config.Edition != "oss" {
		t.Errorf("expected edition 'oss', got '%s'", config.Edition)
	}
}
