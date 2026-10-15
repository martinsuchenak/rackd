package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsHandler(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("needs metrics:read", func(t *testing.T) {
		_, token := createAPIUserForStore(t, store, "metrics-no-role")
		w := performRequest(mux, authReqWithToken(httptest.NewRequest("GET", "/metrics", nil), token))
		if w.Code != http.StatusForbidden {
			t.Fatalf("user without metrics:read got %d, want 403", w.Code)
		}
		w = performRequest(mux, httptest.NewRequest("GET", "/metrics", nil))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous got %d, want 401", w.Code)
		}
	})

	w := performRequest(mux, authReq(httptest.NewRequest("GET", "/metrics", nil)))

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected Content-Type text/plain, got %s", contentType)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty metrics output")
	}

	// Check for key metrics
	expectedMetrics := []string{
		"http_requests_total",
		"devices_total",
		"networks_total",
		"datacenters_total",
		"go_goroutines",
		"process_uptime_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Errorf("Expected metrics to contain %s", metric)
		}
	}
}
