package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestDashboardHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("GetDashboardStats_Empty", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/dashboard", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var stats model.DashboardStats
		if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Verify basic structure
		if stats.StaleThresholdDays != 7 {
			t.Errorf("expected default stale threshold 7, got %d", stats.StaleThresholdDays)
		}
	})

	t.Run("GetDashboardStats_WithParams", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/dashboard?stale_days=14&recent_limit=5", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var stats model.DashboardStats
		if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if stats.StaleThresholdDays != 14 {
			t.Errorf("expected stale threshold 14, got %d", stats.StaleThresholdDays)
		}
	})

	t.Run("GetDashboardStats_Unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dashboard", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("GetUtilizationTrend_MissingResourceID", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/dashboard/trend", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetUtilizationTrend_NetworkType", func(t *testing.T) {
		// Create a network first
		networkBody := `{"name":"Test Network","subnet":"192.168.1.0/24"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(networkBody)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var network model.Network
		json.Unmarshal(w.Body.Bytes(), &network)

		// Get trend for the network
		req = authReq(httptest.NewRequest("GET", "/api/dashboard/trend?type=network&resource_id="+network.ID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var trend []model.UtilizationTrendPoint
		if err := json.Unmarshal(w.Body.Bytes(), &trend); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Empty trend is valid for new network (may be nil or empty array)
		if len(trend) != 0 {
			t.Errorf("expected empty trend for new network, got %d points", len(trend))
		}
	})

	t.Run("GetUtilizationTrend_PoolType", func(t *testing.T) {
		// Create a network and pool first
		networkBody := `{"name":"Test Network 2","subnet":"192.168.2.0/24"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(networkBody)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var network model.Network
		json.Unmarshal(w.Body.Bytes(), &network)

		// Create pool
		poolBody := `{"network_id":"` + network.ID + `","name":"Test Pool","start_ip":"192.168.2.100","end_ip":"192.168.2.200"}`
		req = authReq(httptest.NewRequest("POST", "/api/networks/"+network.ID+"/pools", bytes.NewBufferString(poolBody)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var pool model.NetworkPool
		json.Unmarshal(w.Body.Bytes(), &pool)

		// Get trend for the pool
		req = authReq(httptest.NewRequest("GET", "/api/dashboard/trend?type=pool&resource_id="+pool.ID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}

		var trend []model.UtilizationTrendPoint
		if err := json.Unmarshal(w.Body.Bytes(), &trend); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
	})

	t.Run("GetUtilizationTrend_WithDaysParam", func(t *testing.T) {
		// Create a network first
		networkBody := `{"name":"Test Network 3","subnet":"192.168.3.0/24"}`
		req := authReq(httptest.NewRequest("POST", "/api/networks", bytes.NewBufferString(networkBody)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var network model.Network
		json.Unmarshal(w.Body.Bytes(), &network)

		// Get trend with days parameter
		req = authReq(httptest.NewRequest("GET", "/api/dashboard/trend?type=network&resource_id="+network.ID+"&days=7", nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("GetUtilizationTrend_Unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/dashboard/trend?resource_id=test", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})
}
