# Upgrade and Migration

This document defines version upgrade paths, data migration procedures, and compatibility guidelines for Rackd.

## 1. Version Strategy

### 1.1 Semantic Versioning (MAJOR.MINOR.PATCH)

- **MAJOR**: Incompatible API changes, breaking database changes
- **MINOR**: Backward-compatible functionality, non-breaking schema changes
- **PATCH**: Backward-compatible bug fixes

**Examples:**
- v1.0.0 → v1.0.1 (Patch: Bug fix)
- v1.0.1 → v1.1.0 (Minor: New feature)
- v1.1.0 → v2.0.0 (Major: Breaking changes)

### 1.2 Pre-Release Versions

**Format:** `MAJOR.MINOR.PATCH-PRERELEASE`

**Types:**
- `alpha`: Early development, unstable
- `beta`: Feature complete, testing needed
- `rc`: Release candidate, ready for testing

**Examples:**
- v1.2.0-alpha.1
- v1.2.0-beta.1
- v1.2.0-rc.1

**Version Sorting:**

```go
package version

import (
    "strings"
)

type Version struct {
    Major      int
    Minor      int
    Patch      int
    PreRelease string
}

func Parse(v string) (*Version, error) {
    version := &Version{}

    // Split pre-release
    parts := strings.Split(v, "-")
    versionStr := parts[0]

    // Parse version numbers
    numbers := strings.Split(versionStr, ".")
    if len(numbers) != 3 {
        return nil, &VersionError{Message: "Invalid version format"}
    }

    major, err := strconv.Atoi(numbers[0])
    if err != nil {
        return nil, err
    }
    version.Major = major

    minor, err := strconv.Atoi(numbers[1])
    if err != nil {
        return nil, err
    }
    version.Minor = minor

    patch, err := strconv.Atoi(numbers[2])
    if err != nil {
        return nil, err
    }
    version.Patch = patch

    // Parse pre-release
    if len(parts) > 1 {
        version.PreRelease = parts[1]
    }

    return version, nil
}

func Compare(v1, v2 string) int {
    v1Parsed, err := Parse(v1)
    if err != nil {
        return -1
    }

    v2Parsed, err := Parse(v2)
    if err != nil {
        return 1
    }

    // Compare major
    if v1Parsed.Major != v2Parsed.Major {
        return v1Parsed.Major - v2Parsed.Major
    }

    // Compare minor
    if v1Parsed.Minor != v2Parsed.Minor {
        return v1Parsed.Minor - v2Parsed.Minor
    }

    // Compare patch
    if v1Parsed.Patch != v2Parsed.Patch {
        return v1Parsed.Patch - v2Parsed.Patch
    }

    // Pre-release handling
    if v1Parsed.PreRelease == "" && v2Parsed.PreRelease != "" {
        return 1 // Pre-release < release
    }
    if v1Parsed.PreRelease != "" && v2Parsed.PreRelease == "" {
        return -1
    }

    return 0
}
```

### 1.3 Build Metadata

**Format:** `VERSION+BUILDMETA`

**Examples:**
- v1.2.0+20240120100000.abcdef
- v1.2.0-beta.1+20240120150000.123456

**Components:**
- Version: SemVer version
- Build timestamp: YYYYMMDDHHMMSS
- Commit hash: First 7 characters of git commit

---

## 2. Upgrade Paths

### 2.1 Supported Upgrade Versions

| From Version | To Version | Supported | Notes |
|-------------|-------------|-----------|-------|
| v1.0.x | v1.1.x | ✅ | Direct upgrade |
| v1.1.x | v1.2.x | ✅ | Direct upgrade |
| v1.0.x | v1.2.x | ✅ | Direct upgrade |
| v1.0.x | v2.0.x | ❌ | Must upgrade to v1.2.x first |
| v1.1.x | v2.0.x | ❌ | Must upgrade to v1.2.x first |
| v1.2.x | v2.0.x | ✅ | Direct upgrade |
| v2.0.x | v2.1.x | ✅ | Direct upgrade |
| v2.0.x | v3.0.x | ❌ | Must upgrade to v2.1.x first |

### 2.2 Direct Upgrade Paths

```
v1.0.0 ──┬─> v1.0.1 ──┬─> v1.1.0 ──┬─> v1.2.0 ──┬─> v2.0.0 ──┬─> v3.0.0
           │             │              │              │              │              │
           └─────────────┴──────────────┴──────────────┴──────────────┘              │
                                                                         │
                                   Supported Upgrade Path                         │
                                                                         └────────────────────┘
```

