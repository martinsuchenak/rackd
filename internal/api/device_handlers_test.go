package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeviceHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("CreateDevice", func(t *testing.T) {
		body := `{"name":"server1","description":"Test server","make_model":"Dell R640","os":"Ubuntu 22.04"}`
		req := authReq(httptest.NewRequest("POST", "/api/devices", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("CreateDevice_MissingName", func(t *testing.T) {
		body := `{"description":"No name"}`
		req := authReq(httptest.NewRequest("POST", "/api/devices", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateDevice_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/devices", bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateDevice_Unauthenticated", func(t *testing.T) {
		body := `{"name":"server-noauth"}`
		req := httptest.NewRequest("POST", "/api/devices", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected %d, got %d", http.StatusUnauthorized, w.Code)
		}
	})

	t.Run("ListDevices", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/devices", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("ListDevices_WithFilters", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/devices?tags=web&datacenter_id=dc1", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetDevice_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/devices/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	// Create a device for subsequent tests
	var deviceID string
	t.Run("CreateAndGet", func(t *testing.T) {
		body := `{"name":"server2","tags":["web","prod"],"addresses":[{"ip":"10.0.0.1","port":22,"type":"ssh"}]}`
		req := authReq(httptest.NewRequest("POST", "/api/devices", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		deviceID = resp["id"].(string)

		req = authReq(httptest.NewRequest("GET", "/api/devices/"+deviceID, nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("UpdateDevice", func(t *testing.T) {
		body := `{"name":"server2-updated","os":"Debian 12"}`
		req := authReq(httptest.NewRequest("PUT", "/api/devices/"+deviceID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDevice_WithAddresses", func(t *testing.T) {
		body := `{"addresses":[{"ip":"10.0.0.2","port":443,"type":"https"}]}`
		req := authReq(httptest.NewRequest("PUT", "/api/devices/"+deviceID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDevice_WithTagsAndDomains", func(t *testing.T) {
		body := `{"tags":["updated","test"],"domains":["example.com","test.local"]}`
		req := authReq(httptest.NewRequest("PUT", "/api/devices/"+deviceID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDevice_AllFields", func(t *testing.T) {
		body := `{"description":"Updated desc","make_model":"HP DL380","username":"admin","location":"Rack A1"}`
		req := authReq(httptest.NewRequest("PUT", "/api/devices/"+deviceID, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
		}
	})

	t.Run("UpdateDevice_NotFound", func(t *testing.T) {
		body := `{"name":"Updated"}`
		req := authReq(httptest.NewRequest("PUT", "/api/devices/nonexistent", bytes.NewBufferString(body)))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("UpdateDevice_InvalidJSON", func(t *testing.T) {
		req := authReq(httptest.NewRequest("PUT", "/api/devices/"+deviceID, bytes.NewBufferString("invalid")))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("SearchDevices", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/search?q=server&type=devices", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("SearchDevices_MissingQuery", func(t *testing.T) {
		req := authReq(httptest.NewRequest("GET", "/api/search", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("DeleteDevice", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/devices/"+deviceID, nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}
	})

	t.Run("DeleteDevice_NotFound", func(t *testing.T) {
		req := authReq(httptest.NewRequest("DELETE", "/api/devices/nonexistent", nil))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected %d, got %d", http.StatusNotFound, w.Code)
		}
	})
}
