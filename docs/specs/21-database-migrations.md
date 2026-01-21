# Database Migrations

This document defines the database migration system for Rackd, including schema versioning, migration lifecycle, and implementation patterns.

## 1. Migration System Architecture

### 1.1 Migration Table Schema

```sql
CREATE TABLE schema_migrations (
    version TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    checksum TEXT NOT NULL,
    execution_time_ms INTEGER,
    success INTEGER NOT NULL DEFAULT 1
);
```

### 1.2 Version Tracking Mechanism

All migrations use semantic versioning prefixed with timestamp:

**Version Format:** `YYYYMMDDHHMMSS_version_name`

**Examples:**
- `20240120080000_initial_schema`
- `20240120100000_add_device_tags`
- `20240120200000_add_discovery_scans`

**Schema Versioning:**

| Schema Version | Migration Range | Description |
|---------------|-----------------|-------------|
| v1.0.0 | 20240120* | Initial schema |
| v1.1.0 | 20240121* - 20240130* | Added discovery |
| v1.2.0 | 20240201* - 20240228* | Added relationships |
| v2.0.0 | 20240301* - ... | Major schema changes |

### 1.3 Dependency Graph

Migrations can have dependencies on previous migrations:

```
initial_schema
    ├─> add_device_tags
    ├─> add_network_pools
    └─> add_discovery_scans
          ├─> add_discovery_rules
          └─> add_discovery_services
```

**Dependency Declaration:**

```go
var Migration = &migrate.Migration{
    Name:         "add_discovery_rules",
    Dependencies: []string{"add_discovery_scans"},
    Up:           upFunc,
    Down:         downFunc,
}
```

---

## 2. Schema Versioning

### 2.1 Semantic Versioning for Schema

Schema versions follow: `MAJOR.MINOR.PATCH`

- **MAJOR**: Breaking changes, data structure changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, non-breaking changes

**Version Compatibility:**

| Schema Version | Compatible API | Breaking? |
|---------------|----------------|-----------|
| v1.0.0 → v1.1.0 | v1.0, v1.1 | No |
| v1.1.0 → v2.0.0 | v2.0 only | Yes |

### 2.2 Breaking vs Non-Breaking Classification

**Breaking Changes:**
- Column deletion or rename
- Table deletion or rename
- Changing column data type (lossy)
- Changing primary key
- Adding NOT NULL constraint to existing column
- Removing unique constraint

**Non-Breaking Changes:**
- Adding new column (nullable)
- Adding new table
- Adding new index
- Removing NOT NULL constraint
- Expanding column type (non-lossy)
- Adding default value

### 2.3 Version Format Validation

```go
package migrate

import (
    "regexp"
    "strings"
)

var versionRegex = regexp.MustCompile(`^(\d{14})_([a-z_]+)$`)

func ValidateVersion(version string) error {
    if !versionRegex.MatchString(version) {
        return &ValidationError{
            Message: "Invalid migration version format",
            Format:  "YYYYMMDDHHMMSS_name",
        }
    }

    parts := versionRegex.FindStringSubmatch(version)
    timestamp := parts[1]
    name := parts[2]

    // Validate timestamp
    if _, err := time.Parse("20060102150405", timestamp); err != nil {
        return &ValidationError{
            Message: "Invalid timestamp in version",
            Value:   timestamp,
        }
    }

    // Validate name format
    if strings.HasPrefix(name, "_") || strings.HasSuffix(name, "_") {
        return &ValidationError{
            Message: "Migration name cannot start or end with underscore",
            Value:   name,
        }
    }

    return nil
}
```

---

## 3. Migration Interface Definition

### 3.1 Core Migration Interface

```go
package migrate

import (
    "context"
    "database/sql"
)

// Migration represents a single database migration
type Migration struct {
    Version      string
    Name         string
    Dependencies []string
    Up           func(ctx context.Context, db *sql.DB) error
    Down         func(ctx context.Context, db *sql.DB) error
    Checksum     string
}

// MigrationRunner manages migration execution
type MigrationRunner interface {
    Register(migration *Migration) error
    Up(ctx context.Context, targetVersion string) error
    Down(ctx context.Context, targetVersion string) error
    Status(ctx context.Context) (*MigrationStatus, error)
    Validate(ctx context.Context) error
}

// MigrationStatus represents the current migration state
type MigrationStatus struct {
    CurrentVersion string
    AppliedCount  int
    PendingCount  int
    Applied       []MigrationInfo
    Pending       []MigrationInfo
}

// MigrationInfo provides details about a migration
type MigrationInfo struct {
    Version       string
    Name          string
    AppliedAt     *time.Time
    ExecutionTime *time.Duration
    Success       bool
    Checksum      string
}
```

