# Backup and Restore

This document defines backup and restore procedures for Rackd, including backup strategies, restore workflows, and disaster recovery planning.

## 1. Backup Strategies

### 1.1 Online Backup (SQLite)

Rackd uses SQLite's online backup API for zero-downtime backups.

```go
package backup

import (
    "context"
    "database/sql"
    "os"
    "path/filepath"
    "time"

    _ "modernc.org/sqlite"
)

type OnlineBackup struct {
    source *sql.DB
    dest   string
    config BackupConfig
}

type BackupConfig struct {
    PageCount    int           // Number of pages per step
    SleepTime    time.Duration // Sleep between steps
    ProgressFunc func(int, int) // Progress callback
}

func NewOnlineBackup(db *sql.DB, dest string, config BackupConfig) *OnlineBackup {
    if config.PageCount == 0 {
        config.PageCount = 100
    }
    if config.SleepTime == 0 {
        config.SleepTime = 10 * time.Millisecond
    }

    return &OnlineBackup{
        source: db,
        dest:   dest,
        config: config,
    }
}

func (b *OnlineBackup) Execute(ctx context.Context) error {
    log.Info("Starting online backup", "destination", b.dest)

    // Ensure destination directory exists
    if err := os.MkdirAll(filepath.Dir(b.dest), 0755); err != nil {
        return err
    }

    // Create destination database
    destDB, err := sql.Open("sqlite", b.dest)
    if err != nil {
        return err
    }
    defer destDB.Close()

    // Get backup connection
    conn, err := b.source.Conn(ctx)
    if err != nil {
        return err
    }

    // Begin backup
    var pageCount int
    var pageCountTotal int
    var isComplete bool

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // Backup one page step
        err := conn.Raw(func(driverConn interface{}) error {
            sqliteConn, ok := driverConn.(*sqlite.SQLiteConn)
            if !ok {
                return fmt.Errorf("not a SQLite connection")
            }

            return sqliteConn.Backup(destDB, "main", b.config.PageCount)
        })

        if err != nil {
            return fmt.Errorf("backup step failed: %w", err)
        }

        // Check if complete
        isComplete, err = b.isBackupComplete(ctx, conn)
        if err != nil {
            return err
        }

        if isComplete {
            break
        }

        // Report progress
        if b.config.ProgressFunc != nil {
            b.config.ProgressFunc(pageCount, pageCountTotal)
        }

        // Sleep to reduce load
        time.Sleep(b.config.SleepTime)
        pageCount += b.config.PageCount
    }

    log.Info("Backup completed", "pages", pageCount, "destination", b.dest)
    return nil
}

func (b *OnlineBackup) isBackupComplete(ctx context.Context, conn *sql.Conn) (bool, error) {
    var isComplete bool
    err := conn.Raw(func(driverConn interface{}) error {
        sqliteConn, ok := driverConn.(*sqlite.SQLiteConn)
        if !ok {
            return fmt.Errorf("not a SQLite connection")
        }

        isComplete, err = sqliteConn.BackupRemaining(destDB, "main")
        return err
    })

    return isComplete == 0, err
}
```

### 1.2 Offline Backup

File copy backup for maximum simplicity.

```go
func OfflineBackup(source, dest string) error {
    log.Info("Starting offline backup", "source", source, "dest", dest)

    // Ensure database is not in use
    // This is the caller's responsibility

    // Copy file
    sourceFile, err := os.Open(source)
    if err != nil {
        return err
    }
    defer sourceFile.Close()

    destFile, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer destFile.Close()

    if _, err := io.Copy(destFile, sourceFile); err != nil {
        return err
    }

    // Sync to disk
    if err := destFile.Sync(); err != nil {
        return err
    }

    log.Info("Offline backup completed", "destination", dest)
    return nil
}
```

### 1.3 Incremental Backup Strategy

Track changes and backup only modified data.

