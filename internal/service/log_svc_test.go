package service

import (
	"io"
	"testing"

	appLog "github.com/martinsuchenak/rackd/internal/log"
)

func TestLogService_ListGetAndExport(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "logs", "list", true)
	store.setPermission("user-1", "logs", "read", true)
	store.setPermission("user-1", "logs", "export", true)

	appLog.ClearRecentEntries()
	appLog.Init("console", "debug", io.Discard)
	appLog.Info("test log entry", "component", "audit-test")

	svc := NewLogService(store)

	entries, err := svc.List(userContext("user-1"), nil)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one recent log entry")
	}

	entry, err := svc.Get(userContext("user-1"), entries[0].ID)
	if err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if entry.Message != "test log entry" {
		t.Fatalf("expected message to round-trip, got %q", entry.Message)
	}

	data, err := svc.Export(userContext("user-1"), nil, "csv")
	if err != nil {
		t.Fatalf("Export returned unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected CSV export data")
	}
}
