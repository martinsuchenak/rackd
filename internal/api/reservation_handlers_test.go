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

func TestReservationHandlers(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, false, false)
	defer env.close()

	network := &model.Network{ID: "net-phase2", Name: "phase2-net", Subnet: "10.20.0.0/24"}
	if err := env.store.CreateNetwork(context.Background(), network); err != nil {
		t.Fatalf("failed to seed network: %v", err)
	}
	pool := &model.NetworkPool{ID: "pool-phase2", Name: "phase2-pool", NetworkID: network.ID, StartIP: "10.20.0.10", EndIP: "10.20.0.20"}
	if err := env.store.CreateNetworkPool(context.Background(), pool); err != nil {
		t.Fatalf("failed to seed pool: %v", err)
	}

	t.Run("CreateListReleaseDeleteReservation", func(t *testing.T) {
		createReq := authReq(httptest.NewRequest("POST", "/api/reservations", bytes.NewBufferString(`{"pool_id":"pool-phase2","hostname":"reserved-host","expires_in_days":7}`)))
		createReq.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, createReq)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var created model.Reservation
		if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to decode reservation: %v", err)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/reservations?pool_id=pool-phase2", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/reservations/"+created.ID+"/release", nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/reservations/"+created.ID, nil)))
		if w.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", w.Code)
		}
	})

	t.Run("CreateReservation_BadExpiresDays", func(t *testing.T) {
		req := authReq(httptest.NewRequest("POST", "/api/reservations", bytes.NewBufferString(`{"pool_id":"pool-phase2","expires_in_days":366}`)))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(env.mux, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("Reservation_NotFound", func(t *testing.T) {
		w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/reservations/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		updateReq := authReq(httptest.NewRequest("PUT", "/api/reservations/nonexistent", bytes.NewBufferString(`{"hostname":"missing"}`)))
		updateReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, updateReq)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("DELETE", "/api/reservations/nonexistent", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}

		w = performRequest(env.mux, authReq(httptest.NewRequest("POST", "/api/reservations/nonexistent/release", nil)))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("Reservation_ForbiddenWithoutPermission", func(t *testing.T) {
		_, limitedToken := env.createAPIUser(t, "limited-reservation-user")

		req := authReqWithToken(httptest.NewRequest("GET", "/api/reservations", nil), limitedToken)
		w := performRequest(env.mux, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}

		createReq := authReqWithToken(httptest.NewRequest("POST", "/api/reservations", bytes.NewBufferString(`{"pool_id":"pool-phase2","hostname":"limited-host"}`)), limitedToken)
		createReq.Header.Set("Content-Type", "application/json")
		w = performRequest(env.mux, createReq)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}
