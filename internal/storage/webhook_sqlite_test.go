package storage

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ============================================================================
// Webhook CRUD Tests
// ============================================================================

func TestWebhookOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	webhook := &model.Webhook{
		Name:        "test-webhook",
		URL:         "https://example.com/webhook",
		Secret:      "test-secret",
		Events:      []model.EventType{model.EventTypeDeviceCreated, model.EventTypeDeviceUpdated},
		Active:      true,
		Description: "Test webhook description",
		CreatedBy:   "test-user",
	}

	// Create webhook
	err := storage.CreateWebhook(context.Background(), webhook)
	if err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	if webhook.ID == "" {
		t.Error("webhook ID should be set after creation")
	}
	if webhook.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}
	if webhook.UpdatedAt.IsZero() {
		t.Error("updated_at should be set after creation")
	}

	// Get webhook
	retrieved, err := storage.GetWebhook(context.Background(), webhook.ID)
	if err != nil {
		t.Fatalf("GetWebhook failed: %v", err)
	}

	// Verify fields
	if retrieved.Name != webhook.Name {
		t.Errorf("expected name %s, got %s", webhook.Name, retrieved.Name)
	}
	if retrieved.URL != webhook.URL {
		t.Errorf("expected URL %s, got %s", webhook.URL, retrieved.URL)
	}
	if retrieved.Secret != webhook.Secret {
		t.Errorf("expected secret %s, got %s", webhook.Secret, retrieved.Secret)
	}
	if retrieved.Active != webhook.Active {
		t.Errorf("expected active %v, got %v", webhook.Active, retrieved.Active)
	}
	if retrieved.Description != webhook.Description {
		t.Errorf("expected description %s, got %s", webhook.Description, retrieved.Description)
	}

	// Verify events
	if len(retrieved.Events) != len(webhook.Events) {
		t.Errorf("expected %d events, got %d", len(webhook.Events), len(retrieved.Events))
	}
}

func TestWebhookOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetWebhook(context.Background(), "non-existent-id")
	if err != ErrWebhookNotFound {
		t.Errorf("expected ErrWebhookNotFound, got %v", err)
	}
}

func TestWebhookOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "original-name",
		URL:    "https://example.com/original",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	if err := storage.CreateWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	originalUpdatedAt := webhook.UpdatedAt

	// Update webhook
	webhook.Name = "updated-name"
	webhook.URL = "https://example.com/updated"
	webhook.Events = []model.EventType{model.EventTypeDeviceCreated, model.EventTypeDeviceDeleted}
	webhook.Active = false
	webhook.Description = "Updated description"

	if err := storage.UpdateWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("UpdateWebhook failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetWebhook(context.Background(), webhook.ID)
	if err != nil {
		t.Fatalf("GetWebhook failed: %v", err)
	}

	if retrieved.Name != "updated-name" {
		t.Errorf("expected name 'updated-name', got '%s'", retrieved.Name)
	}
	if retrieved.URL != "https://example.com/updated" {
		t.Errorf("expected URL 'https://example.com/updated', got '%s'", retrieved.URL)
	}
	if len(retrieved.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(retrieved.Events))
	}
	if retrieved.Active != false {
		t.Errorf("expected active false, got %v", retrieved.Active)
	}
	if !retrieved.UpdatedAt.After(originalUpdatedAt) {
		t.Error("updated_at should be updated")
	}
}

func TestWebhookOperations_UpdateNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	webhook := &model.Webhook{
		ID:     "non-existent-id",
		Name:   "test",
		URL:    "https://example.com/test",
		Events: []model.EventType{model.EventTypeDeviceCreated},
	}

	err := storage.UpdateWebhook(context.Background(), webhook)
	if err != ErrWebhookNotFound {
		t.Errorf("expected ErrWebhookNotFound, got %v", err)
	}
}

func TestWebhookOperations_Delete(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "to-delete",
		URL:    "https://example.com/delete",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	if err := storage.CreateWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	// Delete webhook
	if err := storage.DeleteWebhook(context.Background(), webhook.ID); err != nil {
		t.Fatalf("DeleteWebhook failed: %v", err)
	}

	// Verify deletion
	_, err := storage.GetWebhook(context.Background(), webhook.ID)
	if err != ErrWebhookNotFound {
		t.Errorf("expected ErrWebhookNotFound after deletion, got %v", err)
	}
}

func TestWebhookOperations_DeleteNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	err := storage.DeleteWebhook(context.Background(), "non-existent-id")
	if err != ErrWebhookNotFound {
		t.Errorf("expected ErrWebhookNotFound, got %v", err)
	}
}