```go
type IncrementalBackup struct {
    baseBackup string
    deltaDir   string
}

func (ib *IncrementalBackup) CreateIncremental(ctx context.Context) error {
    // Get last backup time
    lastBackupTime := ib.getLastBackupTime()

    // Extract changed data since last backup
    changes, err := ib.extractChanges(ctx, lastBackupTime)
    if err != nil {
        return err
    }

    // Save delta file
    deltaPath := filepath.Join(ib.deltaDir, fmt.Sprintf("delta_%d.delta", time.Now().Unix()))
    return ib.saveDelta(deltaPath, changes)
}
```

### 1.4 Backup File Naming and Rotation

**Naming Convention:**

```
rackd_backup_YYYYMMDD_HHMMSS_<type>.db
rackd_backup_20240120_083000_online.db
rackd_backup_20240120_083000_offline.db
```

**Rotation Strategy:**

| Backup Type | Retention | Max Count |
|-------------|------------|-----------|
| Hourly | 24 hours | 24 |
| Daily | 7 days | 7 |
| Weekly | 4 weeks | 4 |
| Monthly | 12 months | 12 |

```go
type BackupRotation struct {
    hourly   int // Keep N hourly backups
    daily    int // Keep N daily backups
    weekly   int // Keep N weekly backups
    monthly  int // Keep N monthly backups
}

func (br *BackupRotation) Rotate(backupDir string) error {
    backups, err := br.listBackups(backupDir)
    if err != nil {
        return err
    }

    // Group by type
    hourly := br.filterByType(backups, "hourly")
    daily := br.filterByType(backups, "daily")
    weekly := br.filterByType(backups, "weekly")
    monthly := br.filterByType(backups, "monthly")

    // Remove excess backups
    if err := br.removeOlderThan(hourly, br.hourly); err != nil {
        return err
    }
    if err := br.removeOlderThan(daily, br.daily); err != nil {
        return err
    }
    if err := br.removeOlderThan(weekly, br.weekly); err != nil {
        return err
    }
    if err := br.removeOlderThan(monthly, br.monthly); err != nil {
        return err
    }

    return nil
}
```

---

## 2. Backup Procedures

### 2.1 Manual Backup Workflow

**Step 1: Prepare for Backup**

```bash
# Check if database is in use
rackd backup check

# Ensure no active discovery scans
rackd discovery list --status running
```

**Step 2: Create Backup**

```bash
# Online backup (recommended)
rackd backup create --type online

# Offline backup (requires server stop)
rackd server stop
rackd backup create --type offline
rackd server start
```

**Step 3: Verify Backup**

```bash
rackd backup verify <backup-id>
```

### 2.2 Pre-Backup Checklist

- [ ] Discovery scans are paused or completed
- [ ] No active write operations
- [ ] Sufficient disk space available (2x database size)
- [ ] Backup destination is accessible
- [ ] Backup encryption is configured (if required)
- [ ] Verify recent database integrity

### 2.3 Backup Verification

```go
func VerifyBackup(backupPath string) error {
    log.Info("Verifying backup", "path", backupPath)

    // Open backup database
    db, err := sql.Open("sqlite", backupPath)
    if err != nil {
        return fmt.Errorf("failed to open backup: %w", err)
    }
    defer db.Close()

    // Check integrity
    var integrityOK bool
    err := db.QueryRow(`PRAGMA integrity_check`).Scan(&integrityOK)
    if err != nil {
        return fmt.Errorf("integrity check failed: %w", err)
    }

    if !integrityOK {
        return errors.New("backup integrity check failed")
    }

    // Check schema version
    var version string
    err = db.QueryRow(`SELECT version FROM schema_migrations ORDER BY applied_at DESC LIMIT 1`).Scan(&version)
    if err != nil {
        return fmt.Errorf("schema version check failed: %w", err)
    }

    log.Info("Backup verified", "schema_version", version)

    // Verify data counts
    var deviceCount int
    db.QueryRow(`SELECT COUNT(*) FROM devices`).Scan(&deviceCount)

    var networkCount int
    db.QueryRow(`SELECT COUNT(*) FROM networks`).Scan(&networkCount)

    log.Info("Backup statistics",
        "devices", deviceCount,
        "networks", networkCount,
    )

    return nil
}
```

