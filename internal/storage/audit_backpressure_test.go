package storage

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/martinsuchenak/rackd/internal/audit"
)

// A burst bigger than the audit queue dropped entries while the writes they
// recorded succeeded (found by the Strix run). Every entry must land.
func TestAuditBurstLosesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rackd.db")
	s, err := NewSQLiteStorageWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := audit.WithContext(context.Background(), &audit.Context{Source: "test"})

	const writers, perWriter = 20, 250 // 5,000 entries, five times the queue
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				s.auditLog(ctx, "update", "device", "burst", nil)
			}
		}()
	}
	wg.Wait()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	s, err = NewSQLiteStorageWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE resource_id = 'burst'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != writers*perWriter {
		t.Fatalf("%d audit entries recorded, want %d", n, writers*perWriter)
	}
}
