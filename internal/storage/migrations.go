package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

// Migration represents a single database migration
type Migration struct {
	Version  string
	Name     string
	Up       func(ctx context.Context, tx *sql.Tx) error
	Down     func(ctx context.Context, tx *sql.Tx) error
	Checksum string
}

// MigrationRecord represents a migration record in the database
type MigrationRecord struct {
	Version         string
	Name            string
	AppliedAt       time.Time
	Checksum        string
	ExecutionTimeMs int64
	Success         bool
}

// migrations is the ordered list of all migrations
var migrations = []*Migration{
	{
		Version: "20240120080000",
		Name:    "initial_schema",
		Up:      migrateInitialSchemaUp,
		Down:    migrateInitialSchemaDown,
	},
	{
		Version: "20240121080000",
		Name:    "add_pool_tags",
		Up:      migrateAddPoolTagsUp,
		Down:    migrateAddPoolTagsDown,
	},
	{
		Version: "20240122080000",
		Name:    "add_device_hostname",
		Up:      migrateAddDeviceHostnameUp,
		Down:    migrateAddDeviceHostnameDown,
	},
	{
		Version: "20260203000000",
		Name:    "add_relationship_notes",
		Up:      migrateAddRelationshipNotesUp,
		Down:    migrateAddRelationshipNotesDown,
	},
	{
		Version: "20260203110000",
		Name:    "add_fts_search",
		Up:      migrateFTSUp,
		Down:    migrateFTSDown,
	},
	{
		Version: "20260203120000",
		Name:    "add_api_keys",
		Up:      migrateAddAPIKeysUp,
		Down:    migrateAddAPIKeysDown,
	},
	{
		Version: "20260203160000",
		Name:    "add_audit_logs",
		Up:      migrateAddAuditLogsUp,
		Down:    migrateAddAuditLogsDown,
	},
	{
		Version: "20260203170000",
		Name:    "add_audit_source",
		Up:      migrateAddAuditSourceUp,
		Down:    migrateAddAuditSourceDown,
	},
	{
		Version: "20260206120000",
		Name:    "add_users",
		Up:      migrateAddUsersUp,
		Down:    migrateAddUsersDown,
	},
	{
		Version: "20260206130000",
		Name:    "add_rbac",
		Up:      migrateAddRBACUp,
		Down:    migrateAddRBACDown,
	},
	{
		Version: "20260207100000",
		Name:    "add_rbac_missing_permissions",
		Up:      migrateAddRBACMissingPermissionsUp,
		Down:    migrateAddRBACMissingPermissionsDown,
	},
	{
		Version: "20260207110000",
		Name:    "assign_roles_to_existing_admins",
		Up:      migrateAssignRolesToExistingAdminsUp,
		Down:    migrateAssignRolesToExistingAdminsDown,
	},
	{
		Version: "20260207120000",
		Name:    "add_apikey_user_id",
		Up:      migrateAddAPIKeyUserIDUp,
		Down:    migrateAddAPIKeyUserIDDown,
	},
	{
		Version: "20260210100000",
		Name:    "add_oauth_tables",
		Up:      migrateAddOAuthTablesUp,
		Down:    migrateAddOAuthTablesDown,
	},
}

// calculateChecksum generates a checksum for a migration
func calculateChecksum(m *Migration) string {
	data := m.Version + m.Name
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

// RunMigrations runs all pending migrations
func RunMigrations(ctx context.Context, db *sql.DB) error {
	// Create migration tracking table if it doesn't exist
	if err := createMigrationTable(ctx, db); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Run pending migrations
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			continue // Already applied
		}

		if err := runMigration(ctx, db, m); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Version, err)
		}
	}

	return nil
}

