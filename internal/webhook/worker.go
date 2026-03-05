package webhook

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

const (
	// maxConcurrentDeliveries limits the number of simultaneous webhook deliveries
	maxConcurrentDeliveries = 20
)

// Worker processes webhook deliveries in the background
type Worker struct {
	store           storage.WebhookStorage
	deliveryService *DeliveryService
	interval        time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
	sem             chan struct{} // semaphore to bound concurrent deliveries
}

// NewWorker creates a new webhook worker
func NewWorker(store storage.WebhookStorage, config DeliveryConfig) *Worker {
	return &Worker{
		store:           store,
		deliveryService: NewDeliveryService(store, config),
		interval:        30 * time.Second,
		stopCh:          make(chan struct{}),
		sem:             make(chan struct{}, maxConcurrentDeliveries),
	}
}

// Start begins the worker's background processing
func (w *Worker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Subscribe to events
	Subscribe(func(event model.Event) {
		w.handleEvent(event)
	})

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.processRetries()
			w.cleanup()
		}
	}
}

// handleEvent handles an incoming event by delivering to subscribed webhooks
func (w *Worker) handleEvent(event model.Event) {
	ctx := context.Background()

	// Get all webhooks subscribed to this event
	webhooks, err := w.store.GetWebhooksForEvent(ctx, event.Type)
	if err != nil {
		log.Printf("failed to get webhooks for event %s: %v", event.Type, err)
		return
	}

	// Deliver to each webhook (bounded concurrency)
	for _, webhook := range webhooks {
		w.sem <- struct{}{} // acquire
		go func(wh model.Webhook) {
			defer func() { <-w.sem }() // release
			_, err := w.deliveryService.Deliver(ctx, &wh, event)
			if err != nil {
				log.Printf("webhook delivery failed for %s: %v", wh.Name, err)
			}
		}(webhook)
	}
}

// processRetries processes pending retry deliveries
func (w *Worker) processRetries() {
	ctx := context.Background()
	success, failed, err := w.deliveryService.ProcessPendingRetries(ctx)
	if err != nil {
		log.Printf("failed to process webhook retries: %v", err)
	}
	if success > 0 || failed > 0 {
		log.Printf("webhook retries: %d succeeded, %d failed", success, failed)
	}
}

// cleanup removes old delivery records
func (w *Worker) cleanup() {
	if err := w.deliveryService.CleanupOldDeliveries(); err != nil {
		log.Printf("failed to cleanup old webhook deliveries: %v", err)
	}
}

// DeliverEvent delivers an event to all subscribed webhooks immediately
func (w *Worker) DeliverEvent(event model.Event) error {
	w.handleEvent(event)
	return nil
}
