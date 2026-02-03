package storage

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateAuditLog creates a new audit log entry
func (s *SQLiteStorage) CreateAuditLog(log *model.AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	_, err := s.db.Exec(`
		INSERT INTO audit_logs (id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.ID, log.Timestamp, log.Action, log.Resource, log.ResourceID, log.UserID, log.Username, log.IPAddress, log.Changes, log.Status, log.Error)

	return err
}

// ListAuditLogs retrieves audit logs with optional filtering
func (s *SQLiteStorage) ListAuditLogs(filter *model.AuditFilter) ([]model.AuditLog, error) {
	query := `SELECT id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error FROM audit_logs WHERE 1=1`
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

	if filter != nil && filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
		if filter.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var log model.AuditLog
		err := rows.Scan(&log.ID, &log.Timestamp, &log.Action, &log.Resource, &log.ResourceID, &log.UserID, &log.Username, &log.IPAddress, &log.Changes, &log.Status, &log.Error)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// GetAuditLog retrieves a single audit log by ID
func (s *SQLiteStorage) GetAuditLog(id string) (*model.AuditLog, error) {
	var log model.AuditLog
	err := s.db.QueryRow(`
		SELECT id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error
		FROM audit_logs WHERE id = ?
	`, id).Scan(&log.ID, &log.Timestamp, &log.Action, &log.Resource, &log.ResourceID, &log.UserID, &log.Username, &log.IPAddress, &log.Changes, &log.Status, &log.Error)

	if err == sql.ErrNoRows {
		return nil, ErrAuditLogNotFound
	}
	return &log, err
}

// DeleteOldAuditLogs deletes audit logs older than specified days
func (s *SQLiteStorage) DeleteOldAuditLogs(olderThanDays int) error {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	_, err := s.db.Exec("DELETE FROM audit_logs WHERE timestamp < ?", cutoff)
	return err
}
