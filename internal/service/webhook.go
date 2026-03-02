package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
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
	if err := requirePermission(ctx, s.store, "webhooks", "list"); err != nil {
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
	if err := requirePermission(ctx, s.store, "webhooks", "read"); err != nil {
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
	if err := requirePermission(ctx, s.store, "webhooks", "create"); err != nil {
		return nil, err
	}

	// Validate required fields
	if req.Name == "" {
		return nil, ValidationErrors{{Field: "name", Message: "Name is required"}}
	}
	if req.URL == "" {
		return nil, ValidationErrors{{Field: "url", Message: "URL is required"}}
	}
	if err := validateWebhookURL(req.URL); err != nil {
		return nil, err
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
	if err := requirePermission(ctx, s.store, "webhooks", "update"); err != nil {
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
		if err := validateWebhookURL(*req.URL); err != nil {
			return nil, err
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
	if err := requirePermission(ctx, s.store, "webhooks", "delete"); err != nil {
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
	if err := requirePermission(ctx, s.store, "webhooks", "read"); err != nil {
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
	if err := requirePermission(ctx, s.store, "webhooks", "read"); err != nil {
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
	if err := requirePermission(ctx, s.store, "webhooks", "update"); err != nil {
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
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Rackd-Webhook/1.0")
	req.Header.Set("X-Event-Type", string(event.Type))
	req.Header.Set("X-Delivery-ID", uuid.New().String())

	// Add HMAC signature if secret is configured
	if webhook.Secret != "" {
		h := hmac.New(sha256.New, []byte(webhook.Secret))
		h.Write(payload)
		signature := hex.EncodeToString(h.Sum(nil))
		req.Header.Set("X-Signature-256", "sha256="+signature)
	}

	delivery := &model.WebhookDelivery{
		WebhookID: webhook.ID,
		EventType: event.Type,
		Payload:   string(payload),
		CreatedAt: time.Now().UTC(),
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	delivery.Duration = time.Since(startTime).Milliseconds()

	if err != nil {
		delivery.Status = model.DeliveryStatusFailed
		delivery.Error = err.Error()
		return delivery, nil
	}
	defer resp.Body.Close()

	delivery.ResponseCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		delivery.Status = model.DeliveryStatusSuccess
	} else {
		delivery.Status = model.DeliveryStatusFailed
		delivery.Error = fmt.Sprintf("webhook returned status %d", resp.StatusCode)
	}

	return delivery, nil
}

// validateWebhookURL validates that a webhook URL is safe to deliver to
func validateWebhookURL(rawURL string) error {
	if len(rawURL) > 2048 {
		return ValidationErrors{{Field: "url", Message: "URL too long (max 2048 characters)"}}
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return ValidationErrors{{Field: "url", Message: "Invalid URL"}}
	}

	// Only allow http/https
	if u.Scheme != "http" && u.Scheme != "https" {
		return ValidationErrors{{Field: "url", Message: "URL must use http or https scheme"}}
	}

	if u.Host == "" {
		return ValidationErrors{{Field: "url", Message: "URL must include a host"}}
	}

	// Block well-known metadata endpoints and loopback
	hostname := u.Hostname()
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.IsLoopback() || ip.IsUnspecified() {
			return ValidationErrors{{Field: "url", Message: "Loopback and unspecified addresses are not allowed"}}
		}
		// Block link-local / cloud metadata range 169.254.x.x
		if ip.To4() != nil && ip.To4()[0] == 169 && ip.To4()[1] == 254 {
			return ValidationErrors{{Field: "url", Message: "Link-local addresses are not allowed"}}
		}
	}

	return nil
}
