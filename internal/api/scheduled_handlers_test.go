package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestScheduledScanHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, true, true)
	defer env.close()

	network := &model.Network{ID: "sched-net", Name: "sched-net", Subnet: "10.30.0.0/24"}
	if err := env.store.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("failed to seed network: %v", err)
	}

	profileReq := authReq(httptest.NewRequest("POST", "/api/scan-profiles", bytes.NewBufferString(`{"name":"scheduled-profile","scan_type":"quick","timeout_sec":30,"max_workers":10}`)))
	profileReq.Header.Set("Content-Type", "application/json")
	profileResp := performRequest(env.mux, profileReq)
	if profileResp.Code != http.StatusCreated {
		t.Fatalf("failed to create profile: %d %s", profileResp.Code, profileResp.Body.String())
	}
	var profile model.ScanProfile
	if err := json.Unmarshal(profileResp.Body.Bytes(), &profile); err != nil {
		t.Fatalf("failed to decode profile: %v", err)
	}

	t.Run("CreateGetUpdateDeleteScheduledScan", func(t *testing.T) {
		body := `{"name":"nightly","network_id":"sched-net","profile_id":"` + profile.ID + `","cron_expression":"0 2 * * *","enabled":true}`
		req := authReq(httptest.NewRequest("POST", "/api/scheduled-scans", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created model.ScheduledScan
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode scheduled scan: %v", err)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/scheduled-scans/"+created.ID, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/scheduled-scans/"+created.ID, bytes.NewBufferString(`{"name":"nightly-updated","network_id":"sched-net","profile_id":"`+profile.ID+`","cron_expression":"0 3 * * *","enabled":false}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/scheduled-scans/"+created.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("CreateScheduledScan_InvalidJSON", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/scheduled-scans", bytes.NewBufferString("{"))))
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("ScheduledScan_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/scheduled-scans/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/scheduled-scans/nonexistent", bytes.NewBufferString(`{"name":"missing","network_id":"sched-net","profile_id":"`+profile.ID+`","cron_expression":"0 4 * * *","enabled":true}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/scheduled-scans/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("ScheduledScan_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-scheduled-user")

		req := authReqWithToken(httptest.NewRequest("GET", "/api/scheduled-scans", nil), limitedToken)
		w := performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		createReq := authReqWithToken(httptest.NewRequest("POST", "/api/scheduled-scans", bytes.NewBufferString(`{"name":"limited","network_id":"sched-net","profile_id":"`+profile.ID+`","cron_expression":"0 2 * * *","enabled":true}`)), limitedToken)
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