// createMigrationTable creates the schema_migrations table
func createMigrationTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			checksum TEXT NOT NULL,
			execution_time_ms INTEGER,
			success INTEGER NOT NULL DEFAULT 1
		)
	`)
	return err
}

// getAppliedMigrations returns a map of applied migration versions
func getAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]MigrationRecord, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT version, name, applied_at, checksum, execution_time_ms, success
		FROM schema_migrations
		WHERE success = 1
		ORDER BY version
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]MigrationRecord)
	for rows.Next() {
		var r MigrationRecord
		if err := rows.Scan(&r.Version, &r.Name, &r.AppliedAt, &r.Checksum, &r.ExecutionTimeMs, &r.Success); err != nil {
			return nil, err
		}
		applied[r.Version] = r
	}

	return applied, rows.Err()
}

// runMigration runs a single migration within a transaction
func runMigration(ctx context.Context, db *sql.DB, m *Migration) error {
	start := time.Now()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Run the migration
	if err := m.Up(ctx, tx); err != nil {
		return err
	}

	// Record the migration
	duration := time.Since(start)
	checksum := calculateChecksum(m)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (version, name, applied_at, checksum, execution_time_ms, success)
		VALUES (?, ?, ?, ?, ?, 1)
	`, m.Version, m.Name, time.Now().UTC(), checksum, duration.Milliseconds())
	if err != nil {
		return err
	}

	return tx.Commit()
}

// migrateInitialSchemaUp creates all initial tables
func migrateInitialSchemaUp(ctx context.Context, tx *sql.Tx) error {
	// Create datacenters table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS datacenters (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			location TEXT,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("failed to create datacenters table: %w", err)
	}

	// Create networks table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS networks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			subnet TEXT NOT NULL,
			vlan_id INTEGER,
			datacenter_id TEXT,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (datacenter_id) REFERENCES datacenters(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create networks table: %w", err)
	}

	// Create network_pools table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS network_pools (
			id TEXT PRIMARY KEY,
			network_id TEXT NOT NULL,
			name TEXT NOT NULL,
			start_ip TEXT NOT NULL,
			end_ip TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (network_id) REFERENCES networks(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create network_pools table: %w", err)
	}

	// Create devices table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS devices (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			make_model TEXT,
			os TEXT,
			datacenter_id TEXT,
			username TEXT,
			location TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (datacenter_id) REFERENCES datacenters(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create devices table: %w", err)
	}

	// Create addresses table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS addresses (
			id TEXT PRIMARY KEY,
			device_id TEXT NOT NULL,
			ip TEXT NOT NULL,
			port INTEGER,
			type TEXT DEFAULT 'ipv4',
			label TEXT,
			network_id TEXT,
			switch_port TEXT,
			pool_id TEXT,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
			FOREIGN KEY (network_id) REFERENCES networks(id),
			FOREIGN KEY (pool_id) REFERENCES network_pools(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create addresses table: %w", err)
	}

	// Create tags table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS tags (
			device_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			PRIMARY KEY (device_id, tag),
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create tags table: %w", err)
	}

	// Create domains table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS domains (
			device_id TEXT NOT NULL,
			domain TEXT NOT NULL,
			PRIMARY KEY (device_id, domain),
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create domains table: %w", err)
	}

	// Create device_relationships table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS device_relationships (
			parent_id TEXT NOT NULL,
			child_id TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (parent_id, child_id, type),
			FOREIGN KEY (parent_id) REFERENCES devices(id) ON DELETE CASCADE,
			FOREIGN KEY (child_id) REFERENCES devices(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create device_relationships table: %w", err)
	}

	// Create discovered_devices table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS discovered_devices (
			id TEXT PRIMARY KEY,
			ip TEXT NOT NULL,
			mac_address TEXT,
			hostname TEXT,
			network_id TEXT,
			status TEXT DEFAULT 'unknown',
			confidence INTEGER DEFAULT 0,
			os_guess TEXT,
			vendor TEXT,
			open_ports TEXT,
			services TEXT,
			first_seen TIMESTAMP,
			last_seen TIMESTAMP,
			promoted_to_device_id TEXT,
			promoted_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (network_id) REFERENCES networks(id),
			FOREIGN KEY (promoted_to_device_id) REFERENCES devices(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create discovered_devices table: %w", err)
	}

	// Create discovery_scans table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS discovery_scans (
			id TEXT PRIMARY KEY,
			network_id TEXT,
			status TEXT DEFAULT 'pending',
			scan_type TEXT DEFAULT 'full',
			total_hosts INTEGER DEFAULT 0,
			scanned_hosts INTEGER DEFAULT 0,
			found_hosts INTEGER DEFAULT 0,
			progress_percent REAL DEFAULT 0,
			error_message TEXT,
			started_at TIMESTAMP,
			completed_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (network_id) REFERENCES networks(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create discovery_scans table: %w", err)
	}

	// Create discovery_rules table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS discovery_rules (
			id TEXT PRIMARY KEY,
			network_id TEXT UNIQUE,
			enabled INTEGER DEFAULT 1,
			scan_type TEXT DEFAULT 'full',
			interval_hours INTEGER DEFAULT 24,
			exclude_ips TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (network_id) REFERENCES networks(id)
		)
	`); err != nil {
		return fmt.Errorf("failed to create discovery_rules table: %w", err)
	}

	// Create indexes for common queries
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_devices_name ON devices(name)`,
		`CREATE INDEX IF NOT EXISTS idx_devices_datacenter ON devices(datacenter_id)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_device ON addresses(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_ip ON addresses(ip)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_network ON addresses(network_id)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_pool ON addresses(pool_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tags_device ON tags(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_domains_device ON domains(device_id)`,
		`CREATE INDEX IF NOT EXISTS idx_networks_datacenter ON networks(datacenter_id)`,
		`CREATE INDEX IF NOT EXISTS idx_network_pools_network ON network_pools(network_id)`,
		`CREATE INDEX IF NOT EXISTS idx_discovered_devices_network ON discovered_devices(network_id)`,
		`CREATE INDEX IF NOT EXISTS idx_discovered_devices_ip ON discovered_devices(ip)`,
		`CREATE INDEX IF NOT EXISTS idx_discovery_scans_network ON discovery_scans(network_id)`,
		`CREATE INDEX IF NOT EXISTS idx_device_relationships_parent ON device_relationships(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_device_relationships_child ON device_relationships(child_id)`,
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// migrateInitialSchemaDown drops all tables
func migrateInitialSchemaDown(ctx context.Context, tx *sql.Tx) error {
	// Drop tables in reverse order of dependencies
	tables := []string{
		"discovery_rules",
		"discovery_scans",
		"discovered_devices",
		"device_relationships",
		"domains",
		"tags",
		"addresses",
		"devices",
		"network_pools",
		"networks",
		"datacenters",
	}

	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS `+table); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// migrateAddPoolTagsUp creates the pool_tags table
func migrateAddPoolTagsUp(ctx context.Context, tx *sql.Tx) error {
	// Create pool_tags table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS pool_tags (
			pool_id TEXT NOT NULL,
			tag TEXT NOT NULL,
			PRIMARY KEY (pool_id, tag),
			FOREIGN KEY (pool_id) REFERENCES network_pools(id) ON DELETE CASCADE
		)
	`); err != nil {
		return fmt.Errorf("failed to create pool_tags table: %w", err)
	}

	// Create index for pool_tags
	if _, err := tx.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS idx_pool_tags_pool ON pool_tags(pool_id)
	`); err != nil {
		return fmt.Errorf("failed to create pool_tags index: %w", err)
	}

	return nil
}

// migrateAddPoolTagsDown drops the pool_tags table
func migrateAddPoolTagsDown(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS pool_tags`); err != nil {
		return fmt.Errorf("failed to drop pool_tags table: %w", err)
	}
	return nil
}