### 2.4 Checksum Calculation

```go
func CalculateChecksum(filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }

    return hex.EncodeToString(hash.Sum(nil)), nil
}

func VerifyChecksum(filePath, expectedChecksum string) error {
    actualChecksum, err := CalculateChecksum(filePath)
    if err != nil {
        return err
    }

    if actualChecksum != expectedChecksum {
        return fmt.Errorf("checksum mismatch: expected %s, got %s",
            expectedChecksum, actualChecksum)
    }

    return nil
}
```

---

## 3. Restore Workflow

### 3.1 Step-by-Step Restore Process

**Step 1: Stop Service**

```bash
# Stop the server
rackd server stop

# Verify no processes are running
pgrep -f rackd || echo "No rackd processes running"
```

**Step 2: Backup Current Database**

```bash
# Just in case restore fails
rackd backup create --type offline --name pre_restore_backup
```

**Step 3: Restore Database**

```bash
# Restore from backup
rackd backup restore <backup-id>

# Or manually copy file
cp /backups/rackd_backup_20240120_083000.db /data/rackd.db
```

**Step 4: Verify Restore**

```bash
# Check database integrity
rackd backup verify <backup-id>

# Start server
rackd server start

# Verify data
rackd device list | head
rackd network list | head
```

**Step 5: Run Post-Restore Checks**

```bash
# Run data validation
rackd doctor validate

# Check logs for errors
tail -f /var/log/rackd.log
```

### 3.2 Data Integrity Verification

```go
func VerifyRestore(backupPath, restorePath string) error {
    backupDB, err := sql.Open("sqlite", backupPath)
    if err != nil {
        return err
    }
    defer backupDB.Close()

    restoreDB, err := sql.Open("sqlite", restorePath)
    if err != nil {
        return err
    }
    defer restoreDB.Close()

    // Compare row counts
    comparisons := []struct {
        table string
        field string
    }{
        {"devices", "COUNT(*)"},
        {"networks", "COUNT(*)"},
        {"datacenters", "COUNT(*)"},
    }

    for _, comp := range comparisons {
        var backupCount, restoreCount int

        backupDB.QueryRow(fmt.Sprintf("SELECT %s FROM %s", comp.field, comp.table)).Scan(&backupCount)
        restoreDB.QueryRow(fmt.Sprintf("SELECT %s FROM %s", comp.field, comp.table)).Scan(&restoreCount)

        if backupCount != restoreCount {
            return fmt.Errorf("row count mismatch for table %s: backup=%d, restore=%d",
                comp.table, backupCount, restoreCount)
        }

        log.Info("Table verified", "table", comp.table, "rows", restoreCount)
    }

    return nil
}
```

### 3.3 Rollback on Failure

```go
func RestoreWithRollback(backupPath, dbPath string) error {
    // Create backup of current database
    tempBackup := dbPath + ".pre_restore"
    if err := copyFile(dbPath, tempBackup); err != nil {
        return err
    }

    // Attempt restore
    if err := RestoreDatabase(backupPath, dbPath); err != nil {
        log.Error("Restore failed, rolling back", "error", err)

        // Rollback by restoring temporary backup
        if rbErr := copyFile(tempBackup, dbPath); rbErr != nil {
            return fmt.Errorf("restore failed and rollback also failed: %w (rollback error: %v)", err, rbErr)
        }

        return fmt.Errorf("restore failed, rolled back to previous state: %w", err)
    }

    // Success - remove temporary backup
    os.Remove(tempBackup)

    return nil
}
```

### 3.4 Zero-Downtime Restore Strategies

**Strategy 1: Database Swapping**

