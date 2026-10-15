package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Auto-assign and claiming through the API (found by the Strix run): one
// reservation made auto-assign fail with "high contention", and a device
// taking a reserved address left the reservation active.
func TestReservationLifecycleThroughAPI(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	post := func(path string, body any, out any) {
		t.Helper()
		b, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := performRequest(mux, authReq(req))
		if w.Code != http.StatusCreated {
			t.Fatalf("POST %s: %d %s", path, w.Code, w.Body.String())
		}
		if out != nil {
			_ = json.Unmarshal(w.Body.Bytes(), out)
		}
	}

	var network model.Network
	post("/api/v1/networks", map[string]any{"name": "res-net", "subnet": "10.42.0.0/24"}, &network)
	var pool model.NetworkPool
	post("/api/v1/networks/"+network.ID+"/pools", map[string]any{"name": "res-pool", "start_ip": "10.42.0.10", "end_ip": "10.42.0.20"}, &pool)

	var first model.Reservation
	post("/api/v1/reservations", map[string]any{"pool_id": pool.ID, "ip_address": "10.42.0.10"}, &first)
	var auto model.Reservation
	post("/api/v1/reservations", map[string]any{"pool_id": pool.ID}, &auto)
	if auto.IPAddress != "10.42.0.11" {
		t.Fatalf("auto-assigned %s, want 10.42.0.11", auto.IPAddress)
	}

	post("/api/v1/devices", map[string]any{
		"name":      "takes-reserved",
		"addresses": []map[string]any{{"ip": "10.42.0.10", "type": "ipv4", "pool_id": pool.ID, "network_id": network.ID}},
	}, nil)
	w := performRequest(mux, authReq(httptest.NewRequest("GET", "/api/v1/reservations/"+first.ID, nil)))
	var got model.Reservation
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Status != model.ReservationStatusClaimed {
		t.Fatalf("reservation for the device's address is %q, want claimed", got.Status)
	}
}