// migrateAddDeviceHostnameUp adds the hostname column to devices table
func migrateAddDeviceHostnameUp(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE devices ADD COLUMN hostname TEXT DEFAULT ''`); err != nil {
		return fmt.Errorf("failed to add hostname column: %w", err)
	}
	return nil
}

// migrateAddDeviceHostnameDown removes the hostname column from devices table
func migrateAddDeviceHostnameDown(ctx context.Context, tx *sql.Tx) error {
	// SQLite doesn't support DROP COLUMN directly, so we'd need to recreate the table
	// For simplicity, we'll just leave the column (it's safe to have extra columns)
	return nil
}

// migrateAddRelationshipNotesUp adds the notes column to device_relationships table
func migrateAddRelationshipNotesUp(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE device_relationships ADD COLUMN notes TEXT DEFAULT ''`); err != nil {
		return fmt.Errorf("failed to add notes column: %w", err)
	}
	return nil
}

// migrateAddRelationshipNotesDown removes the notes column from device_relationships table
func migrateAddRelationshipNotesDown(ctx context.Context, tx *sql.Tx) error {
	return nil
}

// migrateFTSUp creates FTS5 virtual tables for full-text search
func migrateFTSUp(ctx context.Context, tx *sql.Tx) error {
	// Create standalone FTS5 virtual table for devices
	if _, err := tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS devices_fts USING fts5(
			id UNINDEXED,
			name,
			hostname,
			description,
			make_model,
			os,
			location
		)
	`); err != nil {
		return fmt.Errorf("failed to create devices_fts table: %w", err)
	}

	// Create triggers to keep FTS table in sync
	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS devices_fts_insert AFTER INSERT ON devices BEGIN
			INSERT INTO devices_fts(id, name, hostname, description, make_model, os, location)
			VALUES (new.id, new.name, COALESCE(new.hostname, ''), COALESCE(new.description, ''), 
				   COALESCE(new.make_model, ''), COALESCE(new.os, ''), COALESCE(new.location, ''));
		END
	`); err != nil {
		return fmt.Errorf("failed to create devices_fts_insert trigger: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS devices_fts_delete AFTER DELETE ON devices BEGIN
			DELETE FROM devices_fts WHERE id = old.id;
		END
	`); err != nil {
		return fmt.Errorf("failed to create devices_fts_delete trigger: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS devices_fts_update AFTER UPDATE ON devices BEGIN
			UPDATE devices_fts SET 
				name = new.name,
				hostname = COALESCE(new.hostname, ''),
				description = COALESCE(new.description, ''),
				make_model = COALESCE(new.make_model, ''),
				os = COALESCE(new.os, ''),
				location = COALESCE(new.location, '')
			WHERE id = old.id;
		END
	`); err != nil {
		return fmt.Errorf("failed to create devices_fts_update trigger: %w", err)
	}

	// Populate FTS table with existing data
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO devices_fts(id, name, hostname, description, make_model, os, location)
		SELECT id, name, COALESCE(hostname, ''), COALESCE(description, ''),
			   COALESCE(make_model, ''), COALESCE(os, ''), COALESCE(location, '')
		FROM devices
	`); err != nil {
		return fmt.Errorf("failed to populate devices_fts: %w", err)
	}

	// Create standalone FTS5 virtual table for networks
	if _, err := tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS networks_fts USING fts5(
			id UNINDEXED,
			name,
			subnet,
			description
		)
	`); err != nil {
		return fmt.Errorf("failed to create networks_fts table: %w", err)
	}

	// Network FTS triggers
	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS networks_fts_insert AFTER INSERT ON networks BEGIN
			INSERT INTO networks_fts(id, name, subnet, description)
			VALUES (new.id, new.name, new.subnet, COALESCE(new.description, ''));
		END
	`); err != nil {
		return fmt.Errorf("failed to create networks_fts_insert trigger: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS networks_fts_delete AFTER DELETE ON networks BEGIN
			DELETE FROM networks_fts WHERE id = old.id;
		END
	`); err != nil {
		return fmt.Errorf("failed to create networks_fts_delete trigger: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS networks_fts_update AFTER UPDATE ON networks BEGIN
			UPDATE networks_fts SET 
				name = new.name,
				subnet = new.subnet,
				description = COALESCE(new.description, '')
			WHERE id = old.id;
		END
	`); err != nil {
		return fmt.Errorf("failed to create networks_fts_update trigger: %w", err)
	}

	// Populate networks FTS
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO networks_fts(id, name, subnet, description)
		SELECT id, name, subnet, COALESCE(description, '')
		FROM networks
	`); err != nil {
		return fmt.Errorf("failed to populate networks_fts: %w", err)
	}

	// Create standalone FTS5 virtual table for datacenters
	if _, err := tx.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS datacenters_fts USING fts5(
			id UNINDEXED,
			name,
			location,
			description
		)
	`); err != nil {
		return fmt.Errorf("failed to create datacenters_fts table: %w", err)
	}

	// Datacenter FTS triggers
	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS datacenters_fts_insert AFTER INSERT ON datacenters BEGIN
			INSERT INTO datacenters_fts(id, name, location, description)
			VALUES (new.id, new.name, COALESCE(new.location, ''), COALESCE(new.description, ''));
		END
	`); err != nil {
		return fmt.Errorf("failed to create datacenters_fts_insert trigger: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS datacenters_fts_delete AFTER DELETE ON datacenters BEGIN
			DELETE FROM datacenters_fts WHERE id = old.id;
		END
	`); err != nil {
		return fmt.Errorf("failed to create datacenters_fts_delete trigger: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS datacenters_fts_update AFTER UPDATE ON datacenters BEGIN
			UPDATE datacenters_fts SET 
				name = new.name,
				location = COALESCE(new.location, ''),
				description = COALESCE(new.description, '')
			WHERE id = old.id;
		END
	`); err != nil {
		return fmt.Errorf("failed to create datacenters_fts_update trigger: %w", err)
	}

	// Populate datacenters FTS
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO datacenters_fts(id, name, location, description)
		SELECT id, name, COALESCE(location, ''), COALESCE(description, '')
		FROM datacenters
	`); err != nil {
		return fmt.Errorf("failed to populate datacenters_fts: %w", err)
	}

	return nil
}

// migrateFTSDown drops FTS5 virtual tables and triggers
func migrateFTSDown(ctx context.Context, tx *sql.Tx) error {
	// Drop triggers
	for _, trigger := range []string{
		"devices_fts_insert", "devices_fts_delete", "devices_fts_update",
		"networks_fts_insert", "networks_fts_delete", "networks_fts_update",
		"datacenters_fts_insert", "datacenters_fts_delete", "datacenters_fts_update",
	} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DROP TRIGGER IF EXISTS %s", trigger)); err != nil {
			return fmt.Errorf("failed to drop trigger %s: %w", trigger, err)
		}
	}

	// Drop FTS tables
	for _, table := range []string{"devices_fts", "networks_fts", "datacenters_fts"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// migrateAddAPIKeysUp creates the api_keys table
func migrateAddAPIKeysUp(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			key TEXT NOT NULL UNIQUE,
			description TEXT,
			created_at DATETIME NOT NULL,
			last_used_at DATETIME,
			expires_at DATETIME
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create api_keys table: %w", err)
	}

	// Create indexes
	if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key)`); err != nil {
		return fmt.Errorf("failed to create api_keys key index: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_api_keys_name ON api_keys(name)`); err != nil {
		return fmt.Errorf("failed to create api_keys name index: %w", err)
	}

	return nil
}