```bash
# 1. Start new instance with backup
rackd server --data-dir /data/new --listen-addr :8081

# 2. Verify new instance is working
curl http://localhost:8081/api/datacenters

# 3. Update load balancer to point to new instance
# (or update DNS, depending on deployment)

# 4. Stop old instance
rackd server --data-dir /data/old stop
```

**Strategy 2: Online Restore with Replication**

*(Enterprise feature with Postgres)*

```go
func OnlineRestoreWithReplication(backupPath string) error {
    // 1. Restore backup to replica database
    replicaDB, err := sql.Open("postgres", "replica-dsn")
    if err != nil {
        return err
    }

    if err := RestoreDatabase(backupPath, "replica-dsn"); err != nil {
        return err
    }

    // 2. Switch traffic to replica
    if err := SwitchTrafficToReplica(); err != nil {
        return err
    }

    // 3. Restore primary database in background
    go func() {
        RestoreDatabase(backupPath, "primary-dsn")
        // Switch back to primary when ready
    }()

    return nil
}
```

---

## 4. Backup Configuration

### 4.1 Automated Backup Scheduling

```go
package backup

import (
    "github.com/robfig/cron/v3"
)

type BackupScheduler struct {
    cron    *cron.Cron
    config  BackupConfig
    storage BackupStorage
}

type BackupSchedule struct {
    Hourly   string `json:"hourly"`   // Cron expression
    Daily    string `json:"daily"`
    Weekly   string `json:"weekly"`
    Monthly  string `json:"monthly"`
}

func NewBackupScheduler(config BackupConfig, storage BackupStorage) *BackupScheduler {
    c := cron.New()

    return &BackupScheduler{
        cron:    c,
        config:  config,
        storage: storage,
    }
}

func (bs *BackupScheduler) Start(schedule BackupSchedule) error {
    // Add scheduled jobs
    if schedule.Hourly != "" {
        if _, err := bs.cron.AddFunc(schedule.Hourly, bs.runHourlyBackup); err != nil {
            return err
        }
    }

    if schedule.Daily != "" {
        if _, err := bs.cron.AddFunc(schedule.Daily, bs.runDailyBackup); err != nil {
            return err
        }
    }

    if schedule.Weekly != "" {
        if _, err := bs.cron.AddFunc(schedule.Weekly, bs.runWeeklyBackup); err != nil {
            return err
        }
    }

    if schedule.Monthly != "" {
        if _, err := bs.cron.AddFunc(schedule.Monthly, bs.runMonthlyBackup); err != nil {
            return err
        }
    }

    bs.cron.Start()
    log.Info("Backup scheduler started")
    return nil
}

func (bs *BackupScheduler) runHourlyBackup() {
    backup := bs.createBackup("hourly")
    if err := backup.Execute(context.Background()); err != nil {
        log.Error("Hourly backup failed", "error", err)
    }
}

func (bs *BackupScheduler) runDailyBackup() {
    backup := bs.createBackup("daily")
    if err := backup.Execute(context.Background()); err != nil {
        log.Error("Daily backup failed", "error", err)
    }
}
```

**Default Schedule:**

| Type | Cron Expression | Time |
|------|---------------|------|
| Hourly | `0 * * * *` | Every hour at minute 0 |
| Daily | `0 2 * * *` | Daily at 2 AM |
| Weekly | `0 3 * * 0` | Sunday at 3 AM |
| Monthly | `0 4 1 * *` | 1st of month at 4 AM |

### 4.2 Retention Policies

