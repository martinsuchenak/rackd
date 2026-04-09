package webhook

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func newWebhookTestStore(t *testing.T) *storage.SQLiteStorage {
	t.Helper()
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	return store
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestDeliveryServiceDeliverSuccessAndRetryFailure(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := newWebhookTestStore(t)
		defer store.Close()

		service := NewDeliveryService(store, DeliveryConfig{MaxRetries: 2, RetryBackoff: time.Second, HTTPTimeout: 2 * time.Second, RetentionDays: 1})
		service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("X-Signature-256") == "" {
				t.Fatal("expected HMAC signature header")
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		})}
		webhook := &model.Webhook{Name: "wh-1", URL: "https://example.test/hook", Secret: "secret", Active: true}
		if err := store.CreateWebhook(context.Background(), webhook); err != nil {
			t.Fatalf("CreateWebhook failed: %v", err)
		}
		delivery, err := service.Deliver(context.Background(), webhook, model.Event{Type: model.EventTypeDeviceCreated, Payload: map[string]any{"id": "dev-1"}})
		if err != nil {
			t.Fatalf("Deliver failed: %v", err)
		}
		if delivery.Status != model.DeliveryStatusSuccess {
			t.Fatalf("expected success status, got %+v", delivery)
		}
	})

	t.Run("failure creates retry", func(t *testing.T) {
		store := newWebhookTestStore(t)
		defer store.Close()

		service := NewDeliveryService(store, DeliveryConfig{MaxRetries: 2, RetryBackoff: time.Second, HTTPTimeout: 2 * time.Second, RetentionDays: 1})
		service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("nope")),
				Header:     make(http.Header),
			}, nil
		})}
		webhook := &model.Webhook{Name: "wh-2", URL: "https://example.test/hook", Active: true}
		if err := store.CreateWebhook(context.Background(), webhook); err != nil {
			t.Fatalf("CreateWebhook failed: %v", err)
		}
		delivery, err := service.Deliver(context.Background(), webhook, model.Event{Type: model.EventTypeDeviceCreated, Payload: "x"})
		if err == nil {
			t.Fatal("expected delivery failure")
		}
		if delivery.Status != model.DeliveryStatusRetrying || delivery.NextRetry == nil {
			t.Fatalf("expected retrying status with next retry, got %+v", delivery)
		}
	})
}

func TestDeliveryServiceProcessPendingRetriesAndCleanup(t *testing.T) {
	store := newWebhookTestStore(t)
	defer store.Close()
	ctx := context.Background()

	service := NewDeliveryService(store, DeliveryConfig{MaxRetries: 2, RetryBackoff: time.Second, HTTPTimeout: 2 * time.Second, RetentionDays: 1})
	service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})}

	activeWebhook := &model.Webhook{
		Name:   "active",
		URL:    "https://example.test/active",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	if err := store.CreateWebhook(ctx, activeWebhook); err != nil {
		t.Fatalf("CreateWebhook active failed: %v", err)
	}
	inactiveWebhook := &model.Webhook{
		Name:   "inactive",
		URL:    "https://example.test/inactive",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: false,
	}
	if err := store.CreateWebhook(ctx, inactiveWebhook); err != nil {
		t.Fatalf("CreateWebhook inactive failed: %v", err)
	}

	retrying := &model.WebhookDelivery{
		WebhookID:     activeWebhook.ID,
		EventType:     model.EventTypeDeviceCreated,
		Payload:       `{"id":"1"}`,
		Status:        model.DeliveryStatusRetrying,
		AttemptNumber: 1,
		NextRetry:     ptrTime(time.Now().Add(-time.Minute).UTC()),
	}
	if err := store.CreateDelivery(ctx, retrying); err != nil {
		t.Fatalf("CreateDelivery retrying failed: %v", err)
	}
	abandoned := &model.WebhookDelivery{
		WebhookID:     inactiveWebhook.ID,
		EventType:     model.EventTypeDeviceCreated,
		Payload:       `{"id":"2"}`,
		Status:        model.DeliveryStatusRetrying,
		AttemptNumber: 1,
		NextRetry:     ptrTime(time.Now().Add(-time.Minute).UTC()),
	}
	if err := store.CreateDelivery(ctx, abandoned); err != nil {
		t.Fatalf("CreateDelivery abandoned failed: %v", err)
	}

	success, failed, err := service.ProcessPendingRetries(ctx)
	if err != nil {
		t.Fatalf("ProcessPendingRetries failed: %v", err)
	}
	if success != 1 || failed != 1 {
		t.Fatalf("unexpected retry results success=%d failed=%d", success, failed)
	}

	gotRetrying, err := store.GetDelivery(ctx, retrying.ID)
	if err != nil {
		t.Fatalf("GetDelivery retrying failed: %v", err)
	}
	if gotRetrying.Status != model.DeliveryStatusSuccess {
		t.Fatalf("expected retrying delivery to succeed, got %+v", gotRetrying)
	}

	gotAbandoned, err := store.GetDelivery(ctx, abandoned.ID)
	if err != nil {
		t.Fatalf("GetDelivery abandoned failed: %v", err)
	}
	if gotAbandoned.Status != model.DeliveryStatusAbandoned {
		t.Fatalf("expected inactive webhook delivery to be abandoned, got %+v", gotAbandoned)
	}

	if err := service.CleanupOldDeliveries(); err != nil {
		t.Fatalf("CleanupOldDeliveries failed: %v", err)
	}
}

func TestWorkerDeliverEventAndStop(t *testing.T) {
	store := newWebhookTestStore(t)
	defer store.Close()
	ctx := context.Background()

	wh := &model.Webhook{
		Name:   "worker",
		URL:    "https://example.test/worker",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	if err := store.CreateWebhook(ctx, wh); err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	worker := NewWorker(store, DeliveryConfig{MaxRetries: 1, RetryBackoff: time.Second, HTTPTimeout: 2 * time.Second, RetentionDays: 1})
	worker.interval = 10 * time.Millisecond
	worker.deliveryService.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		}, nil
	})}
	worker.Start()
	defer worker.Stop()

	if err := worker.DeliverEvent(model.Event{Type: model.EventTypeDeviceCreated, Payload: map[string]any{"id": "dev-1"}}); err != nil {
		t.Fatalf("DeliverEvent failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	deliveries, err := store.ListDeliveries(ctx, &model.DeliveryFilter{})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(deliveries) == 0 {
		t.Fatal("expected at least one delivery record")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
