package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateWebhook stores a new webhook
func (s *SQLiteStorage) CreateWebhook(ctx context.Context, webhook *model.Webhook) error {
	if webhook.ID == "" {
		webhook.ID = newUUID()
	}
	webhook.CreatedAt = time.Now().UTC()
	webhook.UpdatedAt = webhook.CreatedAt

	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO webhooks (id, name, url, secret, events, active, description, created_at, updated_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, webhook.ID, webhook.Name, webhook.URL, webhook.Secret, string(eventsJSON),
		webhook.Active, webhook.Description, webhook.CreatedAt, webhook.UpdatedAt, webhook.CreatedBy)

	return err
}

// GetWebhook retrieves a webhook by ID
func (s *SQLiteStorage) GetWebhook(ctx context.Context, id string) (*model.Webhook, error) {
	webhook := &model.Webhook{}
	var eventsJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, url, secret, events, active, description, created_at, updated_at, created_by
		FROM webhooks WHERE id = ?
	`, id).Scan(&webhook.ID, &webhook.Name, &webhook.URL, &webhook.Secret, &eventsJSON,
		&webhook.Active, &webhook.Description, &webhook.CreatedAt, &webhook.UpdatedAt, &webhook.CreatedBy)

	if err == sql.ErrNoRows {
		return nil, ErrWebhookNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(eventsJSON), &webhook.Events); err != nil {
		return nil, err
	}

	return webhook, nil
}

// ListWebhooks retrieves webhooks matching filter criteria
func (s *SQLiteStorage) ListWebhooks(ctx context.Context, filter *model.WebhookFilter) ([]model.Webhook, error) {
	query := `SELECT id, name, url, secret, events, active, description, created_at, updated_at, created_by
		FROM webhooks WHERE 1=1`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.Active != nil {
			conditions = append(conditions, "active = ?")
			args = append(args, *filter.Active)
		}
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanWebhooks(rows)
}

// UpdateWebhook updates an existing webhook
func (s *SQLiteStorage) UpdateWebhook(ctx context.Context, webhook *model.Webhook) error {
	webhook.UpdatedAt = time.Now().UTC()

	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE webhooks SET name = ?, url = ?, secret = ?, events = ?, active = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, webhook.Name, webhook.URL, webhook.Secret, string(eventsJSON), webhook.Active, webhook.Description,
		webhook.UpdatedAt, webhook.ID)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWebhookNotFound
	}

	return nil
}

// DeleteWebhook removes a webhook
func (s *SQLiteStorage) DeleteWebhook(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrWebhookNotFound
	}

	// Also delete associated deliveries
	_, _ = s.db.ExecContext(ctx, `DELETE FROM webhook_deliveries WHERE webhook_id = ?`, id)

	return nil
}

// GetWebhooksForEvent retrieves all active webhooks subscribed to a specific event
func (s *SQLiteStorage) GetWebhooksForEvent(ctx context.Context, eventType model.EventType) ([]model.Webhook, error) {
	// Use JSON functions to ensure exact matches within the events array
	query := `SELECT id, name, url, secret, events, active, description, created_at, updated_at, created_by
		FROM webhooks WHERE active = 1 AND EXISTS (SELECT 1 FROM json_each(events) WHERE value = ?)`

	rows, err := s.db.QueryContext(ctx, query, string(eventType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	webhooks, err := scanWebhooks(rows)
	if err != nil {
		return nil, err
	}

	return webhooks, nil
}

// CreateDelivery stores a new webhook delivery attempt
func (s *SQLiteStorage) CreateDelivery(ctx context.Context, delivery *model.WebhookDelivery) error {
	if delivery.ID == "" {
		delivery.ID = newUUID()
	}
	delivery.CreatedAt = time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (id, webhook_id, event_type, payload, response_code, response_body, error, duration_ms, status, attempt_number, next_retry, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, delivery.ID, delivery.WebhookID, delivery.EventType, delivery.Payload,
		delivery.ResponseCode, delivery.ResponseBody, delivery.Error, delivery.Duration,
		delivery.Status, delivery.AttemptNumber, delivery.NextRetry, delivery.CreatedAt)

	return err
}

// GetDelivery retrieves a delivery by ID
func (s *SQLiteStorage) GetDelivery(ctx context.Context, id string) (*model.WebhookDelivery, error) {
	delivery := &model.WebhookDelivery{}

	err := s.db.QueryRowContext(ctx, `
		SELECT id, webhook_id, event_type, payload, response_code, response_body, error, duration_ms, status, attempt_number, next_retry, created_at
		FROM webhook_deliveries WHERE id = ?
	`, id).Scan(&delivery.ID, &delivery.WebhookID, &delivery.EventType, &delivery.Payload,
		&delivery.ResponseCode, &delivery.ResponseBody, &delivery.Error, &delivery.Duration,
		&delivery.Status, &delivery.AttemptNumber, &delivery.NextRetry, &delivery.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDeliveryNotFound
	}
	if err != nil {
		return nil, err
	}

	return delivery, nil
}

// ListDeliveries retrieves deliveries matching filter criteria
func (s *SQLiteStorage) ListDeliveries(ctx context.Context, filter *model.DeliveryFilter) ([]model.WebhookDelivery, error) {
	query := `SELECT id, webhook_id, event_type, payload, response_code, response_body, error, duration_ms, status, attempt_number, next_retry, created_at
		FROM webhook_deliveries WHERE 1=1`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.WebhookID != "" {
			conditions = append(conditions, "webhook_id = ?")
			args = append(args, filter.WebhookID)
		}
		if filter.Status != "" {
			conditions = append(conditions, "status = ?")
			args = append(args, filter.Status)
		}
		if filter.EventType != "" {
			conditions = append(conditions, "event_type = ?")
			args = append(args, filter.EventType)
		}
		if filter.After != nil {
			conditions = append(conditions, "created_at >= ?")
			args = append(args, filter.After)
		}
		if filter.Before != nil {
			conditions = append(conditions, "created_at <= ?")
			args = append(args, filter.Before)
		}
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	query, args = appendPagination(query, args, &filter.Pagination)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeliveries(rows)
}

// UpdateDelivery updates a delivery record
func (s *SQLiteStorage) UpdateDelivery(ctx context.Context, delivery *model.WebhookDelivery) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE webhook_deliveries SET response_code = ?, response_body = ?, error = ?, duration_ms = ?, status = ?, attempt_number = ?, next_retry = ?
		WHERE id = ?
	`, delivery.ResponseCode, delivery.ResponseBody, delivery.Error, delivery.Duration,
		delivery.Status, delivery.AttemptNumber, delivery.NextRetry, delivery.ID)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDeliveryNotFound
	}

	return nil
}

