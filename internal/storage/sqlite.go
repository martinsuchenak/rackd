package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements ExtendedStorage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dataDir string) (*SQLiteStorage, error) {
	var dbPath string

	if dataDir == ":memory:" {
		dbPath = ":memory:"
	} else {
		// Ensure data directory exists
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
		dbPath = filepath.Join(dataDir, "rackd.db")
	}

	// Open database with SQLite pragma settings
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &SQLiteStorage{db: db}

	// Run migrations
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create default datacenter if none exists
	if err := s.ensureDefaultDatacenter(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure default datacenter: %w", err)
	}

	return s, nil
}

// NewSQLiteStorageWithPath creates a new SQLite storage instance with a specific database file path
func NewSQLiteStorageWithPath(dbPath string) (*SQLiteStorage, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database with SQLite pragma settings
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &SQLiteStorage{db: db}

	// Run migrations
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create default datacenter if none exists
	if err := s.ensureDefaultDatacenter(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure default datacenter: %w", err)
	}

	return s, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// DB returns the underlying database connection for testing
func (s *SQLiteStorage) DB() *sql.DB {
	return s.db
}

// newUUID generates a new UUIDv7
func newUUID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fall back to v4 if v7 generation fails
		return uuid.New().String()
	}
	return id.String()
}

// Device operations

// GetDevice retrieves a device by ID with its addresses, tags, and domains
func (s *SQLiteStorage) GetDevice(id string) (*model.Device, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	// Get the device
	device := &model.Device{}
	var datacenterID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, hostname, description, make_model, os, datacenter_id, username, location, created_at, updated_at
		FROM devices WHERE id = ?
	`, id).Scan(
		&device.ID, &device.Name, &device.Hostname, &device.Description, &device.MakeModel,
		&device.OS, &datacenterID, &device.Username, &device.Location,
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

	return device, nil
}

// getDeviceAddresses retrieves all addresses for a device
func (s *SQLiteStorage) getDeviceAddresses(ctx context.Context, deviceID string) ([]model.Address, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT ip, port, type, label, network_id, switch_port, pool_id
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
		if err := rows.Scan(&addr.IP, &port, &addr.Type, &addr.Label, &networkID, &switchPort, &poolID); err != nil {
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

	// Insert device
	_, err := tx.ExecContext(ctx, `
		INSERT INTO devices (id, name, hostname, description, make_model, os, datacenter_id, username, location, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.Name, device.Hostname, device.Description, device.MakeModel,
		device.OS, nullString(device.DatacenterID), device.Username, device.Location,
		device.CreatedAt, device.UpdatedAt)
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

	return nil
}