// migrateAddAPIKeysDown drops the api_keys table
func migrateAddAPIKeysDown(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS api_keys`); err != nil {
		return fmt.Errorf("failed to drop api_keys table: %w", err)
	}
	return nil
}

// migrateAddAuditLogsUp creates the audit_logs table
func migrateAddAuditLogsUp(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			resource_id TEXT,
			user_id TEXT,
			username TEXT,
			ip_address TEXT,
			changes TEXT,
			status TEXT NOT NULL,
			error TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create audit_logs table: %w", err)
	}

	// Create indexes for common queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource, resource_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)",
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create audit_logs index: %w", err)
		}
	}

	return nil
}

// migrateAddAuditLogsDown drops the audit_logs table
func migrateAddAuditLogsDown(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS audit_logs`); err != nil {
		return fmt.Errorf("failed to drop audit_logs table: %w", err)
	}
	return nil
}

// migrateAddAuditSourceUp adds source column to audit_logs table
func migrateAddAuditSourceUp(ctx context.Context, tx *sql.Tx) error {
	// Add source column
	if _, err := tx.ExecContext(ctx, `ALTER TABLE audit_logs ADD COLUMN source TEXT`); err != nil {
		return fmt.Errorf("failed to add source column to audit_logs: %w", err)
	}

	// Create index on source column
	if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_audit_logs_source ON audit_logs(source)`); err != nil {
		return fmt.Errorf("failed to create idx_audit_logs_source index: %w", err)
	}

	return nil
}

// migrateAddAuditSourceDown removes source column from audit_logs table
func migrateAddAuditSourceDown(ctx context.Context, tx *sql.Tx) error {
	// Drop index first
	if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS idx_audit_logs_source`); err != nil {
		return fmt.Errorf("failed to drop idx_audit_logs_source index: %w", err)
	}

	// SQLite doesn't support ALTER TABLE DROP COLUMN directly, need to recreate table
	if _, err := tx.ExecContext(ctx, `
		CREATE TABLE audit_logs_new (
			id TEXT PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			resource_id TEXT,
			user_id TEXT,
			username TEXT,
			ip_address TEXT,
			changes TEXT,
			status TEXT NOT NULL,
			error TEXT
		)
	`); err != nil {
		return fmt.Errorf("failed to create audit_logs_new table: %w", err)
	}

	// Copy data from old table to new table
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs_new (id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error)
		SELECT id, timestamp, action, resource, resource_id, user_id, username, ip_address, changes, status, error
		FROM audit_logs
	`); err != nil {
		return fmt.Errorf("failed to copy data to audit_logs_new: %w", err)
	}

	// Drop old table
	if _, err := tx.ExecContext(ctx, `DROP TABLE audit_logs`); err != nil {
		return fmt.Errorf("failed to drop old audit_logs table: %w", err)
	}

	// Rename new table to original name
	if _, err := tx.ExecContext(ctx, `ALTER TABLE audit_logs_new RENAME TO audit_logs`); err != nil {
		return fmt.Errorf("failed to rename audit_logs_new to audit_logs: %w", err)
	}

	// Recreate indexes (they were dropped with the table)
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource, resource_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)",
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to recreate audit_logs index: %w", err)
		}
	}

	return nil
}

func migrateAddUsersUp(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			email TEXT UNIQUE,
			full_name TEXT,
			password_hash TEXT NOT NULL,
			is_active INTEGER NOT NULL DEFAULT 1,
			is_admin INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			last_login_at DATETIME
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active)",
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create user index: %w", err)
		}
	}

	return nil
}

func migrateAddUsersDown(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS users`); err != nil {
		return fmt.Errorf("failed to drop users table: %w", err)
	}
	return nil
}

