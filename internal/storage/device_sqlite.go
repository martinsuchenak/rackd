package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// Device operations

// GetDevice retrieves a device by ID with its addresses, tags, and domains
func (s *SQLiteStorage) GetDevice(ctx context.Context, id string) (*model.Device, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	// Get the device
	device := &model.Device{}
	var datacenterID, statusChangedBy sql.NullString
	var decommissionDate, statusChangedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, hostname, description, make_model, os, datacenter_id, username, location,
		       status, decommission_date, status_changed_at, status_changed_by, created_at, updated_at
		FROM devices WHERE id = ?
	`, id).Scan(
		&device.ID, &device.Name, &device.Hostname, &device.Description, &device.MakeModel,
		&device.OS, &datacenterID, &device.Username, &device.Location,
		&device.Status, &decommissionDate, &statusChangedAt, &statusChangedBy,
		&device.CreatedAt, &device.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrDeviceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if datacenterID.Valid {
		device.DatacenterID = datacenterID.String
	}
	if decommissionDate.Valid {
		device.DecommissionDate = &decommissionDate.Time
	}
	if statusChangedAt.Valid {
		device.StatusChangedAt = &statusChangedAt.Time
	}
	if statusChangedBy.Valid {
		device.StatusChangedBy = statusChangedBy.String
	}

	// Get addresses
	addresses, err := s.getDeviceAddresses(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device addresses: %w", err)
	}
	device.Addresses = addresses

	// Get tags
	tags, err := s.getDeviceTags(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device tags: %w", err)
	}
	device.Tags = tags

	// Get domains
	domains, err := s.getDeviceDomains(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device domains: %w", err)
	}
	device.Domains = domains

	// Get custom fields with definitions to get typed values
	customFieldsWithDefs, err := s.GetCustomFieldValuesWithDefinitions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get device custom fields: %w", err)
	}
	// Convert to input format
	for _, cf := range customFieldsWithDefs {
		device.CustomFields = append(device.CustomFields, model.CustomFieldValueInput{
			FieldID: cf.Definition.ID,
			Value:   cf.Value,
		})
	}

	return device, nil
}

// getDeviceAddresses retrieves all addresses for a device
func (s *SQLiteStorage) getDeviceAddresses(ctx context.Context, deviceID string) ([]model.Address, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, ip, port, type, label, network_id, switch_port, pool_id
		FROM addresses WHERE device_id = ?
	`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []model.Address
	for rows.Next() {
		var addr model.Address
		var networkID, switchPort, poolID sql.NullString
		var port sql.NullInt64
		if err := rows.Scan(&addr.ID, &addr.IP, &port, &addr.Type, &addr.Label, &networkID, &switchPort, &poolID); err != nil {
			return nil, err
		}
		if port.Valid {
			p := int(port.Int64)
			addr.Port = &p
		}
		if networkID.Valid {
			addr.NetworkID = networkID.String
		}
		if switchPort.Valid {
			addr.SwitchPort = switchPort.String
		}
		if poolID.Valid {
			addr.PoolID = poolID.String
		}
		addresses = append(addresses, addr)
	}

	if addresses == nil {
		addresses = []model.Address{}
	}

	return addresses, rows.Err()
}

// getDeviceTags retrieves all tags for a device
func (s *SQLiteStorage) getDeviceTags(ctx context.Context, deviceID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag FROM tags WHERE device_id = ?`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	if tags == nil {
		tags = []string{}
	}

	return tags, rows.Err()
}

// getDeviceDomains retrieves all domains for a device
func (s *SQLiteStorage) getDeviceDomains(ctx context.Context, deviceID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT domain FROM domains WHERE device_id = ?`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []string
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, err
		}
		domains = append(domains, domain)
	}

	if domains == nil {
		domains = []string{}
	}

	return domains, rows.Err()
}

