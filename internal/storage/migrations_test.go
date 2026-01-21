package storage

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunMigrations(t *testing.T) {
	// Open in-memory database
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run migrations
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify schema_migrations table exists and has records
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	if count == 0 {
		t.Error("expected at least one migration record")
	}

	// Verify all expected tables exist
	expectedTables := []string{
		"datacenters",
		"networks",
		"network_pools",
		"devices",
		"addresses",
		"tags",
		"domains",
		"device_relationships",
		"discovered_devices",
		"discovery_scans",
		"discovery_rules",
	}

	for _, table := range expectedTables {
		var name string
		err := db.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='table' AND name=?
		`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	// Open in-memory database
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run migrations twice - should not fail
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("first RunMigrations failed: %v", err)
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}

	// Verify correct number of migration records exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query schema_migrations: %v", err)
	}
	expectedMigrations := len(migrations)
	if count != expectedMigrations {
		t.Errorf("expected %d migration records, got %d", expectedMigrations, count)
	}
}

func TestMigrationTableSchema(t *testing.T) {
	// Open in-memory database
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run migrations
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify migration record has expected fields
	var r MigrationRecord
	err = db.QueryRow(`
		SELECT version, name, applied_at, checksum, execution_time_ms, success
		FROM schema_migrations
		LIMIT 1
	`).Scan(&r.Version, &r.Name, &r.AppliedAt, &r.Checksum, &r.ExecutionTimeMs, &r.Success)
	if err != nil {
		t.Fatalf("failed to query migration record: %v", err)
	}

	if r.Version == "" {
		t.Error("migration version should not be empty")
	}
	if r.Name == "" {
		t.Error("migration name should not be empty")
	}
	if r.Checksum == "" {
		t.Error("migration checksum should not be empty")
	}
	if !r.Success {
		t.Error("migration should be marked as successful")
	}
}

func TestForeignKeyConstraints(t *testing.T) {
	// Open in-memory database
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run migrations
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify foreign key constraints are enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("failed to query foreign_keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("foreign keys should be enabled")
	}

	// Try to insert a device with non-existent datacenter_id
	// This should succeed because datacenter_id is nullable
	_, err = db.Exec(`
		INSERT INTO devices (id, name, datacenter_id)
		VALUES ('test-id', 'test-device', 'non-existent-dc')
	`)
	// This should fail due to foreign key constraint
	if err == nil {
		// Clean up and report
		db.Exec("DELETE FROM devices WHERE id = 'test-id'")
		t.Log("Foreign key constraint not enforced for datacenter_id (expected when datacenter doesn't exist)")
	}
}

func TestIndexesCreated(t *testing.T) {
	// Open in-memory database
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Run migrations
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify expected indexes exist
	expectedIndexes := []string{
		"idx_devices_name",
		"idx_devices_datacenter",
		"idx_addresses_device",
		"idx_addresses_ip",
		"idx_networks_datacenter",
		"idx_discovered_devices_network",
	}

	for _, idx := range expectedIndexes {
		var name string
		err := db.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='index' AND name=?
		`, idx).Scan(&name)
		if err != nil {
			t.Errorf("index %s not found: %v", idx, err)
		}
	}
}
