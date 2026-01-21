package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestDiscoveryHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Create a network for discovery tests
	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	store.CreateNetwork(network)

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
}

func TestDiscoveryRuleHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	store.CreateNetwork(network)

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
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	network := &model.Network{Name: "TestNet", Subnet: "192.168.1.0/24"}
	store.CreateNetwork(network)

	discovered := &model.DiscoveredDevice{
		IP:        "192.168.1.100",
		Hostname:  "discovered-host",
		NetworkID: network.ID,
		Status:    "active",
	}
	store.CreateDiscoveredDevice(discovered)

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
}
