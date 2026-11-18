package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Server behaviour the CLI relies on: discovered-host filters, the NAT
// enabled default and a parseable Retry-After.

func TestListDiscoveredDevices_StatusAndPaging(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ctx := context.Background()

	network := &model.Network{Name: "disc-filter", Subnet: "10.9.0.0/24"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}
	for i, status := range []string{"online", "offline", "online", "online"} {
		d := &model.DiscoveredDevice{IP: fmt.Sprintf("10.9.0.%d", i+1), NetworkID: network.ID, Status: status}
		if err := store.CreateDiscoveredDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	list := func(query string) []model.DiscoveredDevice {
		t.Helper()
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, authReq(httptest.NewRequest("GET", "/api/v1/discovery/devices?"+query, nil)))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", query, w.Code, w.Body.String())
		}
		var out []model.DiscoveredDevice
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	if got := list("status=online"); len(got) != 3 {
		t.Errorf("status=online: got %d, want 3", len(got))
	}
	if got := list("status=OFFLINE"); len(got) != 1 || got[0].Status != "offline" {
		t.Errorf("status=OFFLINE: got %+v", got)
	}
	if got := list("status=online&limit=2"); len(got) != 2 {
		t.Errorf("status=online&limit=2: got %d, want 2", len(got))
	}
	if got := list("limit=2&offset=3"); len(got) != 1 {
		t.Errorf("limit=2&offset=3: got %d, want 1", len(got))
	}
	if got := list("offset=10"); len(got) != 0 {
		t.Errorf("offset past the end: got %d, want 0", len(got))
	}
	if got := list("status=unknown"); got == nil {
		t.Error("an empty result should be [], not null")
	}
}

func TestCreateNATMapping_EnabledDefault(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	create := func(name, extra string) model.NATMapping {
		t.Helper()
		body := `{"name": "` + name + `", "external_ip": "203.0.113.50", "external_port": ` + strconv.Itoa(len(name)+1000) +
			`, "internal_ip": "192.168.1.50", "internal_port": 80, "protocol": "tcp"` + extra + `}`
		req := authReq(httptest.NewRequest("POST", "/api/v1/nat", bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("%s: status %d: %s", name, w.Code, w.Body.String())
		}
		var m model.NATMapping
		if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
			t.Fatal(err)
		}
		return m
	}

	if m := create("omitted", ""); !m.Enabled {
		t.Error("a mapping created without enabled should be enabled")
	}
	if m := create("explicit-false", `, "enabled": false`); m.Enabled {
		t.Error(`"enabled": false should create a disabled mapping`)
	}
	if m := create("explicit-true", `, "enabled": true`); !m.Enabled {
		t.Error(`"enabled": true should create an enabled mapping`)
	}
}

func TestRetryAfterIsSeconds(t *testing.T) {
	limiter := NewRateLimiter(1, 30*time.Second)
	handler := RateLimitMiddleware(limiter, NewTrustedProxyResolver(false, ""))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	var w *httptest.ResponseRecorder
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.9:1234"
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", w.Code)
	}
	secs, err := strconv.Atoi(w.Header().Get("Retry-After"))
	if err != nil || secs < 1 || secs > 30 {
		t.Fatalf("Retry-After = %q, want whole seconds in 1..30", w.Header().Get("Retry-After"))
	}

	if got := retryAfterSeconds(time.Now().Add(-time.Second)); got != "1" {
		t.Errorf("past reset: Retry-After = %q, want 1", got)
	}
	if got := retryAfterSeconds(time.Now().Add(1500 * time.Millisecond)); got != "2" {
		t.Errorf("1.5s: Retry-After = %q, want 2 (rounded up)", got)
	}
}
