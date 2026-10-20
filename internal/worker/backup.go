package worker

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// startupDelay is how long the backup worker waits after startup before an
// overdue backup, so it doesn't compete with the server coming up.
const startupDelay = time.Minute

// BackupWorker takes a verified database backup every interval and keeps the
// newest keep. The next backup is due interval after the newest one on disk,
// so restarts (a rescheduled job, say) don't each take a fresh copy.
type BackupWorker struct {
	db       *sql.DB
	dir      string
	interval time.Duration
	keep     int
	now      func() time.Time
	delay    time.Duration
	// failedAt is when the last backup failed, if it did since the newest
	// one on disk; a retry waits, so a full disk isn't hit every minute.
	failedAt time.Time

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewBackupWorker creates a backup worker for db, writing into dir.
func NewBackupWorker(db *sql.DB, dir string, interval time.Duration, keep int) *BackupWorker {
	return &BackupWorker{db: db, dir: dir, interval: interval, keep: keep, now: time.Now, delay: startupDelay}
}

// Start begins taking backups in the background.
func (w *BackupWorker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.wg.Add(1)
	go w.run(ctx)
	log.Info("Scheduled backups enabled", "interval", w.interval, "keep", w.keep, "dir", w.dir)
}

// Stop waits for a backup in progress to finish and stops the worker.
func (w *BackupWorker) Stop() {
	if w.cancel == nil {
		return
	}
	w.cancel()
	w.wg.Wait()
}

func (w *BackupWorker) run(ctx context.Context) {
	defer w.wg.Done()
	for {
		timer := time.NewTimer(w.untilNext())
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		w.backup(ctx)
	}
}

// untilNext is how long until the next backup: interval after the newest one
// on disk, or after a failure the shorter of interval and an hour, but no
// sooner than the startup delay.
func (w *BackupWorker) untilNext() time.Duration {
	now := w.now()
	wait := w.delay
	if last, ok := storage.LatestScheduledBackup(w.dir); ok {
		if due := last.Add(w.interval).Sub(now); due > wait {
			wait = due
		}
	}
	if !w.failedAt.IsZero() {
		if due := w.failedAt.Add(min(w.interval, time.Hour)).Sub(now); due > wait {
			wait = due
		}
	}
	return wait
}

func (w *BackupWorker) backup(ctx context.Context) {
	start := w.now()
	info, err := storage.ScheduledBackup(ctx, w.db, w.dir, w.keep, start)
	if info == nil {
		if ctx.Err() == nil {
			w.failedAt = start
			log.Error("Scheduled backup failed", "dir", w.dir, "error", err)
		}
		return
	}
	w.failedAt = time.Time{}
	log.Info("Scheduled backup taken", "path", info.Path, "size_bytes", info.SizeBytes, "duration", time.Since(start).Round(time.Millisecond))
	if err != nil {
		log.Warn("Scheduled backup", "error", err)
	}
}