### 2.3 Intermediate Version Requirements

**Policy:** Never skip more than one major version.

**Example:**

```bash
# Bad: v1.0.x → v3.0.x (unsupported)
rackd upgrade v3.0.0
# Error: Cannot upgrade from v1.0.5 to v3.0.0. Please upgrade to v1.2.x first.

# Good: v1.0.x → v1.2.x → v3.0.x (supported)
rackd upgrade v1.2.0
rackd upgrade v3.0.0
```

**Validation Logic:**

```go
func ValidateUpgradePath(from, to string) error {
    fromVer, err := Parse(from)
    if err != nil {
        return err
    }

    toVer, err := Parse(to)
    if err != nil {
        return err
    }

    // Cannot downgrade
    if Compare(to, from) < 0 {
        return &ValidationError{
            Message: "Cannot downgrade",
            From:    from,
            To:      to,
        }
    }

    // Check for major version skips
    majorDiff := toVer.Major - fromVer.Major
    if majorDiff > 1 {
        return &ValidationError{
            Message: "Cannot skip major versions",
            From:    from,
            To:      to,
            Required: fmt.Sprintf("v%d.0.0", fromVer.Major+1),
        }
    }

    // Check compatibility matrix
    if !isCompatibleUpgrade(from, to) {
        return &ValidationError{
            Message: "Unsupported upgrade path",
            From:    from,
            To:      to,
        }
    }

    return nil
}
```

### 2.4 Upgrade Path Matrix Table

```markdown
| Current | Supported Upgrades | Recommended |
|---------|------------------|-------------|
| v1.0.0 | v1.0.1, v1.0.2 | v1.2.0 |
| v1.0.1 | v1.0.2, v1.1.0 | v1.2.0 |
| v1.0.2 | v1.1.0 | v1.2.0 |
| v1.1.0 | v1.2.0 | v1.2.0 |
| v1.1.1 | v1.2.0 | v1.2.0 |
| v1.2.0 | v2.0.0 | v2.0.0 |
| v1.2.1 | v2.0.0 | v2.0.0 |
| v2.0.0 | v2.1.0, v2.0.1 | v2.1.0 |
| v2.1.0 | v3.0.0 | v3.0.0 |
```

---

## 3. Data Migration

### 3.1 Schema Migration Handling

**Migration Lifecycle:**

1. **Pre-migration Check**
   - Verify current schema version
   - Validate upgrade path
   - Check database integrity

2. **Migration Execution**
   - Apply schema changes
   - Migrate data
   - Update schema version

3. **Post-migration Validation**
   - Verify data integrity
   - Run data validation queries
   - Compare row counts

4. **Migration Rollback (on failure)**
   - Rollback schema changes
   - Restore original data
   - Log rollback details

**Example:**

```go
func (s *SQLiteStorage) MigrateToVersion(targetVersion string) error {
    // Check current version
    currentVersion, err := s.GetCurrentSchemaVersion()
    if err != nil {
        return err
    }

    // Validate upgrade path
    if err := ValidateUpgradePath(currentVersion, targetVersion); err != nil {
        return err
    }

    // Get migrations to apply
    migrations := s.getMigrations(currentVersion, targetVersion)

    // Apply migrations in transaction
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for _, migration := range migrations {
        log.Info("Applying migration", "version", migration.Version, "name", migration.Name)

        if err := migration.Up(context.Background(), s.db); err != nil {
            log.Error("Migration failed", "version", migration.Version, "error", err)

            // Attempt rollback
            if rbErr := s.rollbackMigrations(tx, currentVersion, migration.Version); rbErr != nil {
                log.Error("Rollback failed", "error", rbErr)
            }

            return fmt.Errorf("migration failed: %w", err)
        }

        log.Info("Migration applied", "version", migration.Version)
    }

    return tx.Commit()
}
```

### 3.2 Data Transformation Requirements

**Column Type Changes:**

```go
// Migration: Add device status column
func up(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Phase 1: Add nullable column
    if _, err := tx.Exec(`ALTER TABLE devices ADD COLUMN status TEXT`); err != nil {
        return err
    }

    // Phase 2: Backfill data
    if _, err := tx.Exec(`UPDATE devices SET status = 'active' WHERE status IS NULL`); err != nil {
        return err
    }

    // Phase 3: Add NOT NULL constraint (after backfill complete)
    // Note: SQLite doesn't support ADD CONSTRAINT, use trigger or recreate table

    return tx.Commit()
}
```