### 3.2 Migration Registry

```go
package migrate

type Registry struct {
    migrations []*Migration
    byVersion  map[string]*Migration
}

func NewRegistry() *Registry {
    return &Registry{
        migrations: make([]*Migration, 0),
        byVersion:  make(map[string]*Migration),
    }
}

func (r *Registry) Register(migration *Migration) error {
    // Validate version format
    if err := ValidateVersion(migration.Version); err != nil {
        return err
    }

    // Check for duplicates
    if _, exists := r.byVersion[migration.Version]; exists {
        return &MigrationError{
            Message: "Migration version already registered",
            Version: migration.Version,
        }
    }

    // Calculate checksum
    migration.Checksum = calculateChecksum(migration)

    r.migrations = append(r.migrations, migration)
    r.byVersion[migration.Version] = migration

    return nil
}

func (r *Registry) Get(version string) (*Migration, bool) {
    m, ok := r.byVersion[version]
    return m, ok
}

func (r *Registry) List() []*Migration {
    return r.migrations
}

func (r *Registry) Sorted() []*Migration {
    sorted := make([]*Migration, len(r.migrations))
    copy(sorted, r.migrations)

    // Sort by version (timestamp is sortable)
    sort.Slice(sorted, func(i, j int) bool {
        return sorted[i].Version < sorted[j].Version
    })

    return sorted
}

// resolveDependencies returns migrations in execution order
func (r *Registry) resolveDependencies(target string) ([]*Migration, error) {
    result := make([]*Migration, 0)
    visited := make(map[string]bool)
    visiting := make(map[string]bool)

    var visit func(version string) error
    visit = func(version string) error {
        migration, ok := r.byVersion[version]
        if !ok {
            return &MigrationError{
                Message: "Migration not found",
                Version: version,
            }
        }

        // Check for cycles
        if visiting[version] {
            return &MigrationError{
                Message: "Circular dependency detected",
                Version: version,
            }
        }

        // Already visited
        if visited[version] {
            return nil
        }

        // Visit dependencies first
        visiting[version] = true
        for _, dep := range migration.Dependencies {
            if err := visit(dep); err != nil {
                return err
            }
        }
        visiting[version] = false

        // Add this migration
        visited[version] = true
        result = append(result, migration)

        return nil
    }

    // Visit target and all its dependencies
    if err := visit(target); err != nil {
        return nil, err
    }

    return result, nil
}
```

---

## 4. Migration File Structure

### 4.1 Naming Conventions

**Format:** `YYYYMMDDHHMMSS_description.go`

**Examples:**
- `20240120080000_initial_schema.go`
- `20240120100000_add_device_tags.go`
- `20240120200000_create_discovery_tables.go`

**Naming Guidelines:**
- Use lowercase with underscores
- Be descriptive but concise
- Use verbs: `add_`, `create_`, `alter_`, `drop_`
- For multiple changes: `update_feature_name`

### 4.2 Package Structure

```
internal/storage/migrations/
├── 20240120080000_initial_schema.go
├── 20240120100000_add_device_tags.go
├── 20240120200000_create_discovery_tables.go
├── 20240120300000_add_network_pools.go
└── registry.go
```

### 4.3 File Template

```go
//go:build ignore

package main

import (
    "database/sql"
    "log"

    _ "modernc.org/sqlite"

    "github.com/martinsuchenak/rackd/internal/storage/migrate"
)

var Migration = &migrate.Migration{
    Version: "20240120080000",
    Name:    "initial_schema",
    Up:      up,
    Down:    down,
}

func up(ctx context.Context, db *sql.DB) error {
    log.Info("Running migration: initial_schema")

    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Create datacenters table
    if _, err := tx.Exec(`
        CREATE TABLE datacenters (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            location TEXT,
            description TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `); err != nil {
        return err
    }

    // Create networks table
    if _, err := tx.Exec(`
        CREATE TABLE networks (
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
        return err
    }

    // ... more tables

    return tx.Commit()
}

