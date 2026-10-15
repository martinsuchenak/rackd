package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestAuditHandlers_SourceFilterAndDetailEndpoints(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, false, false)
	defer env.close()

	now := time.Now().UTC()
	if err := env.store.CreateAuditLog(context.Background(), &model.AuditLog{
		Action:     "update",
		Resource:   "devices",
		ResourceID: "dev-1",
		UserID:     "test-user-id",
		Username:   "testuser",
		IPAddress:  "127.0.0.1",
		Status:     "success",
		Source:     "cli",
		Timestamp:  now,
	}); err != nil {
		t.Fatalf("failed to seed audit log: %v", err)
	}

	w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/v1/audit?source=cli", nil)))
	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"source\":\"cli\"") {
		t.Fatalf("expected source-filtered response, got %s", w.Body.String())
	}

	var logs []model.AuditLog
	if err := json.Unmarshal(w.Body.Bytes(), &logs); err != nil {
		t.Fatalf("failed to decode audit logs: %v", err)
	}
	if len(logs) == 0 {
		t.Fatal("expected audit logs in list response")
	}

	w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/v1/audit/"+logs[0].ID, nil)))
	if w.Code != 200 {
		t.Fatalf("expected detail status 200, got %d", w.Code)
	}

	w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/v1/audit/export?source=cli&format=csv", nil)))
	if w.Code != 200 {
		t.Fatalf("expected export status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "cli") {
		t.Fatalf("expected CSV export to include source, got %s", w.Body.String())
	}
}

// A time filter that doesn't parse must be rejected: ignoring it returned
// every entry, the opposite of what was asked (found by the black-box pass).
func TestTimeFiltersRejectBadValues(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	for _, path := range []string{"/api/v1/audit", "/api/v1/audit/export", "/api/v1/logs", "/api/v1/logs/export"} {
		for query, want := range map[string]int{
			"?start_time=garbage":                                                http.StatusBadRequest,
			"?end_time=%27+AND+%271%27%3D%271%27+--+":                            http.StatusBadRequest,
			"?start_time=2026-10-15T00:00:00Z&end_time=2026-10-14T00:00:00Z":     http.StatusBadRequest,
			"?start_time=2026-10-14T00:00:00.000Z&end_time=2026-10-15T00:00:00Z": http.StatusOK,
		} {
			w := performRequest(mux, authReq(httptest.NewRequest("GET", path+query, nil)))
			if w.Code != want {
				t.Errorf("GET %s%s: got %d, want %d: %s", path, query, w.Code, want, w.Body.String())
			}
		}
	}
}
