package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestAuditService_GetMapsNotFoundAndExportDefaultsToJSON(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "audit", "list", true)
	store.setPermission("user-1", "audit", "read", true)
	store.setPermission("user-1", "audit", "export", true)
	store.auditLogs = []model.AuditLog{{ID: "log-1", Action: "create"}}
	svc := NewAuditService(store)

	_, err := svc.Get(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for missing audit log, got %v", err)
	}

	data, err := svc.Export(userContext("user-1"), nil, "unsupported")
	if err != nil {
		t.Fatalf("Export returned unexpected error: %v", err)
	}
	if len(data) == 0 || data[0] != '[' {
		t.Fatalf("expected JSON export fallback, got %q", string(data))
	}
}