**Table Restructuring:**

```go
// Migration: Split device name into first/last name
func up(ctx context.Context, db *sql.DB) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Phase 1: Add new columns
    if _, err := tx.Exec(`ALTER TABLE devices ADD COLUMN first_name TEXT`); err != nil {
        return err
    }
    if _, err := tx.Exec(`ALTER TABLE devices ADD COLUMN last_name TEXT`); err != nil {
        return err
    }

    // Phase 2: Migrate data
    if _, err := tx.Exec(`
        UPDATE devices
        SET
            first_name = substr(name, 1, instr(name, ' ') - 1),
            last_name = substr(name, instr(name, ' ') + 1)
    `); err != nil {
        return err
    }

    // Phase 3: Add NOT NULL constraints after backfill
    // (implementation depends on database)

    return tx.Commit()
}
```

### 3.3 Migration Testing Procedures

**Test Template:**

```go
package migrations_test

func TestMigrationDataPreservation(t *testing.T) {
    db := setupTestDB(t)

    // Create test data
    testDevice := &Device{
        ID:   "test-device-1",
        Name:  "Test Device",
        OS:    "Ubuntu 22.04",
    }
    if err := CreateDevice(db, testDevice); err != nil {
        t.Fatal(err)
    }

    // Run migration
    if err := Migration20240120.Up(context.Background(), db); err != nil {
        t.Fatal(err)
    }

    // Verify data preserved
    var retrieved Device
    if err := db.QueryRow(`SELECT * FROM devices WHERE id = ?`, testDevice.ID).Scan(&retrieved); err != nil {
        t.Fatal(err)
    }

    if retrieved.Name != testDevice.Name {
        t.Errorf("Name not preserved: expected %s, got %s", testDevice.Name, retrieved.Name)
    }

    if retrieved.OS != testDevice.OS {
        t.Errorf("OS not preserved: expected %s, got %s", testDevice.OS, retrieved.OS)
    }
}

func TestMigrationRollback(t *testing.T) {
    db := setupTestDB(t)

    // Run migration up
    if err := Migration20240120.Up(context.Background(), db); err != nil {
        t.Fatal(err)
    }

    // Verify migration applied
    if !migrationApplied(db, "20240120") {
        t.Error("Migration not marked as applied")
    }

    // Run migration down
    if err := Migration20240120.Down(context.Background(), db); err != nil {
        t.Fatal(err)
    }

    // Verify migration rolled back
    if migrationApplied(db, "20240120") {
        t.Error("Migration not rolled back")
    }

    // Verify schema restored
    if columnExists(db, "devices", "new_column") {
        t.Error("New column still exists after rollback")
    }
}
```

### 3.4 Data Validation Checks

**Pre-Migration Validation:**

```sql
-- Check for data issues before migration
SELECT COUNT(*) FROM devices WHERE name IS NULL OR name = '';

SELECT COUNT(*) FROM devices WHERE id NOT GLOB '[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]-*';

-- Check foreign key integrity
SELECT COUNT(*) FROM devices
WHERE datacenter_id IS NOT NULL
  AND datacenter_id NOT IN (SELECT id FROM datacenters);
```

**Post-Migration Validation:**

```sql
-- Verify row counts match
SELECT 'devices' as table, COUNT(*) as count FROM devices
UNION ALL
SELECT 'networks' as table, COUNT(*) as count FROM networks
UNION ALL
SELECT 'datacenters' as table, COUNT(*) as count FROM datacenters;

-- Verify no NULL in required columns
SELECT COUNT(*) FROM devices WHERE name IS NULL;
SELECT COUNT(*) FROM networks WHERE subnet IS NULL;

-- Verify foreign key constraints
PRAGMA foreign_key_check;
```

---

## 4. Breaking Changes

### 4.1 Breaking Change Classification

**Category 1: Database Schema Changes**

- Column removal
- Column rename
- Data type change (lossy)
- NOT NULL constraint added to existing column
- Primary key change
- Foreign key constraint change

**Category 2: API Changes**

- Endpoint removal
- Endpoint path change
- Request/response structure change
- Required parameter added
- Enum value change
- Authentication change

**Category 3: Configuration Changes**

- Configuration option removal
- Configuration option rename
- Default value change (significant)
- Configuration file format change

### 4.2 Breaking Change Documentation Template

