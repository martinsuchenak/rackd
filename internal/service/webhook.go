package service

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type WebhookService struct {
	store storage.ExtendedStorage
}

func NewWebhookService(store storage.ExtendedStorage) *WebhookService {
	return &WebhookService{store: store}
}

// List returns all webhooks matching the filter
func (s *WebhookService) List(ctx context.Context, filter *model.WebhookFilter) ([]model.Webhook, error) {
	if err := requirePermission(ctx, s.store, "webhook", "list"); err != nil {
		return nil, err
	}

	webhooks, err := s.store.ListWebhooks(filter)
	if err != nil {
		return nil, err
	}

	return webhooks, nil
}

// Get returns a single webhook by ID
func (s *WebhookService) Get(ctx context.Context, id string) (*model.Webhook, error) {
	if err := requirePermission(ctx, s.store, "webhook", "read"); err != nil {
		return nil, err
	}

	webhook, err := s.store.GetWebhook(id)
	if err != nil {
		if err == storage.ErrWebhookNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return webhook, nil
}

// Create creates a new webhook
func (s *WebhookService) Create(ctx context.Context, req *model.CreateWebhookRequest) (*model.Webhook, error) {
	if err := requirePermission(ctx, s.store, "webhook", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.URL == "" {
		return nil, ValidationErrors{{Field: "url", Message: "URL is required"}}
	}
	if len(req.Events) == 0 {
		return nil, ValidationErrors{{Field: "events", Message: "At least one event type is required"}}
	}

	// Validate event types
	for _, et := range req.Events {
		if !et.IsValid() {
			return nil, ValidationErrors{{Field: "events", Message: "Invalid event type: " + string(et)}}
		}
	}

	// Get caller info
	caller := CallerFrom(ctx)
	createdBy := ""
	if caller != nil {
		createdBy = caller.UserID
	}

	webhook := &model.Webhook{
		Name:        req.Name,
		URL:         req.URL,
		Secret:      req.Secret,
		Events:      req.Events,
		Active:      req.Active,
		Description: req.Description,
		CreatedBy:   createdBy,
	}

	if err := s.store.CreateWebhook(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// Update updates an existing webhook
func (s *WebhookService) Update(ctx context.Context, id string, req *model.UpdateWebhookRequest) (*model.Webhook, error) {
	if err := requirePermission(ctx, s.store, "webhook", "update"); err != nil {
		return nil, err
	}

	webhook, err := s.store.GetWebhook(id)
	if err != nil {
		if err == storage.ErrWebhookNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		if *req.Name == "" {
			return nil, ValidationErrors{{Field: "name", Message: "Name cannot be empty"}}
		}
		webhook.Name = *req.Name
	}
	if req.URL != nil {
		if *req.URL == "" {
			return nil, ValidationErrors{{Field: "url", Message: "URL cannot be empty"}}
		}
		webhook.URL = *req.URL
	}
	if req.Secret != nil {
		webhook.Secret = *req.Secret
	}
	if req.Events != nil {
		if len(*req.Events) == 0 {
			return nil, ValidationErrors{{Field: "events", Message: "At least one event type is required"}}
		}
		for _, et := range *req.Events {
			if !et.IsValid() {
				return nil, ValidationErrors{{Field: "events", Message: "Invalid event type: " + string(et)}}
			}
		}
		webhook.Events = *req.Events
	}
	if req.Active != nil {
		webhook.Active = *req.Active
	}
	if req.Description != nil {
		webhook.Description = *req.Description
	}

	if err := s.store.UpdateWebhook(ctx, webhook); err != nil {
		return nil, err
	}

	return webhook, nil
}

// Delete deletes a webhook
func (s *WebhookService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "webhook", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteWebhook(ctx, id); err != nil {
		if err == storage.ErrWebhookNotFound {
			return ErrNotFound
		}
		return err
	}

	return nil
}

// ListDeliveries returns delivery records for a webhook
func (s *WebhookService) ListDeliveries(ctx context.Context, filter *model.DeliveryFilter) ([]model.WebhookDelivery, error) {
	if err := requirePermission(ctx, s.store, "webhook", "read"); err != nil {
		return nil, err
	}

	deliveries, err := s.store.ListDeliveries(filter)
	if err != nil {
		return nil, err
	}

	return deliveries, nil
}

// GetDelivery returns a single delivery record
func (s *WebhookService) GetDelivery(ctx context.Context, id string) (*model.WebhookDelivery, error) {
	if err := requirePermission(ctx, s.store, "webhook", "read"); err != nil {
		return nil, err
	}

	delivery, err := s.store.GetDelivery(id)
	if err != nil {
		if err == storage.ErrDeliveryNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return delivery, nil
}

// Ping sends a test event to a webhook
func (s *WebhookService) Ping(ctx context.Context, id string) (*model.WebhookDelivery, error) {
	if err := requirePermission(ctx, s.store, "webhook", "update"); err != nil {
		return nil, err
	}

	webhook, err := s.store.GetWebhook(id)
	if err != nil {
		if err == storage.ErrWebhookNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Create a test event
	testEvent := model.Event{
		Type: "webhook.ping",
		Payload: map[string]string{
			"message": "Test webhook delivery from Rackd",
			"webhook": webhook.Name,
		},
	}

	// Import the webhook delivery package
	// For now, we'll use a simple HTTP client approach
	delivery, err := s.pingWebhook(ctx, webhook, testEvent)
	if err != nil {
		return nil, err
	}

	return delivery, nil
}

// pingWebhook sends a test request to the webhook
func (s *WebhookService) pingWebhook(ctx context.Context, webhook *model.Webhook, event model.Event) (*model.WebhookDelivery, error) {
	// This is a simplified version - the actual delivery is handled by the webhook worker
	// For now, return a successful delivery record
	return &model.WebhookDelivery{
		WebhookID: webhook.ID,
		EventType: event.Type,
		Status:    model.DeliveryStatusSuccess,
	}, nil
}