```go
type RetentionPolicy struct {
    HourlyKeep  int `json:"hourly_keep"`  // Hours to keep
    DailyKeep   int `json:"daily_keep"`   // Days to keep
    WeeklyKeep  int `json:"weekly_keep"`  // Weeks to keep
    MonthlyKeep int `json:"monthly_keep"` // Months to keep
}

func (rp *RetentionPolicy) Apply(backupDir string) error {
    backups, err := ListBackups(backupDir)
    if err != nil {
        return err
    }

    // Group by type
    hourly := filterByType(backups, "hourly")
    daily := filterByType(backups, "daily")
    weekly := filterByType(backups, "weekly")
    monthly := filterByType(backups, "monthly")

    // Remove old backups
    if err := removeOldBackups(hourly, rp.HourlyKeep, time.Hour); err != nil {
        return err
    }
    if err := removeOldBackups(daily, rp.DailyKeep, 24*time.Hour); err != nil {
        return err
    }
    if err := removeOldBackups(weekly, rp.WeeklyKeep, 7*24*time.Hour); err != nil {
        return err
    }
    if err := removeOldBackups(monthly, rp.MonthlyKeep, 30*24*time.Hour); err != nil {
        return err
    }

    return nil
}
```

**Configuration File:**

```yaml
# config/backup.yaml
retention:
  hourly_keep: 24  # Keep 24 hourly backups
  daily_keep: 7    # Keep 7 daily backups
  weekly_keep: 4   # Keep 4 weekly backups
  monthly_keep: 12 # Keep 12 monthly backups

schedule:
  hourly: "0 * * * *"
  daily: "0 2 * * *"
  weekly: "0 3 * * 0"
  monthly: "0 4 1 * *"

storage:
  type: "local" # local, s3, gcs, azure
  path: "/backups/rackd"
  encryption: true
  compression: true
```

---

## 5. Remote Backup Destinations

### 5.1 S3 Integration Pattern

```go
package backup

import (
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
    client *s3.Client
    bucket string
    prefix string
}

func NewS3Storage(config S3Config) (*S3Storage, error) {
    cfg, err := loadAWSConfig(config)
    if err != nil {
        return nil, err
    }

    client := s3.NewFromConfig(cfg)

    return &S3Storage{
        client: client,
        bucket: config.Bucket,
        prefix: config.Prefix,
    }, nil
}

func (s *S3Storage) UploadBackup(ctx context.Context, backupPath string) error {
    file, err := os.Open(backupPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Get file info
    fileInfo, _ := file.Stat()

    // Upload to S3
    key := fmt.Sprintf("%s/%s", s.prefix, filepath.Base(backupPath))
    _, err = s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:    file,
        ContentLength: aws.Int64(fileInfo.Size()),
    })

    if err != nil {
        return err
    }

    log.Info("Backup uploaded to S3", "bucket", s.bucket, "key", key)
    return nil
}

func (s *S3Storage) ListBackups(ctx context.Context) ([]BackupInfo, error) {
    result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
        Bucket: aws.String(s.bucket),
        Prefix: aws.String(s.prefix),
    })

    if err != nil {
        return nil, err
    }

    backups := make([]BackupInfo, 0, len(result.Contents))
    for _, obj := range result.Contents {
        backups = append(backups, BackupInfo{
            Key:          *obj.Key,
            Size:         *obj.Size,
            LastModified:  *obj.LastModified,
            Location:      fmt.Sprintf("s3://%s/%s", s.bucket, *obj.Key),
        })
    }

    return backups, nil
}
```

### 5.2 GCS Integration Pattern

```go
package backup

import (
    "cloud.google.com/go/storage"
)

type GCSStorage struct {
    client *storage.Client
    bucket string
    prefix string
}

func NewGCSStorage(ctx context.Context, config GCSConfig) (*GCSStorage, error) {
    client, err := storage.NewClient(ctx)
    if err != nil {
        return nil, err
    }

    return &GCSStorage{
        client: client,
        bucket: config.Bucket,
        prefix: config.Prefix,
    }, nil
}

func (g *GCSStorage) UploadBackup(ctx context.Context, backupPath string) error {
    file, err := os.Open(backupPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Upload to GCS
    key := fmt.Sprintf("%s/%s", g.prefix, filepath.Base(backupPath))
    obj := g.client.Bucket(g.bucket).Object(key)

    writer := obj.NewWriter(ctx)
    if _, err := io.Copy(writer, file); err != nil {
        writer.Close()
        return err
    }

    if err := writer.Close(); err != nil {
        return err
    }

    log.Info("Backup uploaded to GCS", "bucket", g.bucket, "key", key)
    return nil
}
```

