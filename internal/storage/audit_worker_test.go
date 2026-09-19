package storage

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestAuditWorkerStopsAndDrainsOnClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit-worker-test.db")

	baseline := runtime.NumGoroutine()

	s, err := NewSQLiteStorageWithPath(path)
	if err != nil {
		t.Fatalf("NewSQLiteStorageWithPath() error: %v", err)
	}

	// Queue entries directly; the buffer holds 1000 so these sends never block.
	for i := 0; i < 25; i++ {
		s.auditChan <- &model.AuditLog{Action: "test.action", Resource: "test", Status: "success"}
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Close waits on the worker's WaitGroup, so the goroutine is in final
	// teardown when Close returns; give it a moment to disappear.
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > baseline && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > baseline {
		t.Fatalf("audit worker goroutine still running after Close (goroutines: baseline=%d now=%d)", baseline, n)
	}

	// Entries queued before Close must have been drained and persisted.
	check, err := NewSQLiteStorageWithPath(path)
	if err != nil {
		t.Fatalf("reopen for drain check: %v", err)
	}
	defer func() { _ = check.Close() }()

	logs, err := check.ListAuditLogs(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListAuditLogs() error: %v", err)
	}
	if len(logs) != 25 {
		t.Errorf("drained audit entries = %d, want 25", len(logs))
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	s, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage() error: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("first Close() error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close() error: %v", err)
	}
}
