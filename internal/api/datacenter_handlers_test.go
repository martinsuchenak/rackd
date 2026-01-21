package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func init() {
	log.Init("console", "error", io.Discard)
}

func setupTestHandler(t *testing.T) (*Handler, storage.ExtendedStorage) {
	t.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	return NewHandler(store), store
}

func TestDatacenterHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("CreateDatacenter", func(t *testing.T) {
		body := `{"name":"DC1","location":"NYC","description":"Test DC"}`
		req := httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("CreateDatacenter_MissingName", func(t *testing.T) {
		body := `{"location":"NYC"}`
		req := httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateDatacenter_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("ListDatacenters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/datacenters", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetDatacenter_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/datacenters/nonexistent", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Create a datacenter for subsequent tests
	var dcID string
	t.Run("CreateAndGet", func(t *testing.T) {
		body := `{"name":"DC2","location":"LA"}`
		req := httptest.NewRequest("POST", "/api/datacenters", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		dcID = resp["id"].(string)

		req = httptest.NewRequest("GET", "/api/datacenters/"+dcID, nil)
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("UpdateDatacenter", func(t *testing.T) {
		body := `{"name":"DC2-Updated"}`
		req := httptest.NewRequest("PUT", "/api/datacenters/"+dcID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDatacenter_NotFound", func(t *testing.T) {
		body := `{"name":"Updated"}`
		req := httptest.NewRequest("PUT", "/api/datacenters/nonexistent", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateDatacenter_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/api/datacenters/"+dcID, bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetDatacenterDevices", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/datacenters/"+dcID+"/devices", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetDatacenterDevices_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/datacenters/nonexistent/devices", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("DeleteDatacenter", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/datacenters/"+dcID, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("DeleteDatacenter_NotFound", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/datacenters/nonexistent", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}