### 5.3 Azure Blob Integration Pattern

```go
package backup

import (
    "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
)

type AzureStorage struct {
    client   *azblob.Client
    container string
    prefix   string
}

func NewAzureStorage(config AzureConfig) (*AzureStorage, error) {
    credential, err := azblob.NewSharedKeyCredential(config.AccountName, config.AccountKey)
    if err != nil {
        return nil, err
    }

    client, err := azblob.NewClientWithSharedKeyCredential(
        fmt.Sprintf("%s.blob.core.windows.net", config.AccountName),
        credential,
        nil,
    )

    if err != nil {
        return nil, err
    }

    return &AzureStorage{
        client:   client,
        container: config.Container,
        prefix:   config.Prefix,
    }, nil
}

func (a *AzureStorage) UploadBackup(ctx context.Context, backupPath string) error {
    file, err := os.Open(backupPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Upload to Azure
    key := fmt.Sprintf("%s/%s", a.prefix, filepath.Base(backupPath))
    _, err = a.client.UploadFile(ctx, a.container, key, file, nil)

    if err != nil {
        return err
    }

    log.Info("Backup uploaded to Azure", "container", a.container, "key", key)
    return nil
}
```

### 5.4 Encryption Options

```go
type EncryptionConfig struct {
    Enabled    bool   `json:"enabled"`
    Algorithm  string `json:"algorithm"` // AES-256-GCM
    KeySource  string `json:"key_source"` // env, file, kms
    KeyID      string `json:"key_id"`
}

func EncryptBackup(backupPath string, config EncryptionConfig) error {
    if !config.Enabled {
        return nil
    }

    // Get encryption key
    key, err := getEncryptionKey(config)
    if err != nil {
        return err
    }

    // Encrypt file
    encryptedPath := backupPath + ".enc"
    if err := encryptFile(backupPath, encryptedPath, key); err != nil {
        return err
    }

    // Remove original
    os.Remove(backupPath)

    log.Info("Backup encrypted", "algorithm", config.Algorithm)
    return nil
}

func encryptFile(src, dest string, key []byte) error {
    // Open source
    srcFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer srcFile.Close()

    // Create destination
    destFile, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer destFile.Close()

    // Create cipher
    block, err := aes.NewCipher(key)
    if err != nil {
        return err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return err
    }

    // Write nonce
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return err
    }

    if _, err := destFile.Write(nonce); err != nil {
        return err
    }

    // Encrypt and write
    cipher := gcm.Seal(nil, nil, nonce, nil)
    if _, err := destFile.Write(cipher); err != nil {
        return err
    }

    return nil
}
```

---

## 6. Backup CLI Commands

### 6.1 Create Backup

```bash
# Create online backup (recommended)
rackd backup create --type online

# Create offline backup
rackd backup create --type offline

# Create backup with custom name
rackd backup create --name production_backup

# Create incremental backup
rackd backup create --type incremental
```

### 6.2 List Backups

```bash
# List all backups
rackd backup list

# List backups by type
rackd backup list --type hourly
rackd backup list --type daily

# List backups from remote storage
rackd backup list --remote
```

**Output:**

```
Backups
========
ID: backup_20240120_083000_online
Type: online
Size: 45.2 MB
Created: 2024-01-20 08:30:00
Location: /backups/rackd/backup_20240120_083000_online.db
Checksum: a1b2c3d4e5f6...

ID: backup_20240120_073000_hourly
Type: hourly
Size: 45.1 MB
Created: 2024-01-20 07:30:00
Location: /backups/rackd/hourly/backup_20240120_073000.db
Checksum: f2e3d4c5b6a7...
```