```markdown
# Breaking Change: [Brief Title]

## Version
- **Affected Version:** v1.2.0
- **Introduced In:** v2.0.0
- **Migration Guide:** [Link]

## Description
[Detailed description of the change and why it's breaking]

## Impact

### API Changes
| Endpoint | Old Behavior | New Behavior | Migration Required |
|----------|--------------|--------------|------------------|
| POST /api/devices | Field X required | Field X optional | Update client code |
| GET /api/devices/{id} | Returns field Y | Field Y removed | Remove field Y usage |

### Database Changes
| Table | Change | Migration Required |
|-------|---------|------------------|
| devices | Column X renamed to Y | Update queries |
| networks | Column Z type changed | Update queries |

### Configuration Changes
| Option | Old Value | New Value | Migration Required |
|--------|-----------|-----------|------------------|
| discovery.timeout | 30s | discovery.scan_timeout | Update config |

## Migration Steps

### Step 1: [Title]
[Detailed instructions]

### Step 2: [Title]
[Detailed instructions]

### Step 3: [Title]
[Detailed instructions]

## Rollback Plan
[Instructions for rolling back to previous version]

## Testing
[Testing recommendations]

## Related Issues
- [Issue #1](url)
- [Issue #2](url)
```

### 4.3 Deprecation Notices

**Deprecation Timeline:**

| Version | Feature | Deprecation Notice | Removal Version | Timeline |
|---------|---------|-------------------|-----------------|----------|
| v1.2.0 | Field X in API | v1.2.0 | v2.0.0 | 1 major version |
| v2.0.0 | Configuration option Y | v2.0.0 | v2.1.0 | 1 minor version |
| v2.0.0 | Endpoint Z | v2.0.0 | v3.0.0 | 1 major version |

**Minimum Deprecation Policy:**
- API changes: 1 major version (12 months)
- Database changes: 1 major version (12 months)
- Configuration changes: 1 minor version (6 months)

**Deprecation Header:**

```http
HTTP/1.1 200 OK
Content-Type: application/json
X-Rackd-API-Deprecated: true
X-Rackd-API-Deprecation-Date: 2024-06-01
X-Rackd-API-Removal-Version: v2.0.0
X-Rackd-API-Removal-Date: 2025-01-01
```

### 4.4 Migration Guide Format

```markdown
# Migration Guide: v1.2.0 to v2.0.0

## Overview
This guide covers migrating from Rackd v1.2.0 to v2.0.0.

## Breaking Changes Summary

### 1. API Changes
- **Endpoint Removal:** `/api/legacy/endpoint` removed
- **Request Structure:** `POST /api/devices` now requires `category` field
- **Response Structure:** `Device` object no longer includes `legacy_field`

### 2. Database Changes
- **Column Rename:** `devices.make_model` → `devices.make` and `devices.model`
- **New Constraint:** `devices.name` must be unique

### 3. Configuration Changes
- **Option Removal:** `legacy_mode` configuration option removed

## Pre-Upgrade Checklist

- [ ] Backup current database
- [ ] Review breaking changes above
- [ ] Test upgrade in staging environment
- [ ] Plan downtime (if required)
- [ ] Notify users of scheduled maintenance

## Upgrade Procedure

### Option 1: Automatic Upgrade

```bash
rackd upgrade prepare v2.0.0
rackd upgrade execute v2.0.0
```

### Option 2: Manual Upgrade

1. Stop Rackd server
2. Backup database
3. Download v2.0.0 binary
4. Replace binary
5. Start Rackd server
6. Verify data integrity

## Data Migration Notes

### Column Renaming

The `make_model` column has been split into `make` and `model` columns.

**Automatic Migration:**
The upgrade process will automatically migrate existing data:
- Values containing both make and model will be split on space
- Values without space will go to `make` column
- `model` column will be NULL if not applicable

**Manual Review:**
After upgrade, review devices and update `make` and `model` fields as needed.

## Post-Upgrade Verification

### API Testing
```bash
# Test device creation with new field
curl -X POST http://localhost:8080/api/devices \
  -H "Content-Type: application/json" \
  -d '{"name":"test", "category":"server"}'

# Verify response structure
curl http://localhost:8080/api/devices | jq '.[0] | keys'
```

### Data Integrity
```bash
# Verify all devices have valid name
rackd db validate --table devices --column name