func down(ctx context.Context, db *sql.DB) error {
    log.Info("Rolling back migration: initial_schema")

    // Drop tables in reverse order of dependencies
    tables := []string{
        "device_relationships",
        "addresses",
        "tags",
        "devices",
        "networks",
        "datacenters",
    }

    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for _, table := range tables {
        if _, err := tx.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

---

## 5. Migration Lifecycle

### 5.1 Creation Workflow

**Step 1: Generate Migration File**

```bash
rackd migration create add_device_location
```

**Step 2: Write Migration**

Edit generated file to implement `Up` and `Down` functions.

**Step 3: Test Migration**

```bash
rackd migration up --dry-run
```

**Step 4: Apply Migration**

```bash
rackd migration up
```

### 5.2 Testing Procedures

**Dry Run:**

```go
func (r *Runner) dryRunUp(ctx context.Context, migration *Migration) error {
    // Get SQL that would be executed
    sqlStatements, err := r.extractSQL(migration.Up)
    if err != nil {
        return err
    }

    // Validate SQL syntax
    for _, stmt := range sqlStatements {
        if err := r.validateSQL(stmt); err != nil {
            return fmt.Errorf("invalid SQL: %w", err)
        }
    }

    // Display SQL without executing
    fmt.Println("Would execute:")
    for _, stmt := range sqlStatements {
        fmt.Println(stmt)
        fmt.Println("--")
    }

    return nil
}
```

**Validation:**

```go
func (r *Runner) validateSQL(sql string) error {
    // Check for dangerous operations
    dangerous := []string{
        "DROP DATABASE",
        "DROP SCHEMA",
        "TRUNCATE",
    }

    upper := strings.ToUpper(sql)
    for _, danger := range dangerous {
        if strings.Contains(upper, danger) {
            return &MigrationError{
                Message: "Dangerous SQL operation",
                SQL:     sql,
            }
        }
    }

    return nil
}
```

### 5.3 Deployment Process

**Production Migration Steps:**

1. **Pre-migration checklist:**
   - [ ] Backup database
   - [ ] Review migration code
   - [ ] Test in staging environment
   - [ ] Verify no running discovery scans
   - [ ] Notify users of potential downtime

2. **Execute migration:**
   ```bash
   rackd migration up
   ```

3. **Post-migration verification:**
   - [ ] Verify schema version
   - [ ] Run data validation queries
   - [ ] Test API endpoints
   - [ ] Check application logs

### 5.4 Rollback Procedures

**Automatic Rollback on Error:**

```go
func (r *Runner) Up(ctx context.Context, targetVersion string) error {
    migrations := r.registry.Sorted()
    applied := r.getAppliedMigrations()

    for _, migration := range migrations {
        if migration.Version > targetVersion {
            break
        }

        if _, alreadyApplied := applied[migration.Version]; alreadyApplied {
            continue
        }

        log.Info("Applying migration", "version", migration.Version, "name", migration.Name)

        start := time.Now()
        err := migration.Up(ctx, r.db)
        duration := time.Since(start)

        if err != nil {
            log.Error("Migration failed", "version", migration.Version, "error", err)

            // Record failure
            r.recordMigration(migration, duration, false)

            // Attempt rollback
            if err := r.rollbackPrevious(ctx, migration.Version); err != nil {
                log.Error("Rollback failed", "error", err)
            }

            return &MigrationError{
                Message: "Migration failed and rolled back",
                Version: migration.Version,
                Err:     err,
            }
        }

        // Record success
        r.recordMigration(migration, duration, true)
        log.Info("Migration applied", "version", migration.Version, "duration", duration)
    }

    return nil
}

func (r *Runner) rollbackPrevious(ctx context.Context, currentVersion string) error {
    applied := r.getAppliedMigrations()
    migrations := r.registry.Sorted()

    // Rollback migrations in reverse order
    for i := len(migrations) - 1; i >= 0; i-- {
        migration := migrations[i]

        if migration.Version >= currentVersion {
            break
        }

        if _, wasApplied := applied[migration.Version]; wasApplied {
            log.Warn("Rolling back migration", "version", migration.Version)

            if err := migration.Down(ctx, r.db); err != nil {
                log.Error("Rollback failed", "version", migration.Version, "error", err)
                continue
            }

            r.deleteMigrationRecord(migration.Version)
        }
    }

    return nil
}
```

---

## 6. Migration Implementation Patterns

### 6.1 SQLite Migration Implementation

```go
package storage

import (
    "context"
    "database/sql"
)

type SQLiteStorage struct {
    db *sql.DB
    runner migrate.MigrationRunner
}

func (s *SQLiteStorage) Migrate(ctx context.Context) error {
    return s.runner.Up(ctx, "")
}

func (s *SQLiteStorage) RegisterMigrations() error {
    registry := migrate.NewRegistry()

    // Register all migrations
    registry.Register(migrations.InitialSchema)
    registry.Register(migrations.AddDeviceTags)
    registry.Register(migrations.CreateDiscoveryTables)

    s.runner = migrate.NewSQLiteRunner(s.db, registry)
    return nil
}

func NewSQLiteRunner(db *sql.DB, registry *Registry) MigrationRunner {
    return &SQLiteRunner{
        db:       db,
        registry:  registry,
    }
}

type SQLiteRunner struct {
    db      *sql.DB
    registry *Registry
}

func (r *SQLiteRunner) Up(ctx context.Context, targetVersion string) error {
    // Create migration table if not exists
    if err := r.createMigrationTable(ctx); err != nil {
        return err
    }

    // Get applied migrations
    applied, err := r.getAppliedMigrations(ctx)
    if err != nil {
        return err
    }

    // Get migrations to apply
    toApply := r.getMigrationsToApply(applied, targetVersion)

    // Apply migrations in transaction
    for _, migration := range toApply {
        if err := r.applyMigration(ctx, migration); err != nil {
            return err
        }
    }

    return nil
}

func (r *SQLiteRunner) applyMigration(ctx context.Context, migration *Migration) error {
    log.Info("Applying migration", "version", migration.Version)

    start := time.Now()
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Execute migration
    if err := migration.Up(ctx, r.db); err != nil {
        return err
    }

    // Record migration
    duration := time.Since(start)
    if err := r.recordMigration(ctx, tx, migration, duration, true); err != nil {
        return err
    }

    return tx.Commit()
}

func (r *SQLiteRunner) recordMigration(ctx context.Context, tx *sql.Tx, migration *Migration, duration time.Duration, success bool) error {
    _, err := tx.ExecContext(ctx, `
        INSERT INTO schema_migrations (version, name, applied_at, execution_time_ms, success, checksum)
        VALUES (?, ?, ?, ?, ?, ?)
    `,
        migration.Version,
        migration.Name,
        time.Now().UTC(),
        duration.Milliseconds(),
        success,
        migration.Checksum,
    )
    return err
}

func (r *SQLiteRunner) createMigrationTable(ctx context.Context) error {
    _, err := r.db.ExecContext(ctx, `
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
```

### 6.2 Postgres Migration Implementation

```go
package postgres

import (
    "context"
    "database/sql"

    "github.com/martinsuchenak/rackd/internal/storage/migrate"
)

type PostgresRunner struct {
    db       *sql.DB
    registry *Registry
}

func NewPostgresRunner(db *sql.DB, registry *Registry) MigrationRunner {
    return &PostgresRunner{
        db:       db,
        registry:  registry,
    }
}

func (r *PostgresRunner) Up(ctx context.Context, targetVersion string) error {
    // Create migration table if not exists
    if err := r.createMigrationTable(ctx); err != nil {
        return err
    }

    // Use advisory lock to prevent concurrent migrations
    lockID := 123456789 // Unique lock ID for migrations
    if err := r.acquireLock(ctx, lockID); err != nil {
        return fmt.Errorf("migration in progress: %w", err)
    }
    defer r.releaseLock(ctx, lockID)

    // Get applied migrations
    applied, err := r.getAppliedMigrations(ctx)
    if err != nil {
        return err
    }

    // Get migrations to apply
    toApply := r.getMigrationsToApply(applied, targetVersion)

    // Apply migrations
    for _, migration := range toApply {
        if err := r.applyMigration(ctx, migration); err != nil {
            return err
        }
    }

    return nil
}

func (r *PostgresRunner) createMigrationTable(ctx context.Context) error {
    _, err := r.db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            checksum TEXT NOT NULL,
            execution_time_ms INTEGER,
            success INTEGER NOT NULL DEFAULT TRUE
        )
    `)
    return err
}

func (r *PostgresRunner) acquireLock(ctx context.Context, lockID int64) error {
    _, err := r.db.ExecContext(ctx, `
        SELECT pg_advisory_lock($1)
    `, lockID)
    return err
}

func (r *PostgresRunner) releaseLock(ctx context.Context, lockID int64) error {
    _, err := r.db.ExecContext(ctx, `
        SELECT pg_advisory_unlock($1)
    `, lockID)
    return err
}
```

### 6.3 Data Migration Patterns

**Adding New Column with Default:**

```go
func up(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Add nullable column first
    if _, err := tx.Exec(`ALTER TABLE devices ADD COLUMN location TEXT`); err != nil {
        return err
    }

    // Backfill data for existing rows
    if _, err := tx.Exec(`
        UPDATE devices
        SET location = 'unknown'
        WHERE location IS NULL
    `); err != nil {
        return err
    }

    // Add NOT NULL constraint
    if _, err := tx.Exec(`ALTER TABLE devices ALTER COLUMN location SET NOT NULL`); err != nil {
        return err
    }

    return tx.Commit()
}
```

**Renaming Column:**

```go
func up(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // SQLite doesn't support ALTER COLUMN directly
    // Use ALTER TABLE with RENAME
    if _, err := tx.Exec(`ALTER TABLE devices RENAME COLUMN make_model TO make`); err != nil {
        return err
    }

    return tx.Commit()
}
```

### 6.4 Index Creation/Modification

**Creating Index:**

```go
func up(ctx context.Context, db *sql.DB) error {
    // Create index for device names
    _, err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_devices_name ON devices(name)
    `)
    return err
}

