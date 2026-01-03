package storage

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// MigrateToV2 migrates from schema v1 (location text) to v2 (datacenter_id reference)
// - Creates datacenters table if it doesn't exist
// - Converts existing location strings to datacenter references
func (ss *SQLiteStorage) MigrateToV2() error {
	// Check if already migrated
	var version int
	err := ss.db.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("checking migration version: %w", err)
	}
	if version >= 2 {
		return nil // Already migrated
	}

	tx, err := ss.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if devices table has the datacenter_id column
	// If it doesn't, we need to migrate
	var datacenterIDColumn string
	err = tx.QueryRow(`
		SELECT name FROM pragma_table_info('devices')
		WHERE name='datacenter_id'
	`).Scan(&datacenterIDColumn)

	needsMigration := (err == sql.ErrNoRows)

	if needsMigration {
		// Devices table needs migration - first ensure datacenters table exists
		var tableName string
		err = tx.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='table' AND name='datacenters'
		`).Scan(&tableName)

		if err == sql.ErrNoRows {
			// Table doesn't exist - create it
			_, err = tx.Exec(`
				CREATE TABLE datacenters (
					id TEXT PRIMARY KEY,
					name TEXT NOT NULL UNIQUE,
					location TEXT,
					description TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
					updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
				)
			`)
			if err != nil {
				return fmt.Errorf("creating datacenters table: %w", err)
			}

			_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_datacenters_name ON datacenters(name)`)
			if err != nil {
				return fmt.Errorf("creating datacenters index: %w", err)
			}

			// Create trigger for updated_at
			_, err = tx.Exec(`
				CREATE TRIGGER IF NOT EXISTS update_datacenters_timestamp
				AFTER UPDATE ON datacenters
				FOR EACH ROW
				BEGIN
					UPDATE datacenters SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
				END
			`)
			if err != nil {
				return fmt.Errorf("creating datacenters trigger: %w", err)
			}
		}

		// Get unique location values from existing devices
		rows, err := tx.Query(`
			SELECT DISTINCT location
			FROM devices
			WHERE location IS NOT NULL AND location != ''
			ORDER BY location
		`)
		if err != nil {
			return fmt.Errorf("querying existing locations: %w", err)
		}
		defer rows.Close()

		var locations []string
		for rows.Next() {
			var loc string
			if err := rows.Scan(&loc); err != nil {
				return fmt.Errorf("scanning location: %w", err)
			}
			locations = append(locations, loc)
		}
		rows.Close()

		// Create datacenter entries from unique locations
		for _, location := range locations {
			dcID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO datacenters (id, name, location, description, created_at, updated_at)
				VALUES (?, ?, '', '', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			`, dcID, location)
			if err != nil {
				return fmt.Errorf("creating datacenter for %s: %w", location, err)
			}
		}

		// Add new datacenter_id column (allowing NULL initially)
		_, err = tx.Exec(`ALTER TABLE devices ADD COLUMN datacenter_id TEXT`)
		if err != nil {
			// Column might already exist
			if !isDuplicateColumnError(err) {
				return fmt.Errorf("adding datacenter_id column: %w", err)
			}
		}

		// Update devices to reference new datacenters
		_, err = tx.Exec(`
			UPDATE devices
			SET datacenter_id = (
				SELECT id FROM datacenters WHERE name = devices.location
			)
			WHERE location IS NOT NULL AND location != ''
		`)
		if err != nil {
			return fmt.Errorf("updating device datacenter references: %w", err)
		}

		// Drop old location column by recreating the table
		_, err = tx.Exec(`
			CREATE TABLE devices_new (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				description TEXT,
				make_model TEXT,
				os TEXT,
				datacenter_id TEXT,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (datacenter_id) REFERENCES datacenters(id) ON DELETE SET NULL
			)
		`)
		if err != nil {
			return fmt.Errorf("creating new devices table: %w", err)
		}

		_, err = tx.Exec(`
			INSERT INTO devices_new (id, name, description, make_model, os, datacenter_id, created_at, updated_at)
			SELECT id, name, description, make_model, os, datacenter_id, created_at, updated_at
			FROM devices
		`)
		if err != nil {
			return fmt.Errorf("migrating device data: %w", err)
		}

		_, err = tx.Exec(`DROP TABLE devices`)
		if err != nil {
			return fmt.Errorf("dropping old devices table: %w", err)
		}

		_, err = tx.Exec(`ALTER TABLE devices_new RENAME TO devices`)
		if err != nil {
			return fmt.Errorf("renaming devices table: %w", err)
		}

		// Recreate indexes
		_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_devices_name ON devices(name)`)
		if err != nil {
			return fmt.Errorf("recreating devices index: %w", err)
		}

		// Recreate the update trigger
		_, err = tx.Exec(`
			CREATE TRIGGER IF NOT EXISTS update_devices_timestamp
			AFTER UPDATE ON devices
			FOR EACH ROW
			BEGIN
				UPDATE devices SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END
		`)
		if err != nil {
			return fmt.Errorf("recreating devices trigger: %w", err)
		}
	}

	// Create schema_migrations table if it doesn't exist
	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	// Update migration version
	_, err = tx.Exec(`INSERT OR IGNORE INTO schema_migrations (version) VALUES (2)`)
	if err != nil {
		return fmt.Errorf("setting migration version: %w", err)
	}

	return tx.Commit()
}

// isDuplicateColumnError checks if the error is about duplicate column
func isDuplicateColumnError(err error) bool {
	return err != nil && (err.Error() == "duplicate column name: datacenter_id" ||
		err.Error() == "table devices has no column named location")
}