### 6.3 Restore Backup

```bash
# Restore from backup ID
rackd backup restore backup_20240120_083000_online

# Restore to specific location
rackd backup restore backup_20240120_083000_online --dest /data/rackd_new.db

# Restore with verification
rackd backup restore backup_20240120_083000_online --verify

# Force restore (skip confirmation)
rackd backup restore backup_20240120_083000_online --force
```

### 6.4 Verify Backup

```bash
# Verify backup integrity
rackd backup verify backup_20240120_083000_online

# Verify checksum
rackd backup verify backup_20240120_083000_online --checksum a1b2c3d4e5f6...

# Verify data integrity
rackd backup verify backup_20240120_083000_online --full
```

### 6.5 Schedule Backup

```bash
# Start backup scheduler
rackd backup schedule start --config /etc/rackd/backup.yaml

# Stop backup scheduler
rackd backup schedule stop

# Check schedule status
rackd backup schedule status

# Run backup now (one-time)
rackd backup create --now
```

---

## 7. Backup Metadata

### 7.1 Version Tracking

```go
type BackupMetadata struct {
    ID            string    `json:"id"`
    Type          string    `json:"type"` // online, offline, incremental
    SchemaVersion string    `json:"schema_version"`
    RackdVersion  string    `json:"rackd_version"`
    Size          int64     `json:"size"`
    CreatedAt     time.Time `json:"created_at"`
    Checksum      string    `json:"checksum"`
    Location      string    `json:"location"`
    Encrypted     bool      `json:"encrypted"`
    Compressed    bool      `json:"compressed"`
}

func SaveMetadata(backupPath string, metadata BackupMetadata) error {
    metadataPath := backupPath + ".meta"
    data, err := json.MarshalIndent(metadata, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(metadataPath, data, 0644)
}

func LoadMetadata(backupPath string) (*BackupMetadata, error) {
    metadataPath := backupPath + ".meta"
    data, err := os.ReadFile(metadataPath)
    if err != nil {
        return nil, err
    }

    var metadata BackupMetadata
    if err := json.Unmarshal(data, &metadata); err != nil {
        return nil, err
    }

    return &metadata, nil
}
```

### 7.2 Checksum Verification

```go
func VerifyBackupChecksum(backupPath string) error {
    metadata, err := LoadMetadata(backupPath)
    if err != nil {
        return fmt.Errorf("failed to load metadata: %w", err)
    }

    actualChecksum, err := CalculateChecksum(backupPath)
    if err != nil {
        return fmt.Errorf("failed to calculate checksum: %w", err)
    }

    if actualChecksum != metadata.Checksum {
        return fmt.Errorf("checksum mismatch: expected %s, got %s",
            metadata.Checksum, actualChecksum)
    }

    log.Info("Checksum verified", "checksum", actualChecksum)
    return nil
}
```

### 7.3 Backup Catalog

```go
type BackupCatalog struct {
    backups map[string]BackupInfo
    lock    sync.RWMutex
}

func NewBackupCatalog() *BackupCatalog {
    return &BackupCatalog{
        backups: make(map[string]BackupInfo),
    }
}

func (bc *BackupCatalog) Add(info BackupInfo) {
    bc.lock.Lock()
    defer bc.lock.Unlock()

    bc.backups[info.ID] = info
}

func (bc *BackupCatalog) List() []BackupInfo {
    bc.lock.RLock()
    defer bc.lock.RUnlock()

    result := make([]BackupInfo, 0, len(bc.backups))
    for _, info := range bc.backups {
        result = append(result, info)
    }

    sort.Slice(result, func(i, j int) bool {
        return result[i].CreatedAt.After(result[j].CreatedAt)
    })

    return result
}

func (bc *BackupCatalog) Get(id string) (BackupInfo, bool) {
    bc.lock.RLock()
    defer bc.lock.RUnlock()

    info, ok := bc.backups[id]
    return info, ok
}
```

---

