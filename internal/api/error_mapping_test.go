package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Bad input and references to missing records are the client's mistake: they
// must get a 4xx, never a 500 (found by the black-box pass).
func TestBadInputIsNotAServerError(t *testing.T) {
	env := setupExtendedTestHandler(t, false, true, true, true)
	defer env.close()

	do := func(method, path, body string) *httptest.ResponseRecorder {
		var req *http.Request
		if body == "" {
			req = httptest.NewRequest(method, path, nil)
		} else {
			req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
		}
		return performRequest(env.mux, authReq(req))
	}

	device := do("POST", "/api/v1/devices", `{"name":"err-map-device"}`)
	if device.Code != http.StatusCreated {
		t.Fatalf("create device: %d %s", device.Code, device.Body.String())
	}
	var dev model.Device
	_ = json.Unmarshal(device.Body.Bytes(), &dev)
	missing := "01a13e45-0000-7000-8000-000000000000"

	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"credential with unknown type", "POST", "/api/v1/credentials", `{"name":"c","type":"bogus"}`, http.StatusBadRequest},
		{"credential missing its secret", "POST", "/api/v1/credentials", `{"name":"c","type":"snmp_v2c"}`, http.StatusBadRequest},
		{"scan profile with unknown type", "POST", "/api/v1/scan-profiles", `{"name":"p","scan_type":"bogus"}`, http.StatusBadRequest},
		{"scan profile with too many workers", "POST", "/api/v1/scan-profiles", `{"name":"p","scan_type":"quick","max_workers":1000}`, http.StatusBadRequest},
		{"scan profile without a name", "POST", "/api/v1/scan-profiles", `{"scan_type":"quick"}`, http.StatusBadRequest},
		{"scheduled scan without a profile", "POST", "/api/v1/scheduled-scans", `{"name":"s","network_id":"n","cron_expression":"0 * * * *"}`, http.StatusBadRequest},
		{"grant role without ids", "POST", "/api/v1/users/grant-role", `{}`, http.StatusBadRequest},
		{"revoke role without ids", "POST", "/api/v1/users/revoke-role", `{}`, http.StatusBadRequest},
		{"relationship to a missing device", "POST", "/api/v1/devices/" + dev.ID + "/relationships", `{"child_id":"` + missing + `","type":"contains"}`, http.StatusBadRequest},
		{"scan of a missing network", "POST", "/api/v1/discovery/networks/" + missing + "/scan", `{"scan_type":"quick"}`, http.StatusNotFound},
		{"delete a missing user", "DELETE", "/api/v1/users/" + missing, "", http.StatusNotFound},
		{"search with a NUL byte", "GET", "/api/v1/search?q=a%00b", "", http.StatusOK},
		{"device search with a NUL byte", "GET", "/api/v1/devices/search?q=%00", "", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := do(tt.method, tt.path, tt.body)
			if w.Code != tt.want {
				t.Fatalf("got %d, want %d: %s", w.Code, tt.want, w.Body.String())
			}
		})
	}

	t.Run("scan profile defaults the optional fields", func(t *testing.T) {
		w := do("POST", "/api/v1/scan-profiles", `{"name":"defaults","scan_type":"quick"}`)
		if w.Code != http.StatusCreated {
			t.Fatalf("got %d: %s", w.Code, w.Body.String())
		}
		var p model.ScanProfile
		_ = json.Unmarshal(w.Body.Bytes(), &p)
		// The scan_profiles column defaults, which the service applies too.
		if p.TimeoutSec != 30 || p.MaxWorkers != 10 {
			t.Fatalf("got timeout %d, workers %d", p.TimeoutSec, p.MaxWorkers)
		}
	})
}