func TestWebhookOperations_ListAll(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create multiple webhooks
	webhooks := []struct {
		name   string
		active bool
	}{
		{"webhook1", true},
		{"webhook2", true},
		{"webhook3", false},
	}

	for _, w := range webhooks {
		webhook := &model.Webhook{
			Name:   w.name,
			URL:    "https://example.com/" + w.name,
			Events: []model.EventType{model.EventTypeDeviceCreated},
			Active: w.active,
		}
		if err := storage.CreateWebhook(context.Background(), webhook); err != nil {
			t.Fatalf("CreateWebhook failed: %v", err)
		}
	}

	// List all webhooks
	result, err := storage.ListWebhooks(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListWebhooks failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 webhooks, got %d", len(result))
	}
}

func TestWebhookOperations_ListWithActiveFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhooks with different active states
	active := true
	inactive := false

	storage.CreateWebhook(context.Background(), &model.Webhook{
		Name: "active-1", URL: "https://example.com/1", Events: []model.EventType{model.EventTypeDeviceCreated}, Active: true,
	})
	storage.CreateWebhook(context.Background(), &model.Webhook{
		Name: "active-2", URL: "https://example.com/2", Events: []model.EventType{model.EventTypeDeviceCreated}, Active: true,
	})
	storage.CreateWebhook(context.Background(), &model.Webhook{
		Name: "inactive-1", URL: "https://example.com/3", Events: []model.EventType{model.EventTypeDeviceCreated}, Active: false,
	})

	// Filter by active=true
	result, err := storage.ListWebhooks(context.Background(), &model.WebhookFilter{Active: &active})
	if err != nil {
		t.Fatalf("ListWebhooks failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 active webhooks, got %d", len(result))
	}

	// Filter by active=false
	result, err = storage.ListWebhooks(context.Background(), &model.WebhookFilter{Active: &inactive})
	if err != nil {
		t.Fatalf("ListWebhooks failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 inactive webhook, got %d", len(result))
	}
}

func TestWebhookOperations_GetWebhooksForEvent(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhooks with different event subscriptions
	storage.CreateWebhook(context.Background(), &model.Webhook{
		Name: "device-webhook", URL: "https://example.com/device",
		Events: []model.EventType{model.EventTypeDeviceCreated, model.EventTypeDeviceUpdated},
		Active: true,
	})
	storage.CreateWebhook(context.Background(), &model.Webhook{
		Name: "network-webhook", URL: "https://example.com/network",
		Events: []model.EventType{model.EventTypeNetworkCreated},
		Active: true,
	})
	storage.CreateWebhook(context.Background(), &model.Webhook{
		Name: "inactive-webhook", URL: "https://example.com/inactive",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: false,
	})

	// Get webhooks for device.created event
	result, err := storage.GetWebhooksForEvent(context.Background(), model.EventTypeDeviceCreated)
	if err != nil {
		t.Fatalf("GetWebhooksForEvent failed: %v", err)
	}

	// Should only return active webhooks subscribed to device.created
	if len(result) != 1 {
		t.Errorf("expected 1 webhook for device.created, got %d", len(result))
	}
	if len(result) > 0 && result[0].Name != "device-webhook" {
		t.Errorf("expected 'device-webhook', got '%s'", result[0].Name)
	}

	// Get webhooks for network.created event
	result, err = storage.GetWebhooksForEvent(context.Background(), model.EventTypeNetworkCreated)
	if err != nil {
		t.Fatalf("GetWebhooksForEvent failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 webhook for network.created, got %d", len(result))
	}
}

// ============================================================================
// Delivery CRUD Tests
// ============================================================================

func TestDeliveryOperations_CreateAndGet(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook first
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	if err := storage.CreateWebhook(context.Background(), webhook); err != nil {
		t.Fatalf("CreateWebhook failed: %v", err)
	}

	nextRetry := time.Now().Add(5 * time.Minute)
	delivery := &model.WebhookDelivery{
		WebhookID:     webhook.ID,
		EventType:     model.EventTypeDeviceCreated,
		Payload:       `{"test": "data"}`,
		Status:        model.DeliveryStatusPending,
		AttemptNumber: 1,
		NextRetry:     &nextRetry,
	}

	// Create delivery
	err := storage.CreateDelivery(context.Background(), delivery)
	if err != nil {
		t.Fatalf("CreateDelivery failed: %v", err)
	}

	if delivery.ID == "" {
		t.Error("delivery ID should be set after creation")
	}
	if delivery.CreatedAt.IsZero() {
		t.Error("created_at should be set after creation")
	}

	// Get delivery
	retrieved, err := storage.GetDelivery(context.Background(), delivery.ID)
	if err != nil {
		t.Fatalf("GetDelivery failed: %v", err)
	}

	// Verify fields
	if retrieved.WebhookID != webhook.ID {
		t.Errorf("expected webhook_id %s, got %s", webhook.ID, retrieved.WebhookID)
	}
	if retrieved.EventType != model.EventTypeDeviceCreated {
		t.Errorf("expected event_type %s, got %s", model.EventTypeDeviceCreated, retrieved.EventType)
	}
	if retrieved.Status != model.DeliveryStatusPending {
		t.Errorf("expected status %s, got %s", model.DeliveryStatusPending, retrieved.Status)
	}
}

