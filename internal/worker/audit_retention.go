package worker

import (
	"context"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// AuditRetentionWorker deletes audit entries older than the retention period,
// once at start and then daily.
type AuditRetentionWorker struct {
	store    storage.AuditStorage
	days     int
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewAuditRetentionWorker keeps days of audit history; days <= 0 keeps it all,
// and Start does nothing.
func NewAuditRetentionWorker(store storage.AuditStorage, days int) *AuditRetentionWorker {
	return &AuditRetentionWorker{store: store, days: days, interval: 24 * time.Hour}
}

func (w *AuditRetentionWorker) Start() {
	if w.days <= 0 {
		log.Info("Audit log retention off; keeping all entries")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			w.prune(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	log.Info("Audit log retention started", "retention_days", w.days)
}

func (w *AuditRetentionWorker) prune(ctx context.Context) {
	if err := w.store.DeleteOldAuditLogs(ctx, w.days); err != nil && ctx.Err() == nil {
		log.Error("Failed to delete old audit entries", "retention_days", w.days, "error", err)
	}
}

func (w *AuditRetentionWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
		w.wg.Wait()
	}
}
