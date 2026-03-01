package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// ========================================
// DNS Provider Operations
// ========================================

// CreateDNSProvider creates a new DNS provider configuration
func (s *SQLiteStorage) CreateDNSProvider(ctx context.Context, provider *model.DNSProviderConfig) error {
	if provider == nil {
		return fmt.Errorf("provider is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.createDNSProviderInTx(ctx, tx, provider); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "dns_provider", provider.ID, provider)
	return nil
}

// createDNSProviderInTx creates a DNS provider within an existing transaction
func (s *SQLiteStorage) createDNSProviderInTx(ctx context.Context, tx *sql.Tx, provider *model.DNSProviderConfig) error {
	// Generate ID if not provided
	if provider.ID == "" {
		provider.ID = newUUID()
	}

	now := time.Now().UTC()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	_, err := tx.ExecContext(ctx, `
		INSERT INTO dns_provider_configs (id, name, type, endpoint, token, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, provider.ID, provider.Name, provider.Type, provider.Endpoint, provider.Token,
		provider.Description, provider.CreatedAt, provider.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create DNS provider: %w", err)
	}

	return nil
}

// GetDNSProvider retrieves a DNS provider by ID
func (s *SQLiteStorage) GetDNSProvider(id string) (*model.DNSProviderConfig, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	provider := &model.DNSProviderConfig{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, type, endpoint, token, description, created_at, updated_at
		FROM dns_provider_configs WHERE id = ?
	`, id).Scan(
		&provider.ID, &provider.Name, &provider.Type, &provider.Endpoint,
		&provider.Token, &provider.Description, &provider.CreatedAt, &provider.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDNSProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS provider: %w", err)
	}

	return provider, nil
}

// GetDNSProviderByName retrieves a DNS provider by name
func (s *SQLiteStorage) GetDNSProviderByName(name string) (*model.DNSProviderConfig, error) {
	if name == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	provider := &model.DNSProviderConfig{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, type, endpoint, token, description, created_at, updated_at
		FROM dns_provider_configs WHERE name = ?
	`, name).Scan(
		&provider.ID, &provider.Name, &provider.Type, &provider.Endpoint,
		&provider.Token, &provider.Description, &provider.CreatedAt, &provider.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDNSProviderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS provider by name: %w", err)
	}

	return provider, nil
}

// ListDNSProviders retrieves all DNS providers matching the filter criteria
func (s *SQLiteStorage) ListDNSProviders(filter *model.DNSProviderFilter) ([]model.DNSProviderConfig, error) {
	ctx := context.Background()

	query := `SELECT id, name, type, endpoint, token, description, created_at, updated_at FROM dns_provider_configs`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.Type != "" {
			conditions = append(conditions, "type = ?")
			args = append(args, filter.Type)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS providers: %w", err)
	}
	defer rows.Close()

	var providers []model.DNSProviderConfig
	for rows.Next() {
		var provider model.DNSProviderConfig
		if err := rows.Scan(
			&provider.ID, &provider.Name, &provider.Type, &provider.Endpoint,
			&provider.Token, &provider.Description, &provider.CreatedAt, &provider.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan DNS provider: %w", err)
		}
		providers = append(providers, provider)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if providers == nil {
		providers = []model.DNSProviderConfig{}
	}

	return providers, nil
}

// UpdateDNSProvider updates an existing DNS provider
func (s *SQLiteStorage) UpdateDNSProvider(ctx context.Context, provider *model.DNSProviderConfig) error {
	if provider == nil {
		return fmt.Errorf("provider is nil")
	}
	if provider.ID == "" {
		return ErrInvalidID
	}

	// Check if provider exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dns_provider_configs WHERE id = ?)`, provider.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check DNS provider existence: %w", err)
	}
	if !exists {
		return ErrDNSProviderNotFound
	}

	provider.UpdatedAt = time.Now().UTC()

	_, err = s.db.ExecContext(ctx, `
		UPDATE dns_provider_configs SET name = ?, type = ?, endpoint = ?, token = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, provider.Name, provider.Type, provider.Endpoint, provider.Token,
		provider.Description, provider.UpdatedAt, provider.ID)

	if err != nil {
		return fmt.Errorf("failed to update DNS provider: %w", err)
	}

	s.auditLog(ctx, "update", "dns_provider", provider.ID, provider)
	return nil
}

// DeleteDNSProvider removes a DNS provider by ID
func (s *SQLiteStorage) DeleteDNSProvider(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	// Check if provider exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dns_provider_configs WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check DNS provider existence: %w", err)
	}
	if !exists {
		return ErrDNSProviderNotFound
	}

	// Check if provider is in use by any zones
	var zoneCount int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dns_zones WHERE provider_id = ?`, id).Scan(&zoneCount)
	if err != nil {
		return fmt.Errorf("failed to check DNS provider usage: %w", err)
	}
	if zoneCount > 0 {
		return fmt.Errorf("cannot delete DNS provider: %d zone(s) still reference it", zoneCount)
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM dns_provider_configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete DNS provider: %w", err)
	}

	s.auditLog(ctx, "delete", "dns_provider", id, nil)
	return nil
}

// ========================================
// DNS Zone Operations
// ========================================

// CreateDNSZone creates a new DNS zone
func (s *SQLiteStorage) CreateDNSZone(ctx context.Context, zone *model.DNSZone) error {
	if zone == nil {
		return fmt.Errorf("zone is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.createDNSZoneInTx(ctx, tx, zone); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "dns_zone", zone.ID, zone)
	return nil
}

// createDNSZoneInTx creates a DNS zone within an existing transaction
func (s *SQLiteStorage) createDNSZoneInTx(ctx context.Context, tx *sql.Tx, zone *model.DNSZone) error {
	// Generate ID if not provided
	if zone.ID == "" {
		zone.ID = newUUID()
	}

	now := time.Now().UTC()
	zone.CreatedAt = now
	zone.UpdatedAt = now

	// Set default sync status if not provided
	if zone.LastSyncStatus == "" {
		zone.LastSyncStatus = model.SyncStatusSuccess
	}

	var networkIDParam, ptrZoneParam, lastSyncErrorParam sql.NullString
	if zone.NetworkID != nil {
		networkIDParam = sql.NullString{String: *zone.NetworkID, Valid: true}
	}
	if zone.PTRZone != nil {
		ptrZoneParam = sql.NullString{String: *zone.PTRZone, Valid: true}
	}
	if zone.LastSyncError != nil {
		lastSyncErrorParam = sql.NullString{String: *zone.LastSyncError, Valid: true}
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO dns_zones (id, name, provider_id, network_id, auto_sync, create_ptr, ptr_zone,
		                      ttl, description, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, zone.ID, zone.Name, zone.ProviderID, networkIDParam,
		zone.AutoSync, zone.CreatePTR, ptrZoneParam,
		zone.TTL, zone.Description, nullTime(zone.LastSyncAt),
		zone.LastSyncStatus, lastSyncErrorParam, zone.CreatedAt, zone.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create DNS zone: %w", err)
	}

	return nil
}

// GetDNSZone retrieves a DNS zone by ID
func (s *SQLiteStorage) GetDNSZone(id string) (*model.DNSZone, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	zone := &model.DNSZone{}
	var networkID, ptrZone, lastSyncError sql.NullString
	var lastSyncAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, provider_id, network_id, auto_sync, create_ptr, ptr_zone,
		       ttl, description, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
		FROM dns_zones WHERE id = ?
	`, id).Scan(
		&zone.ID, &zone.Name, &zone.ProviderID, &networkID,
		&zone.AutoSync, &zone.CreatePTR, &ptrZone,
		&zone.TTL, &zone.Description, &lastSyncAt,
		&zone.LastSyncStatus, &lastSyncError, &zone.CreatedAt, &zone.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDNSZoneNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS zone: %w", err)
	}

	if networkID.Valid {
		zone.NetworkID = &networkID.String
	}
	if ptrZone.Valid {
		zone.PTRZone = &ptrZone.String
	}
	if lastSyncAt.Valid {
		zone.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncError.Valid {
		zone.LastSyncError = &lastSyncError.String
	}

	return zone, nil
}

// GetDNSZoneByName retrieves a DNS zone by name
func (s *SQLiteStorage) GetDNSZoneByName(name string) (*model.DNSZone, error) {
	if name == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	zone := &model.DNSZone{}
	var networkID, ptrZone, lastSyncError sql.NullString
	var lastSyncAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, provider_id, network_id, auto_sync, create_ptr, ptr_zone,
		       ttl, description, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
		FROM dns_zones WHERE name = ?
	`, name).Scan(
		&zone.ID, &zone.Name, &zone.ProviderID, &networkID,
		&zone.AutoSync, &zone.CreatePTR, &ptrZone,
		&zone.TTL, &zone.Description, &lastSyncAt,
		&zone.LastSyncStatus, &lastSyncError, &zone.CreatedAt, &zone.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDNSZoneNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS zone by name: %w", err)
	}

	if networkID.Valid {
		zone.NetworkID = &networkID.String
	}
	if ptrZone.Valid {
		zone.PTRZone = &ptrZone.String
	}
	if lastSyncAt.Valid {
		zone.LastSyncAt = &lastSyncAt.Time
	}
	if lastSyncError.Valid {
		zone.LastSyncError = &lastSyncError.String
	}

	return zone, nil
}

// ListDNSZones retrieves all DNS zones matching the filter criteria
func (s *SQLiteStorage) ListDNSZones(filter *model.DNSZoneFilter) ([]model.DNSZone, error) {
	ctx := context.Background()

	query := `SELECT id, name, provider_id, network_id, auto_sync, create_ptr, ptr_zone,
	        ttl, description, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
	        FROM dns_zones`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.ProviderID != "" {
			conditions = append(conditions, "provider_id = ?")
			args = append(args, filter.ProviderID)
		}
		if filter.NetworkID != nil {
			conditions = append(conditions, "network_id = ?")
			args = append(args, *filter.NetworkID)
		}
		if filter.AutoSync != nil {
			conditions = append(conditions, "auto_sync = ?")
			args = append(args, *filter.AutoSync)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS zones: %w", err)
	}
	defer rows.Close()

	var zones []model.DNSZone
	for rows.Next() {
		var zone model.DNSZone
		var networkID, ptrZone, lastSyncError sql.NullString
		var lastSyncAt sql.NullTime

		if err := rows.Scan(
			&zone.ID, &zone.Name, &zone.ProviderID, &networkID,
			&zone.AutoSync, &zone.CreatePTR, &ptrZone,
			&zone.TTL, &zone.Description, &lastSyncAt,
			&zone.LastSyncStatus, &lastSyncError, &zone.CreatedAt, &zone.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan DNS zone: %w", err)
		}

		if networkID.Valid {
			zone.NetworkID = &networkID.String
		}
		if ptrZone.Valid {
			zone.PTRZone = &ptrZone.String
		}
		if lastSyncAt.Valid {
			zone.LastSyncAt = &lastSyncAt.Time
		}
		if lastSyncError.Valid {
			zone.LastSyncError = &lastSyncError.String
		}

		zones = append(zones, zone)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if zones == nil {
		zones = []model.DNSZone{}
	}

	return zones, nil
}

// UpdateDNSZone updates an existing DNS zone
func (s *SQLiteStorage) UpdateDNSZone(ctx context.Context, zone *model.DNSZone) error {
	if zone == nil {
		return fmt.Errorf("zone is nil")
	}
	if zone.ID == "" {
		return ErrInvalidID
	}

	// Check if zone exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dns_zones WHERE id = ?)`, zone.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check DNS zone existence: %w", err)
	}
	if !exists {
		return ErrDNSZoneNotFound
	}

	zone.UpdatedAt = time.Now().UTC()

	var networkIDParam, ptrZoneParam, lastSyncErrorParam sql.NullString
	if zone.NetworkID != nil {
		networkIDParam = sql.NullString{String: *zone.NetworkID, Valid: true}
	}
	if zone.PTRZone != nil {
		ptrZoneParam = sql.NullString{String: *zone.PTRZone, Valid: true}
	}
	if zone.LastSyncError != nil {
		lastSyncErrorParam = sql.NullString{String: *zone.LastSyncError, Valid: true}
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE dns_zones SET name = ?, provider_id = ?, network_id = ?, auto_sync = ?, create_ptr = ?,
		                    ptr_zone = ?, ttl = ?, description = ?, last_sync_at = ?, last_sync_status = ?,
		                    last_sync_error = ?, updated_at = ?
		WHERE id = ?
	`, zone.Name, zone.ProviderID, networkIDParam,
		zone.AutoSync, zone.CreatePTR, ptrZoneParam,
		zone.TTL, zone.Description, nullTime(zone.LastSyncAt),
		zone.LastSyncStatus, lastSyncErrorParam, zone.UpdatedAt, zone.ID)

	if err != nil {
		return fmt.Errorf("failed to update DNS zone: %w", err)
	}

	s.auditLog(ctx, "update", "dns_zone", zone.ID, zone)
	return nil
}

// DeleteDNSZone removes a DNS zone by ID
func (s *SQLiteStorage) DeleteDNSZone(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.deleteDNSZoneInTx(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "delete", "dns_zone", id, nil)
	return nil
}

// deleteDNSZoneInTx deletes a DNS zone within an existing transaction
func (s *SQLiteStorage) deleteDNSZoneInTx(ctx context.Context, tx *sql.Tx, id string) error {
	// Check if zone exists
	var exists bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dns_zones WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check DNS zone existence: %w", err)
	}
	if !exists {
		return ErrDNSZoneNotFound
	}

	// Delete the zone (records will cascade delete via foreign key)
	_, err = tx.ExecContext(ctx, `DELETE FROM dns_zones WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete DNS zone: %w", err)
	}

	return nil
}

// GetDNSZonesByNetwork retrieves all DNS zones for a specific network
func (s *SQLiteStorage) GetDNSZonesByNetwork(networkID string) ([]model.DNSZone, error) {
	if networkID == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	query := `SELECT id, name, provider_id, network_id, auto_sync, create_ptr, ptr_zone,
	        ttl, description, last_sync_at, last_sync_status, last_sync_error, created_at, updated_at
	        FROM dns_zones WHERE network_id = ? ORDER BY name`

	rows, err := s.db.QueryContext(ctx, query, networkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS zones by network: %w", err)
	}
	defer rows.Close()

	var zones []model.DNSZone
	for rows.Next() {
		var zone model.DNSZone
		var netID, ptrZone, lastSyncError sql.NullString
		var lastSyncAt sql.NullTime

		if err := rows.Scan(
			&zone.ID, &zone.Name, &zone.ProviderID, &netID,
			&zone.AutoSync, &zone.CreatePTR, &ptrZone,
			&zone.TTL, &zone.Description, &lastSyncAt,
			&zone.LastSyncStatus, &lastSyncError, &zone.CreatedAt, &zone.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan DNS zone: %w", err)
		}

		if netID.Valid {
			zone.NetworkID = &netID.String
		}
		if ptrZone.Valid {
			zone.PTRZone = &ptrZone.String
		}
		if lastSyncAt.Valid {
			zone.LastSyncAt = &lastSyncAt.Time
		}
		if lastSyncError.Valid {
			zone.LastSyncError = &lastSyncError.String
		}

		zones = append(zones, zone)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if zones == nil {
		zones = []model.DNSZone{}
	}

	return zones, nil
}

// GetDNSZonesByProvider retrieves all DNS zones for a specific provider
func (s *SQLiteStorage) GetDNSZonesByProvider(providerID string) ([]model.DNSZone, error) {
	if providerID == "" {
		return nil, ErrInvalidID
	}

	return s.ListDNSZones(&model.DNSZoneFilter{ProviderID: providerID})
}

// ========================================
// DNS Record Operations
// ========================================

// CreateDNSRecord creates a new DNS record
func (s *SQLiteStorage) CreateDNSRecord(ctx context.Context, record *model.DNSRecord) error {
	if record == nil {
		return fmt.Errorf("record is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.createDNSRecordInTx(ctx, tx, record); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "dns_record", record.ID, record)
	return nil
}

// createDNSRecordInTx creates a DNS record within an existing transaction
func (s *SQLiteStorage) createDNSRecordInTx(ctx context.Context, tx *sql.Tx, record *model.DNSRecord) error {
	// Generate ID if not provided
	if record.ID == "" {
		record.ID = newUUID()
	}

	now := time.Now().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now

	// Set default sync status if not provided
	if record.SyncStatus == "" {
		record.SyncStatus = model.RecordSyncStatusPending
	}

	var deviceIDParam, errorMessageParam sql.NullString
	if record.DeviceID != nil {
		deviceIDParam = sql.NullString{String: *record.DeviceID, Valid: true}
	}
	if record.ErrorMessage != nil {
		errorMessageParam = sql.NullString{String: *record.ErrorMessage, Valid: true}
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO dns_records (id, zone_id, device_id, name, type, value, ttl, sync_status, last_sync_at, error_message, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.ZoneID, deviceIDParam,
		record.Name, record.Type, record.Value, record.TTL,
		record.SyncStatus, nullTime(record.LastSyncAt), errorMessageParam,
		record.CreatedAt, record.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create DNS record: %w", err)
	}

	return nil
}

// GetDNSRecord retrieves a DNS record by ID
func (s *SQLiteStorage) GetDNSRecord(id string) (*model.DNSRecord, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	record := &model.DNSRecord{}
	var deviceID, errorMessage sql.NullString
	var lastSyncAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, zone_id, device_id, name, type, value, ttl, sync_status, last_sync_at, error_message, created_at, updated_at
		FROM dns_records WHERE id = ?
	`, id).Scan(
		&record.ID, &record.ZoneID, &deviceID,
		&record.Name, &record.Type, &record.Value, &record.TTL,
		&record.SyncStatus, &lastSyncAt, &errorMessage,
		&record.CreatedAt, &record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDNSRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS record: %w", err)
	}

	if deviceID.Valid {
		record.DeviceID = &deviceID.String
	}
	if lastSyncAt.Valid {
		record.LastSyncAt = &lastSyncAt.Time
	}
	if errorMessage.Valid {
		record.ErrorMessage = &errorMessage.String
	}

	return record, nil
}

// GetDNSRecordByName retrieves a DNS record by zone, name, and type
func (s *SQLiteStorage) GetDNSRecordByName(zoneID, name string, recordType string) (*model.DNSRecord, error) {
	if zoneID == "" || name == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	record := &model.DNSRecord{}
	var deviceID, errorMessage sql.NullString
	var lastSyncAt sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, zone_id, device_id, name, type, value, ttl, sync_status, last_sync_at, error_message, created_at, updated_at
		FROM dns_records WHERE zone_id = ? AND name = ? AND type = ?
	`, zoneID, name, recordType).Scan(
		&record.ID, &record.ZoneID, &deviceID,
		&record.Name, &record.Type, &record.Value, &record.TTL,
		&record.SyncStatus, &lastSyncAt, &errorMessage,
		&record.CreatedAt, &record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrDNSRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS record by name: %w", err)
	}

	if deviceID.Valid {
		record.DeviceID = &deviceID.String
	}
	if lastSyncAt.Valid {
		record.LastSyncAt = &lastSyncAt.Time
	}
	if errorMessage.Valid {
		record.ErrorMessage = &errorMessage.String
	}

	return record, nil
}

// ListDNSRecords retrieves all DNS records matching the filter criteria
func (s *SQLiteStorage) ListDNSRecords(filter *model.DNSRecordFilter) ([]model.DNSRecord, error) {
	ctx := context.Background()

	query := `SELECT id, zone_id, device_id, name, type, value, ttl, sync_status, last_sync_at, error_message, created_at, updated_at
	        FROM dns_records`
	var args []any
	var conditions []string

	if filter != nil {
		if filter.ZoneID != "" {
			conditions = append(conditions, "zone_id = ?")
			args = append(args, filter.ZoneID)
		}
		if filter.DeviceID != nil {
			conditions = append(conditions, "device_id = ?")
			args = append(args, *filter.DeviceID)
		}
		if filter.Type != "" {
			conditions = append(conditions, "type = ?")
			args = append(args, filter.Type)
		}
		if filter.SyncStatus != nil {
			conditions = append(conditions, "sync_status = ?")
			args = append(args, *filter.SyncStatus)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY zone_id, name, type"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}
	defer rows.Close()

	var records []model.DNSRecord
	for rows.Next() {
		var record model.DNSRecord
		var deviceID, errorMessage sql.NullString
		var lastSyncAt sql.NullTime

		if err := rows.Scan(
			&record.ID, &record.ZoneID, &deviceID,
			&record.Name, &record.Type, &record.Value, &record.TTL,
			&record.SyncStatus, &lastSyncAt, &errorMessage,
			&record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan DNS record: %w", err)
		}

		if deviceID.Valid {
			record.DeviceID = &deviceID.String
		}
		if lastSyncAt.Valid {
			record.LastSyncAt = &lastSyncAt.Time
		}
		if errorMessage.Valid {
			record.ErrorMessage = &errorMessage.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if records == nil {
		records = []model.DNSRecord{}
	}

	return records, nil
}

// UpdateDNSRecord updates an existing DNS record
func (s *SQLiteStorage) UpdateDNSRecord(ctx context.Context, record *model.DNSRecord) error {
	if record == nil {
		return fmt.Errorf("record is nil")
	}
	if record.ID == "" {
		return ErrInvalidID
	}

	// Check if record exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dns_records WHERE id = ?)`, record.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check DNS record existence: %w", err)
	}
	if !exists {
		return ErrDNSRecordNotFound
	}

	record.UpdatedAt = time.Now().UTC()

	var deviceIDParam, errorMessageParam sql.NullString
	if record.DeviceID != nil {
		deviceIDParam = sql.NullString{String: *record.DeviceID, Valid: true}
	}
	if record.ErrorMessage != nil {
		errorMessageParam = sql.NullString{String: *record.ErrorMessage, Valid: true}
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE dns_records SET zone_id = ?, device_id = ?, name = ?, type = ?, value = ?,
		                    ttl = ?, sync_status = ?, last_sync_at = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`, record.ZoneID, deviceIDParam, record.Name, record.Type,
		record.Value, record.TTL, record.SyncStatus, nullTime(record.LastSyncAt),
		errorMessageParam, record.UpdatedAt, record.ID)

	if err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}

	s.auditLog(ctx, "update", "dns_record", record.ID, record)
	return nil
}

// DeleteDNSRecord removes a DNS record by ID
func (s *SQLiteStorage) DeleteDNSRecord(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	// Check if record exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dns_records WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check DNS record existence: %w", err)
	}
	if !exists {
		return ErrDNSRecordNotFound
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM dns_records WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete DNS record: %w", err)
	}

	s.auditLog(ctx, "delete", "dns_record", id, nil)
	return nil
}

// DeleteDNSRecordsByZone removes all DNS records for a specific zone
func (s *SQLiteStorage) DeleteDNSRecordsByZone(ctx context.Context, zoneID string) error {
	if zoneID == "" {
		return ErrInvalidID
	}

	// Check if zone exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dns_zones WHERE id = ?)`, zoneID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check DNS zone existence: %w", err)
	}
	if !exists {
		return ErrDNSZoneNotFound
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM dns_records WHERE zone_id = ?`, zoneID)
	if err != nil {
		return fmt.Errorf("failed to delete DNS records by zone: %w", err)
	}

	s.auditLog(ctx, "delete", "dns_records", zoneID, nil)
	return nil
}

// DeleteDNSRecordsByDevice removes all DNS records for a specific device
func (s *SQLiteStorage) DeleteDNSRecordsByDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return ErrInvalidID
	}

	// Check if device exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE id = ?)`, deviceID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check device existence: %w", err)
	}
	if !exists {
		return ErrDeviceNotFound
	}

	_, err = s.db.ExecContext(ctx, `DELETE FROM dns_records WHERE device_id = ?`, deviceID)
	if err != nil {
		return fmt.Errorf("failed to delete DNS records by device: %w", err)
	}

	s.auditLog(ctx, "delete", "dns_records", deviceID, nil)
	return nil
}

// GetDNSRecordsByDevice retrieves all DNS records for a specific device
func (s *SQLiteStorage) GetDNSRecordsByDevice(deviceID string) ([]model.DNSRecord, error) {
	if deviceID == "" {
		return nil, ErrInvalidID
	}

	// Check if device exists
	var exists bool
	ctx := context.Background()
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE id = ?)`, deviceID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check device existence: %w", err)
	}
	if !exists {
		return nil, ErrDeviceNotFound
	}

	query := `SELECT id, zone_id, device_id, name, type, value, ttl, sync_status, last_sync_at, error_message, created_at, updated_at
	        FROM dns_records WHERE device_id = ? ORDER BY zone_id, name, type`

	rows, err := s.db.QueryContext(ctx, query, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS records by device: %w", err)
	}
	defer rows.Close()

	var records []model.DNSRecord
	for rows.Next() {
		var record model.DNSRecord
		var devID, errorMessage sql.NullString
		var lastSyncAt sql.NullTime

		if err := rows.Scan(
			&record.ID, &record.ZoneID, &devID,
			&record.Name, &record.Type, &record.Value, &record.TTL,
			&record.SyncStatus, &lastSyncAt, &errorMessage,
			&record.CreatedAt, &record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan DNS record: %w", err)
		}

		if devID.Valid {
			record.DeviceID = &devID.String
		}
		if lastSyncAt.Valid {
			record.LastSyncAt = &lastSyncAt.Time
		}
		if errorMessage.Valid {
			record.ErrorMessage = &errorMessage.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if records == nil {
		records = []model.DNSRecord{}
	}

	return records, nil
}
