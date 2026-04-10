package api

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	appLog "github.com/martinsuchenak/rackd/internal/log"
)

func TestLogHandlers_ListGetAndExport(t *testing.T) {
	env := setupExtendedTestHandler(t, false, false, false, false)
	defer env.close()

	appLog.ClearRecentEntries()
	appLog.Init("console", "debug", io.Discard)
	appLog.Info("phase2-log-entry", "component", "api-test")

	w := performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/logs?source=api-test", nil)))
	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "phase2-log-entry") {
		t.Fatalf("expected logs response to include test entry, got %s", w.Body.String())
	}

	var entries []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to decode logs response: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected log entries in list response")
	}

	id, _ := entries[0]["id"].(string)
	w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/logs/"+id, nil)))
	if w.Code != 200 {
		t.Fatalf("expected detail status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "phase2-log-entry") {
		t.Fatalf("expected detail response to include test entry, got %s", w.Body.String())
	}

	w = performRequest(env.mux, authReq(httptest.NewRequest("GET", "/api/logs/export?source=api-test&format=csv", nil)))
	if w.Code != 200 {
		t.Fatalf("expected export status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "phase2-log-entry") {
		t.Fatalf("expected CSV export to include test entry, got %s", w.Body.String())
	}
}
