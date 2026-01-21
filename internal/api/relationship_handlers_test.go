package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestRelationshipHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Create two devices for relationship tests
	device1 := &model.Device{Name: "parent-device"}
	device2 := &model.Device{Name: "child-device"}
	device3 := &model.Device{Name: "another-child"}
	store.CreateDevice(device1)
	store.CreateDevice(device2)
	store.CreateDevice(device3)

	t.Run("AddRelationship_Contains", func(t *testing.T) {
		body := `{"child_id":"` + device2.ID + `","type":"contains"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("AddRelationship_ConnectedTo", func(t *testing.T) {
		body := `{"child_id":"` + device3.ID + `","type":"connected_to"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("AddRelationship_DependsOn", func(t *testing.T) {
		body := `{"child_id":"` + device2.ID + `","type":"depends_on"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device3.ID+"/relationships", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
		}
	})

	t.Run("AddRelationship_InvalidType", func(t *testing.T) {
		body := `{"child_id":"` + device2.ID + `","type":"invalid"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("AddRelationship_MissingChildID", func(t *testing.T) {
		body := `{"type":"contains"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("AddRelationship_MissingType", func(t *testing.T) {
		body := `{"child_id":"` + device2.ID + `"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("AddRelationship_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString("invalid"))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetRelationships", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/devices/"+device1.ID+"/relationships", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}

		var rels []model.DeviceRelationship
		json.NewDecoder(w.Body).Decode(&rels)
		if len(rels) < 1 {
			t.Errorf("expected at least 1 relationship, got %d", len(rels))
		}
	})

	t.Run("GetRelatedDevices_Contains", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/devices/"+device1.ID+"/related?type=contains", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetRelatedDevices_ConnectedTo", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/devices/"+device1.ID+"/related?type=connected_to", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("GetRelatedDevices_NoFilter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/devices/"+device1.ID+"/related", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("RemoveRelationship", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/api/devices/"+device1.ID+"/relationships/"+device2.ID+"/contains", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Errorf("expected %d, got %d", http.StatusNoContent, w.Code)
		}
	})
}