// insertDeviceAddresses inserts addresses for a device within a transaction
func (s *SQLiteStorage) insertDeviceAddresses(ctx context.Context, tx *sql.Tx, deviceID string, addresses []model.Address) error {
	for _, addr := range addresses {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO addresses (id, device_id, ip, port, type, label, network_id, switch_port, pool_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, newUUID(), deviceID, addr.IP, nullIntPtr(addr.Port), addr.Type, addr.Label,
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

	// Check if device exists
	var exists bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE id = ?)`, device.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check device existence: %w", err)
	}
	if !exists {
		return ErrDeviceNotFound
	}

	device.UpdatedAt = time.Now().UTC()

	// Update device
	_, err = tx.ExecContext(ctx, `
		UPDATE devices SET
			name = ?, hostname = ?, description = ?, make_model = ?, os = ?, datacenter_id = ?,
			username = ?, location = ?, updated_at = ?
		WHERE id = ?
	`, device.Name, device.Hostname, device.Description, device.MakeModel, device.OS,
		nullString(device.DatacenterID), device.Username, device.Location,
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
func (s *SQLiteStorage) ListDevices(filter *model.DeviceFilter) ([]model.Device, error) {
	ctx := context.Background()

	query := `SELECT id, name, hostname, description, make_model, os, datacenter_id, username, location, created_at, updated_at FROM devices`
	var args []interface{}
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

		if len(filter.Tags) > 0 {
			// Match devices that have ALL specified tags
			for _, tag := range filter.Tags {
				conditions = append(conditions, "id IN (SELECT device_id FROM tags WHERE tag = ?)")
				args = append(args, tag)
			}
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var device model.Device
		var datacenterID sql.NullString
		if err := rows.Scan(
			&device.ID, &device.Name, &device.Hostname, &device.Description, &device.MakeModel,
			&device.OS, &datacenterID, &device.Username, &device.Location,
			&device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		if datacenterID.Valid {
			device.DatacenterID = datacenterID.String
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
func (s *SQLiteStorage) SearchDevices(query string) ([]model.Device, error) {
	if query == "" {
		return s.ListDevices(nil)
	}

	ctx := context.Background()
	ftsQuery := escapeFTSQuery(query)
	likePattern := "%" + query + "%"

	// Use UNION to combine FTS results with tag/domain/address matches
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os, 
		       d.datacenter_id, d.username, d.location, d.created_at, d.updated_at
		FROM devices d
		INNER JOIN devices_fts fts ON d.id = fts.id
		WHERE devices_fts MATCH ?
		UNION
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os, 
		       d.datacenter_id, d.username, d.location, d.created_at, d.updated_at
		FROM devices d
		INNER JOIN tags t ON d.id = t.device_id
		WHERE t.tag LIKE ?
		UNION
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os, 
		       d.datacenter_id, d.username, d.location, d.created_at, d.updated_at
		FROM devices d
		INNER JOIN domains dm ON d.id = dm.device_id
		WHERE dm.domain LIKE ?
		UNION
		SELECT DISTINCT d.id, d.name, d.hostname, d.description, d.make_model, d.os, 
		       d.datacenter_id, d.username, d.location, d.created_at, d.updated_at
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
		var datacenterID sql.NullString
		if err := rows.Scan(
			&device.ID, &device.Name, &device.Hostname, &device.Description, &device.MakeModel,
			&device.OS, &datacenterID, &device.Username, &device.Location,
			&device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		if datacenterID.Valid {
			device.DatacenterID = datacenterID.String
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

// escapeFTSQuery escapes special FTS5 characters and adds prefix matching
func escapeFTSQuery(query string) string {
	// Escape double quotes by doubling them
	escaped := strings.ReplaceAll(query, `"`, `""`)
	// Wrap in quotes and add * for prefix matching
	return `"` + escaped + `"*`
}

// nullString returns a sql.NullString for empty strings
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullInt returns a sql.NullInt64 for zero values
func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

// nullIntPtr returns a sql.NullInt64 for nil pointer values
func nullIntPtr(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

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
func (s *SQLiteStorage) ListDatacenters(filter *model.DatacenterFilter) ([]model.Datacenter, error) {
	ctx := context.Background()

	query := `SELECT id, name, location, description, created_at, updated_at FROM datacenters`
	var args []interface{}

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
func (s *SQLiteStorage) SearchDatacenters(query string) ([]model.Datacenter, error) {
	if query == "" {
		return s.ListDatacenters(nil)
	}

	ctx := context.Background()
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
func (s *SQLiteStorage) GetDatacenter(id string) (*model.Datacenter, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

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
func (s *SQLiteStorage) GetDatacenterDevices(datacenterID string) ([]model.Device, error) {
	if datacenterID == "" {
		return nil, ErrInvalidID
	}

	// First check if the datacenter exists
	ctx := context.Background()
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM datacenters WHERE id = ?)`, datacenterID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check datacenter existence: %w", err)
	}
	if !exists {
		return nil, ErrDatacenterNotFound
	}

	// Use ListDevices with a filter
	return s.ListDevices(&model.DeviceFilter{DatacenterID: datacenterID})
}

// Network operations

// ListNetworks retrieves all networks matching the filter criteria
func (s *SQLiteStorage) ListNetworks(filter *model.NetworkFilter) ([]model.Network, error) {
	ctx := context.Background()

	query := `SELECT id, name, subnet, vlan_id, datacenter_id, description, created_at, updated_at FROM networks`
	var args []interface{}
	var conditions []string

	if filter != nil {
		if filter.Name != "" {
			conditions = append(conditions, "name LIKE ?")
			args = append(args, "%"+filter.Name+"%")
		}
		if filter.DatacenterID != "" {
			conditions = append(conditions, "datacenter_id = ?")
			args = append(args, filter.DatacenterID)
		}
		if filter.VLANID > 0 {
			conditions = append(conditions, "vlan_id = ?")
			args = append(args, filter.VLANID)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}
	defer rows.Close()

	var networks []model.Network
	for rows.Next() {
		var network model.Network
		var vlanID sql.NullInt64
		var datacenterID sql.NullString
		if err := rows.Scan(
			&network.ID, &network.Name, &network.Subnet, &vlanID,
			&datacenterID, &network.Description, &network.CreatedAt, &network.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan network: %w", err)
		}
		if vlanID.Valid {
			network.VLANID = int(vlanID.Int64)
		}
		if datacenterID.Valid {
			network.DatacenterID = datacenterID.String
		}
		networks = append(networks, network)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if networks == nil {
		networks = []model.Network{}
	}

	return networks, nil
}

// SearchNetworks performs a full-text search across network fields using FTS5
func (s *SQLiteStorage) SearchNetworks(query string) ([]model.Network, error) {
	if query == "" {
		return s.ListNetworks(nil)
	}

	ctx := context.Background()
	ftsQuery := escapeFTSQuery(query)

	rows, err := s.db.QueryContext(ctx, `
		SELECT n.id, n.name, n.subnet, n.vlan_id, n.datacenter_id, n.description, 
		       n.created_at, n.updated_at
		FROM networks n
		INNER JOIN networks_fts fts ON n.id = fts.id
		WHERE networks_fts MATCH ?
		ORDER BY n.name
	`, ftsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to search networks: %w", err)
	}
	defer rows.Close()

	var networks []model.Network
	for rows.Next() {
		var network model.Network
		var vlanID sql.NullInt64
		var datacenterID sql.NullString
		if err := rows.Scan(
			&network.ID, &network.Name, &network.Subnet, &vlanID,
			&datacenterID, &network.Description, &network.CreatedAt, &network.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan network: %w", err)
		}
		if vlanID.Valid {
			network.VLANID = int(vlanID.Int64)
		}
		if datacenterID.Valid {
			network.DatacenterID = datacenterID.String
		}
		networks = append(networks, network)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if networks == nil {
		networks = []model.Network{}
	}

	return networks, nil
}

// GetNetwork retrieves a network by ID
func (s *SQLiteStorage) GetNetwork(id string) (*model.Network, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	network := &model.Network{}
	var vlanID sql.NullInt64
	var datacenterID sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, subnet, vlan_id, datacenter_id, description, created_at, updated_at
		FROM networks WHERE id = ?
	`, id).Scan(
		&network.ID, &network.Name, &network.Subnet, &vlanID,
		&datacenterID, &network.Description, &network.CreatedAt, &network.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNetworkNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}

	if vlanID.Valid {
		network.VLANID = int(vlanID.Int64)
	}
	if datacenterID.Valid {
		network.DatacenterID = datacenterID.String
	}

	return network, nil
}

// CreateNetwork creates a new network
func (s *SQLiteStorage) CreateNetwork(ctx context.Context, network *model.Network) error {
	if network == nil {
		return fmt.Errorf("network is nil")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.createNetworkInTx(ctx, tx, network); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "network", network.ID, network)
	return nil
}

// createNetworkInTx creates a network within an existing transaction
func (s *SQLiteStorage) createNetworkInTx(ctx context.Context, tx *sql.Tx, network *model.Network) error {

	// Generate ID if not provided
	if network.ID == "" {
		network.ID = newUUID()
	}

	now := time.Now().UTC()
	network.CreatedAt = now
	network.UpdatedAt = now

	_, err := tx.ExecContext(ctx, `
		INSERT INTO networks (id, name, subnet, vlan_id, datacenter_id, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, network.ID, network.Name, network.Subnet, nullInt(network.VLANID),
		nullString(network.DatacenterID), network.Description, network.CreatedAt, network.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	return nil
}

// UpdateNetwork updates an existing network
func (s *SQLiteStorage) UpdateNetwork(ctx context.Context, network *model.Network) error {
	if network == nil {
		return fmt.Errorf("network is nil")
	}
	if network.ID == "" {
		return ErrInvalidID
	}

	// Check if network exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, network.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check network existence: %w", err)
	}
	if !exists {
		return ErrNetworkNotFound
	}

	network.UpdatedAt = time.Now().UTC()

	_, err = s.db.ExecContext(ctx, `
		UPDATE networks SET name = ?, subnet = ?, vlan_id = ?, datacenter_id = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, network.Name, network.Subnet, nullInt(network.VLANID),
		nullString(network.DatacenterID), network.Description, network.UpdatedAt, network.ID)

	if err != nil {
		return fmt.Errorf("failed to update network: %w", err)
	}

	s.auditLog(ctx, "update", "network", network.ID, network)
	return nil
}

// DeleteNetwork removes a network by ID
func (s *SQLiteStorage) DeleteNetwork(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.deleteNetworkInTx(ctx, tx, id); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "delete", "network", id, nil)
	return nil
}

// deleteNetworkInTx deletes a network within an existing transaction
func (s *SQLiteStorage) deleteNetworkInTx(ctx context.Context, tx *sql.Tx, id string) error {

	// Check if network exists
	var exists bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check network existence: %w", err)
	}
	if !exists {
		return ErrNetworkNotFound
	}

	// Unlink addresses from this network (set network_id to NULL)
	_, err = tx.ExecContext(ctx, `UPDATE addresses SET network_id = NULL WHERE network_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to unlink addresses: %w", err)
	}

	// Delete network pools (cascades via foreign key, but explicit for clarity)
	_, err = tx.ExecContext(ctx, `DELETE FROM network_pools WHERE network_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete network pools: %w", err)
	}

	// Delete the network
	_, err = tx.ExecContext(ctx, `DELETE FROM networks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	return nil
}

// GetNetworkDevices retrieves all devices that have addresses in a network
func (s *SQLiteStorage) GetNetworkDevices(networkID string) ([]model.Device, error) {
	if networkID == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	// Check if network exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, networkID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check network existence: %w", err)
	}
	if !exists {
		return nil, ErrNetworkNotFound
	}

	// Use ListDevices with a filter
	return s.ListDevices(&model.DeviceFilter{NetworkID: networkID})
}

// GetNetworkUtilization calculates IP usage for a network based on its CIDR and assigned addresses
func (s *SQLiteStorage) GetNetworkUtilization(networkID string) (*model.NetworkUtilization, error) {
	if networkID == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	// Get the network to retrieve its subnet
	network, err := s.GetNetwork(networkID)
	if err != nil {
		return nil, err
	}

	// Calculate total IPs from CIDR
	totalIPs, err := calculateCIDRSize(network.Subnet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse subnet CIDR: %w", err)
	}

	// Count used IPs (addresses assigned to this network)
	var usedIPs int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT ip) FROM addresses WHERE network_id = ?
	`, networkID).Scan(&usedIPs)
	if err != nil {
		return nil, fmt.Errorf("failed to count used IPs: %w", err)
	}

	availableIPs := totalIPs - usedIPs
	if availableIPs < 0 {
		availableIPs = 0
	}

	var utilization float64
	if totalIPs > 0 {
		utilization = float64(usedIPs) / float64(totalIPs) * 100
	}

	return &model.NetworkUtilization{
		NetworkID:    networkID,
		TotalIPs:     totalIPs,
		UsedIPs:      usedIPs,
		AvailableIPs: availableIPs,
		Utilization:  utilization,
	}, nil
}

// calculateCIDRSize calculates the number of usable host IPs in a CIDR block
func calculateCIDRSize(cidr string) (int, error) {
	// Parse CIDR (e.g., "192.168.1.0/24")
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return 0, err
	}

	// Get the prefix length
	ones, bits := ipNet.Mask.Size()
	if bits == 0 {
		return 0, fmt.Errorf("invalid mask")
	}

	// Calculate total addresses: 2^(bits - ones)
	hostBits := bits - ones
	if hostBits <= 0 {
		return 1, nil // /32 or /128 has 1 address
	}

	// For large subnets, cap at a reasonable number
	if hostBits > 20 {
		// For subnets larger than /12, return a capped value
		return 1 << 20, nil // ~1 million
	}

	total := 1 << hostBits

	// Subtract network and broadcast addresses for IPv4 subnets with more than 2 hosts
	if bits == 32 && hostBits >= 2 {
		total -= 2 // Subtract network and broadcast addresses
	}

	if total < 0 {
		total = 0
	}

	return total, nil
}

