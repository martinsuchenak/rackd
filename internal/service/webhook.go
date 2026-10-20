package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/storage"
	wh "github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/model"
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

	webhooks, err := s.store.ListWebhooks(ctx, filter)
	if err != nil {
		return nil, err
	}
	if !s.canSeeURLs(ctx) {
		for i := range webhooks {
			webhooks[i].URL = RedactWebhookURL(webhooks[i].URL)
		}
	}

	return webhooks, nil
}

// canSeeURLs reports whether the caller may see full webhook URLs: those who
// can change webhooks. A URL can itself be a credential (an incoming-webhook
// URL, a ?token= or user:pass@), so readers see only its scheme and host.
func (s *WebhookService) canSeeURLs(ctx context.Context) bool {
	return requirePermission(ctx, s.store, "webhooks", "update") == nil
}

// RedactWebhookURL keeps a URL's scheme and host and hides the rest.
func RedactWebhookURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "(hidden)"
	}
	redacted := u.Scheme + "://" + u.Host + "/"
	if u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.User != nil {
		redacted += "…"
	}
	return redacted
}

// Get returns a single webhook by ID
func (s *WebhookService) Get(ctx context.Context, id string) (*model.Webhook, error) {
	if err := requirePermission(ctx, s.store, "webhooks", "read"); err != nil {
		return nil, err
	}

	webhook, err := s.store.GetWebhook(ctx, id)
	if err != nil {
		if err == storage.ErrWebhookNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !s.canSeeURLs(ctx) {
		webhook.URL = RedactWebhookURL(webhook.URL)
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
		if !wh.IsKnownEventType(et) {
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
	recordAudit(ctx, s.store, "create", "webhook", webhook.ID, webhookAudit(webhook))

	return webhook, nil
}

// Update updates an existing webhook
func (s *WebhookService) Update(ctx context.Context, id string, req *model.UpdateWebhookRequest) (*model.Webhook, error) {
	if err := requirePermission(ctx, s.store, "webhooks", "update"); err != nil {
		return nil, err
	}

	webhook, err := s.store.GetWebhook(ctx, id)
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
			if !wh.IsKnownEventType(et) {
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
	recordAudit(ctx, s.store, "update", "webhook", id, webhookAudit(webhook))

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
	recordAudit(ctx, s.store, "delete", "webhook", id, nil)

	return nil
}

// webhookAudit describes a webhook for the audit log: the URL only to its
// host, since it can be a credential, and never the secret.
func webhookAudit(w *model.Webhook) map[string]any {
	return map[string]any{"name": w.Name, "url": RedactWebhookURL(w.URL), "events": w.Events, "active": w.Active, "has_secret": w.Secret != ""}
}

// ListDeliveries returns delivery records for a webhook
func (s *WebhookService) ListDeliveries(ctx context.Context, filter *model.DeliveryFilter) ([]model.WebhookDelivery, error) {
	if err := requirePermission(ctx, s.store, "webhooks", "read"); err != nil {
		return nil, err
	}

	deliveries, err := s.store.ListDeliveries(ctx, filter)
	if err != nil {
		return nil, err
	}
	if !s.canSeeDeliveryResponses(ctx) {
		for i := range deliveries {
			hideDeliveryResponse(&deliveries[i])
		}
	}

	return deliveries, nil
}

// canSeeDeliveryResponses reports whether the caller may see what a
// webhook's endpoint answered. The answer can be an internal service's own
// response, so only someone who can set the URL sees it; anyone else could
// read from wherever the webhook points.
func (s *WebhookService) canSeeDeliveryResponses(ctx context.Context) bool {
	return requirePermission(ctx, s.store, "webhooks", "update") == nil
}

// hideDeliveryResponse keeps whether a delivery failed, but not the
// endpoint's answer.
func hideDeliveryResponse(d *model.WebhookDelivery) {
	d.ResponseCode = 0
	d.ResponseBody = ""
	if d.Error != "" {
		d.Error = "delivery failed (someone who can edit the webhook can see why)"
	}
}

// GetDelivery returns a single delivery record
func (s *WebhookService) GetDelivery(ctx context.Context, id string) (*model.WebhookDelivery, error) {
	if err := requirePermission(ctx, s.store, "webhooks", "read"); err != nil {
		return nil, err
	}

	delivery, err := s.store.GetDelivery(ctx, id)
	if err != nil {
		if err == storage.ErrDeliveryNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !s.canSeeDeliveryResponses(ctx) {
		hideDeliveryResponse(delivery)
	}

	return delivery, nil
}

// Ping sends a test event to a webhook
func (s *WebhookService) Ping(ctx context.Context, id string) (*model.WebhookDelivery, error) {
	if err := requirePermission(ctx, s.store, "webhooks", "update"); err != nil {
		return nil, err
	}

	webhook, err := s.store.GetWebhook(ctx, id)
	if err != nil {
		if err == storage.ErrWebhookNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// Create a test event
	testEvent := model.Event{
		Type:      "webhook.ping",
		Timestamp: time.Now().UTC(),
		Payload: map[string]string{
			"message": "Test webhook delivery from Rackd",
			"webhook": webhook.Name,
		},
	}
	delivery, err := s.pingWebhook(ctx, webhook, testEvent)
	if err != nil {
		return nil, err
	}
	if delivery.Status == model.DeliveryStatusFailed {
		return delivery, fmt.Errorf("%w: %s", ErrWebhookPingFailed, delivery.Error)
	}
	return delivery, nil
}

// ErrWebhookPingFailed is returned when a test delivery doesn't reach the
// webhook's URL, so the API can answer 502 with the cause.
var ErrWebhookPingFailed = errors.New("webhook ping failed")

// pingWebhook sends a test event in the same format, with the same headers and
// signature, as real deliveries.
func (s *WebhookService) pingWebhook(ctx context.Context, webhook *model.Webhook, event model.Event) (*model.WebhookDelivery, error) {
	delivery := &model.WebhookDelivery{
		ID:        uuid.New().String(),
		WebhookID: webhook.ID,
		EventType: event.Type,
		CreatedAt: time.Now().UTC(),
	}
	body, err := wh.NewMessage(event, delivery.ID)
	if err != nil {
		return nil, err
	}
	delivery.Payload = string(body)

	req, err := wh.NewRequest(ctx, webhook, event.Type, delivery.ID, body, time.Now())
	if err != nil {
		return nil, err
	}
	// Runs inside the API request, so stay below the default REQUEST_TIMEOUT
	// (30s) and report the real failure rather than a generic timeout.
	client := wh.NewSecureHTTPClient(20 * time.Second)

	startTime := time.Now()
	delivery.ResponseCode, err = wh.Send(client, req)
	delivery.Duration = time.Since(startTime).Milliseconds()
	if err != nil {
		delivery.Status = model.DeliveryStatusFailed
		delivery.Error = err.Error()
	} else {
		delivery.Status = model.DeliveryStatusSuccess
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

	// Block loopback, link-local (cloud metadata), unspecified, multicast
	// and private-range (RFC1918/CGNAT/ULA) destinations, including
	// hostnames that resolve into those ranges. Without this, an
	// authenticated webhook creator can make the server deliver signed
	// requests to internal services (SSRF).
	if err := wh.ValidateOutboundHost(u.Hostname()); err != nil {
		return ValidationErrors{{Field: "url", Message: "Loopback, link-local, multicast and private network addresses are not allowed (an administrator can allow private ones with SSRF_ALLOWED_TARGETS)"}}
	}

	return nil
}
