package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Datacenter operations

// ensureDefaultDatacenter creates a default datacenter if none exists
func (s *SQLiteStorage) ensureDefaultDatacenter(ctx context.Context) error {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM datacenters`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check datacenter count: %w", err)
	}

	if count == 0 {
		defaultDC := &model.Datacenter{
			ID:          newUUID(),
			Name:        "Default",
			Location:    "",
			Description: "Default datacenter",
		}
		if err := s.CreateDatacenter(ctx, defaultDC); err != nil {
			return fmt.Errorf("failed to create default datacenter: %w", err)
		}
	}

	return nil
}

// ListDatacenters retrieves all datacenters matching the filter criteria
func (s *SQLiteStorage) ListDatacenters(ctx context.Context, filter *model.DatacenterFilter) ([]model.Datacenter, error) {

	query := `SELECT id, name, location, description, created_at, updated_at FROM datacenters`
	var args []any

	if filter != nil && filter.Name != "" {
		query += " WHERE name LIKE ?"
		args = append(args, "%"+filter.Name+"%")
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list datacenters: %w", err)
	}
	defer rows.Close()

	var datacenters []model.Datacenter
	for rows.Next() {
		var dc model.Datacenter
		if err := rows.Scan(&dc.ID, &dc.Name, &dc.Location, &dc.Description, &dc.CreatedAt, &dc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan datacenter: %w", err)
		}
		datacenters = append(datacenters, dc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if datacenters == nil {
		datacenters = []model.Datacenter{}
	}

	return datacenters, nil
}

// SearchDatacenters performs a full-text search across datacenter fields using FTS5
func (s *SQLiteStorage) SearchDatacenters(ctx context.Context, query string) ([]model.Datacenter, error) {
	if query == "" {
		return s.ListDatacenters(ctx, nil)
	}

	ftsQuery := escapeFTSQuery(query)

	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.name, d.location, d.description, d.created_at, d.updated_at
		FROM datacenters d
		INNER JOIN datacenters_fts fts ON d.id = fts.id
		WHERE datacenters_fts MATCH ?
		ORDER BY d.name
	`, ftsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search datacenters: %w", err)
	}
	defer rows.Close()

	var datacenters []model.Datacenter
	for rows.Next() {
		var dc model.Datacenter
		if err := rows.Scan(&dc.ID, &dc.Name, &dc.Location, &dc.Description, &dc.CreatedAt, &dc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan datacenter: %w", err)
		}
		datacenters = append(datacenters, dc)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if datacenters == nil {
		datacenters = []model.Datacenter{}
	}

	return datacenters, nil
}

// GetDatacenter retrieves a datacenter by ID
func (s *SQLiteStorage) GetDatacenter(ctx context.Context, id string) (*model.Datacenter, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	dc := &model.Datacenter{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, location, description, created_at, updated_at
		FROM datacenters WHERE id = ?
	`, id).Scan(&dc.ID, &dc.Name, &dc.Location, &dc.Description, &dc.CreatedAt, &dc.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDatacenterNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get datacenter: %w", err)
	}

	return dc, nil
}

// CreateDatacenter creates a new datacenter
func (s *SQLiteStorage) CreateDatacenter(ctx context.Context, dc *model.Datacenter) error {
	if dc == nil {
		return fmt.Errorf("datacenter is nil")
	}

	// Generate ID if not provided
	if dc.ID == "" {
		dc.ID = newUUID()
	}

	now := time.Now().UTC()
	dc.CreatedAt = now
	dc.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO datacenters (id, name, location, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, dc.ID, dc.Name, dc.Location, dc.Description, dc.CreatedAt, dc.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create datacenter: %w", err)
	}

	s.auditLog(ctx, "create", "datacenter", dc.ID, dc)
	return nil
}

// UpdateDatacenter updates an existing datacenter
func (s *SQLiteStorage) UpdateDatacenter(ctx context.Context, dc *model.Datacenter) error {
	if dc == nil {
		return fmt.Errorf("datacenter is nil")
	}
	if dc.ID == "" {
		return ErrInvalidID
	}

	// Check if datacenter exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM datacenters WHERE id = ?)`, dc.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check datacenter existence: %w", err)
	}
	if !exists {
		return ErrDatacenterNotFound
	}

	dc.UpdatedAt = time.Now().UTC()

	_, err = s.db.ExecContext(ctx, `
		UPDATE datacenters SET name = ?, location = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, dc.Name, dc.Location, dc.Description, dc.UpdatedAt, dc.ID)

	if err != nil {
		return fmt.Errorf("failed to update datacenter: %w", err)
	}

	s.auditLog(ctx, "update", "datacenter", dc.ID, dc)
	return nil
}

// DeleteDatacenter removes a datacenter by ID
func (s *SQLiteStorage) DeleteDatacenter(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	// Check if datacenter exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM datacenters WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check datacenter existence: %w", err)
	}
	if !exists {
		return ErrDatacenterNotFound
	}

	// Check for dependent devices
	var deviceCount int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM devices WHERE datacenter_id = ?`, id).Scan(&deviceCount)
	if err != nil {
		return fmt.Errorf("failed to check dependent devices: %w", err)
	}

	// Unlink devices from datacenter (set datacenter_id to NULL)
	if deviceCount > 0 {
		_, err = s.db.ExecContext(ctx, `UPDATE devices SET datacenter_id = NULL WHERE datacenter_id = ?`, id)
		if err != nil {
			return fmt.Errorf("failed to unlink devices: %w", err)
		}
	}

	// Delete the datacenter
	_, err = s.db.ExecContext(ctx, `DELETE FROM datacenters WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete datacenter: %w", err)
	}

	s.auditLog(ctx, "delete", "datacenter", id, nil)
	return nil
}

// GetDatacenterDevices retrieves all devices in a datacenter
func (s *SQLiteStorage) GetDatacenterDevices(ctx context.Context, datacenterID string) ([]model.Device, error) {
	if datacenterID == "" {
		return nil, ErrInvalidID
	}

	// First check if the datacenter exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM datacenters WHERE id = ?)`, datacenterID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check datacenter existence: %w", err)
	}
	if !exists {
		return nil, ErrDatacenterNotFound
	}

	// Use ListDevices with a filter
	return s.ListDevices(ctx, &model.DeviceFilter{DatacenterID: datacenterID})
}