// Pool operations (P2-007)

// CreateNetworkPool creates a new network pool
func (s *SQLiteStorage) CreateNetworkPool(ctx context.Context, pool *model.NetworkPool) error {
	if pool == nil {
		return fmt.Errorf("pool is nil")
	}

	// Validate network exists
	var networkExists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, pool.NetworkID).Scan(&networkExists)
	if err != nil {
		return fmt.Errorf("failed to check network existence: %w", err)
	}
	if !networkExists {
		return ErrNetworkNotFound
	}

	// Generate ID if not provided
	if pool.ID == "" {
		pool.ID = newUUID()
	}

	now := time.Now().UTC()
	pool.CreatedAt = now
	pool.UpdatedAt = now

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert pool
	_, err = tx.ExecContext(ctx, `
		INSERT INTO network_pools (id, network_id, name, start_ip, end_ip, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, pool.ID, pool.NetworkID, pool.Name, pool.StartIP, pool.EndIP, pool.Description, pool.CreatedAt, pool.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create network pool: %w", err)
	}

	// Insert tags
	if err := s.insertPoolTags(ctx, tx, pool.ID, pool.Tags); err != nil {
		return fmt.Errorf("failed to insert pool tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "create", "pool", pool.ID, pool)
	return nil
}

// insertPoolTags inserts tags for a pool within a transaction
func (s *SQLiteStorage) insertPoolTags(ctx context.Context, tx *sql.Tx, poolID string, tags []string) error {
	for _, tag := range tags {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO pool_tags (pool_id, tag) VALUES (?, ?)
		`, poolID, tag)
		if err != nil {
			return err
		}
	}
	return nil
}

