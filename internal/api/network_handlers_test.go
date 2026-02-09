package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNetworkHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("CreateNetwork", func(t *testing.T) {
		body := `{"name":"Net1","subnet":"10.0.0.0/24","vlan_id":100}`
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("CreateNetwork_MissingName", func(t *testing.T) {
		body := `{"subnet":"10.0.0.0/24"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNetwork_MissingSubnet", func(t *testing.T) {
		body := `{"name":"Net"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNetwork_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNetwork_Unauthenticated", func(t *testing.T) {
		body := `{"name":"Net-noauth","subnet":"10.0.0.0/24"}`
		req := httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("ListNetworks", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListNetworks_WithFilters", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks?name=Net1&vlan_id=100", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetNetwork_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	var netID string
	t.Run("CreateAndGet", func(t *testing.T) {
		body := `{"name":"Net2","subnet":"192.168.0.0/24","vlan_id":200}`
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		netID = resp["id"].(string)

		req = authReq(httptest.NewRequest("GET", "/api/networks/"+netID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("UpdateNetwork", func(t *testing.T) {
		body := `{"name":"Net2-Updated","vlan_id":201}`
		req := authReq(httptest.NewRequest("PUT", "/api/networks/"+netID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateNetwork_NotFound", func(t *testing.T) {
		body := `{"name":"Updated"}`
		req := authReq(httptest.NewRequest("PUT", "/api/networks/nonexistent", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateNetwork_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("PUT", "/api/networks/"+netID, bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetNetworkDevices", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks/"+netID+"/devices", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetNetworkDevices_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks/nonexistent/devices", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("GetNetworkUtilization", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks/"+netID+"/utilization", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("GetNetworkUtilization_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks/nonexistent/utilization", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("ListNetworkPools_Empty", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks/"+netID+"/pools", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListNetworkPools_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/networks/nonexistent/pools", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	var poolID string
	t.Run("CreateNetworkPool", func(t *testing.T) {
		body := `{"name":"Pool1","start_ip":"192.168.0.10","end_ip":"192.168.0.50"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks/"+netID+"/pools", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		poolID = resp["id"].(string)
	})

	t.Run("CreateNetworkPool_MissingName", func(t *testing.T) {
		body := `{"start_ip":"192.168.0.10","end_ip":"192.168.0.50"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks/"+netID+"/pools", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNetworkPool_MissingIPs", func(t *testing.T) {
		body := `{"name":"Pool"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks/"+netID+"/pools", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateNetworkPool_NetworkNotFound", func(t *testing.T) {
		body := `{"name":"Pool","start_ip":"10.0.0.1","end_ip":"10.0.0.10"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks/nonexistent/pools", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("CreateNetworkPool_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/networks/"+netID+"/pools", bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetNetworkPool", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/pools/"+poolID, nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetNetworkPool_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/pools/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateNetworkPool", func(t *testing.T) {
		body := `{"name":"Pool1-Updated","tags":["prod","web"]}`
		req := authReq(httptest.NewRequest("PUT", "/api/pools/"+poolID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateNetworkPool_NotFound", func(t *testing.T) {
		body := `{"name":"Updated"}`
		req := authReq(httptest.NewRequest("PUT", "/api/pools/nonexistent", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateNetworkPool_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("PUT", "/api/pools/"+poolID, bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetNextIP", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/pools/"+poolID+"/next-ip", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("GetNextIP_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/pools/nonexistent/next-ip", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("GetPoolHeatmap", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/pools/"+poolID+"/heatmap", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("GetPoolHeatmap_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/pools/nonexistent/heatmap", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteNetworkPool", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/pools/"+poolID, nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("DeleteNetworkPool_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/pools/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteNetwork", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/networks/"+netID, nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("DeleteNetwork_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/networks/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}