# Verify no NULL in new required fields
rackd db validate --table devices --column category
```

## Rollback Procedure

If issues are encountered after upgrade, follow these steps to rollback to v1.2.0:

1. Stop Rackd server
2. Restore database from pre-upgrade backup
3. Download v1.2.0 binary
4. Replace binary
5. Start Rackd server
6. Verify rollback successful

## Support

If you encounter issues during upgrade:
- Check troubleshooting guide: [Link]
- Review known issues: [Link]
- Create support ticket: [Link]

## Related Documentation

- [Breaking Changes Documentation](breaking-changes-v2.0.0.md)
- [API Reference](api-reference.md)
- [Troubleshooting](troubleshooting.md)
```

---

## 5. Upgrade Procedures

### 5.1 Pre-Upgrade Checklist

**Database:**
- [ ] Database backed up
- [ ] Database integrity verified
- [ ] Disk space available (at least 2x current size)
- [ ] No active long-running queries

**Application:**
- [ ] Review breaking changes
- [ ] Test in staging environment
- [ ] Configuration validated
- [ ] Dependencies compatible

**Operations:**
- [ ] Maintenance window scheduled
- [ ] Users notified
- [ ] Rollback plan prepared
- [ ] Support team notified

### 5.2 Backup Requirements

**Backup Types Required:**

1. **Full Database Backup**
   ```bash
   rackd backup create --type online --name pre_upgrade_backup
   ```

2. **Configuration Backup**
   ```bash
   cp /etc/rackd/config.yaml /backups/config.yaml.backup
   ```

3. **Backup Verification**
   ```bash
   rackd backup verify pre_upgrade_backup
   ```

### 5.3 Step-by-Step Upgrade Process

**Automated Upgrade:**

```bash
# 1. Check upgrade compatibility
rackd upgrade check --version v2.0.0

# 2. Prepare upgrade
rackd upgrade prepare v2.0.0

# 3. Verify changes
rackd upgrade verify v2.0.0

# 4. Execute upgrade
rackd upgrade execute v2.0.0

# 5. Post-upgrade validation
rackd upgrade validate
```

**Manual Upgrade:**

```bash
# 1. Stop server
rackd server stop

# 2. Backup
rackd backup create --type online --name manual_backup

# 3. Download new version
wget https://github.com/martinsuchenak/rackd/releases/download/v2.0.0/rackd_2.0.0_linux_amd64

# 4. Replace binary
chmod +x rackd_2.0.0_linux_amd64
mv rackd_2.0.0_linux_amd64 /usr/local/bin/rackd

# 5. Start server
rackd server start

# 6. Verify upgrade
rackd version
rackd doctor validate
```

### 5.4 Post-Upgrade Verification

**Data Validation:**

```sql
-- Verify all devices exist
SELECT COUNT(*) FROM devices;

-- Verify no orphaned records
SELECT COUNT(*) FROM addresses WHERE device_id NOT IN (SELECT id FROM devices);

-- Verify foreign key integrity
PRAGMA foreign_key_check;

-- Verify indexes
SELECT * FROM pragma_index_list('devices');
```

**API Validation:**

```bash
# Test basic API endpoints
curl -f http://localhost:8080/api/datacenters || echo "Datacenters endpoint failed"
curl -f http://localhost:8080/api/devices || echo "Devices endpoint failed"

# Test authentication (if configured)
curl -f -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/devices || echo "Auth failed"

# Test device operations
curl -X POST http://localhost:8080/api/devices -H "Content-Type: application/json" \
  -d '{"name":"test-upgrade-device"}' || echo "Device creation failed"
```

**Performance Validation:**

```bash
# Run performance tests
rackd benchmark --duration 60s

# Check response times
curl -w "@curl-format.txt" http://localhost:8080/api/devices

# Verify no memory leaks
rackd debug profile memory --duration 5m
```

### 5.5 Rollback Triggers

**Immediate Rollback:**
- Upgrade process fails
- Data corruption detected
- Critical API errors
- Application won't start

**Planned Rollback:**
- Performance degradation > 50%
- Error rate > 10%
- Data inconsistencies found
- Business critical issues

### 5.6 Rollback Procedures

**Binary Rollback:**

```bash
# 1. Stop server
rackd server stop

# 2. Restore previous binary
mv /usr/local/bin/rackd /usr/local/bin/rackd.new
mv /usr/local/bin/rackd.backup /usr/local/bin/rackd

# 3. Restore configuration (if changed)
cp /backups/config.yaml.backup /etc/rackd/config.yaml

# 4. Start server
rackd server start

# 5. Verify rollback
rackd version
rackd doctor validate
```