// getPoolTags retrieves all tags for a pool
func (s *SQLiteStorage) getPoolTags(ctx context.Context, poolID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag FROM pool_tags WHERE pool_id = ?`, poolID)
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

// GetNetworkPool retrieves a network pool by ID
func (s *SQLiteStorage) GetNetworkPool(id string) (*model.NetworkPool, error) {
	if id == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	pool := &model.NetworkPool{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, network_id, name, start_ip, end_ip, description, created_at, updated_at
		FROM network_pools WHERE id = ?
	`, id).Scan(
		&pool.ID, &pool.NetworkID, &pool.Name, &pool.StartIP, &pool.EndIP,
		&pool.Description, &pool.CreatedAt, &pool.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrPoolNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get network pool: %w", err)
	}

	// Get tags
	tags, err := s.getPoolTags(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool tags: %w", err)
	}
	pool.Tags = tags

	return pool, nil
}

// UpdateNetworkPool updates an existing network pool
func (s *SQLiteStorage) UpdateNetworkPool(ctx context.Context, pool *model.NetworkPool) error {
	if pool == nil {
		return fmt.Errorf("pool is nil")
	}
	if pool.ID == "" {
		return ErrInvalidID
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if pool exists
	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM network_pools WHERE id = ?)`, pool.ID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check pool existence: %w", err)
	}
	if !exists {
		return ErrPoolNotFound
	}

	pool.UpdatedAt = time.Now().UTC()

	// Update pool
	_, err = tx.ExecContext(ctx, `
		UPDATE network_pools SET name = ?, start_ip = ?, end_ip = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, pool.Name, pool.StartIP, pool.EndIP, pool.Description, pool.UpdatedAt, pool.ID)
	if err != nil {
		return fmt.Errorf("failed to update network pool: %w", err)
	}

	// Delete existing tags and reinsert
	if _, err := tx.ExecContext(ctx, `DELETE FROM pool_tags WHERE pool_id = ?`, pool.ID); err != nil {
		return fmt.Errorf("failed to delete pool tags: %w", err)
	}
	if err := s.insertPoolTags(ctx, tx, pool.ID, pool.Tags); err != nil {
		return fmt.Errorf("failed to insert pool tags: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "update", "pool", pool.ID, pool)
	return nil
}

// DeleteNetworkPool removes a network pool by ID
func (s *SQLiteStorage) DeleteNetworkPool(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidID
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if pool exists
	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM network_pools WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check pool existence: %w", err)
	}
	if !exists {
		return ErrPoolNotFound
	}

	// Unlink addresses from this pool (set pool_id to NULL)
	_, err = tx.ExecContext(ctx, `UPDATE addresses SET pool_id = NULL WHERE pool_id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to unlink addresses: %w", err)
	}

	// Delete pool (tags cascade via foreign key)
	_, err = tx.ExecContext(ctx, `DELETE FROM network_pools WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete network pool: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.auditLog(ctx, "delete", "pool", id, nil)
	return nil
}

// ListNetworkPools retrieves pools matching the filter criteria
func (s *SQLiteStorage) ListNetworkPools(filter *model.NetworkPoolFilter) ([]model.NetworkPool, error) {
	ctx := context.Background()

	query := `SELECT id, network_id, name, start_ip, end_ip, description, created_at, updated_at FROM network_pools`
	var args []interface{}
	var conditions []string

	if filter != nil {
		if filter.NetworkID != "" {
			conditions = append(conditions, "network_id = ?")
			args = append(args, filter.NetworkID)
		}

		if len(filter.Tags) > 0 {
			// Match pools that have ALL specified tags
			for _, tag := range filter.Tags {
				conditions = append(conditions, "id IN (SELECT pool_id FROM pool_tags WHERE tag = ?)")
				args = append(args, tag)
			}
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + conditions[0]
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list network pools: %w", err)
	}
	defer rows.Close()

	var pools []model.NetworkPool
	for rows.Next() {
		var pool model.NetworkPool
		if err := rows.Scan(
			&pool.ID, &pool.NetworkID, &pool.Name, &pool.StartIP, &pool.EndIP,
			&pool.Description, &pool.CreatedAt, &pool.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan network pool: %w", err)
		}
		pools = append(pools, pool)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load tags for each pool
	for i := range pools {
		tags, err := s.getPoolTags(ctx, pools[i].ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for pool %s: %w", pools[i].ID, err)
		}
		pools[i].Tags = tags
	}

	if pools == nil {
		pools = []model.NetworkPool{}
	}

	return pools, nil
}

// GetNextAvailableIP finds the first unused IP address in a pool's range
func (s *SQLiteStorage) GetNextAvailableIP(poolID string) (string, error) {
	if poolID == "" {
		return "", ErrInvalidID
	}

	ctx := context.Background()

	// Get the pool
	pool, err := s.GetNetworkPool(poolID)
	if err != nil {
		return "", err
	}

	// Parse start and end IPs
	startIP := net.ParseIP(pool.StartIP)
	endIP := net.ParseIP(pool.EndIP)

	if startIP == nil || endIP == nil {
		return "", fmt.Errorf("invalid IP range: %s - %s", pool.StartIP, pool.EndIP)
	}

	// Convert to IPv4 if applicable
	startIP = startIP.To4()
	endIP = endIP.To4()

	if startIP == nil || endIP == nil {
		// Handle IPv6 or other formats
		return "", fmt.Errorf("only IPv4 addresses are currently supported")
	}

	// Get all used IPs in this pool
	usedIPs := make(map[string]bool)
	rows, err := s.db.QueryContext(ctx, `SELECT ip FROM addresses WHERE pool_id = ?`, poolID)
	if err != nil {
		return "", fmt.Errorf("failed to query used IPs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return "", fmt.Errorf("failed to scan IP: %w", err)
		}
		usedIPs[ip] = true
	}
	if err := rows.Err(); err != nil {
		return "", err
	}

	// Iterate through the range to find the first available IP
	current := make(net.IP, len(startIP))
	copy(current, startIP)

	for {
		ipStr := current.String()
		if !usedIPs[ipStr] {
			return ipStr, nil
		}

		// Increment IP
		if !incrementIP(current, endIP) {
			break
		}
	}

	return "", ErrIPNotAvailable
}

// incrementIP increments an IP address by 1, returns false if it exceeds endIP
func incrementIP(ip net.IP, endIP net.IP) bool {
	// Increment from the last byte
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}

	// Check if we've exceeded the end IP
	for i := 0; i < len(ip); i++ {
		if ip[i] < endIP[i] {
			return true
		}
		if ip[i] > endIP[i] {
			return false
		}
	}

	// IPs are equal, this is the last valid IP
	return true
}

// ValidateIPInPool checks if an IP address is within a pool's range
func (s *SQLiteStorage) ValidateIPInPool(poolID, ip string) (bool, error) {
	if poolID == "" {
		return false, ErrInvalidID
	}

	// Get the pool
	pool, err := s.GetNetworkPool(poolID)
	if err != nil {
		return false, err
	}

	// Parse all IPs
	checkIP := net.ParseIP(ip)
	startIP := net.ParseIP(pool.StartIP)
	endIP := net.ParseIP(pool.EndIP)

	if checkIP == nil || startIP == nil || endIP == nil {
		return false, fmt.Errorf("invalid IP address")
	}

	// Convert to IPv4 if applicable
	checkIP = checkIP.To4()
	startIP = startIP.To4()
	endIP = endIP.To4()

	if checkIP == nil || startIP == nil || endIP == nil {
		return false, fmt.Errorf("only IPv4 addresses are currently supported")
	}

	// Check if checkIP is within range [startIP, endIP]
	return ipInRange(checkIP, startIP, endIP), nil
}

// ipInRange checks if ip is within the range [start, end] (inclusive)
func ipInRange(ip, start, end net.IP) bool {
	// Compare with start: ip >= start
	for i := 0; i < len(ip); i++ {
		if ip[i] < start[i] {
			return false
		}
		if ip[i] > start[i] {
			break
		}
	}

	// Compare with end: ip <= end
	for i := 0; i < len(ip); i++ {
		if ip[i] > end[i] {
			return false
		}
		if ip[i] < end[i] {
			break
		}
	}

	return true
}

// GetPoolHeatmap returns the status of all IPs in a pool's range
func (s *SQLiteStorage) GetPoolHeatmap(poolID string) ([]IPStatus, error) {
	if poolID == "" {
		return nil, ErrInvalidID
	}

	ctx := context.Background()

	// Get the pool
	pool, err := s.GetNetworkPool(poolID)
	if err != nil {
		return nil, err
	}

	// Parse start and end IPs
	startIP := net.ParseIP(pool.StartIP)
	endIP := net.ParseIP(pool.EndIP)

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("invalid IP range: %s - %s", pool.StartIP, pool.EndIP)
	}

	// Convert to IPv4
	startIP = startIP.To4()
	endIP = endIP.To4()

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("only IPv4 addresses are currently supported")
	}

	// Get all addresses in this pool with their device IDs
	addressMap := make(map[string]string) // ip -> device_id
	rows, err := s.db.QueryContext(ctx, `SELECT ip, device_id FROM addresses WHERE pool_id = ?`, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to query addresses: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip, deviceID string
		if err := rows.Scan(&ip, &deviceID); err != nil {
			return nil, fmt.Errorf("failed to scan address: %w", err)
		}
		addressMap[ip] = deviceID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build heatmap
	var heatmap []IPStatus
	current := make(net.IP, len(startIP))
	copy(current, startIP)

	// Safety limit: don't enumerate more than 65536 IPs to prevent memory issues
	const maxIPs = 65536
	count := 0

	for count < maxIPs {
		ipStr := current.String()
		status := IPStatus{
			IP:     ipStr,
			Status: "available",
		}

		if deviceID, exists := addressMap[ipStr]; exists {
			status.Status = "used"
			status.DeviceID = deviceID
		}

		heatmap = append(heatmap, status)
		count++

		// Check if we've reached the end
		if current.Equal(endIP) {
			break
		}

		// Increment IP
		if !incrementIP(current, endIP) {
			break
		}
	}

	if heatmap == nil {
		heatmap = []IPStatus{}
	}

	return heatmap, nil
}

// Relationship operations - stub implementations for now (will be completed in P2-008)

func (s *SQLiteStorage) AddRelationship(ctx context.Context, parentID, childID, relationshipType, notes string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO device_relationships (parent_id, child_id, type, notes)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (parent_id, child_id, type) DO UPDATE SET notes = excluded.notes
	`, parentID, childID, relationshipType, notes)
	if err != nil {
		return err
	}
	s.auditLog(ctx, "add", "relationship", parentID+":"+childID, nil)
	return nil
}

func (s *SQLiteStorage) RemoveRelationship(ctx context.Context, parentID, childID, relationshipType string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM device_relationships
		WHERE parent_id = ? AND child_id = ? AND type = ?
	`, parentID, childID, relationshipType)
	if err != nil {
		return err
	}
	s.auditLog(ctx, "remove", "relationship", parentID+":"+childID, nil)
	return nil
}