func down(ctx context.Context, db *sql.DB) error {
    _, err := db.Exec(`DROP INDEX IF EXISTS idx_devices_name`)
    return err
}
```

**Composite Index:**

```go
func up(ctx context.Context, db *sql.DB) error {
    // Create composite index for queries filtering by datacenter and name
    _, err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_devices_datacenter_name
        ON devices(datacenter_id, name)
    `)
    return err
}
```

---

## 7. Migration CLI Commands

### 7.1 List Migrations

```bash
rackd migration list
```

**Output:**

```
Migration Status
===============
Current Version: 20240120200000 (v1.2.0)
Applied: 4 migrations
Pending: 2 migrations

Applied Migrations:
- 20240120080000  initial_schema                   v1.0.0    2024-01-20 08:00:15    120ms
- 20240120100000  add_device_tags                  v1.0.0    2024-01-20 10:00:22    45ms
- 20240120200000  create_discovery_tables        v1.2.0    2024-01-20 12:00:10    280ms
- 20240120300000  add_network_pools                v1.3.0    2024-01-20 13:00:05    95ms

Pending Migrations:
- 20240120400000  add_device_relationships          v1.4.0
- 20240120500000  add_discovery_services            v1.5.0
```

### 7.2 Apply Migrations

```bash
# Apply all pending migrations
rackd migration up

# Apply up to specific version
rackd migration up 20240120400000

# Dry run - don't execute, just show SQL
rackd migration up --dry-run

# Force re-apply specific migration
rackd migration up --force 20240120100000
```

### 7.3 Rollback Migrations

```bash
# Rollback to previous version
rackd migration down

# Rollback to specific version
rackd migration down 20240120100000

# Dry run rollback
rackd migration down --dry-run
```

### 7.4 Create Migration

```bash
rackd migration create add_device_status

# Output:
# Created: internal/storage/migrations/20240120120000_add_device_status.go
# Please edit the file to implement Up and Down functions
```

### 7.5 Migration Status

```bash
rackd migration status

# Output:
# Database Version: 20240120200000
# Schema Version: v1.2.0
# Migrations Applied: 4
# Migrations Pending: 2
# Database: /data/rackd.db
```

---

## 8. Testing Strategies

### 8.1 Up/Down Migration Testing

```go
package migrations_test

func TestMigrationUpAndDown(t *testing.T) {
    db := setupTestDB(t)

    // Apply up migration
    ctx := context.Background()
    if err := migration.Up(ctx, db); err != nil {
        t.Fatalf("Migration up failed: %v", err)
    }

    // Verify schema changes
    var tableName string
    err := db.QueryRow(`
        SELECT name FROM sqlite_master WHERE type='table' AND name='devices'
    `).Scan(&tableName)
    if err != nil {
        t.Fatalf("Table not created: %v", err)
    }

    // Apply down migration
    if err := migration.Down(ctx, db); err != nil {
        t.Fatalf("Migration down failed: %v", err)
    }

    // Verify rollback
    err = db.QueryRow(`
        SELECT name FROM sqlite_master WHERE type='table' AND name='devices'
    `).Scan(&tableName)
    if err == nil {
        t.Error("Table should not exist after down migration")
    }
}
```

### 8.2 Data Migration Testing

```go
func TestMigrationPreservesData(t *testing.T) {
    db := setupTestDB(t)

    // Create test data before migration
    testDevice := model.Device{
        Name: "test-device",
        OS:   "Ubuntu 22.04",
    }
    // ... insert device

    // Run migration
    ctx := context.Background()
    if err := migration.Up(ctx, db); err != nil {
        t.Fatalf("Migration failed: %v", err)
    }

    // Verify data is preserved
    var retrieved model.Device
    err := db.QueryRow(`
        SELECT id, name, os FROM devices WHERE id = ?
    `, testDevice.ID).Scan(&retrieved.ID, &retrieved.Name, &retrieved.OS)

    if err != nil {
        t.Fatalf("Failed to retrieve device after migration: %v", err)
    }

    if retrieved.Name != testDevice.Name {
        t.Errorf("Name changed: expected %s, got %s", testDevice.Name, retrieved.Name)
    }
}
```

### 8.3 Performance Testing

```go
func BenchmarkMigrationUp(b *testing.B) {
    db := setupTestDB(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        // Clean database
        resetDB(b, db)

        // Run migration
        ctx := context.Background()
        if err := migration.Up(ctx, db); err != nil {
            b.Fatalf("Migration failed: %v", err)
        }
    }
}
```

### 8.4 Parallel Migration Safety

```go
func TestConcurrentMigrationSafety(t *testing.T) {
    // SQLite doesn't support concurrent writes
    // Postgres uses advisory locks
    // This test verifies locking mechanism

    if testing.Short() {
        t.Skip("Skipping in short mode")
    }

    db := setupPostgresTestDB(t)
    ctx := context.Background()

    errors := make(chan error, 2)

    // Try to run migration concurrently
    for i := 0; i < 2; i++ {
        go func() {
            errors <- runner.Up(ctx, "")
        }()
    }

    // One should succeed, one should fail
    successCount := 0
    for i := 0; i < 2; i++ {
        err := <-errors
        if err == nil {
            successCount++
        }
    }

    if successCount != 1 {
        t.Errorf("Expected 1 successful migration, got %d", successCount)
    }
}
```

---

## 9. Best Practices

### 9.1 Idempotent Migrations

```go
// Use CREATE IF NOT EXISTS
func up(ctx context.Context, db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS tags (
            device_id TEXT NOT NULL,
            tag TEXT NOT NULL,
            PRIMARY KEY (device_id, tag),
            FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
        )
    `)
    return err
}

