package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAPIServesV1AndDeprecatedAlias(t *testing.T) {
	mux := http.NewServeMux()
	HandleAPI(mux, "GET /widgets/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.PathValue("id")))
	})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/widgets/w%201", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "w 1" {
		t.Fatalf("v1: %d %q", rec.Code, rec.Body.String())
	}
	for _, h := range []string{"Deprecation", "Sunset", "Link"} {
		if v := rec.Header().Get(h); v != "" {
			t.Errorf("v1 response has %s: %q", h, v)
		}
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/widgets/w%201", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "w 1" {
		t.Fatalf("legacy: %d %q", rec.Code, rec.Body.String())
	}
	if got, want := rec.Header().Get("Deprecation"), "@1791936000"; got != want {
		t.Errorf("Deprecation = %q, want %q", got, want)
	}
	if got, want := rec.Header().Get("Sunset"), "Wed, 14 Apr 2027 00:00:00 GMT"; got != want {
		t.Errorf("Sunset = %q, want %q", got, want)
	}
	if got, want := rec.Header().Get("Link"), `</api/v1/widgets/w%201>; rel="successor-version"`; got != want {
		t.Errorf("Link = %q, want %q", got, want)
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/widgets/1", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("wrong method on v1 = %d, want 405", rec.Code)
	}
}

func TestHandleAPIRejectsBadPattern(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("pattern without a method should panic")
		}
	}()
	HandleAPI(http.NewServeMux(), "/widgets", func(http.ResponseWriter, *http.Request) {})
}

func TestRegisteredRoutesVersioning(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<html>")) })

	for _, tc := range []struct {
		method, path string
		status       int
		deprecated   bool
	}{
		{"GET", "/api/v1/datacenters", http.StatusOK, false},
		{"GET", "/api/datacenters", http.StatusOK, true},
		{"GET", "/api/v1/no-such-endpoint", http.StatusNotFound, false},
		{"GET", "/api/no-such-endpoint", http.StatusNotFound, false},
		{"POST", "/api/v1/dashboard", http.StatusMethodNotAllowed, false},
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, authReq(httptest.NewRequest(tc.method, tc.path, nil)))
		if rec.Code != tc.status {
			t.Errorf("%s %s = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.status, rec.Body.String())
		}
		if got := rec.Header().Get("Deprecation") != ""; got != tc.deprecated {
			t.Errorf("%s %s deprecated = %v, want %v", tc.method, tc.path, got, tc.deprecated)
		}
		if rec.Code == http.StatusNotFound && rec.Header().Get("Content-Type") != "application/json" {
			t.Errorf("%s %s 404 is not JSON: %q", tc.method, tc.path, rec.Body.String())
		}
	}
}
