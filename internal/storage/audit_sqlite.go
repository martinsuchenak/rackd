package storage

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateAuditLog creates a new audit log entry
func (s *SQLiteStorage) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = nowUTC()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.ID, log.Timestamp, log.Action, log.Resource, log.ResourceID, log.UserID, log.Username, log.IPAddress, log.Changes, log.Status, log.Error, log.Source)

	return err
}

// ListAuditLogs retrieves audit logs with optional filtering
func (s *SQLiteStorage) ListAuditLogs(ctx context.Context, filter *model.AuditFilter) ([]model.AuditLog, error) {
	query := `SELECT id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error, source FROM audit_logs WHERE 1=1`
	args := []interface{}{}

	if filter != nil {
		if filter.Resource != "" {
			query += " AND resource = ?"
			args = append(args, filter.Resource)
		}
		if filter.ResourceID != "" {
			query += " AND resource_id = ?"
			args = append(args, filter.ResourceID)
		}
		if filter.UserID != "" {
			query += " AND user_id = ?"
			args = append(args, filter.UserID)
		}
		if filter.Action != "" {
			query += " AND action = ?"
			args = append(args, filter.Action)
		}
		if filter.Source != "" {
			query += " AND source = ?"
			args = append(args, filter.Source)
		}
		if filter.StartTime != nil {
			query += " AND timestamp >= ?"
			args = append(args, filter.StartTime)
		}
		if filter.EndTime != nil {
			query += " AND timestamp <= ?"
			args = append(args, filter.EndTime)
		}
	}

	query += " ORDER BY timestamp DESC"

	query, args = appendPagination(query, args, &filter.Pagination)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var log model.AuditLog
		var source sql.NullString
		err := rows.Scan(&log.ID, &log.Timestamp, &log.Action, &log.Resource, &log.ResourceID, &log.UserID, &log.Username, &log.IPAddress, &log.Changes, &log.Status, &log.Error, &source)
		if err != nil {
			return nil, err
		}
		if source.Valid {
			log.Source = source.String
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetAuditLog retrieves a single audit log by ID
func (s *SQLiteStorage) GetAuditLog(ctx context.Context, id string) (*model.AuditLog, error) {
	var log model.AuditLog
	var source sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error, source
		FROM audit_logs WHERE id = ?
	`, id).Scan(&log.ID, &log.Timestamp, &log.Action, &log.Resource, &log.ResourceID, &log.UserID, &log.Username, &log.IPAddress, &log.Changes, &log.Status, &log.Error, &source)

	if err == sql.ErrNoRows {
		return nil, ErrAuditLogNotFound
	}
	if source.Valid {
		log.Source = source.String
	}
	return &log, err
}

// DeleteOldAuditLogs deletes audit logs older than specified days
func (s *SQLiteStorage) DeleteOldAuditLogs(ctx context.Context, olderThanDays int) error {
	cutoff := nowUTC().AddDate(0, 0, -olderThanDays)
	_, err := s.db.ExecContext(ctx, "DELETE FROM audit_logs WHERE timestamp < ?", cutoff)
	return err
}