func migrateAddRBACUp(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			created_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create permissions table: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS roles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			is_system INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create roles table: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id TEXT NOT NULL,
			permission_id TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (role_id, permission_id),
			FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
			FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create role_permissions table: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS user_roles (
			user_id TEXT NOT NULL,
			role_id TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			PRIMARY KEY (user_id, role_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create user_roles table: %w", err)
	}

	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource, action)",
		"CREATE INDEX IF NOT EXISTS idx_role_permissions_role ON role_permissions(role_id)",
		"CREATE INDEX IF NOT EXISTS idx_role_permissions_permission ON role_permissions(permission_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id)",
	}

	for _, idx := range indexes {
		if _, err := tx.ExecContext(ctx, idx); err != nil {
			return fmt.Errorf("failed to create rbac index: %w", err)
		}
	}

	now := time.Now()

	defaultPermissions := [][]string{
		{"device:list", "devices", "list"},
		{"device:create", "devices", "create"},
		{"device:read", "devices", "read"},
		{"device:update", "devices", "update"},
		{"device:delete", "devices", "delete"},
		{"network:list", "networks", "list"},
		{"network:create", "networks", "create"},
		{"network:read", "networks", "read"},
		{"network:update", "networks", "update"},
		{"network:delete", "networks", "delete"},
		{"datacenter:list", "datacenters", "list"},
		{"datacenter:create", "datacenters", "create"},
		{"datacenter:read", "datacenters", "read"},
		{"datacenter:update", "datacenters", "update"},
		{"datacenter:delete", "datacenters", "delete"},
		{"discovery:list", "discovery", "list"},
		{"discovery:create", "discovery", "create"},
		{"discovery:read", "discovery", "read"},
		{"discovery:delete", "discovery", "delete"},
		{"user:list", "users", "list"},
		{"user:create", "users", "create"},
		{"user:read", "users", "read"},
		{"user:update", "users", "update"},
		{"user:delete", "users", "delete"},
		{"role:list", "roles", "list"},
		{"role:create", "roles", "create"},
		{"role:read", "roles", "read"},
		{"role:update", "roles", "update"},
		{"role:delete", "roles", "delete"},
		{"audit:list", "audit", "list"},
		{"apikey:list", "apikeys", "list"},
		{"apikey:create", "apikeys", "create"},
		{"apikey:read", "apikeys", "read"},
		{"apikey:update", "apikeys", "update"},
		{"apikey:delete", "apikeys", "delete"},
	}

	for _, perm := range defaultPermissions {
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO permissions (id, name, resource, action, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, newUUID(), perm[0], perm[1], perm[2], now)
		if err != nil {
			return fmt.Errorf("failed to insert default permission: %w", err)
		}
	}

	roles := [][]any{
		{"admin", "Full administrative access", true},
		{"operator", "Can manage devices, networks, and discovery", true},
		{"viewer", "Read-only access", true},
	}

	for _, role := range roles {
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO roles (id, name, description, is_system, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, newUUID(), role[0], role[1], role[2], now, now)
		if err != nil {
			return fmt.Errorf("failed to insert default role: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO role_permissions (role_id, permission_id, created_at)
		SELECT r.id, p.id, ?
		FROM roles r, permissions p
		WHERE r.name = 'admin'
	`, now)
	if err != nil {
		return fmt.Errorf("failed to assign permissions to admin role: %w", err)
	}

	operatorPerms := []string{
		"device:list", "device:create", "device:read", "device:update",
		"network:list", "network:create", "network:read", "network:update",
		"datacenter:list", "datacenter:read",
		"discovery:list", "discovery:create", "discovery:read",
	}
	for _, permName := range operatorPerms {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO role_permissions (role_id, permission_id, created_at)
			SELECT r.id, p.id, ?
			FROM roles r, permissions p
			WHERE r.name = 'operator' AND p.name = ?
		`, now, permName)
		if err != nil {
			return fmt.Errorf("failed to assign operator permission: %w", err)
		}
	}

	viewerPerms := []string{
		"device:list", "device:read",
		"network:list", "network:read",
		"datacenter:list", "datacenter:read",
		"discovery:list", "discovery:read",
	}
	for _, permName := range viewerPerms {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO role_permissions (role_id, permission_id, created_at)
			SELECT r.id, p.id, ?
			FROM roles r, permissions p
			WHERE r.name = 'viewer' AND p.name = ?
		`, now, permName)
		if err != nil {
			return fmt.Errorf("failed to assign viewer permission: %w", err)
		}
	}

	return nil
}

func migrateAddRBACDown(ctx context.Context, tx *sql.Tx) error {
	tables := []string{"user_roles", "role_permissions", "roles", "permissions"}

	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS `+table); err != nil {
			return fmt.Errorf("failed to drop %s table: %w", table, err)
		}
	}

	return nil
}

