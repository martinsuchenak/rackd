package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
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

	w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/audit?source=cli", nil)))
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

	w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/audit/"+logs[0].ID, nil)))
	if w.Code != 200 {
		t.Fatalf("expected detail status 200, got %d", w.Code)
	}

	w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/audit/export?source=cli&format=csv", nil)))
	if w.Code != 200 {
		t.Fatalf("expected export status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "cli") {
		t.Fatalf("expected CSV export to include source, got %s", w.Body.String())
	}
}
