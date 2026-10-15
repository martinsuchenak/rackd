package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/martinsuchenak/rackd/internal/version"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// Request headers sent with every webhook delivery.
const (
	HeaderEvent     = "X-Rackd-Event"
	HeaderDelivery  = "X-Rackd-Delivery"
	HeaderTimestamp = "X-Rackd-Timestamp"
	HeaderSignature = "X-Rackd-Signature"
)

// Message is the JSON body of a webhook request.
type Message struct {
	Event model.EventType `json:"event"`
	// DeliveryID identifies the delivery and stays the same across retries, so
	// receivers can ignore duplicates.
	DeliveryID string `json:"delivery_id"`
	// Timestamp is when the event happened.
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// NewMessage returns the body for delivering event under deliveryID.
func NewMessage(event model.Event, deliveryID string) ([]byte, error) {
	ts := event.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	body, err := json.Marshal(Message{Event: event.Type, DeliveryID: deliveryID, Timestamp: ts, Data: event.Payload})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize event: %w", err)
	}
	return body, nil
}

// Sign returns the X-Rackd-Signature value for body sent at unix time ts:
// "sha256=" followed by the hex HMAC-SHA256 of "<ts>.<body>" keyed with
// secret. Signing the timestamp lets receivers reject replayed requests.
func Sign(secret string, ts int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// NewRequest builds the POST that delivers body to webhook, timestamped and
// signed at sentAt. The signature is omitted when the webhook has no secret.
func NewRequest(ctx context.Context, webhook *model.Webhook, eventType model.EventType, deliveryID string, body []byte, sentAt time.Time) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	ts := sentAt.Unix()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Rackd-Webhook/"+version.Version)
	req.Header.Set(HeaderEvent, string(eventType))
	req.Header.Set(HeaderDelivery, deliveryID)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(ts, 10))
	if webhook.Secret != "" {
		req.Header.Set(HeaderSignature, Sign(webhook.Secret, ts, body))
	}
	return req, nil
}

// Send performs req and returns the response status. Any non-2xx status is
// also returned as an error.
func Send(client *http.Client, req *http.Request) (int, error) {
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", withoutURL(err))
	}
	defer func() { _ = resp.Body.Close() }()
	// Drain the body without loading it fully into memory: a hostile target
	// could otherwise return an arbitrarily large response (× concurrency × retries).
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

// withoutURL drops the request URL net/http puts in its errors. A webhook URL
// can itself be the secret (Slack and Teams incoming webhooks, a ?token=), and
// delivery errors are stored and shown to anyone who can read deliveries.
func withoutURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s: %w", urlErr.Op, urlErr.Err)
	}
	return err
}