// DeleteOldDeliveries removes delivery records older than specified days
func (s *SQLiteStorage) DeleteOldDeliveries(ctx context.Context, olderThanDays int) error {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhook_deliveries WHERE created_at < ?`, cutoff)
	return err
}

// GetPendingDeliveries retrieves deliveries that are pending retry
func (s *SQLiteStorage) GetPendingDeliveries(ctx context.Context, limit int) ([]model.WebhookDelivery, error) {
	query := `SELECT id, webhook_id, event_type, payload, response_code, response_body, error, duration_ms, status, attempt_number, next_retry, created_at
		FROM webhook_deliveries
		WHERE status IN (?, ?) AND next_retry IS NOT NULL AND next_retry <= ?
		ORDER BY next_retry ASC`

	if limit > 0 {
		query += " LIMIT ?"
	}

	rows, err := s.db.QueryContext(ctx, query, model.DeliveryStatusPending, model.DeliveryStatusRetrying, time.Now().UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDeliveries(rows)
}

// scanWebhooks helper function
func scanWebhooks(rows *sql.Rows) ([]model.Webhook, error) {
	var webhooks []model.Webhook
	for rows.Next() {
		var w model.Webhook
		var eventsJSON string
		var createdBy sql.NullString
		var secret, description sql.NullString

		if err := rows.Scan(&w.ID, &w.Name, &w.URL, &secret, &eventsJSON,
			&w.Active, &description, &w.CreatedAt, &w.UpdatedAt, &createdBy); err != nil {
			return nil, err
		}

		w.Secret = secret.String
		w.Description = description.String
		w.CreatedBy = createdBy.String

		if err := json.Unmarshal([]byte(eventsJSON), &w.Events); err != nil {
			return nil, err
		}

		webhooks = append(webhooks, w)
	}
	return webhooks, rows.Err()
}

// scanDeliveries helper function
func scanDeliveries(rows *sql.Rows) ([]model.WebhookDelivery, error) {
	var deliveries []model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		var responseBody, errMsg, nextRetry sql.NullString
		var responseCode sql.NullInt64
		var duration sql.NullInt64

		if err := rows.Scan(&d.ID, &d.WebhookID, &d.EventType, &d.Payload,
			&responseCode, &responseBody, &errMsg, &duration,
			&d.Status, &d.AttemptNumber, &nextRetry, &d.CreatedAt); err != nil {
			return nil, err
		}

		if responseCode.Valid {
			d.ResponseCode = int(responseCode.Int64)
		}
		d.ResponseBody = responseBody.String
		d.Error = errMsg.String
		if duration.Valid {
			d.Duration = duration.Int64
		}
		if nextRetry.Valid && nextRetry.String != "" {
			t, err := time.Parse(time.RFC3339, nextRetry.String)
			if err == nil {
				d.NextRetry = &t
			}
		}

		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}
