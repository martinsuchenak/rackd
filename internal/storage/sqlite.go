package storage

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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
		SELECT id, name, description, make_model, os, datacenter_id, username, location, created_at, updated_at
		FROM devices WHERE id = ?
	`, id).Scan(
		&device.ID, &device.Name, &device.Description, &device.MakeModel,
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
			addr.Port = int(port.Int64)
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
func (s *SQLiteStorage) CreateDevice(device *model.Device) error {
	if device == nil {
		return fmt.Errorf("device is nil")
	}

	ctx := context.Background()

	// Generate ID if not provided
	if device.ID == "" {
		device.ID = newUUID()
	}

	now := time.Now().UTC()
	device.CreatedAt = now
	device.UpdatedAt = now

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert device
	_, err = tx.ExecContext(ctx, `
		INSERT INTO devices (id, name, description, make_model, os, datacenter_id, username, location, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, device.ID, device.Name, device.Description, device.MakeModel,
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

	return tx.Commit()
}

// insertDeviceAddresses inserts addresses for a device within a transaction
func (s *SQLiteStorage) insertDeviceAddresses(ctx context.Context, tx *sql.Tx, deviceID string, addresses []model.Address) error {
	for _, addr := range addresses {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO addresses (id, device_id, ip, port, type, label, network_id, switch_port, pool_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, newUUID(), deviceID, addr.IP, nullInt(addr.Port), addr.Type, addr.Label,
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
func (s *SQLiteStorage) UpdateDevice(device *model.Device) error {
	if device == nil {
		return fmt.Errorf("device is nil")
	}
	if device.ID == "" {
		return ErrInvalidID
	}

	ctx := context.Background()

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if device exists
	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE id = ?)`, device.ID).Scan(&exists)
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
			name = ?, description = ?, make_model = ?, os = ?, datacenter_id = ?,
			username = ?, location = ?, updated_at = ?
		WHERE id = ?
	`, device.Name, device.Description, device.MakeModel, device.OS,
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

	return tx.Commit()
}

// DeleteDevice removes a device and all related data (cascades via foreign keys)
func (s *SQLiteStorage) DeleteDevice(id string) error {
	if id == "" {
		return ErrInvalidID
	}

	ctx := context.Background()

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check if device exists
	var exists bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM devices WHERE id = ?)`, id).Scan(&exists)
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

	return tx.Commit()
}

// ListDevices retrieves devices matching the filter criteria
func (s *SQLiteStorage) ListDevices(filter *model.DeviceFilter) ([]model.Device, error) {
	ctx := context.Background()

	query := `SELECT id, name, description, make_model, os, datacenter_id, username, location, created_at, updated_at FROM devices`
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
			&device.ID, &device.Name, &device.Description, &device.MakeModel,
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

// SearchDevices performs a text search across device fields
func (s *SQLiteStorage) SearchDevices(query string) ([]model.Device, error) {
	if query == "" {
		return s.ListDevices(nil)
	}

	ctx := context.Background()
	searchPattern := "%" + query + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT d.id, d.name, d.description, d.make_model, d.os, d.datacenter_id, d.username, d.location, d.created_at, d.updated_at
		FROM devices d
		LEFT JOIN addresses a ON d.id = a.device_id
		LEFT JOIN tags t ON d.id = t.device_id
		LEFT JOIN domains dm ON d.id = dm.device_id
		WHERE d.name LIKE ? OR d.description LIKE ? OR d.make_model LIKE ? OR d.os LIKE ?
		   OR d.location LIKE ? OR a.ip LIKE ? OR t.tag LIKE ? OR dm.domain LIKE ?
		ORDER BY d.name
	`, searchPattern, searchPattern, searchPattern, searchPattern,
		searchPattern, searchPattern, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search devices: %w", err)
	}
	defer rows.Close()

	var devices []model.Device
	for rows.Next() {
		var device model.Device
		var datacenterID sql.NullString
		if err := rows.Scan(
			&device.ID, &device.Name, &device.Description, &device.MakeModel,
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

// Datacenter operations

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
func (s *SQLiteStorage) CreateDatacenter(dc *model.Datacenter) error {
	if dc == nil {
		return fmt.Errorf("datacenter is nil")
	}

	ctx := context.Background()

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

	return nil
}

// UpdateDatacenter updates an existing datacenter
func (s *SQLiteStorage) UpdateDatacenter(dc *model.Datacenter) error {
	if dc == nil {
		return fmt.Errorf("datacenter is nil")
	}
	if dc.ID == "" {
		return ErrInvalidID
	}

	ctx := context.Background()

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

	return nil
}

// DeleteDatacenter removes a datacenter by ID
func (s *SQLiteStorage) DeleteDatacenter(id string) error {
	if id == "" {
		return ErrInvalidID
	}

	ctx := context.Background()

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
func (s *SQLiteStorage) CreateNetwork(network *model.Network) error {
	if network == nil {
		return fmt.Errorf("network is nil")
	}

	ctx := context.Background()

	// Generate ID if not provided
	if network.ID == "" {
		network.ID = newUUID()
	}

	now := time.Now().UTC()
	network.CreatedAt = now
	network.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
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
func (s *SQLiteStorage) UpdateNetwork(network *model.Network) error {
	if network == nil {
		return fmt.Errorf("network is nil")
	}
	if network.ID == "" {
		return ErrInvalidID
	}

	ctx := context.Background()

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

	return nil
}

// DeleteNetwork removes a network by ID
func (s *SQLiteStorage) DeleteNetwork(id string) error {
	if id == "" {
		return ErrInvalidID
	}

	ctx := context.Background()

	// Check if network exists
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM networks WHERE id = ?)`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check network existence: %w", err)
	}
	if !exists {
		return ErrNetworkNotFound
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

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

	return tx.Commit()
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

// Pool operations - stub implementations for now (will be completed in P2-007)

func (s *SQLiteStorage) CreateNetworkPool(pool *model.NetworkPool) error {
	return nil
}

func (s *SQLiteStorage) UpdateNetworkPool(pool *model.NetworkPool) error {
	return nil
}

func (s *SQLiteStorage) DeleteNetworkPool(id string) error {
	return nil
}

func (s *SQLiteStorage) GetNetworkPool(id string) (*model.NetworkPool, error) {
	return nil, ErrPoolNotFound
}

func (s *SQLiteStorage) ListNetworkPools(filter *model.NetworkPoolFilter) ([]model.NetworkPool, error) {
	return nil, nil
}

func (s *SQLiteStorage) GetNextAvailableIP(poolID string) (string, error) {
	return "", ErrIPNotAvailable
}

func (s *SQLiteStorage) ValidateIPInPool(poolID, ip string) (bool, error) {
	return false, nil
}

func (s *SQLiteStorage) GetPoolHeatmap(poolID string) ([]IPStatus, error) {
	return nil, nil
}

// Relationship operations - stub implementations for now (will be completed in P2-008)

func (s *SQLiteStorage) AddRelationship(parentID, childID, relationshipType string) error {
	return nil
}

func (s *SQLiteStorage) RemoveRelationship(parentID, childID, relationshipType string) error {
	return nil
}

func (s *SQLiteStorage) GetRelationships(deviceID string) ([]model.DeviceRelationship, error) {
	return nil, nil
}

func (s *SQLiteStorage) GetRelatedDevices(deviceID, relationshipType string) ([]model.Device, error) {
	return nil, nil
}

// Discovery operations - stub implementations for now (will be completed in P2-009)

func (s *SQLiteStorage) CreateDiscoveredDevice(device *model.DiscoveredDevice) error {
	return nil
}

func (s *SQLiteStorage) UpdateDiscoveredDevice(device *model.DiscoveredDevice) error {
	return nil
}

func (s *SQLiteStorage) GetDiscoveredDevice(id string) (*model.DiscoveredDevice, error) {
	return nil, ErrDiscoveryNotFound
}

func (s *SQLiteStorage) GetDiscoveredDeviceByIP(networkID, ip string) (*model.DiscoveredDevice, error) {
	return nil, ErrDiscoveryNotFound
}

func (s *SQLiteStorage) ListDiscoveredDevices(networkID string) ([]model.DiscoveredDevice, error) {
	return nil, nil
}

func (s *SQLiteStorage) DeleteDiscoveredDevice(id string) error {
	return nil
}

func (s *SQLiteStorage) PromoteDiscoveredDevice(discoveredID, deviceID string) error {
	return nil
}

func (s *SQLiteStorage) CreateDiscoveryScan(scan *model.DiscoveryScan) error {
	return nil
}

func (s *SQLiteStorage) UpdateDiscoveryScan(scan *model.DiscoveryScan) error {
	return nil
}

func (s *SQLiteStorage) GetDiscoveryScan(id string) (*model.DiscoveryScan, error) {
	return nil, ErrScanNotFound
}

func (s *SQLiteStorage) ListDiscoveryScans(networkID string) ([]model.DiscoveryScan, error) {
	return nil, nil
}

func (s *SQLiteStorage) GetDiscoveryRule(networkID string) (*model.DiscoveryRule, error) {
	return nil, ErrRuleNotFound
}

func (s *SQLiteStorage) SaveDiscoveryRule(rule *model.DiscoveryRule) error {
	return nil
}

func (s *SQLiteStorage) ListDiscoveryRules() ([]model.DiscoveryRule, error) {
	return nil, nil
}

func (s *SQLiteStorage) CleanupOldDiscoveries(olderThanDays int) error {
	return nil
}
