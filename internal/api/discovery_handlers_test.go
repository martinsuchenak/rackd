package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type mockScanner struct {
	store storage.ExtendedStorage
}

func (m *mockScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
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

func (m *mockScanner) GetScanStatus(scanID string) (*model.DiscoveryScan, error) {
	return m.store.GetDiscoveryScan(scanID)
}

func (m *mockScanner) CancelScan(scanID string) error {
	return nil
}

func setupTestHandlerWithScanner(t *testing.T) (*Handler, storage.ExtendedStorage, discovery.Scanner) {
	t.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	scanner := &mockScanner{store: store}
	return NewHandler(store, scanner), store, scanner
}

func TestDiscoveryHandlers(t *testing.T) {
	h, store, _ := setupTestHandlerWithScanner(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Create a network for discovery tests
	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	store.CreateNetwork(context.Background(), network)

	var scanID string

	t.Run("StartScan", func(t *testing.T) {
		body := `{"scan_type":"quick"}`
		req := httptest.NewRequest("POST", "/api/discovery/networks/"+network.ID+"/scan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected %d, got %d: %s", http.StatusAccepted, w.Code, w.Body.String())
		}

		var scan model.DiscoveryScan
		json.NewDecoder(w.Body).Decode(&scan)
		scanID = scan.ID
	})

	t.Run("StartScan_Full", func(t *testing.T) {
		body := `{"scan_type":"full"}`
		req := httptest.NewRequest("POST", "/api/discovery/networks/"+network.ID+"/scan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected %d, got %d", http.StatusAccepted, w.Code)
		}
	})

	t.Run("StartScan_Deep", func(t *testing.T) {
		body := `{"scan_type":"deep"}`
		req := httptest.NewRequest("POST", "/api/discovery/networks/"+network.ID+"/scan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected %d, got %d", http.StatusAccepted, w.Code)
		}
	})

	t.Run("StartScan_DefaultType", func(t *testing.T) {
		body := `{}`
		req := httptest.NewRequest("POST", "/api/discovery/networks/"+network.ID+"/scan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Errorf("expected %d, got %d", http.StatusAccepted, w.Code)
		}
	})

	t.Run("StartScan_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/discovery/networks/"+network.ID+"/scan", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		// Invalid JSON defaults to quick scan type
		if w.Code != http.StatusAccepted {
			t.Errorf("expected %d, got %d", http.StatusAccepted, w.Code)
		}
	})

	t.Run("StartScan_InvalidType", func(t *testing.T) {
		body := `{"scan_type":"invalid"}`
		req := httptest.NewRequest("POST", "/api/discovery/networks/"+network.ID+"/scan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("ListScans", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/scans", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListScans_WithNetworkFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/scans?network_id="+network.ID, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetScan", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/scans/"+scanID, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("GetScan_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/scans/nonexistent", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("ListDiscoveredDevices", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/devices", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListDiscoveredDevices_WithNetworkFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/devices?network_id="+network.ID, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestDiscoveryRuleHandlers(t *testing.T) {
	h, store, _ := setupTestHandlerWithScanner(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	store.CreateNetwork(context.Background(), network)

	var ruleID string

	t.Run("CreateDiscoveryRule", func(t *testing.T) {
		body := `{"network_id":"` + network.ID + `","enabled":true,"scan_type":"full","interval_hours":24}`
		req := httptest.NewRequest("POST", "/api/discovery/rules", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var rule model.DiscoveryRule
		json.NewDecoder(w.Body).Decode(&rule)
		ruleID = rule.ID
	})

	t.Run("CreateDiscoveryRule_WithDefaults", func(t *testing.T) {
		body := `{"network_id":"` + network.ID + `","enabled":true}`
		req := httptest.NewRequest("POST", "/api/discovery/rules", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var rule model.DiscoveryRule
		json.NewDecoder(w.Body).Decode(&rule)
		if rule.ScanType != "quick" {
			t.Errorf("expected default scan_type 'quick', got '%s'", rule.ScanType)
		}
		if rule.IntervalHours != 24 {
			t.Errorf("expected default interval_hours 24, got %d", rule.IntervalHours)
		}
	})

	t.Run("CreateDiscoveryRule_MissingNetworkID", func(t *testing.T) {
		body := `{"enabled":true}`
		req := httptest.NewRequest("POST", "/api/discovery/rules", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateDiscoveryRule_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/discovery/rules", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("ListDiscoveryRules", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/rules", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetDiscoveryRule", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/rules/"+ruleID, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("GetDiscoveryRule_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/rules/nonexistent", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateDiscoveryRule", func(t *testing.T) {
		body := `{"enabled":false,"interval_hours":12}`
		req := httptest.NewRequest("PUT", "/api/discovery/rules/"+ruleID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDiscoveryRule_WithScanType", func(t *testing.T) {
		body := `{"enabled":true,"scan_type":"deep","exclude_ips":"192.168.1.1,192.168.1.2"}`
		req := httptest.NewRequest("PUT", "/api/discovery/rules/"+ruleID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDiscoveryRule_NotFound", func(t *testing.T) {
		body := `{"enabled":false}`
		req := httptest.NewRequest("PUT", "/api/discovery/rules/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateDiscoveryRule_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/discovery/rules/"+ruleID, bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("DeleteDiscoveryRule", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/discovery/rules/"+ruleID, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("DeleteDiscoveryRule_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/discovery/rules/nonexistent", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}

func TestPromoteDevice(t *testing.T) {
	h, store, _ := setupTestHandlerWithScanner(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	store.CreateNetwork(context.Background(), network)

	discovered := &model.DiscoveredDevice{
		IP:        "192.168.1.100",
		Hostname:  "discovered-host",
		NetworkID: network.ID,
		Status:    "active",
	}
	store.CreateDiscoveredDevice(context.Background(), discovered)

	t.Run("PromoteDevice", func(t *testing.T) {
		body := `{"name":"promoted-device","make_model":"Dell R640"}`
		req := httptest.NewRequest("POST", "/api/discovery/devices/"+discovered.ID+"/promote", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var device model.Device
		json.NewDecoder(w.Body).Decode(&device)
		if device.Name != "promoted-device" {
			t.Errorf("expected name 'promoted-device', got '%s'", device.Name)
		}
	})

	t.Run("PromoteDevice_WithDatacenter", func(t *testing.T) {
		// Create a datacenter first
		dc := &model.Datacenter{Name: "Test DC", Location: "NYC"}
		store.CreateDatacenter(context.Background(), dc)

		// Create another discovered device
		discovered2 := &model.DiscoveredDevice{
			IP:        "192.168.1.101",
			Hostname:  "discovered-host-2",
			NetworkID: network.ID,
			Status:    "active",
		}
		store.CreateDiscoveredDevice(context.Background(), discovered2)

		body := `{"name":"promoted-device-2","make_model":"HP DL380","datacenter_id":"` + dc.ID + `"}`
		req := httptest.NewRequest("POST", "/api/discovery/devices/"+discovered2.ID+"/promote", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("PromoteDevice_UseHostname", func(t *testing.T) {
		// Create another discovered device
		discovered3 := &model.DiscoveredDevice{
			IP:        "192.168.1.102",
			Hostname:  "auto-named-host",
			NetworkID: network.ID,
			Status:    "active",
		}
		store.CreateDiscoveredDevice(context.Background(), discovered3)

		// Empty name should use hostname
		body := `{}`
		req := httptest.NewRequest("POST", "/api/discovery/devices/"+discovered3.ID+"/promote", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var device model.Device
		json.NewDecoder(w.Body).Decode(&device)
		if device.Name != "auto-named-host" {
			t.Errorf("expected name 'auto-named-host', got '%s'", device.Name)
		}
	})

	t.Run("PromoteDevice_NotFound", func(t *testing.T) {
		body := `{"name":"test"}`
		req := httptest.NewRequest("POST", "/api/discovery/devices/nonexistent/promote", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("PromoteDevice_InvalidJSON", func(t *testing.T) {
		// Create another discovered device
		discovered4 := &model.DiscoveredDevice{
			IP:        "192.168.1.103",
			Hostname:  "test-host",
			NetworkID: network.ID,
			Status:    "active",
		}
		store.CreateDiscoveredDevice(context.Background(), discovered4)

		req := httptest.NewRequest("POST", "/api/discovery/devices/"+discovered4.ID+"/promote", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
}