func migrateAddRBACMissingPermissionsUp(ctx context.Context, tx *sql.Tx) error {
	now := time.Now()

	// Add missing permissions for resources that weren't covered in the initial RBAC migration
	newPermissions := [][]string{
		{"pool:list", "pools", "list"},
		{"pool:create", "pools", "create"},
		{"pool:read", "pools", "read"},
		{"pool:update", "pools", "update"},
		{"pool:delete", "pools", "delete"},
		{"credential:list", "credentials", "list"},
		{"credential:create", "credentials", "create"},
		{"credential:read", "credentials", "read"},
		{"credential:update", "credentials", "update"},
		{"credential:delete", "credentials", "delete"},
		{"scan-profile:list", "scan-profiles", "list"},
		{"scan-profile:create", "scan-profiles", "create"},
		{"scan-profile:read", "scan-profiles", "read"},
		{"scan-profile:update", "scan-profiles", "update"},
		{"scan-profile:delete", "scan-profiles", "delete"},
		{"scheduled-scan:list", "scheduled-scans", "list"},
		{"scheduled-scan:create", "scheduled-scans", "create"},
		{"scheduled-scan:read", "scheduled-scans", "read"},
		{"scheduled-scan:update", "scheduled-scans", "update"},
		{"scheduled-scan:delete", "scheduled-scans", "delete"},
		{"relationship:list", "relationships", "list"},
		{"relationship:create", "relationships", "create"},
		{"relationship:read", "relationships", "read"},
		{"relationship:update", "relationships", "update"},
		{"relationship:delete", "relationships", "delete"},
		{"search:read", "search", "read"},
	}

	for _, perm := range newPermissions {
		_, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO permissions (id, name, resource, action, created_at)
			VALUES (?, ?, ?, ?, ?)
		`, newUUID(), perm[0], perm[1], perm[2], now)
		if err != nil {
			return fmt.Errorf("failed to insert permission %s: %w", perm[0], err)
		}
	}

	// Grant all new permissions to admin role
	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO role_permissions (role_id, permission_id, created_at)
		SELECT r.id, p.id, ?
		FROM roles r, permissions p
		WHERE r.name = 'admin'
		AND p.id NOT IN (SELECT permission_id FROM role_permissions WHERE role_id = r.id)
	`, now)
	if err != nil {
		return fmt.Errorf("failed to assign new permissions to admin role: %w", err)
	}

	// Grant operator permissions for pools, credentials, scan-profiles, scheduled-scans, relationships, search
	operatorPerms := []string{
		"pool:list", "pool:create", "pool:read", "pool:update",
		"credential:list", "credential:create", "credential:read", "credential:update",
		"scan-profile:list", "scan-profile:create", "scan-profile:read", "scan-profile:update",
		"scheduled-scan:list", "scheduled-scan:create", "scheduled-scan:read", "scheduled-scan:update",
		"relationship:list", "relationship:create", "relationship:read", "relationship:update",
		"search:read",
	}
	for _, permName := range operatorPerms {
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO role_permissions (role_id, permission_id, created_at)
			SELECT r.id, p.id, ?
			FROM roles r, permissions p
			WHERE r.name = 'operator' AND p.name = ?
		`, now, permName)
		if err != nil {
			return fmt.Errorf("failed to assign operator permission %s: %w", permName, err)
		}
	}

	// Grant viewer read permissions
	viewerPerms := []string{
		"pool:list", "pool:read",
		"credential:list", "credential:read",
		"scan-profile:list", "scan-profile:read",
		"scheduled-scan:list", "scheduled-scan:read",
		"relationship:list", "relationship:read",
		"search:read",
	}
	for _, permName := range viewerPerms {
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO role_permissions (role_id, permission_id, created_at)
			SELECT r.id, p.id, ?
			FROM roles r, permissions p
			WHERE r.name = 'viewer' AND p.name = ?
		`, now, permName)
		if err != nil {
			return fmt.Errorf("failed to assign viewer permission %s: %w", permName, err)
		}
	}

	return nil
}

