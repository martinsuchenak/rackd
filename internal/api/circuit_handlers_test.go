package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestCircuitHandlers(t *testing.T) {
	h, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("CreateGetUpdateDeleteCircuit", func(t *testing.T) {
		createReq := authReq(httptest.NewRequest("POST", "/api/circuits", bytes.NewBufferString(`{"name":"primary-wan","circuit_id":"CKT-100","provider":"ISP","type":"fiber"}`)))
		createReq.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, createReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created model.Circuit
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode circuit: %v", err)
		}
		if created.ID == "" {
			t.Fatal("expected created circuit ID")
		}

		w = httptest.NewRecorder()
		mux.ServeHTTP(w, authReq(httptest.NewRequest("GET", "/api/circuits/"+created.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/circuits/"+created.ID, bytes.NewBufferString(`{"name":"primary-wan-updated","status":"maintenance"}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, updateReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = httptest.NewRecorder()
		mux.ServeHTTP(w, authReq(httptest.NewRequest("DELETE", "/api/circuits/"+created.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Circuit_InvalidJSONAndValidation", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, authReq(httptest.NewRequest("POST", "/api/circuits", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}

		req := authReq(httptest.NewRequest("POST", "/api/circuits", bytes.NewBufferString(`{"name":"missing-provider","circuit_id":"CKT-101"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}

		req = authReq(httptest.NewRequest("PUT", "/api/circuits/nonexistent", bytes.NewBufferString(`{"status":"invalid"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
			t.Fatalf("expected 404 or 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Circuit_NotFound", func(t *testing.T) {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, authReq(httptest.NewRequest("GET", "/api/circuits/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		req := authReq(httptest.NewRequest("DELETE", "/api/circuits/nonexistent", nil))
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Circuit_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := createAPIUserForStore(t, store, "limited-circuit-user")

		w := httptest.NewRecorder()
		mux.ServeHTTP(w, authReqWithToken(httptest.NewRequest("GET", "/api/circuits", nil), limitedToken))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		req := authReqWithToken(httptest.NewRequest("POST", "/api/circuits", bytes.NewBufferString(`{"name":"limited","circuit_id":"CKT-200","provider":"ISP"}`)), limitedToken)
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
