package storage

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/model"
)

var ErrScheduledScanNotFound = errors.New("scheduled scan not found")

type ScheduledScanStorage interface {
	Create(scan *model.ScheduledScan) error
	Update(scan *model.ScheduledScan) error
	Get(id string) (*model.ScheduledScan, error)
	List(networkID string) ([]model.ScheduledScan, error)
	Delete(id string) error
}

type SQLiteScheduledScanStorage struct {
	db *sql.DB
}

func NewSQLiteScheduledScanStorage(db *sql.DB) (*SQLiteScheduledScanStorage, error) {
	s := &SQLiteScheduledScanStorage{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteScheduledScanStorage) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS scheduled_scans (
			id TEXT PRIMARY KEY,
			network_id TEXT NOT NULL,
			profile_id TEXT NOT NULL,
			name TEXT NOT NULL,
			cron_expression TEXT NOT NULL,
			enabled INTEGER DEFAULT 1,
			description TEXT,
			last_run_at TIMESTAMP,
			next_run_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *SQLiteScheduledScanStorage) Create(scan *model.ScheduledScan) error {
	if err := scan.Validate(); err != nil {
		return err
	}
	if scan.ID == "" {
		scan.ID = uuid.Must(uuid.NewV7()).String()
	}
	now := time.Now()
	scan.CreatedAt = now
	scan.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO scheduled_scans (id, network_id, profile_id, name, cron_expression, enabled, description, last_run_at, next_run_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, scan.ID, scan.NetworkID, scan.ProfileID, scan.Name, scan.CronExpression, scan.Enabled, scan.Description, scan.LastRunAt, scan.NextRunAt, scan.CreatedAt, scan.UpdatedAt)
	return err
}

func (s *SQLiteScheduledScanStorage) Update(scan *model.ScheduledScan) error {
	scan.UpdatedAt = time.Now()

	res, err := s.db.Exec(`
		UPDATE scheduled_scans SET network_id=?, profile_id=?, name=?, cron_expression=?, enabled=?, description=?, last_run_at=?, next_run_at=?, updated_at=?
		WHERE id=?
	`, scan.NetworkID, scan.ProfileID, scan.Name, scan.CronExpression, scan.Enabled, scan.Description, scan.LastRunAt, scan.NextRunAt, scan.UpdatedAt, scan.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrScheduledScanNotFound
	}
	return nil
}

func (s *SQLiteScheduledScanStorage) Get(id string) (*model.ScheduledScan, error) {
	row := s.db.QueryRow(`SELECT id, network_id, profile_id, name, cron_expression, enabled, description, last_run_at, next_run_at, created_at, updated_at FROM scheduled_scans WHERE id=?`, id)
	return s.scanScheduledScan(row)
}

func (s *SQLiteScheduledScanStorage) List(networkID string) ([]model.ScheduledScan, error) {
	var rows *sql.Rows
	var err error
	if networkID == "" {
		rows, err = s.db.Query(`SELECT id, network_id, profile_id, name, cron_expression, enabled, description, last_run_at, next_run_at, created_at, updated_at FROM scheduled_scans ORDER BY name`)
	} else {
		rows, err = s.db.Query(`SELECT id, network_id, profile_id, name, cron_expression, enabled, description, last_run_at, next_run_at, created_at, updated_at FROM scheduled_scans WHERE network_id=? ORDER BY name`, networkID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []model.ScheduledScan
	for rows.Next() {
		var scan model.ScheduledScan
		var description sql.NullString
		var lastRunAt, nextRunAt sql.NullTime
		err := rows.Scan(&scan.ID, &scan.NetworkID, &scan.ProfileID, &scan.Name, &scan.CronExpression, &scan.Enabled, &description, &lastRunAt, &nextRunAt, &scan.CreatedAt, &scan.UpdatedAt)
		if err != nil {
			return nil, err
		}
		scan.Description = description.String
		if lastRunAt.Valid {
			scan.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			scan.NextRunAt = &nextRunAt.Time
		}
		scans = append(scans, scan)
	}
	return scans, rows.Err()
}

func (s *SQLiteScheduledScanStorage) Delete(id string) error {
	res, err := s.db.Exec(`DELETE FROM scheduled_scans WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrScheduledScanNotFound
	}
	return nil
}

func (s *SQLiteScheduledScanStorage) scanScheduledScan(row *sql.Row) (*model.ScheduledScan, error) {
	var scan model.ScheduledScan
	var description sql.NullString
	var lastRunAt, nextRunAt sql.NullTime
	err := row.Scan(&scan.ID, &scan.NetworkID, &scan.ProfileID, &scan.Name, &scan.CronExpression, &scan.Enabled, &description, &lastRunAt, &nextRunAt, &scan.CreatedAt, &scan.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrScheduledScanNotFound
	}
	if err != nil {
		return nil, err
	}
	scan.Description = description.String
	if lastRunAt.Valid {
		scan.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		scan.NextRunAt = &nextRunAt.Time
	}
	return &scan, nil
}