func TestDeliveryOperations_GetNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	_, err := storage.GetDelivery(context.Background(), "non-existent-id")
	if err != ErrDeliveryNotFound {
		t.Errorf("expected ErrDeliveryNotFound, got %v", err)
	}
}

func TestDeliveryOperations_Update(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook and delivery
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	storage.CreateWebhook(context.Background(), webhook)

	delivery := &model.WebhookDelivery{
		WebhookID: webhook.ID,
		EventType: model.EventTypeDeviceCreated,
		Payload:   `{"test": "data"}`,
		Status:    model.DeliveryStatusPending,
	}
	storage.CreateDelivery(context.Background(), delivery)

	// Update delivery
	delivery.Status = model.DeliveryStatusSuccess
	delivery.ResponseCode = 200
	delivery.ResponseBody = `{"received": true}`
	delivery.Duration = 150
	delivery.AttemptNumber = 1

	if err := storage.UpdateDelivery(context.Background(), delivery); err != nil {
		t.Fatalf("UpdateDelivery failed: %v", err)
	}

	// Verify update
	retrieved, err := storage.GetDelivery(context.Background(), delivery.ID)
	if err != nil {
		t.Fatalf("GetDelivery failed: %v", err)
	}

	if retrieved.Status != model.DeliveryStatusSuccess {
		t.Errorf("expected status 'success', got '%s'", retrieved.Status)
	}
	if retrieved.ResponseCode != 200 {
		t.Errorf("expected response_code 200, got %d", retrieved.ResponseCode)
	}
	if retrieved.Duration != 150 {
		t.Errorf("expected duration 150, got %d", retrieved.Duration)
	}
}

func TestDeliveryOperations_UpdateNotFound(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	delivery := &model.WebhookDelivery{
		ID:     "non-existent-id",
		Status: model.DeliveryStatusSuccess,
	}

	err := storage.UpdateDelivery(context.Background(), delivery)
	if err != ErrDeliveryNotFound {
		t.Errorf("expected ErrDeliveryNotFound, got %v", err)
	}
}

func TestDeliveryOperations_ListWithFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated, model.EventTypeNetworkCreated},
		Active: true,
	}
	storage.CreateWebhook(context.Background(), webhook)

	// Create deliveries with different statuses
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID, EventType: model.EventTypeDeviceCreated,
		Payload: `{}`, Status: model.DeliveryStatusSuccess,
	})
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID, EventType: model.EventTypeDeviceCreated,
		Payload: `{}`, Status: model.DeliveryStatusFailed,
	})
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID, EventType: model.EventTypeNetworkCreated,
		Payload: `{}`, Status: model.DeliveryStatusSuccess,
	})

	// Filter by webhook ID
	result, err := storage.ListDeliveries(context.Background(), &model.DeliveryFilter{WebhookID: webhook.ID})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 deliveries, got %d", len(result))
	}

	// Filter by status
	result, err = storage.ListDeliveries(context.Background(), &model.DeliveryFilter{Status: model.DeliveryStatusSuccess})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 successful deliveries, got %d", len(result))
	}

	// Filter by event type
	result, err = storage.ListDeliveries(context.Background(), &model.DeliveryFilter{EventType: model.EventTypeNetworkCreated})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 network.created delivery, got %d", len(result))
	}
}

func TestDeliveryOperations_ListWithLimit(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	storage.CreateWebhook(context.Background(), webhook)

	// Create multiple deliveries
	for i := 0; i < 10; i++ {
		storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
			WebhookID: webhook.ID,
			EventType: model.EventTypeDeviceCreated,
			Payload:   `{}`,
			Status:    model.DeliveryStatusSuccess,
		})
	}

	// List with limit
	result, err := storage.ListDeliveries(context.Background(), &model.DeliveryFilter{Limit: 5})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 5 {
		t.Errorf("expected 5 deliveries with limit, got %d", len(result))
	}
}