func (s *SQLiteStorage) UpdateRelationshipNotes(ctx context.Context, parentID, childID, relationshipType, notes string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE device_relationships
		SET notes = ?
		WHERE parent_id = ? AND child_id = ? AND type = ?
	`, notes, parentID, childID, relationshipType)
	if err != nil {
		return err
	}
	s.auditLog(ctx, "update", "relationship", parentID+":"+childID, nil)
	return nil
}

func (s *SQLiteStorage) GetRelationships(deviceID string) ([]model.DeviceRelationship, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `
		SELECT parent_id, child_id, type, notes, created_at
		FROM device_relationships
		WHERE parent_id = ? OR child_id = ?
	`, deviceID, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []model.DeviceRelationship
	for rows.Next() {
		var r model.DeviceRelationship
		if err := rows.Scan(&r.ParentID, &r.ChildID, &r.Type, &r.Notes, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *SQLiteStorage) ListAllRelationships() ([]model.DeviceRelationship, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `
		SELECT parent_id, child_id, type, notes, created_at
		FROM device_relationships
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []model.DeviceRelationship
	for rows.Next() {
		var r model.DeviceRelationship
		if err := rows.Scan(&r.ParentID, &r.ChildID, &r.Type, &r.Notes, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}

func (s *SQLiteStorage) GetRelatedDevices(deviceID, relationshipType string) ([]model.Device, error) {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.name, d.description, d.make_model, d.os, d.datacenter_id,
		       d.username, d.location, d.created_at, d.updated_at
		FROM devices d
		JOIN device_relationships r ON (d.id = r.child_id OR d.id = r.parent_id)
		WHERE (r.parent_id = ? OR r.child_id = ?) AND r.type = ? AND d.id != ?
	`, deviceID, deviceID, relationshipType, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var d model.Device
		var dcID sql.NullString
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.MakeModel, &d.OS,
			&dcID, &d.Username, &d.Location, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		d.DatacenterID = dcID.String
		devices = append(devices, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch related data after closing rows
	for i := range devices {
		devices[i].Addresses, _ = s.getDeviceAddresses(ctx, devices[i].ID)
		devices[i].Tags, _ = s.getDeviceTags(ctx, devices[i].ID)
		devices[i].Domains, _ = s.getDeviceDomains(ctx, devices[i].ID)
	}
	return devices, nil
}

// auditLog creates an audit log entry asynchronously
func (s *SQLiteStorage) auditLog(ctx context.Context, action, resource, resourceID string, changes interface{}) {
	auditCtx, ok := audit.FromContext(ctx)
	if !ok {
		return
	}

	go func() {
		var changesStr string
		if changes != nil {
			if str, ok := changes.(string); ok {
				changesStr = str
			} else {
				changesBytes, err := json.Marshal(changes)
				if err == nil {
					changesStr = string(changesBytes)
				}
			}
		}

		auditLog := &model.AuditLog{
			Timestamp:  time.Now(),
			Action:     action,
			Resource:   resource,
			ResourceID: resourceID,
			UserID:     auditCtx.UserID,
			Username:   auditCtx.Username,
			IPAddress:  auditCtx.IPAddress,
			Changes:    changesStr,
			Source:     auditCtx.Source,
			Status:     "success",
		}

		if err := s.CreateAuditLog(auditLog); err != nil {
			log.Error("Failed to create audit log", "error", err)
		}
	}()
}
