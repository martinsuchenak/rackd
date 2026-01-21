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
	store.CreateDevice(device1)
	store.CreateDevice(device2)

	t.Run("AddRelationship", func(t *testing.T) {
		body := `{"child_id":"` + device2.ID + `","type":"contains"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString(body))
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

	t.Run("AddRelationship_MissingFields", func(t *testing.T) {
		body := `{"child_id":"` + device2.ID + `"}`
		req := httptest.NewRequest("POST", "/api/devices/"+device1.ID+"/relationships", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
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
		if len(rels) != 1 {
			t.Errorf("expected 1 relationship, got %d", len(rels))
		}
	})

	t.Run("GetRelatedDevices", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/devices/"+device1.ID+"/related?type=contains", nil)
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
