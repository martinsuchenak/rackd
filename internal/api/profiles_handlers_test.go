package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestProfileHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, true, false)
	defer env.close()

	t.Run("CreateGetUpdateDeleteProfile", func(t *testing.T) {
		body := `{"name":"phase2-profile","scan_type":"quick","timeout_sec":30,"max_workers":10}`
		req := authReq(httptest.NewRequest("POST", "/api/scan-profiles", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created model.ScanProfile
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode profile: %v", err)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/scan-profiles/"+created.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/scan-profiles/"+created.ID, bytes.NewBufferString(`{"name":"phase2-profile-updated","scan_type":"quick","timeout_sec":45,"max_workers":10}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/scan-profiles/"+created.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("CreateProfile_InvalidJSON", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/scan-profiles", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Profile_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/scan-profiles/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/scan-profiles/nonexistent", bytes.NewBufferString(`{"name":"missing","scan_type":"quick","timeout_sec":30,"max_workers":10}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/scan-profiles/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("Profile_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-profile-user")

		req := authReqWithToken(httptest.NewRequest("GET", "/api/scan-profiles", nil), limitedToken)
		w := performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		createReq := authReqWithToken(httptest.NewRequest("POST", "/api/scan-profiles", bytes.NewBufferString(`{"name":"limited","scan_type":"quick","timeout_sec":30,"max_workers":10}`)), limitedToken)
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
