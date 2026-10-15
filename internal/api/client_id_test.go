package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A client may choose an id on create only if it is a UUID: the web UI and
// exports put ids in URLs and spreadsheets, so "../users/<id>" must never be
// stored.
func TestCreateRejectsIDsThatAreNotUUIDs(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	post := func(path, body string) *httptest.ResponseRecorder {
		req := authReq(httptest.NewRequest("POST", path, bytes.NewBufferString(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		return w
	}

	for _, tc := range []struct{ path, body string }{
		{"/api/v1/devices", `{"id":"../users/test-user-id","name":"decom-sw-old"}`},
		{"/api/v1/devices", `{"id":"=HYPERLINK(\"https://evil\")","name":"a"}`},
		{"/api/v1/devices/bulk", `[{"id":"x?force=1","name":"a"}]`},
		{"/api/v1/networks", `{"id":"..","name":"n","subnet":"10.9.0.0/24"}`},
		{"/api/v1/networks/bulk", `[{"id":"../keys/1","name":"n","subnet":"10.9.1.0/24"}]`},
		{"/api/v1/datacenters", `{"id":"dc/../../users","name":"d"}`},
	} {
		w := post(tc.path, tc.body)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "id must be a UUID") {
			t.Errorf("POST %s %s = %d %s, want 400 on id", tc.path, tc.body, w.Code, w.Body.String())
		}
	}

	// A UUID is kept, as imports need.
	const id = "0190f3c2-7d3e-7a6b-9c1d-2e3f4a5b6c7d"
	if w := post("/api/v1/devices", `{"id":"`+id+`","name":"imported"}`); w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), id) {
		t.Errorf("create with a UUID = %d %s", w.Code, w.Body.String())
	}

	// Bulk networks are now validated like single ones.
	if w := post("/api/v1/networks/bulk", `[{"name":"","subnet":"not-a-cidr"}]`); w.Code != http.StatusBadRequest {
		t.Errorf("bulk network with no name and a bad subnet = %d %s, want 400", w.Code, w.Body.String())
	}
}