func migrateAddRBACMissingPermissionsDown(ctx context.Context, tx *sql.Tx) error {
	permNames := []string{
		"pool:list", "pool:create", "pool:read", "pool:update", "pool:delete",
		"credential:list", "credential:create", "credential:read", "credential:update", "credential:delete",
		"scan-profile:list", "scan-profile:create", "scan-profile:read", "scan-profile:update", "scan-profile:delete",
		"scheduled-scan:list", "scheduled-scan:create", "scheduled-scan:read", "scheduled-scan:update", "scheduled-scan:delete",
		"relationship:list", "relationship:create", "relationship:read", "relationship:update", "relationship:delete",
		"search:read",
	}
	for _, name := range permNames {
		if _, err := tx.ExecContext(ctx, `DELETE FROM permissions WHERE name = ?`, name); err != nil {
			return fmt.Errorf("failed to delete permission %s: %w", name, err)
		}
	}
	return nil
}

// migrateAssignRolesToExistingAdminsUp assigns the admin role to any existing users
// with is_admin=true who don't already have it. This fixes the case where users were
// created before RBAC was introduced and thus have no entries in user_roles.
func migrateAssignRolesToExistingAdminsUp(ctx context.Context, tx *sql.Tx) error {
	now := time.Now()

	_, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO user_roles (user_id, role_id, created_at)
		SELECT u.id, r.id, ?
		FROM users u, roles r
		WHERE u.is_admin = 1
		AND r.name = 'admin'
		AND NOT EXISTS (
			SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_id = r.id
		)
	`, now)
	if err != nil {
		return fmt.Errorf("failed to assign admin role to existing admin users: %w", err)
	}

	return nil
}

func migrateAssignRolesToExistingAdminsDown(ctx context.Context, tx *sql.Tx) error {
	// No-op: removing role assignments could lock out admin users
	return nil
}

// migrateAddAPIKeyUserIDUp adds user_id column to api_keys table
func migrateAddAPIKeyUserIDUp(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `ALTER TABLE api_keys ADD COLUMN user_id TEXT REFERENCES users(id)`); err != nil {
		return fmt.Errorf("failed to add user_id column to api_keys: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id)`); err != nil {
		return fmt.Errorf("failed to create api_keys user_id index: %w", err)
	}
	return nil
}

// migrateAddAPIKeyUserIDDown removes user_id column from api_keys table
func migrateAddAPIKeyUserIDDown(ctx context.Context, tx *sql.Tx) error {
	// SQLite doesn't support DROP COLUMN before 3.35.0, so recreate the table
	queries := []string{
		`CREATE TABLE api_keys_backup AS SELECT id, name, key, description, created_at, last_used_at, expires_at FROM api_keys`,
		`DROP TABLE api_keys`,
		`ALTER TABLE api_keys_backup RENAME TO api_keys`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_name ON api_keys(name)`,
	}
	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("failed to remove user_id column from api_keys: %w", err)
		}
	}
	return nil
}
