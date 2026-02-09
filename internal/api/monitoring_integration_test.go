package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMonitoringEndpoints(t *testing.T) {
	handler, store := setupTestHandler(t)
	defer store.Close()

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	tests := []struct {
		name           string
		path           string
		needsAuth      bool
		expectedStatus int
		checkBody      func(t *testing.T, body string)
	}{
		{
			name:           "Healthz endpoint",
			path:           "/healthz",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if body != "ok" {
					t.Errorf("Expected body 'ok', got %s", body)
				}
			},
		},
		{
			name:           "Readyz endpoint",
			path:           "/readyz",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				if !strings.Contains(body, "healthy") {
					t.Errorf("Expected body to contain 'healthy', got %s", body)
				}
				if !strings.Contains(body, "database") {
					t.Errorf("Expected body to contain 'database', got %s", body)
				}
			},
		},
		{
			name:           "Metrics endpoint",
			path:           "/metrics",
			needsAuth:      true,
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body string) {
				expectedMetrics := []string{
					"http_requests_total",
					"devices_total",
					"networks_total",
					"go_goroutines",
				}
				for _, metric := range expectedMetrics {
					if !strings.Contains(body, metric) {
						t.Errorf("Expected metrics to contain %s", metric)
					}
				}
			},
		},
		{
			name:           "Metrics endpoint unauthenticated",
			path:           "/metrics",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.needsAuth {
				req = authReq(req)
			}
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkBody != nil {
				tt.checkBody(t, w.Body.String())
			}
		})
	}
}
