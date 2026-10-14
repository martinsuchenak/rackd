package webhook

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

const (
	// MaxRetries is the maximum number of delivery retry attempts
	MaxRetries = 5

	// RetryBackoffBase is the base duration for exponential backoff
	RetryBackoffBase = 10 * time.Second

	// HTTPTimeout is the timeout for HTTP requests
	HTTPTimeout = 30 * time.Second
)

// DeliveryConfig holds configuration for webhook delivery
type DeliveryConfig struct {
	MaxRetries    int
	RetryBackoff  time.Duration
	HTTPTimeout   time.Duration
	RetentionDays int
}

// DefaultDeliveryConfig returns the default delivery configuration
func DefaultDeliveryConfig() DeliveryConfig {
	return DeliveryConfig{
		MaxRetries:    MaxRetries,
		RetryBackoff:  RetryBackoffBase,
		HTTPTimeout:   HTTPTimeout,
		RetentionDays: 30,
	}
}

// DeliveryService handles webhook delivery with retries
type DeliveryService struct {
	store  storage.WebhookStorage
	client *http.Client
	config DeliveryConfig
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(store storage.WebhookStorage, config DeliveryConfig) *DeliveryService {
	return &DeliveryService{
		store:  store,
		client: NewSecureHTTPClient(config.HTTPTimeout),
		config: config,
	}
}

// Deliver sends a webhook to a specific endpoint
func (s *DeliveryService) Deliver(ctx context.Context, webhook *model.Webhook, event model.Event) (*model.WebhookDelivery, error) {
	delivery := &model.WebhookDelivery{
		ID:            uuid.New().String(),
		WebhookID:     webhook.ID,
		EventType:     event.Type,
		Status:        model.DeliveryStatusPending,
		AttemptNumber: 1,
		CreatedAt:     time.Now().UTC(),
	}
	body, err := NewMessage(event, delivery.ID)
	if err != nil {
		return nil, err
	}
	delivery.Payload = string(body)

	// Attempt delivery
	startTime := time.Now()
	delivery.ResponseCode, err = s.send(ctx, webhook, delivery)
	delivery.Duration = time.Since(startTime).Milliseconds()

	if err != nil {
		delivery.Error = err.Error()
		if s.config.MaxRetries > 0 {
			delivery.Status = model.DeliveryStatusRetrying
			nextRetry := time.Now().UTC().Add(s.calculateBackoff(1))
			delivery.NextRetry = &nextRetry
		} else {
			delivery.Status = model.DeliveryStatusFailed
		}
	} else {
		delivery.Status = model.DeliveryStatusSuccess
	}

	// Store delivery record
	if storeErr := s.store.CreateDelivery(ctx, delivery); storeErr != nil {
		return delivery, fmt.Errorf("delivery succeeded but failed to store record: %w", storeErr)
	}

	return delivery, err
}

// send posts a delivery's stored body, timestamped and signed now. Retries
// resend the same body and delivery ID.
func (s *DeliveryService) send(ctx context.Context, webhook *model.Webhook, delivery *model.WebhookDelivery) (int, error) {
	req, err := NewRequest(ctx, webhook, delivery.EventType, delivery.ID, []byte(delivery.Payload), time.Now())
	if err != nil {
		return 0, err
	}
	return Send(s.client, req)
}

// calculateBackoff calculates the backoff duration for a given attempt
func (s *DeliveryService) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: base * 2^attempt
	backoff := s.config.RetryBackoff * time.Duration(1<<uint(attempt))
	// Cap at 1 hour
	if backoff > time.Hour {
		backoff = time.Hour
	}
	return backoff
}

// ProcessPendingRetries processes all pending webhook deliveries
func (s *DeliveryService) ProcessPendingRetries(ctx context.Context) (int, int, error) {
	deliveries, err := s.store.GetPendingDeliveries(ctx, 100)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get pending deliveries: %w", err)
	}

	var successCount, failCount int

	for _, delivery := range deliveries {
		// Get the webhook
		webhook, err := s.store.GetWebhook(ctx, delivery.WebhookID)
		if err != nil {
			// Webhook was deleted, abandon the delivery
			delivery.Status = model.DeliveryStatusAbandoned
			if err := s.store.UpdateDelivery(ctx, &delivery); err != nil {
				log.Error("Failed to persist abandoned delivery status", "delivery_id", delivery.ID, "error", err)
			}
			failCount++
			continue
		}

		// Only deliver to active webhooks
		if !webhook.Active {
			delivery.Status = model.DeliveryStatusAbandoned
			if err := s.store.UpdateDelivery(ctx, &delivery); err != nil {
				log.Error("Failed to persist abandoned delivery status", "delivery_id", delivery.ID, "error", err)
			}
			failCount++
			continue
		}

		// Increment attempt counter
		delivery.AttemptNumber++

		// Attempt delivery
		startTime := time.Now()
		delivery.ResponseCode, err = s.send(ctx, webhook, &delivery)
		duration := time.Since(startTime)

		delivery.Duration = duration.Milliseconds()
		delivery.NextRetry = nil // Clear next retry time

		if err != nil {
			delivery.Error = err.Error()

			// Check if we should retry again
			if delivery.AttemptNumber < s.config.MaxRetries {
				delivery.Status = model.DeliveryStatusRetrying
				nextRetry := time.Now().UTC().Add(s.calculateBackoff(delivery.AttemptNumber))
				delivery.NextRetry = &nextRetry
			} else {
				delivery.Status = model.DeliveryStatusFailed
			}
			failCount++
		} else {
			delivery.Status = model.DeliveryStatusSuccess
			delivery.Error = ""
			successCount++
		}

		if updateErr := s.store.UpdateDelivery(ctx, &delivery); updateErr != nil {
			log.Error("Failed to update webhook delivery", "delivery_id", delivery.ID, "error", updateErr)
		}
	}

	return successCount, failCount, nil
}

// CleanupOldDeliveries removes old delivery records
func (s *DeliveryService) CleanupOldDeliveries(ctx context.Context) error {
	return s.store.DeleteOldDeliveries(ctx, s.config.RetentionDays)
}