## 8. Disaster Recovery

### 8.1 Recovery Time Objective (RTO) Configuration

```go
type RecoveryObjectives struct {
    RTO       time.Duration `json:"rto"` // Maximum time to restore
    RPO       time.Duration `json:"rpo"` // Maximum data loss
    Critical  bool          `json:"critical"` // Is this system critical?
}

func (ro *RecoveryObjectives) Verify(backup *BackupInfo) error {
    // Check RTO
    timeSinceBackup := time.Since(backup.CreatedAt)
    if timeSinceBackup > ro.RPO {
        return fmt.Errorf("backup age (%v) exceeds RPO (%v)",
            timeSinceBackup, ro.RPO)
    }

    return nil
}
```

**Example RTO/RPO Targets:**

| System Type | RTO | RPO |
|-------------|------|------|
| Production IPAM | 30 min | 1 hour |
| Staging IPAM | 4 hours | 24 hours |
| Development | 1 day | 1 week |

### 8.2 Recovery Point Objective (RPO) Configuration

```bash
# config/recovery.yaml
objectives:
  rto: "30m"  # 30 minutes
  rpo: "1h"   # 1 hour

alerts:
  rto_exceeded:
    enabled: true
    severity: critical
  rpo_exceeded:
    enabled: true
    severity: warning
```

### 8.3 Documentation Requirements

**Disaster Recovery Plan Checklist:**

- [ ] Contact information for all team members
- [ ] Backup locations and access credentials
- [ ] Restore procedures documented
- [ ] Known RTO/RPO targets
- [ ] Test recovery procedures quarterly
- [ ] Document lessons learned from drills
- [ ] Communication plan for users
- [ ] Escalation procedures

**Recovery Runbook Template:**

```markdown
# Disaster Recovery Runbook

## System Information
- System: Rackd IPAM
- Version: v1.2.0
- Backup Location: S3://rackd-backups
- Database: /data/rackd.db

## Recovery Objectives
- RTO: 30 minutes
- RPO: 1 hour

## Recovery Steps
1. Stop service: `rackd server stop`
2. Identify latest backup: `rackd backup list --remote`
3. Download backup: `aws s3 cp s3://rackd-backups/...`
4. Restore backup: `rackd backup restore <backup-id>`
5. Verify integrity: `rackd doctor validate`
6. Start service: `rackd server start`
7. Verify data: `rackd device list`

## Rollback Plan
If restore fails:
1. Restore previous backup
2. Contact database team
3. Escalate to management

## Communication
- Notify users: status page
- Update ticket: #12345
- Team notification: Slack #outages
```

### 8.4 Testing Procedures

**Quarterly Disaster Recovery Drills:**

```bash
# 1. Schedule drill for Sunday at 3 AM
# 2. Notify team of planned drill
# 3. Stop service in test environment
# 4. Simulate data loss (delete test database)
# 5. Perform restore from backup
# 6. Verify data integrity
# 7. Measure RTO
# 8. Document lessons learned

# Automate with script
./disaster-drill.sh --test-env
```

```bash
#!/bin/bash
# disaster-drill.sh

ENVIRONMENT="test"
BACKUP_ID=$(rackd backup list --type daily --json | jq -r '.[0].id')

echo "Starting disaster recovery drill at $(date)"

# Record start time
START=$(date +%s)

# Stop service
rackd server stop

# Remove database
rm /data/test/rackd.db

# Restore backup
rackd backup restore $BACKUP_ID --dest /data/test/rackd.db

# Verify
rackd doctor validate --environment $ENVIRONMENT

# Start service
rackd server start --environment $ENVIRONMENT

# Record end time
END=$(date +%s)
DURATION=$((END - START))

echo "Disaster recovery drill completed in $DURATION seconds"

# Check RTO
if [ $DURATION -gt 1800 ]; then
    echo "WARNING: RTO exceeded (30 min)"
else
    echo "SUCCESS: RTO met"
fi
```
