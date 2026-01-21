package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// CreateDiscoveredDevice inserts a new discovered device
func (s *SQLiteStorage) CreateDiscoveredDevice(device *model.DiscoveredDevice) error {
	ctx := context.Background()
	if device.ID == "" {
		device.ID = newUUID()
	}
	now := time.Now()
	device.CreatedAt = now
	device.UpdatedAt = now
	if device.FirstSeen.IsZero() {
		device.FirstSeen = now
	}
	device.LastSeen = now

	openPorts, _ := json.Marshal(device.OpenPorts)
	services, _ := json.Marshal(device.Services)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO discovered_devices (id, ip, mac_address, hostname, network_id, status, confidence,
			os_guess, vendor, open_ports, services, first_seen, last_seen, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.IP, device.MACAddress, device.Hostname, device.NetworkID, device.Status,
		device.Confidence, device.OSGuess, device.Vendor, string(openPorts), string(services),
		device.FirstSeen, device.LastSeen, device.CreatedAt, device.UpdatedAt)
	return err
}

// UpdateDiscoveredDevice updates an existing discovered device
func (s *SQLiteStorage) UpdateDiscoveredDevice(device *model.DiscoveredDevice) error {
	ctx := context.Background()
	device.UpdatedAt = time.Now()
	device.LastSeen = device.UpdatedAt

	openPorts, _ := json.Marshal(device.OpenPorts)
	services, _ := json.Marshal(device.Services)

	result, err := s.db.ExecContext(ctx, `
		UPDATE discovered_devices SET ip = ?, mac_address = ?, hostname = ?, network_id = ?,
			status = ?, confidence = ?, os_guess = ?, vendor = ?, open_ports = ?, services = ?,
			last_seen = ?, updated_at = ?
		WHERE id = ?
	`, device.IP, device.MACAddress, device.Hostname, device.NetworkID, device.Status,
		device.Confidence, device.OSGuess, device.Vendor, string(openPorts), string(services),
		device.LastSeen, device.UpdatedAt, device.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDiscoveryNotFound
	}
	return nil
}

// GetDiscoveredDevice retrieves a discovered device by ID
func (s *SQLiteStorage) GetDiscoveredDevice(id string) (*model.DiscoveredDevice, error) {
	ctx := context.Background()
	var d model.DiscoveredDevice
	var openPorts, services, promotedToDeviceID sql.NullString
	var promotedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, ip, mac_address, hostname, network_id, status, confidence, os_guess, vendor,
			open_ports, services, first_seen, last_seen, promoted_to_device_id, promoted_at,
			created_at, updated_at
		FROM discovered_devices WHERE id = ?
	`, id).Scan(&d.ID, &d.IP, &d.MACAddress, &d.Hostname, &d.NetworkID, &d.Status, &d.Confidence,
		&d.OSGuess, &d.Vendor, &openPorts, &services, &d.FirstSeen, &d.LastSeen,
		&promotedToDeviceID, &promotedAt, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrDiscoveryNotFound
	}
	if err != nil {
		return nil, err
	}

	if openPorts.Valid {
		json.Unmarshal([]byte(openPorts.String), &d.OpenPorts)
	}
	if services.Valid {
		json.Unmarshal([]byte(services.String), &d.Services)
	}
	if promotedToDeviceID.Valid {
		d.PromotedToDeviceID = promotedToDeviceID.String
	}
	if promotedAt.Valid {
		d.PromotedAt = &promotedAt.Time
	}
	return &d, nil
}

// GetDiscoveredDeviceByIP retrieves a discovered device by network and IP
func (s *SQLiteStorage) GetDiscoveredDeviceByIP(networkID, ip string) (*model.DiscoveredDevice, error) {
	ctx := context.Background()
	var d model.DiscoveredDevice
	var openPorts, services, promotedToDeviceID sql.NullString
	var promotedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, ip, mac_address, hostname, network_id, status, confidence, os_guess, vendor,
			open_ports, services, first_seen, last_seen, promoted_to_device_id, promoted_at,
			created_at, updated_at
		FROM discovered_devices WHERE network_id = ? AND ip = ?
	`, networkID, ip).Scan(&d.ID, &d.IP, &d.MACAddress, &d.Hostname, &d.NetworkID, &d.Status,
		&d.Confidence, &d.OSGuess, &d.Vendor, &openPorts, &services, &d.FirstSeen, &d.LastSeen,
		&promotedToDeviceID, &promotedAt, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrDiscoveryNotFound
	}
	if err != nil {
		return nil, err
	}

	if openPorts.Valid {
		json.Unmarshal([]byte(openPorts.String), &d.OpenPorts)
	}
	if services.Valid {
		json.Unmarshal([]byte(services.String), &d.Services)
	}
	if promotedToDeviceID.Valid {
		d.PromotedToDeviceID = promotedToDeviceID.String
	}
	if promotedAt.Valid {
		d.PromotedAt = &promotedAt.Time
	}
	return &d, nil
}

