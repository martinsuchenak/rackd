package worker

import (
	"context"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// ReservationExpiryWorker marks reservations past their expiry as expired,
// once at start and then every minute, so their addresses become free again.
// Creating a reservation and picking the next free address already ignore
// lapsed reservations; this brings their status and the pool views up to
// date.
type ReservationExpiryWorker struct {
	store    storage.ReservationStorage
	interval time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewReservationExpiryWorker(store storage.ReservationStorage) *ReservationExpiryWorker {
	return &ReservationExpiryWorker{store: store, interval: time.Minute}
}

func (w *ReservationExpiryWorker) Start() {
	ctx, cancel := context.WithCancel(audit.WithContext(context.Background(), &audit.Context{Source: "system"}))
	w.cancel = cancel
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			w.expire(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (w *ReservationExpiryWorker) expire(ctx context.Context) {
	n, err := w.store.ExpireReservations(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Error("Failed to expire reservations", "error", err)
		}
		return
	}
	if n > 0 {
		log.Info("Expired reservations", "count", n)
	}
}

func (w *ReservationExpiryWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
		w.wg.Wait()
	}
}
