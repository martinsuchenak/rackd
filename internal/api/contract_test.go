package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestContract(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		req := authReq(httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	t.Run("duplicate role name is 409", func(t *testing.T) {
		if w := do("POST", "/api/v1/roles", `{"name":"helpdesk"}`); w.Code != http.StatusCreated {
			t.Fatalf("create: %d %s", w.Code, w.Body.String())
		}
		w := do("POST", "/api/v1/roles", `{"name":"helpdesk"}`)
		if w.Code != http.StatusConflict {
			t.Errorf("duplicate: %d %s, want 409", w.Code, w.Body.String())
		}
	})

	t.Run("reservation expiry takes either form", func(t *testing.T) {
		w := do("POST", "/api/v1/networks", `{"name":"n","subnet":"10.50.0.0/24"}`)
		var network struct{ ID string }
		_ = json.Unmarshal(w.Body.Bytes(), &network)
		w = do("POST", "/api/v1/networks/"+network.ID+"/pools", `{"name":"p","start_ip":"10.50.0.10","end_ip":"10.50.0.20"}`)
		var pool struct{ ID string }
		_ = json.Unmarshal(w.Body.Bytes(), &pool)
		if pool.ID == "" {
			t.Fatalf("pool: %d %s", w.Code, w.Body.String())
		}

		w = do("POST", "/api/v1/reservations", `{"pool_id":"`+pool.ID+`","expires_in_days":10}`)
		var res struct {
			ID        string     `json:"id"`
			ExpiresAt *time.Time `json:"expires_at"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if w.Code != http.StatusCreated || res.ExpiresAt == nil {
			t.Fatalf("create with expires_in_days: %d %s", w.Code, w.Body.String())
		}

		// Update with days, as the CLI sends, now takes effect.
		w = do("PUT", "/api/v1/reservations/"+res.ID, `{"expires_in_days":30}`)
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if w.Code != http.StatusOK || res.ExpiresAt == nil || time.Until(*res.ExpiresAt) < 29*24*time.Hour {
			t.Errorf("update with expires_in_days: %d %s", w.Code, w.Body.String())
		}
		if w := do("PUT", "/api/v1/reservations/"+res.ID, `{"expires_in_days":5,"expires_at":"2030-01-01T00:00:00Z"}`); w.Code != http.StatusBadRequest {
			t.Errorf("both forms: %d, want 400", w.Code)
		}

		// The expiration can be removed.
		w = do("PUT", "/api/v1/reservations/"+res.ID, `{"clear_expiry":true}`)
		res.ExpiresAt = nil
		var cleared map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &cleared)
		if w.Code != http.StatusOK || cleared["expires_at"] != nil {
			t.Errorf("clear_expiry: %d %s", w.Code, w.Body.String())
		}
	})
}