// CreateDevice creates a new device with its addresses, tags, and domains
func (s *SQLiteStorage) CreateDevice(ctx context.Context, device *model.Device) error {
	if device == nil {
		return fmt.Errorf("device is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.createDeviceInTx(ctx, tx, device); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "device", device.ID, device)
	return nil
}

// createDeviceInTx creates a device within an existing transaction
func (s *SQLiteStorage) createDeviceInTx(ctx context.Context, tx *sql.Tx, device *model.Device) error {

	// Generate ID if not provided
	if device.ID == "" {
		device.ID = newUUID()
	}

	now := time.Now().UTC()
	device.CreatedAt = now
	device.UpdatedAt = now

	// Set default status if not provided
	if device.Status == "" {
		device.Status = model.DeviceStatusActive
	}

	// Set status changed at for new devices
	device.StatusChangedAt = &now

	// Insert device
	_, err := tx.ExecContext(ctx, `
		INSERT INTO devices (id, name, hostname, description, make_model, os, datacenter_id, username, location,
		                     status, decommission_date, status_changed_at, status_changed_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.Name, device.Hostname, device.Description, device.MakeModel,
		device.OS, nullString(device.DatacenterID), device.Username, device.Location,
		device.Status, nullTime(device.DecommissionDate), nullTime(device.StatusChangedAt),
		nullString(device.StatusChangedBy), device.CreatedAt, device.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert device: %w", err)
	}

	// Insert addresses
	if err := s.insertDeviceAddresses(ctx, tx, device.ID, device.Addresses); err != nil {
		return fmt.Errorf("failed to insert addresses: %w", err)
	}

	// Insert tags
	if err := s.insertDeviceTags(ctx, tx, device.ID, device.Tags); err != nil {
		return fmt.Errorf("failed to insert tags: %w", err)
	}

	// Insert domains
	if err := s.insertDeviceDomains(ctx, tx, device.ID, device.Domains); err != nil {
		return fmt.Errorf("failed to insert domains: %w", err)
	}

	// Insert custom fields
	if err := s.insertDeviceCustomFields(ctx, tx, device.ID, device.CustomFields); err != nil {
		return fmt.Errorf("failed to insert custom fields: %w", err)
	}

	return nil
}

// insertDeviceAddresses inserts addresses for a device within a transaction
func (s *SQLiteStorage) insertDeviceAddresses(ctx context.Context, tx *sql.Tx, deviceID string, addresses []model.Address) error {
	for _, addr := range addresses {
		id := addr.ID
		if id == "" {
			id = newUUID()
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO addresses (id, device_id, ip, port, type, label, network_id, switch_port, pool_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, deviceID, addr.IP, nullIntPtr(addr.Port), addr.Type, addr.Label,
			nullString(addr.NetworkID), nullString(addr.SwitchPort), nullString(addr.PoolID))
		if err != nil {
			return err
		}
	}
	return nil
}

// insertDeviceTags inserts tags for a device within a transaction
func (s *SQLiteStorage) insertDeviceTags(ctx context.Context, tx *sql.Tx, deviceID string, tags []string) error {
	for _, tag := range tags {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO tags (device_id, tag) VALUES (?, ?)
		`, deviceID, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

// insertDeviceDomains inserts domains for a device within a transaction
func (s *SQLiteStorage) insertDeviceDomains(ctx context.Context, tx *sql.Tx, deviceID string, domains []string) error {
	for _, domain := range domains {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO domains (device_id, domain) VALUES (?, ?)
		`, deviceID, domain)
		if err != nil {
			return err
		}
	}
	return nil
}

// insertDeviceCustomFields inserts custom fields for a device within a transaction
func (s *SQLiteStorage) insertDeviceCustomFields(ctx context.Context, tx *sql.Tx, deviceID string, customFields []model.CustomFieldValueInput) error {
	for _, cf := range customFields {
		if cf.FieldID == "" {
			continue
		}

		// Get the field type using the transaction context to avoid locking issues
		var fieldType model.CustomFieldType
		err := tx.QueryRowContext(ctx, `SELECT type FROM custom_field_definitions WHERE id = ?`, cf.FieldID).Scan(&fieldType)
		if err != nil {
			continue // Skip invalid field IDs
		}

		// Create the value record
		value := &model.CustomFieldValue{
			DeviceID: deviceID,
			FieldID:  cf.FieldID,
		}
		value.SetValue(fieldType, cf.Value)

		// Insert the value
		_, err = tx.ExecContext(ctx, `
			INSERT INTO custom_field_values (id, device_id, field_id, string_value, number_value, bool_value)
			VALUES (?, ?, ?, ?, ?, ?)
		`, newUUID(), value.DeviceID, value.FieldID, value.StringValue, value.NumberValue, value.BoolValue)
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateDevice updates an existing device and its related data
func (s *SQLiteStorage) UpdateDevice(ctx context.Context, device *model.Device) error {
	if device == nil {
		return fmt.Errorf("device is nil")
	}
	if device.ID == "" {
		return ErrInvalidID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.updateDeviceInTx(ctx, tx, device); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "update", "device", device.ID, device)
	return nil
}

// updateDeviceInTx updates a device within an existing transaction
func (s *SQLiteStorage) updateDeviceInTx(ctx context.Context, tx *sql.Tx, device *model.Device) error {

	// Check if device exists and get current status
	var currentStatus model.DeviceStatus
	err := tx.QueryRowContext(ctx, `SELECT status FROM devices WHERE id = ?`, device.ID).Scan(&currentStatus)
	if err == sql.ErrNoRows {
		return ErrDeviceNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to check device existence: %w", err)
	}

	device.UpdatedAt = time.Now().UTC()

	// Track status changes
	if device.Status != "" && device.Status != currentStatus {
		now := time.Now().UTC()
		device.StatusChangedAt = &now
		// StatusChangedBy should be set by the service layer from context
	} else if device.Status == "" {
		// Keep existing status if not provided
		device.Status = currentStatus
	}

	// Update device
	_, err = tx.ExecContext(ctx, `
		UPDATE devices SET
			name = ?, hostname = ?, description = ?, make_model = ?, os = ?, datacenter_id = ?,
			username = ?, location = ?, status = ?, decommission_date = ?,
			status_changed_at = ?, status_changed_by = ?, updated_at = ?
		WHERE id = ?
	`, device.Name, device.Hostname, device.Description, device.MakeModel, device.OS,
		nullString(device.DatacenterID), device.Username, device.Location,
		device.Status, nullTime(device.DecommissionDate),
		nullTime(device.StatusChangedAt), nullString(device.StatusChangedBy),
		device.UpdatedAt, device.ID)
	if err != nil {
		return fmt.Errorf("failed to update device: %w", err)
	}

	// Delete existing addresses, tags, domains and reinsert
	if _, err := tx.ExecContext(ctx, `DELETE FROM addresses WHERE device_id = ?`, device.ID); err != nil {
		return fmt.Errorf("failed to delete addresses: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tags WHERE device_id = ?`, device.ID); err != nil {
		return fmt.Errorf("failed to delete tags: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM domains WHERE device_id = ?`, device.ID); err != nil {
		return fmt.Errorf("failed to delete domains: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM custom_field_values WHERE device_id = ?`, device.ID); err != nil {
		return fmt.Errorf("failed to delete custom field values: %w", err)
	}

	// Insert new addresses, tags, domains
	if err := s.insertDeviceAddresses(ctx, tx, device.ID, device.Addresses); err != nil {
		return fmt.Errorf("failed to insert addresses: %w", err)
	}
	if err := s.insertDeviceTags(ctx, tx, device.ID, device.Tags); err != nil {
		return fmt.Errorf("failed to insert tags: %w", err)
	}
	if err := s.insertDeviceDomains(ctx, tx, device.ID, device.Domains); err != nil {
		return fmt.Errorf("failed to insert domains: %w", err)
	}
	if err := s.insertDeviceCustomFields(ctx, tx, device.ID, device.CustomFields); err != nil {
		return fmt.Errorf("failed to insert custom fields: %w", err)
	}

	return nil
}

// DeleteDevice removes a device and all related data (cascades via foreign keys)
func (s *SQLiteStorage) DeleteDevice(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.deleteDeviceInTx(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "delete", "device", id, nil)
	return nil
}

// deleteDeviceInTx deletes a device within an existing transaction
func (s *SQLiteStorage) deleteDeviceInTx(ctx context.Context, tx *sql.Tx, id string) error {

	// Check if device exists
	var exists bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check device existence: %w", err)
	}
	if !exists {
		return ErrDeviceNotFound
	}

	// Delete the device (cascades to addresses, tags, domains, relationships)
	_, err = tx.ExecContext(ctx, `DELETE FROM devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// ListDevices retrieves devices matching the filter criteria
func (s *SQLiteStorage) ListDevices(ctx context.Context, filter *model.DeviceFilter) ([]model.Device, error) {

	query := `SELECT id, name, hostname, description, make_model, os, datacenter_id, username, location,
	          status, decommission_date, status_changed_at, status_changed_by, created_at, updated_at
	          FROM devices`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.DatacenterID != "" {
			conditions = append(conditions, "datacenter_id = ?")
			args = append(args, filter.DatacenterID)
		}

		if filter.NetworkID != "" {
			conditions = append(conditions, "id IN (SELECT device_id FROM addresses WHERE network_id = ?)")
			args = append(args, filter.NetworkID)
		}

		if filter.PoolID != "" {
			conditions = append(conditions, "id IN (SELECT device_id FROM addresses WHERE pool_id = ?)")
			args = append(args, filter.PoolID)
		}

		if filter.Status != "" {
			conditions = append(conditions, "status = ?")
			args = append(args, filter.Status)
		}

		if len(filter.Tags) > 0 {
			// Match devices that have ALL specified tags
			for _, tag := range filter.Tags {
				conditions = append(conditions, "id IN (SELECT device_id FROM tags WHERE tag = ?)")
				args = append(args, tag)
			}
		}

		if filter.StaleDays > 0 {
			// Filter devices not seen in discovery for X days
			staleCutoff := time.Now().AddDate(0, 0, -filter.StaleDays)
			conditions = append(conditions, `status = 'active' AND NOT EXISTS (
				SELECT 1 FROM discovered_devices dd
				WHERE dd.promoted_to_device_id = devices.id
				AND dd.last_seen >= ?
			)`)
			args = append(args, staleCutoff)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	var pg *model.Pagination
	if filter != nil {
		pg = &filter.Pagination
	}
	query, args = appendPagination(query, args, pg)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var device model.Device
		var datacenterID, statusChangedBy sql.NullString
		var decommissionDate, statusChangedAt sql.NullTime
		if err := rows.Scan(
			&device.ID, &device.Name, &device.Hostname, &device.Description, &device.MakeModel,
			&device.OS, &datacenterID, &device.Username, &device.Location,
			&device.Status, &decommissionDate, &statusChangedAt, &statusChangedBy,
			&device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		if datacenterID.Valid {
			device.DatacenterID = datacenterID.String
		}
		if decommissionDate.Valid {
			device.DecommissionDate = &decommissionDate.Time
		}
		if statusChangedAt.Valid {
			device.StatusChangedAt = &statusChangedAt.Time
		}
		if statusChangedBy.Valid {
			device.StatusChangedBy = statusChangedBy.String
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load addresses, tags, and domains for each device
	for i := range devices {
		addresses, err := s.getDeviceAddresses(ctx, devices[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get addresses for device %s: %w", devices[i].ID, err)
		}
		devices[i].Addresses = addresses

		tags, err := s.getDeviceTags(ctx, devices[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for device %s: %w", devices[i].ID, err)
		}
		devices[i].Tags = tags

		domains, err := s.getDeviceDomains(ctx, devices[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get domains for device %s: %w", devices[i].ID, err)
		}
		devices[i].Domains = domains
	}

	if devices == nil {
		devices = []model.Device{}
	}

	return devices, nil
}

// SearchDevices performs a full-text search across device fields using FTS5
func (s *SQLiteStorage) SearchDevices(ctx context.Context, query string) ([]model.Device, error) {
	if query == "" {
		return s.ListDevices(ctx, nil)
	}
	ftsQuery := escapeFTSQuery(query)
	likePattern := "%" + query + "%"

	// Use UNION to combine FTS results with tag/domain/address matches
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os,
		       d.datacenter_id, d.username, d.location,
		       d.status, d.decommission_date, d.status_changed_at, d.status_changed_by,
		       d.created_at, d.updated_at
		FROM devices d
		INNER JOIN devices_fts fts ON d.id = fts.id
		WHERE devices_fts MATCH ?
		UNION
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os,
		       d.datacenter_id, d.username, d.location,
		       d.status, d.decommission_date, d.status_changed_at, d.status_changed_by,
		       d.created_at, d.updated_at
		FROM devices d
		INNER JOIN tags t ON d.id = t.device_id
		WHERE t.tag LIKE ?
		UNION
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os,
		       d.datacenter_id, d.username, d.location,
		       d.status, d.decommission_date, d.status_changed_at, d.status_changed_by,
		       d.created_at, d.updated_at
		FROM devices d
		INNER JOIN domains dm ON d.id = dm.device_id
		WHERE dm.domain LIKE ?
		UNION
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os,
		       d.datacenter_id, d.username, d.location,
		       d.status, d.decommission_date, d.status_changed_at, d.status_changed_by,
		       d.created_at, d.updated_at
		FROM devices d
		INNER JOIN addresses a ON d.id = a.device_id
		WHERE a.ip LIKE ?
		ORDER BY name
	`, ftsQuery, likePattern, likePattern, likePattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search devices: %w", err)
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var device model.Device
		var datacenterID, statusChangedBy sql.NullString
		var decommissionDate, statusChangedAt sql.NullTime
		if err := rows.Scan(
			&device.ID, &device.Name, &device.Hostname, &device.Description, &device.MakeModel,
			&device.OS, &datacenterID, &device.Username, &device.Location,
			&device.Status, &decommissionDate, &statusChangedAt, &statusChangedBy,
			&device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		if datacenterID.Valid {
			device.DatacenterID = datacenterID.String
		}
		if decommissionDate.Valid {
			device.DecommissionDate = &decommissionDate.Time
		}
		if statusChangedAt.Valid {
			device.StatusChangedAt = &statusChangedAt.Time
		}
		if statusChangedBy.Valid {
			device.StatusChangedBy = statusChangedBy.String
		}
		devices = append(devices, device)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load addresses, tags, and domains for each device
	for i := range devices {
		addresses, err := s.getDeviceAddresses(ctx, devices[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get addresses for device %s: %w", devices[i].ID, err)
		}
		devices[i].Addresses = addresses

		tags, err := s.getDeviceTags(ctx, devices[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for device %s: %w", devices[i].ID, err)
		}
		devices[i].Tags = tags

		domains, err := s.getDeviceDomains(ctx, devices[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get domains for device %s: %w", devices[i].ID, err)
		}
		devices[i].Domains = domains
	}

	if devices == nil {
		devices = []model.Device{}
	}

	return devices, nil
}

// GetDeviceStatusCounts returns the count of devices by status
func (s *SQLiteStorage) GetDeviceStatusCounts(ctx context.Context) (map[model.DeviceStatus]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) as count
		FROM devices
		GROUP BY status
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get device status counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[model.DeviceStatus]int)
	for rows.Next() {
		var status model.DeviceStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}
		counts[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}

// escapeFTSQuery escapes special FTS5 characters and adds prefix matching
func escapeFTSQuery(query string) string {
	// Escape double quotes by doubling them
	escaped := strings.ReplaceAll(query, `"`, `""`)
	// Wrap in quotes and add * for prefix matching
	return `"` + escaped + `"*`
}
