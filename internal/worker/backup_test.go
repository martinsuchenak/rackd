package worker

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The next backup is due interval after the newest one on disk, so a
// restart doesn't take a fresh copy; an overdue one waits out the startup
// delay, and a failure waits before retrying.
func TestBackupWorkerSchedule(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	w := NewBackupWorker(nil, dir, 24*time.Hour, 7)
	w.now = func() time.Time { return now }

	if got := w.untilNext(); got != startupDelay {
		t.Errorf("no backups yet: wait %v, want the startup delay", got)
	}

	last := now.Add(-3 * time.Hour)
	name := "rackd-scheduled-" + last.Format("20060102T150405.000000000Z") + ".db"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := w.untilNext(); got != 21*time.Hour {
		t.Errorf("backup 3h ago: wait %v, want 21h", got)
	}

	now = now.Add(30 * time.Hour) // overdue
	if got := w.untilNext(); got != startupDelay {
		t.Errorf("overdue: wait %v, want the startup delay", got)
	}
	w.failedAt = now
	if got := w.untilNext(); got != time.Hour {
		t.Errorf("after a failure: wait %v, want an hour", got)
	}
	w.failedAt = time.Time{}

	// A backup dated in the future waits one interval from now, not from it.
	future := now.Add(48 * time.Hour)
	name = "rackd-scheduled-" + future.Format("20060102T150405.000000000Z") + ".db"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := w.untilNext(); got != 24*time.Hour {
		t.Errorf("backup dated in the future: wait %v, want 24h", got)
	}
}