func TestDeliveryOperations_GetPendingDeliveries(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	storage.CreateWebhook(context.Background(), webhook)

	// Create deliveries with different statuses
	// Use UTC times formatted as RFC3339 for proper SQLite string comparison
	pastTime := time.Now().UTC().Add(-1 * time.Hour)
	futureTime := time.Now().UTC().Add(1 * time.Hour)

	// Pending delivery ready for retry
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID, EventType: model.EventTypeDeviceCreated,
		Payload: `{}`, Status: model.DeliveryStatusPending,
		NextRetry: &pastTime,
	})

	// Retrying delivery ready for retry
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID, EventType: model.EventTypeDeviceCreated,
		Payload: `{}`, Status: model.DeliveryStatusRetrying,
		NextRetry: &pastTime,
	})

	// Pending delivery not ready yet
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID, EventType: model.EventTypeDeviceCreated,
		Payload: `{}`, Status: model.DeliveryStatusPending,
		NextRetry: &futureTime,
	})

	// Successful delivery (should not be returned)
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID, EventType: model.EventTypeDeviceCreated,
		Payload: `{}`, Status: model.DeliveryStatusSuccess,
	})

	// Get pending deliveries
	result, err := storage.GetPendingDeliveries(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetPendingDeliveries failed: %v", err)
	}

	// Should only return pending/retrying with next_retry in the past
	if len(result) != 2 {
		t.Errorf("expected 2 pending deliveries, got %d", len(result))
	}
}

func TestDeliveryOperations_DeleteOldDeliveries(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	storage.CreateWebhook(context.Background(), webhook)

	// Create some deliveries
	for i := 0; i < 5; i++ {
		storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
			WebhookID: webhook.ID,
			EventType: model.EventTypeDeviceCreated,
			Payload:   `{}`,
			Status:    model.DeliveryStatusSuccess,
		})
	}

	// Delete deliveries older than 0 days (all)
	err := storage.DeleteOldDeliveries(context.Background(), 0)
	if err != nil {
		t.Fatalf("DeleteOldDeliveries failed: %v", err)
	}

	// Verify all deleted
	result, err := storage.ListDeliveries(context.Background(), &model.DeliveryFilter{WebhookID: webhook.ID})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 deliveries after cleanup, got %d", len(result))
	}
}

func TestWebhookOperations_DeleteCascadesToDeliveries(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	storage.CreateWebhook(context.Background(), webhook)

	// Create deliveries
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID,
		EventType: model.EventTypeDeviceCreated,
		Payload:   `{}`,
		Status:    model.DeliveryStatusSuccess,
	})

	// Delete webhook
	if err := storage.DeleteWebhook(context.Background(), webhook.ID); err != nil {
		t.Fatalf("DeleteWebhook failed: %v", err)
	}

	// Verify deliveries are also deleted
	result, err := storage.ListDeliveries(context.Background(), &model.DeliveryFilter{WebhookID: webhook.ID})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 deliveries after webhook deletion, got %d", len(result))
	}
}

func TestDeliveryOperations_ListWithTimeFilter(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	// Create webhook
	webhook := &model.Webhook{
		Name:   "test-webhook",
		URL:    "https://example.com/webhook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	}
	storage.CreateWebhook(context.Background(), webhook)

	// Create delivery
	storage.CreateDelivery(context.Background(), &model.WebhookDelivery{
		WebhookID: webhook.ID,
		EventType: model.EventTypeDeviceCreated,
		Payload:   `{}`,
		Status:    model.DeliveryStatusSuccess,
	})

	// Get the created delivery to know its timestamp
	deliveries, _ := storage.ListDeliveries(context.Background(), &model.DeliveryFilter{WebhookID: webhook.ID})
	if len(deliveries) == 0 {
		t.Fatal("expected at least one delivery")
	}
	createdTime := deliveries[0].CreatedAt

	// Filter with After
	after := createdTime.Add(-1 * time.Hour)
	result, err := storage.ListDeliveries(context.Background(), &model.DeliveryFilter{After: &after})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 delivery with After filter, got %d", len(result))
	}

	// Filter with Before (should exclude the delivery)
	before := createdTime.Add(-1 * time.Hour)
	result, err = storage.ListDeliveries(context.Background(), &model.DeliveryFilter{Before: &before})
	if err != nil {
		t.Fatalf("ListDeliveries failed: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 deliveries with Before filter, got %d", len(result))
	}
}