**Database Rollback:**

```bash
# 1. Stop server
rackd server stop

# 2. Restore database backup
rackd backup restore pre_upgrade_backup

# 3. Verify restore
rackd backup verify pre_upgrade_backup

# 4. Start server
rackd server start

# 5. Validate data
rackd doctor validate
```

**Data Recovery (Partial Failure):**

```bash
# If upgrade completed but data is corrupted
rackd doctor repair --table devices

# Export data before full restore
rackd export devices --format csv --output devices_backup.csv

# Restore database
rackd backup restore pre_upgrade_backup

# Re-import new data (if available)
rackd import devices --input devices_backup.csv
```

---

## 6. Upgrade CLI Commands

### 6.1 Upgrade Check

```bash
# Check if upgrade is available
rackd upgrade check

# Check specific version compatibility
rackd upgrade check --version v2.0.0

# Output:
# Current version: v1.2.0
# Latest version: v2.0.0
# Upgrade available: Yes
# Compatible: Yes
# Breaking changes: 3
# Estimated downtime: 5 minutes
```

### 6.2 Upgrade Prepare

```bash
# Prepare for upgrade
rackd upgrade prepare v2.0.0

# Output:
# Downloading v2.0.0...
# Download complete: rackd_2.0.0_linux_amd64 (15 MB)
# Creating backup: pre_upgrade_backup_20240120120000
# Backup complete
# Reading breaking changes...
# 3 breaking changes found
# See: /tmp/rackd_upgrade_changes.txt
```

### 6.3 Upgrade Execute

```bash
# Execute upgrade
rackd upgrade execute v2.0.0

# Options:
rackd upgrade execute v2.0.0 \
  --no-backup \          # Skip backup creation
  --force \               # Skip confirmation
  --dry-run \            # Show what would be done
  --config /path/to/config.yaml

# Output:
# Stopping server...
# Server stopped
# Running database migrations...
# Applied 3 migrations
# Starting server...
# Server started
# Validating upgrade...
# Upgrade successful
```

### 6.4 Upgrade Verify

```bash
# Verify upgrade
rackd upgrade verify

# Output:
# Verifying API endpoints...
# ✓ GET /api/datacenters
# ✓ GET /api/devices
# Verifying database integrity...
# ✓ Foreign key constraints valid
# ✓ Indexes present
# Verifying data counts...
# ✓ Devices: 1234
# ✓ Networks: 56
# ✓ Datacenters: 3
# Upgrade verification successful
```

### 6.5 Upgrade Rollback

```bash
# Rollback to previous version
rackd upgrade rollback

# Rollback to specific version
rackd upgrade rollback --version v1.2.0

# Restore specific backup
rackd upgrade rollback --backup pre_upgrade_backup_20240120120000

# Output:
# Stopping server...
# Server stopped
# Restoring database...
# Database restored
# Restoring binary...
# Binary restored
# Starting server...
# Server started
# Rollback successful
```

---

## 7. Testing Procedures

### 7.1 Upgrade Testing in Staging

**Test Plan:**

1. **Environment Setup**
   ```bash
   # Clone staging data from production
   rackd backup restore production_backup --environment staging

   # Configure staging environment
   cp staging/config.yaml /etc/rackd/config.yaml
   ```

2. **Pre-Upgrade Verification**
   ```bash
   # Verify staging is working
   rackd doctor validate --environment staging

   # Record baseline metrics
   rackd metrics snapshot --name pre_upgrade
   ```

3. **Execute Upgrade**
   ```bash
   # Perform upgrade
   rackd upgrade execute v2.0.0

   # Verify upgrade
   rackd upgrade verify
   ```

4. **Post-Upgrade Testing**
   ```bash
   # Run API tests
   rackd test api --environment staging

   # Run data validation
   rackd doctor validate --environment staging

   # Compare metrics
   rackd metrics compare --baseline pre_upgrade
   ```

5. **Load Testing**
   ```bash
   # Simulate production load
   rackd benchmark --duration 1h --environment staging
   ```

### 7.2 Data Validation Testing

**Automated Validation:**

