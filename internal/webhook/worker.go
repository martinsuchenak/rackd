package webhook

import (
	"context"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

const (
	// maxConcurrentDeliveries limits the number of simultaneous webhook deliveries
	maxConcurrentDeliveries = 20
)

// Worker processes webhook deliveries in the background
// Worker delivers events to subscribed webhooks and retries failed deliveries.
type Worker struct {
	store           storage.WebhookStorage
	deliveryService *DeliveryService
	interval        time.Duration
	sem             chan struct{} // bounds concurrent deliveries

	ctx         context.Context
	cancel      context.CancelFunc
	unsubscribe func()

	mu       sync.Mutex
	stopping bool
	wg       sync.WaitGroup // the run loop and in-flight deliveries
}

// NewWorker returns a worker; Start begins delivering.
func NewWorker(store storage.WebhookStorage, config DeliveryConfig) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		store:           store,
		deliveryService: NewDeliveryService(store, config),
		interval:        30 * time.Second,
		sem:             make(chan struct{}, maxConcurrentDeliveries),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start subscribes to events and starts retrying failed deliveries.
func (w *Worker) Start() {
	w.unsubscribe = Subscribe(w.handleEvent)
	w.wg.Add(1)
	go w.run()
}

// Stop stops taking events, cancels in-flight deliveries (they are retried on
// the next start) and waits for them to finish.
func (w *Worker) Stop() {
	w.mu.Lock()
	if w.stopping {
		w.mu.Unlock()
		return
	}
	w.stopping = true
	w.mu.Unlock()
	if w.unsubscribe != nil {
		w.unsubscribe()
	}
	w.cancel()
	w.wg.Wait()
}

// track counts a delivery as in flight unless the worker is stopping.
func (w *Worker) track() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopping {
		return false
	}
	w.wg.Add(1)
	return true
}

func (w *Worker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.processRetries()
			w.cleanup()
		}
	}
}

func (w *Worker) handleEvent(event model.Event) {
	webhooks, err := w.store.GetWebhooksForEvent(w.ctx, event.Type)
	if err != nil {
		log.Error("Failed to get webhooks for event", "event_type", event.Type, "error", err)
		return
	}

	for _, webhook := range webhooks {
		if !w.track() {
			return
		}
		select {
		case w.sem <- struct{}{}: // acquire
		case <-w.ctx.Done():
			w.wg.Done()
			return
		}
		go func(wh model.Webhook) {
			defer w.wg.Done()
			defer func() { <-w.sem }() // release
			if _, err := w.deliveryService.Deliver(w.ctx, &wh, event); err != nil {
				log.Warn("Webhook delivery failed", "webhook", wh.Name, "event_type", event.Type, "error", err)
			}
		}(webhook)
	}
}

func (w *Worker) processRetries() {
	success, failed, err := w.deliveryService.ProcessPendingRetries(w.ctx)
	if err != nil {
		log.Error("Failed to process webhook retries", "error", err)
	}
	if success > 0 || failed > 0 {
		log.Info("Webhook retries", "succeeded", success, "failed", failed)
	}
}

func (w *Worker) cleanup() {
	if err := w.deliveryService.CleanupOldDeliveries(w.ctx); err != nil {
		log.Error("Failed to clean up old webhook deliveries", "error", err)
	}
}

// DeliverEvent delivers event to its subscribed webhooks, as if it had been
// published.
func (w *Worker) DeliverEvent(event model.Event) error {
	w.handleEvent(event)
	return nil
}