// Use ALTER TABLE ... IF EXISTS (Postgres)
// or check table existence (SQLite)
func up(ctx context.Context, db *sql.DB) error {
    var exists bool
    err := db.QueryRow(`
        SELECT COUNT(*) FROM sqlite_master
        WHERE type='table' AND name='tags'
    `).Scan(&exists)
    if err != nil {
        return err
    }

    if !exists {
        // Create table
    }

    return nil
}
```

### 9.2 Backward Compatibility

```go
// Add column as nullable first
func up(ctx context.Context, db *sql.DB) error {
    _, err := db.Exec(`
        ALTER TABLE devices ADD COLUMN status TEXT DEFAULT 'unknown'
    `)
    return err
}

// Old code still works (new column has default)
// New code can use the new column
```

### 9.3 Data Preservation

```go
// Always use transactions for data changes
func up(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Create new table
    if _, err := tx.Exec(`CREATE TABLE devices_new (...)`); err != nil {
        return err
    }

    // Copy data
    if _, err := tx.Exec(`INSERT INTO devices_new SELECT * FROM devices`); err != nil {
        return err
    }

    // Drop old table
    if _, err := tx.Exec(`DROP TABLE devices`); err != nil {
        return err
    }

    // Rename new table
    if _, err := tx.Exec(`ALTER TABLE devices_new RENAME TO devices`); err != nil {
        return err
    }

    return tx.Commit()
}
```

### 9.4 Minimizing Downtime

**Strategies:**
1. Use online migrations (ALTER TABLE without locking)
2. For large tables, use incremental migrations:
   - Create new table
   - Copy data in batches
   - Switch over
   - Clean up old table
3. For critical migrations, use blue-green deployment

**Example: Incremental Data Migration**

```go
func up(ctx context.Context, db *sql.DB) error {
    // Phase 1: Add new nullable column
    if _, err := db.Exec(`ALTER TABLE devices ADD COLUMN status TEXT`); err != nil {
        return err
    }

    // Phase 2: Backfill data in batches (background job)
    // This can be done after deployment without downtime

    // Phase 3: Add NOT NULL constraint (after backfill complete)
    // if _, err := db.Exec(`ALTER TABLE devices ALTER COLUMN status SET NOT NULL`); err != nil {
    //     return err
    // }

    return nil
}
```
