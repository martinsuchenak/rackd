package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
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
		store: store,
		client: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		config: config,
	}
}

// Deliver sends a webhook to a specific endpoint
func (s *DeliveryService) Deliver(ctx context.Context, webhook *model.Webhook, event model.Event) (*model.WebhookDelivery, error) {
	payload, err := ToJSON(event)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}

	delivery := &model.WebhookDelivery{
		WebhookID:     webhook.ID,
		EventType:     event.Type,
		Payload:       payload,
		Status:        model.DeliveryStatusPending,
		AttemptNumber: 1,
		CreatedAt:     time.Now().UTC(),
	}

	// Attempt delivery
	startTime := time.Now()
	err = s.sendHTTPRequest(ctx, webhook, event.Type, payload)
	duration := time.Since(startTime)

	delivery.Duration = duration.Milliseconds()

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
		delivery.ResponseCode = 200
	}

	// Store delivery record
	if storeErr := s.store.CreateDelivery(ctx, delivery); storeErr != nil {
		return delivery, fmt.Errorf("delivery succeeded but failed to store record: %w", storeErr)
	}

	return delivery, err
}

// sendHTTPRequest sends the webhook payload via HTTP
func (s *DeliveryService) sendHTTPRequest(ctx context.Context, webhook *model.Webhook, eventType model.EventType, payload string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Rackd-Webhook/1.0")
	req.Header.Set("X-Event-Type", string(eventType))
	req.Header.Set("X-Delivery-ID", uuid.New().String())

	// Add HMAC signature if secret is configured
	if webhook.Secret != "" {
		signature := ComputeHMAC(payload, webhook.Secret)
		req.Header.Set("X-Signature-256", "sha256="+signature)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body (for logging)
	body, _ := io.ReadAll(resp.Body)
	_ = body // We don't use it but read it to drain the connection

	// Consider 2xx responses as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("webhook returned status %d", resp.StatusCode)
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
			_ = s.store.UpdateDelivery(ctx, &delivery)
			failCount++
			continue
		}

		// Only deliver to active webhooks
		if !webhook.Active {
			delivery.Status = model.DeliveryStatusAbandoned
			_ = s.store.UpdateDelivery(ctx, &delivery)
			failCount++
			continue
		}

		// Increment attempt counter
		delivery.AttemptNumber++

		// Attempt delivery
		startTime := time.Now()
		err = s.sendHTTPRequest(ctx, webhook, delivery.EventType, delivery.Payload)
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
			delivery.ResponseCode = 200
			delivery.Error = ""
			successCount++
		}

		if updateErr := s.store.UpdateDelivery(ctx, &delivery); updateErr != nil {
			// Log but continue processing
			fmt.Printf("failed to update delivery %s: %v\n", delivery.ID, updateErr)
		}
	}

	return successCount, failCount, nil
}

// CleanupOldDeliveries removes old delivery records
func (s *DeliveryService) CleanupOldDeliveries() error {
	return s.store.DeleteOldDeliveries(context.Background(), s.config.RetentionDays)
}

// ComputeHMAC computes an HMAC-SHA256 signature for the payload
func ComputeHMAC(payload, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