// ListDiscoveredDevices returns all discovered devices for a network
func (s *SQLiteStorage) ListDiscoveredDevices(networkID string) ([]model.DiscoveredDevice, error) {
	ctx := context.Background()
	query := `SELECT id, ip, mac_address, hostname, network_id, status, confidence, os_guess, vendor,
		open_ports, services, first_seen, last_seen, promoted_to_device_id, promoted_at,
		created_at, updated_at FROM discovered_devices`
	var args []any
	if networkID != "" {
		query += " WHERE network_id = ?"
		args = append(args, networkID)
	}
	query += " ORDER BY last_seen DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.DiscoveredDevice
	for rows.Next() {
		var d model.DiscoveredDevice
		var openPorts, services, promotedToDeviceID sql.NullString
		var promotedAt sql.NullTime
		if err := rows.Scan(&d.ID, &d.IP, &d.MACAddress, &d.Hostname, &d.NetworkID, &d.Status,
			&d.Confidence, &d.OSGuess, &d.Vendor, &openPorts, &services, &d.FirstSeen, &d.LastSeen,
			&promotedToDeviceID, &promotedAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		if openPorts.Valid {
			json.Unmarshal([]byte(openPorts.String), &d.OpenPorts)
		}
		if services.Valid {
			json.Unmarshal([]byte(services.String), &d.Services)
		}
		if promotedToDeviceID.Valid {
			d.PromotedToDeviceID = promotedToDeviceID.String
		}
		if promotedAt.Valid {
			d.PromotedAt = &promotedAt.Time
		}
		devices = append(devices, d)
	}
	return devices, rows.Err()
}

// DeleteDiscoveredDevice removes a discovered device
func (s *SQLiteStorage) DeleteDiscoveredDevice(id string) error {
	result, err := s.db.ExecContext(context.Background(),
		"DELETE FROM discovered_devices WHERE id = ?", id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDiscoveryNotFound
	}
	return nil
}

// PromoteDiscoveredDevice links a discovered device to a created device
func (s *SQLiteStorage) PromoteDiscoveredDevice(discoveredID, deviceID string) error {
	now := time.Now()
	result, err := s.db.ExecContext(context.Background(), `
		UPDATE discovered_devices SET promoted_to_device_id = ?, promoted_at = ?, updated_at = ?
		WHERE id = ?
	`, deviceID, now, now, discoveredID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrDiscoveryNotFound
	}
	return nil
}

// CreateDiscoveryScan inserts a new discovery scan
func (s *SQLiteStorage) CreateDiscoveryScan(scan *model.DiscoveryScan) error {
	ctx := context.Background()
	if scan.ID == "" {
		scan.ID = newUUID()
	}
	now := time.Now()
	scan.CreatedAt = now
	scan.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO discovery_scans (id, network_id, status, scan_type, total_hosts, scanned_hosts,
			found_hosts, progress_percent, error_message, started_at, completed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, scan.ID, scan.NetworkID, scan.Status, scan.ScanType, scan.TotalHosts, scan.ScannedHosts,
		scan.FoundHosts, scan.ProgressPercent, scan.ErrorMessage, scan.StartedAt, scan.CompletedAt,
		scan.CreatedAt, scan.UpdatedAt)
	return err
}

// UpdateDiscoveryScan updates an existing discovery scan
func (s *SQLiteStorage) UpdateDiscoveryScan(scan *model.DiscoveryScan) error {
	scan.UpdatedAt = time.Now()
	result, err := s.db.ExecContext(context.Background(), `
		UPDATE discovery_scans SET status = ?, scan_type = ?, total_hosts = ?, scanned_hosts = ?,
			found_hosts = ?, progress_percent = ?, error_message = ?, started_at = ?, completed_at = ?,
			updated_at = ?
		WHERE id = ?
	`, scan.Status, scan.ScanType, scan.TotalHosts, scan.ScannedHosts, scan.FoundHosts,
		scan.ProgressPercent, scan.ErrorMessage, scan.StartedAt, scan.CompletedAt, scan.UpdatedAt, scan.ID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrScanNotFound
	}
	return nil
}

// GetDiscoveryScan retrieves a discovery scan by ID
func (s *SQLiteStorage) GetDiscoveryScan(id string) (*model.DiscoveryScan, error) {
	var scan model.DiscoveryScan
	var startedAt, completedAt sql.NullTime
	err := s.db.QueryRowContext(context.Background(), `
		SELECT id, network_id, status, scan_type, total_hosts, scanned_hosts, found_hosts,
			progress_percent, error_message, started_at, completed_at, created_at, updated_at
		FROM discovery_scans WHERE id = ?
	`, id).Scan(&scan.ID, &scan.NetworkID, &scan.Status, &scan.ScanType, &scan.TotalHosts,
		&scan.ScannedHosts, &scan.FoundHosts, &scan.ProgressPercent, &scan.ErrorMessage,
		&startedAt, &completedAt, &scan.CreatedAt, &scan.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrScanNotFound
	}
	if err != nil {
		return nil, err
	}
	if startedAt.Valid {
		scan.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		scan.CompletedAt = &completedAt.Time
	}
	return &scan, nil
}

// ListDiscoveryScans returns all scans for a network
func (s *SQLiteStorage) ListDiscoveryScans(networkID string) ([]model.DiscoveryScan, error) {
	query := `SELECT id, network_id, status, scan_type, total_hosts, scanned_hosts, found_hosts,
		progress_percent, error_message, started_at, completed_at, created_at, updated_at
		FROM discovery_scans`
	var args []any
	if networkID != "" {
		query += " WHERE network_id = ?"
		args = append(args, networkID)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(context.Background(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scans []model.DiscoveryScan
	for rows.Next() {
		var scan model.DiscoveryScan
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(&scan.ID, &scan.NetworkID, &scan.Status, &scan.ScanType, &scan.TotalHosts,
			&scan.ScannedHosts, &scan.FoundHosts, &scan.ProgressPercent, &scan.ErrorMessage,
			&startedAt, &completedAt, &scan.CreatedAt, &scan.UpdatedAt); err != nil {
			return nil, err
		}
		if startedAt.Valid {
			scan.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			scan.CompletedAt = &completedAt.Time
		}
		scans = append(scans, scan)
	}
	return scans, rows.Err()
}

// GetDiscoveryRule retrieves a discovery rule by network ID
func (s *SQLiteStorage) GetDiscoveryRule(networkID string) (*model.DiscoveryRule, error) {
	var rule model.DiscoveryRule
	var enabled int
	err := s.db.QueryRowContext(context.Background(), `
		SELECT id, network_id, enabled, scan_type, interval_hours, exclude_ips, created_at, updated_at
		FROM discovery_rules WHERE network_id = ?
	`, networkID).Scan(&rule.ID, &rule.NetworkID, &enabled, &rule.ScanType, &rule.IntervalHours,
		&rule.ExcludeIPs, &rule.CreatedAt, &rule.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrRuleNotFound
	}
	if err != nil {
		return nil, err
	}
	rule.Enabled = enabled == 1
	return &rule, nil
}

// SaveDiscoveryRule creates or updates a discovery rule (upsert)
func (s *SQLiteStorage) SaveDiscoveryRule(rule *model.DiscoveryRule) error {
	ctx := context.Background()
	if rule.ID == "" {
		rule.ID = newUUID()
	}
	now := time.Now()
	rule.UpdatedAt = now

	enabled := 0
	if rule.Enabled {
		enabled = 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO discovery_rules (id, network_id, enabled, scan_type, interval_hours, exclude_ips, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(network_id) DO UPDATE SET
			enabled = excluded.enabled, scan_type = excluded.scan_type,
			interval_hours = excluded.interval_hours, exclude_ips = excluded.exclude_ips,
			updated_at = excluded.updated_at
	`, rule.ID, rule.NetworkID, enabled, rule.ScanType, rule.IntervalHours, rule.ExcludeIPs, now, now)
	return err
}

// ListDiscoveryRules returns all discovery rules
func (s *SQLiteStorage) ListDiscoveryRules() ([]model.DiscoveryRule, error) {
	rows, err := s.db.QueryContext(context.Background(), `
		SELECT id, network_id, enabled, scan_type, interval_hours, exclude_ips, created_at, updated_at
		FROM discovery_rules ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.DiscoveryRule
	for rows.Next() {
		var rule model.DiscoveryRule
		var enabled int
		if err := rows.Scan(&rule.ID, &rule.NetworkID, &enabled, &rule.ScanType, &rule.IntervalHours,
			&rule.ExcludeIPs, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rule.Enabled = enabled == 1
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// CleanupOldDiscoveries removes discovered devices older than specified days
func (s *SQLiteStorage) CleanupOldDiscoveries(olderThanDays int) error {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)
	_, err := s.db.ExecContext(context.Background(), `
		DELETE FROM discovered_devices WHERE last_seen < ? AND promoted_to_device_id IS NULL
	`, cutoff)
	return err
}
