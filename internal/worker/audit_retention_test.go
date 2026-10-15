package worker

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestAuditRetentionDeletesOnlyOldEntries(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	for _, age := range []time.Duration{100 * 24 * time.Hour, time.Hour} {
		if err := store.CreateAuditLog(ctx, &model.AuditLog{Timestamp: time.Now().UTC().Add(-age), Action: "update", Resource: "device", Status: "success"}); err != nil {
			t.Fatal(err)
		}
	}

	NewAuditRetentionWorker(store, 90).prune(ctx)
	logs, err := store.ListAuditLogs(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 || time.Since(logs[0].Timestamp) > 24*time.Hour {
		t.Errorf("after pruning: %+v, want only the recent entry", logs)
	}

	// 0 keeps everything: Start does nothing.
	w := NewAuditRetentionWorker(store, 0)
	w.Start()
	w.Stop()
}
