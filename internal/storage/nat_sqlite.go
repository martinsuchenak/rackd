package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateNATMapping creates a new NAT mapping
func (s *SQLiteStorage) CreateNATMapping(ctx context.Context, mapping *model.NATMapping) error {
	// Generate ID if not provided
	if mapping.ID == "" {
		mapping.ID = newUUID()
	}

	mapping.CreatedAt = time.Now().UTC()
	mapping.UpdatedAt = mapping.CreatedAt

	tagsJSON, _ := json.Marshal(mapping.Tags)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO nat_mappings (
			id, name, external_ip, external_port, internal_ip, internal_port,
			protocol, device_id, description, enabled, datacenter_id, network_id,
			tags, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		mapping.ID, mapping.Name, mapping.ExternalIP, mapping.ExternalPort,
		mapping.InternalIP, mapping.InternalPort, mapping.Protocol,
		nullString(mapping.DeviceID), mapping.Description, mapping.Enabled,
		nullString(mapping.DatacenterID), nullString(mapping.NetworkID),
		string(tagsJSON), mapping.CreatedAt, mapping.UpdatedAt,
	)

	return err
}

// GetNATMapping retrieves a NAT mapping by ID
func (s *SQLiteStorage) GetNATMapping(ctx context.Context, id string) (*model.NATMapping, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	mapping := &model.NATMapping{}
	var tagsJSON string
	var deviceID, datacenterID, networkID sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, external_ip, external_port, internal_ip, internal_port,
			protocol, device_id, description, enabled, datacenter_id, network_id,
			tags, created_at, updated_at
		FROM nat_mappings WHERE id = ?
	`, id).Scan(
		&mapping.ID, &mapping.Name, &mapping.ExternalIP, &mapping.ExternalPort,
		&mapping.InternalIP, &mapping.InternalPort, &mapping.Protocol,
		&deviceID, &mapping.Description, &mapping.Enabled,
		&datacenterID, &networkID,
		&tagsJSON, &mapping.CreatedAt, &mapping.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNATNotFound
	}
	if err != nil {
		return nil, err
	}

	if deviceID.Valid {
		mapping.DeviceID = deviceID.String
	}
	if datacenterID.Valid {
		mapping.DatacenterID = datacenterID.String
	}
	if networkID.Valid {
		mapping.NetworkID = networkID.String
	}

	if err := json.Unmarshal([]byte(tagsJSON), &mapping.Tags); err != nil {
		mapping.Tags = []string{}
	}

	return mapping, nil
}

// ListNATMappings lists NAT mappings with optional filtering
func (s *SQLiteStorage) ListNATMappings(ctx context.Context, filter *model.NATFilter) ([]model.NATMapping, error) {
	query := `SELECT id, name, external_ip, external_port, internal_ip, internal_port,
		protocol, device_id, description, enabled, datacenter_id, network_id,
		tags, created_at, updated_at
		FROM nat_mappings`

	var args []any
	var conditions []string

	if filter != nil {
		if filter.ExternalIP != "" {
			conditions = append(conditions, "external_ip = ?")
			args = append(args, filter.ExternalIP)
		}
		if filter.InternalIP != "" {
			conditions = append(conditions, "internal_ip = ?")
			args = append(args, filter.InternalIP)
		}
		if filter.Protocol != "" {
			conditions = append(conditions, "protocol = ?")
			args = append(args, filter.Protocol)
		}
		if filter.DeviceID != "" {
			conditions = append(conditions, "device_id = ?")
			args = append(args, filter.DeviceID)
		}
		if filter.DatacenterID != "" {
			conditions = append(conditions, "datacenter_id = ?")
			args = append(args, filter.DatacenterID)
		}
		if filter.NetworkID != "" {
			conditions = append(conditions, "network_id = ?")
			args = append(args, filter.NetworkID)
		}
		if filter.Enabled != nil {
			conditions = append(conditions, "enabled = ?")
			args = append(args, *filter.Enabled)
		}
		if len(filter.Tags) > 0 {
			for _, tag := range filter.Tags {
				conditions = append(conditions, "tags LIKE ?")
				args = append(args, "%\""+tag+"\"%")
			}
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}

	query += " ORDER BY name"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list NAT mappings: %w", err)
	}
	defer rows.Close()

	var mappings []model.NATMapping
	for rows.Next() {
		var mapping model.NATMapping
		var tagsJSON string
		var deviceID, datacenterID, networkID sql.NullString

		if err := rows.Scan(
			&mapping.ID, &mapping.Name, &mapping.ExternalIP, &mapping.ExternalPort,
			&mapping.InternalIP, &mapping.InternalPort, &mapping.Protocol,
			&deviceID, &mapping.Description, &mapping.Enabled,
			&datacenterID, &networkID,
			&tagsJSON, &mapping.CreatedAt, &mapping.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan NAT mapping: %w", err)
		}

		if deviceID.Valid {
			mapping.DeviceID = deviceID.String
		}
		if datacenterID.Valid {
			mapping.DatacenterID = datacenterID.String
		}
		if networkID.Valid {
			mapping.NetworkID = networkID.String
		}

		if err := json.Unmarshal([]byte(tagsJSON), &mapping.Tags); err != nil {
			mapping.Tags = []string{}
		}

		mappings = append(mappings, mapping)
	}

	if mappings == nil {
		mappings = []model.NATMapping{}
	}

	return mappings, nil
}

// UpdateNATMapping updates an existing NAT mapping
func (s *SQLiteStorage) UpdateNATMapping(ctx context.Context, mapping *model.NATMapping) error {
	if mapping.ID == "" {
		return ErrInvalidID
	}

	mapping.UpdatedAt = time.Now().UTC()

	tagsJSON, _ := json.Marshal(mapping.Tags)

	result, err := s.db.ExecContext(ctx, `
		UPDATE nat_mappings SET
			name = ?, external_ip = ?, external_port = ?, internal_ip = ?, internal_port = ?,
			protocol = ?, device_id = ?, description = ?, enabled = ?, datacenter_id = ?, network_id = ?,
			tags = ?, updated_at = ?
		WHERE id = ?
	`,
		mapping.Name, mapping.ExternalIP, mapping.ExternalPort,
		mapping.InternalIP, mapping.InternalPort, mapping.Protocol,
		nullString(mapping.DeviceID), mapping.Description, mapping.Enabled,
		nullString(mapping.DatacenterID), nullString(mapping.NetworkID),
		string(tagsJSON), mapping.UpdatedAt, mapping.ID,
	)

	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNATNotFound
	}

	return nil
}

// DeleteNATMapping deletes a NAT mapping
func (s *SQLiteStorage) DeleteNATMapping(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM nat_mappings WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrNATNotFound
	}

	return nil
}

// GetNATMappingsByDevice retrieves all NAT mappings for a device
func (s *SQLiteStorage) GetNATMappingsByDevice(ctx context.Context, deviceID string) ([]model.NATMapping, error) {
	return s.ListNATMappings(ctx, &model.NATFilter{DeviceID: deviceID})
}

// GetNATMappingsByDatacenter retrieves all NAT mappings for a datacenter
func (s *SQLiteStorage) GetNATMappingsByDatacenter(ctx context.Context, datacenterID string) ([]model.NATMapping, error) {
	return s.ListNATMappings(ctx, &model.NATFilter{DatacenterID: datacenterID})
}