```go
func ValidateUpgrade(db *sql.DB) error {
    validations := []struct {
        Name string
        Check func(*sql.DB) error
    }{
        {"Row counts", validateRowCounts},
        {"Foreign keys", validateForeignKeys},
        {"Indexes", validateIndexes},
        {"Data integrity", validateDataIntegrity},
    }

    for _, validation := range validations {
        if err := validation.Check(db); err != nil {
            return fmt.Errorf("validation failed: %s: %w", validation.Name, err)
        }
        log.Info("Validation passed", "check", validation.Name)
    }

    return nil
}

func validateRowCounts(db *sql.DB) error {
    // Compare row counts before/after upgrade
    var deviceCount, networkCount, datacenterCount int
    db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&deviceCount)
    db.QueryRow("SELECT COUNT(*) FROM networks").Scan(&networkCount)
    db.QueryRow("SELECT COUNT(*) FROM datacenters").Scan(&datacenterCount)

    log.Info("Row counts", "devices", deviceCount, "networks", networkCount, "datacenters", datacenterCount)

    // Compare with baseline
    baseline := loadBaselineCounts()
    if deviceCount != baseline.Devices {
        return fmt.Errorf("device count mismatch: expected %d, got %d",
            baseline.Devices, deviceCount)
    }

    return nil
}
```

**Manual Validation Checklist:**

- [ ] Device count matches baseline
- [ ] Network count matches baseline
- [ ] Datacenter count matches baseline
- [ ] No orphaned addresses
- [ ] No orphaned tags
- [ ] Foreign key constraints valid
- [ ] All indexes present
- [ ] No NULL in required fields
- [ ] API endpoints responding
- [ ] Authentication working (if configured)

### 7.3 Performance Regression Testing

**Baseline Comparison:**

```bash
# Before upgrade
rackd benchmark --name baseline --duration 10m --output baseline.json

# After upgrade
rackd benchmark --name current --duration 10m --output current.json

# Compare
rackd benchmark compare baseline.json current.json

# Output:
# API GET /api/devices:
#   Baseline: p50=50ms, p95=100ms, p99=150ms
#   Current:  p50=60ms, p95=120ms, p99=180ms
#   Regression: +20% (WARNING)
```

**Regression Detection:**

```go
func DetectPerformanceRegression(baseline, current *BenchmarkResult) []*Regression {
    regressions := make([]*Regression, 0)

    for _, endpoint := range baseline.Endpoints {
        currentEndpoint := current.GetEndpoint(endpoint.Name)

        // Calculate regression
        p50Regression := (currentEndpoint.P50 - endpoint.P50) / endpoint.P50
        p95Regression := (currentEndpoint.P95 - endpoint.P95) / endpoint.P95
        p99Regression := (currentEndpoint.P99 - endpoint.P99) / endpoint.P99

        // Check threshold
        if p50Regression > 0.2 || p95Regression > 0.2 || p99Regression > 0.2 {
            regressions = append(regressions, &Regression{
                Endpoint:   endpoint.Name,
                P50:        p50Regression,
                P95:        p95Regression,
                P99:        p99Regression,
                Baseline:   endpoint,
                Current:    currentEndpoint,
            })
        }
    }

    return regressions
}
```

### 7.4 Smoke Testing Checklist

**Post-Upgrade Smoke Tests:**

```bash
#!/bin/bash
# smoke-test.sh

echo "Running post-upgrade smoke tests..."

# Test 1: Server is running
curl -f http://localhost:8080/api/datacenters || exit 1
echo "✓ Server is running"

# Test 2: Create device
curl -X POST http://localhost:8080/api/devices \
  -H "Content-Type: application/json" \
  -d '{"name":"smoke-test-device"}' \
  -f || exit 1
echo "✓ Device creation works"

# Test 3: List devices
curl -f http://localhost:8080/api/devices | jq '. | length > 0' || exit 1
echo "✓ Device listing works"

# Test 4: Get device
DEVICE_ID=$(curl -s http://localhost:8080/api/devices | jq -r '.[0].id')
curl -f http://localhost:8080/api/devices/$DEVICE_ID || exit 1
echo "✓ Device retrieval works"

# Test 5: Update device
curl -X PUT http://localhost:8080/api/devices/$DEVICE_ID \
  -H "Content-Type: application/json" \
  -d '{"name":"updated-smoke-test"}' \
  -f || exit 1
echo "✓ Device update works"

# Test 6: Delete device
curl -X DELETE http://localhost:8080/api/devices/$DEVICE_ID -f || exit 1
echo "✓ Device deletion works"

echo ""
echo "All smoke tests passed!"
```

---

## 8. Compatibility Notes

### 8.1 Go Version Compatibility

