package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestWebhookService_CreateRejectsLinkLocalAndTracksCreator(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "webhooks", "create", true)
	svc := NewWebhookService(store)

	_, err := svc.Create(userContext("user-1"), &model.CreateWebhookRequest{
		Name:   "blocked",
		URL:    "http://169.254.169.254/hooks",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for link-local URL, got %v", err)
	}

	webhook, err := svc.Create(userContext("user-1"), &model.CreateWebhookRequest{
		Name:   "inventory-webhook",
		URL:    "https://example.com/hooks",
		Events: []model.EventType{model.EventTypeDeviceCreated},
		Active: true,
	})
	if err != nil {
		t.Fatalf("expected valid webhook create to succeed, got %v", err)
	}
	if webhook.CreatedBy != "user-1" {
		t.Fatalf("expected created_by to track caller, got %q", webhook.CreatedBy)
	}
}

func TestWebhookService_UpdateRejectsInvalidEventsAndMissingWebhook(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "webhooks", "update", true)
	store.webhooks["webhook-1"] = &model.Webhook{
		ID:     "webhook-1",
		Name:   "inventory",
		URL:    "https://example.com/hook",
		Events: []model.EventType{model.EventTypeDeviceCreated},
	}
	svc := NewWebhookService(store)

	invalidEvents := []model.EventType{"not.real"}
	_, err := svc.Update(userContext("user-1"), "webhook-1", &model.UpdateWebhookRequest{Events: &invalidEvents})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for invalid event type, got %v", err)
	}

	_, err = svc.Update(userContext("user-1"), "missing", &model.UpdateWebhookRequest{})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for missing webhook, got %v", err)
	}
}

func TestWebhookService_DeleteMapsMissingWebhookAndURLHelperRejectsBadSchemes(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "webhooks", "delete", true)
	svc := NewWebhookService(store)

	err := svc.Delete(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for missing webhook delete, got %v", err)
	}

	err = validateWebhookURL("ftp://example.com/hook")
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for unsupported URL scheme, got %v", err)
	}
}
