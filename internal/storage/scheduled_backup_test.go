package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScheduledBackupsAreVerifiedAndPruned(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	dir := t.TempDir()
	// Other backups in the same directory are never pruned.
	for _, name := range []string{"rackd-pre-migrate-20260101T000000.000000000Z-up.db", "mine.db"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	if _, ok := LatestScheduledBackup(dir); ok {
		t.Fatal("found a scheduled backup in an empty schedule")
	}
	start := time.Date(2026, 10, 20, 1, 0, 0, 0, time.UTC)
	for i := range 4 {
		info, err := ScheduledBackup(ctx, s.DB(), dir, 3, start.Add(time.Duration(i)*time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		if info.Migrations == 0 {
			t.Errorf("backup %d has no migration history", i)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := []string{
		"mine.db",
		"rackd-pre-migrate-20260101T000000.000000000Z-up.db",
		"rackd-scheduled-20261020T020000.000000000Z.db",
		"rackd-scheduled-20261020T030000.000000000Z.db",
		"rackd-scheduled-20261020T040000.000000000Z.db",
	}
	if len(names) != len(want) {
		t.Fatalf("files %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("files %v, want %v", names, want)
		}
	}
	if last, ok := LatestScheduledBackup(dir); !ok || !last.Equal(start.Add(3*time.Hour)) {
		t.Errorf("latest = %v %v, want %v", last, ok, start.Add(3*time.Hour))
	}
}