| Rackd Version | Go 1.24 | Go 1.25 | Go 1.26 | Notes |
|---------------|-----------|-----------|-----------|------|
| v1.0.0 | ✅ | ✅ | ✅ | Minimum Go 1.24 |
| v1.1.0 | ✅ | ✅ | ✅ | |
| v1.2.0 | ❌ | ✅ | ✅ | Requires Go 1.25+ |
| v2.0.0 | ❌ | ❌ | ✅ | Requires Go 1.26+ |

### 8.2 OS Compatibility Matrix

| OS | v1.0.x | v1.1.x | v1.2.x | v2.0.x |
|----|---------|---------|---------|---------|
| Ubuntu 20.04 | ✅ | ✅ | ✅ | ✅ |
| Ubuntu 22.04 | ✅ | ✅ | ✅ | ✅ |
| Ubuntu 24.04 | ✅ | ✅ | ✅ | ✅ |
| Debian 11 (Bullseye) | ✅ | ✅ | ✅ | ✅ |
| Debian 12 (Bookworm) | ✅ | ✅ | ✅ | ✅ |
| CentOS 8 | ✅ | ✅ | ✅ | ❌ |
| CentOS 9 Stream | ✅ | ✅ | ✅ | ✅ |
| Rocky Linux 9 | ✅ | ✅ | ✅ | ✅ |
| RHEL 8 | ✅ | ✅ | ✅ | ❌ |
| RHEL 9 | ✅ | ✅ | ✅ | ✅ |
| macOS 12 (Monterey) | ✅ | ✅ | ✅ | ✅ |
| macOS 13 (Ventura) | ✅ | ✅ | ✅ | ✅ |
| macOS 14 (Sonoma) | ✅ | ✅ | ✅ | ✅ |
| Windows Server 2019 | ✅ | ✅ | ✅ | ✅ |
| Windows Server 2022 | ✅ | ✅ | ✅ | ✅ |

### 8.3 Dependency Compatibility

| Dependency | v1.0.x | v1.1.x | v1.2.x | v2.0.x | Notes |
|-----------|---------|---------|---------|---------|------|
| modernc.org/sqlite | v1.40.x | v1.42.x | v1.42.x | v1.44.x | |
| paularlott/cli | v0.6.x | v0.6.x | v0.7.x | v0.8.x | |
| paularlott/logger | v0.2.x | v0.2.x | v0.3.x | v0.3.x | |
| paularlott/mcp | v0.8.x | v0.8.x | v0.9.x | v0.10.x | |

### 8.4 Database Version Compatibility

**SQLite:**

| Rackd Version | SQLite Version | Notes |
|---------------|----------------|------|
| v1.0.x - v1.2.x | 3.38.x+ | Requires WAL mode |
| v2.0.x+ | 3.40.x+ | Requires new features |

**Postgres (Enterprise):**

| Rackd Version | Postgres Version | Notes |
|---------------|------------------|------|
| v1.0.x - v1.2.x | 14.x+ | Full support |
| v2.0.x+ | 15.x+ | Requires new features |

### 8.5 Configuration Compatibility

**Deprecated Options:**

| Option | Deprecated In | Removed In | Replacement |
|--------|---------------|------------|-------------|
| `discovery.interval` | v1.2.0 | v2.0.0 | `discovery.scan_interval` |
| `api.token` | v1.2.0 | v2.0.0 | `api.auth_token` |
| `log.json` | v1.1.0 | v2.0.0 | `log.format=json` |

**New Options:**

| Option | Added In | Description |
|--------|----------|-------------|
| `discovery.scan_interval` | v1.2.0 | Interval between discovery scans |
| `api.auth_token` | v1.2.0 | API authentication token |
| `api.rate_limit` | v2.0.0 | API rate limiting configuration |
| `monitoring.enabled` | v2.0.0 | Enable monitoring metrics |

### 8.6 Upgrade Path Summary

**Quick Reference:**

```
Current Version → Upgrade Path

v1.0.x
  ├─> v1.2.0 ─> v2.0.0
  └─> v1.1.x ─> v1.2.0 ─> v2.0.0

v1.1.x
  └─> v1.2.0 ─> v2.0.0

v1.2.x
  └─> v2.0.0

v2.0.x
  ├─> v2.1.x ─> v3.0.0
  └─> v2.0.x ─> v3.0.0

v2.1.x
  └─> v3.0.0
```

**Recommendations:**
- Stay within 1 major version
- Upgrade to latest minor before major upgrade
- Test in staging before production
- Always backup before upgrade
- Review breaking changes documentation
