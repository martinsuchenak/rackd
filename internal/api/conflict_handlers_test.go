package api

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestConflictHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, false, false)
	defer env.close()

	conflict := &model.Conflict{
		ID:          "phase2-conflict",
		Type:        model.ConflictTypeDuplicateIP,
		Status:      model.ConflictStatusActive,
		Description: "duplicate IP",
		IPAddress:   "10.50.0.10",
		DeviceIDs:   []string{"device-a", "device-b"},
		DetectedAt:  time.Now().UTC(),
	}
	if err := env.store.CreateConflict(context.Background(), conflict); err != nil {
		t.Fatalf("failed to seed conflict: %v", err)
	}

	t.Run("ListGetResolveDeleteConflict", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/conflicts", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/conflicts/"+conflict.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		resolveReq := authReq(httptest.NewRequest("POST", "/api/conflicts/"+conflict.ID+"/resolve", bytes.NewBufferString(`{"conflict_id":"`+conflict.ID+`","keep_device_id":"device-a","notes":"resolved"}`)))
		resolveReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, resolveReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/conflicts/summary", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		detectReq := authReq(httptest.NewRequest("POST", "/api/conflicts/detect?type=duplicate_ip", nil))
		w = performRequest(env.mux, detectReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/conflicts/"+conflict.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Conflict_InvalidJSONAndValidation", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/conflicts/phase2-conflict/resolve", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}

		req := authReq(httptest.NewRequest("POST", "/api/conflicts/phase2-conflict/resolve", bytes.NewBufferString(`{"notes":"missing id"}`)))
		req.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/conflicts/detect?type=invalid", nil)))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Conflict_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/conflicts/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		resolveReq := authReq(httptest.NewRequest("POST", "/api/conflicts/nonexistent/resolve", bytes.NewBufferString(`{"conflict_id":"nonexistent","notes":"missing"}`)))
		resolveReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, resolveReq)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/conflicts/nonexistent", nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Conflict_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-conflict-user")

		w := performRequest(env.mux, authReqWithToken(httptest.NewRequest("GET", "/api/conflicts", nil), limitedToken))
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		req := authReqWithToken(httptest.NewRequest("POST", "/api/conflicts/detect", nil), limitedToken)
		w = performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
