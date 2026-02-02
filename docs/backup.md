# Backup & Restore

This guide covers backup strategies, restore procedures, and data migration for Rackd.

## SQLite Backup Strategies

### 1. File-Based Backup

The simplest approach is copying the SQLite database file:

```bash
# Stop Rackd service
systemctl stop rackd

# Copy database file
cp /var/lib/rackd/rackd.db /backup/rackd-$(date +%Y%m%d-%H%M%S).db

# Start Rackd service
systemctl start rackd
```

### 2. Online Backup (Recommended)

Use SQLite's `.backup` command for consistent backups without stopping the service:

```bash
# Create backup while service is running
sqlite3 /var/lib/rackd/rackd.db ".backup /backup/rackd-$(date +%Y%m%d-%H%M%S).db"
```

### 3. WAL Mode Backup

If using WAL mode, backup both database and WAL files:

```bash
# Checkpoint WAL file first
sqlite3 /var/lib/rackd/rackd.db "PRAGMA wal_checkpoint(FULL);"

# Backup database file
cp /var/lib/rackd/rackd.db /backup/rackd-$(date +%Y%m%d-%H%M%S).db
```

## Automated Backups

### Cron-Based Backup Script

Create `/usr/local/bin/rackd-backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/backup/rackd"
DB_PATH="/var/lib/rackd/rackd.db"
RETENTION_DAYS=30

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Create backup
BACKUP_FILE="$BACKUP_DIR/rackd-$(date +%Y%m%d-%H%M%S).db"
sqlite3 "$DB_PATH" ".backup $BACKUP_FILE"

# Compress backup
gzip "$BACKUP_FILE"

# Remove old backups
find "$BACKUP_DIR" -name "*.db.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: $BACKUP_FILE.gz"
```

Add to crontab:

```bash
# Daily backup at 2 AM
0 2 * * * /usr/local/bin/rackd-backup.sh
```

### Systemd Timer

Create `/etc/systemd/system/rackd-backup.service`:

```ini
[Unit]
Description=Rackd Database Backup
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/rackd-backup.sh
User=rackd
```

Create `/etc/systemd/system/rackd-backup.timer`:

```ini
[Unit]
Description=Run Rackd backup daily
Requires=rackd-backup.service

[Timer]
OnCalendar=daily
Persistent=true

[Install]
WantedBy=timers.target
```

Enable the timer:

```bash
systemctl enable rackd-backup.timer
systemctl start rackd-backup.timer
```

## Restore Procedures

### Basic Restore

```bash
# Stop Rackd service
systemctl stop rackd

# Backup current database (optional)
cp /var/lib/rackd/rackd.db /var/lib/rackd/rackd.db.backup

# Restore from backup
cp /backup/rackd-20240101-020000.db /var/lib/rackd/rackd.db

# Set correct permissions
chown rackd:rackd /var/lib/rackd/rackd.db
chmod 644 /var/lib/rackd/rackd.db

# Start Rackd service
systemctl start rackd
```

### Restore from Compressed Backup

```bash
# Decompress backup
gunzip /backup/rackd-20240101-020000.db.gz

# Follow basic restore procedure
```

### Verify Restore

```bash
# Check database integrity
sqlite3 /var/lib/rackd/rackd.db "PRAGMA integrity_check;"

# Verify Rackd can start
rackd server --dry-run

# Check web UI and API
curl http://localhost:8080/api/health
```

## Disaster Recovery

### Recovery Plan

1. **Assess Damage**
   - Identify what data is lost
   - Determine last known good backup
   - Check backup integrity

2. **Prepare New Environment**
   ```bash
   # Install Rackd
   curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-linux-amd64
   chmod +x rackd-linux-amd64
   
   # Create data directory
   mkdir -p /var/lib/rackd
   ```

3. **Restore Database**
   ```bash
   # Copy backup to new location
   cp /backup/rackd-latest.db /var/lib/rackd/rackd.db
   
   # Verify integrity
   sqlite3 /var/lib/rackd/rackd.db "PRAGMA integrity_check;"
   ```

4. **Start Services**
   ```bash
   # Start Rackd
   ./rackd-linux-amd64 server
   
   # Verify functionality
   curl http://localhost:8080/api/health
   ```

### Recovery Testing

Regularly test recovery procedures:

```bash
#!/bin/bash
# Test restore script
TEST_DIR="/tmp/rackd-test"
mkdir -p "$TEST_DIR"

# Copy backup
cp /backup/rackd-latest.db "$TEST_DIR/rackd.db"

# Test database
sqlite3 "$TEST_DIR/rackd.db" "PRAGMA integrity_check;"

# Test Rackd startup
RACKD_DATA_DIR="$TEST_DIR" ./rackd server --dry-run

echo "Recovery test completed"
rm -rf "$TEST_DIR"
```

## Data Export/Import

### Export Data

Export specific data types using the CLI:

```bash
# Export all devices
rackd device list --format json > devices.json

# Export networks
rackd network list --format json > networks.json

# Export datacenters
rackd datacenter list --format json > datacenters.json
```

### Import Data

```bash
# Import devices
cat devices.json | jq -r '.[] | @json' | while read device; do
  echo "$device" | rackd device create --from-json
done

# Import networks
cat networks.json | jq -r '.[] | @json' | while read network; do
  echo "$network" | rackd network create --from-json
done
```

### Full Database Export

```bash
# Export entire database as SQL
sqlite3 /var/lib/rackd/rackd.db .dump > rackd-export.sql

# Export as CSV (per table)
sqlite3 /var/lib/rackd/rackd.db -header -csv "SELECT * FROM devices;" > devices.csv
```

### Full Database Import

```bash
# Import from SQL dump
sqlite3 /var/lib/rackd/rackd-new.db < rackd-export.sql
```

## Migration Between Instances

### Same Version Migration

1. **Export from source**
   ```bash
   # Create backup
   sqlite3 /var/lib/rackd/rackd.db ".backup /tmp/rackd-migration.db"
   ```

2. **Transfer to destination**
   ```bash
   # Copy database file
   scp /tmp/rackd-migration.db user@destination:/tmp/
   ```

3. **Import to destination**
   ```bash
   # Stop destination service
   systemctl stop rackd
   
   # Replace database
   cp /tmp/rackd-migration.db /var/lib/rackd/rackd.db
   
   # Start service
   systemctl start rackd
   ```

### Cross-Version Migration

1. **Export data from source**
   ```bash
   # Export using CLI
   rackd device list --format json > devices.json
   rackd network list --format json > networks.json
   rackd datacenter list --format json > datacenters.json
   ```

2. **Setup destination with new version**
   ```bash
   # Install new Rackd version
   # Initialize fresh database
   ./rackd server --init-db
   ```

3. **Import data to destination**
   ```bash
   # Import each data type
   # Handle any schema differences
   ```

### Migration Validation

```bash
#!/bin/bash
# Validate migration
SOURCE_DB="/var/lib/rackd/source.db"
DEST_DB="/var/lib/rackd/dest.db"

# Compare record counts
echo "Devices:"
echo "Source: $(sqlite3 $SOURCE_DB 'SELECT COUNT(*) FROM devices;')"
echo "Dest:   $(sqlite3 $DEST_DB 'SELECT COUNT(*) FROM devices;')"

echo "Networks:"
echo "Source: $(sqlite3 $SOURCE_DB 'SELECT COUNT(*) FROM networks;')"
echo "Dest:   $(sqlite3 $DEST_DB 'SELECT COUNT(*) FROM networks;')"
```

## Best Practices

### Backup Strategy

- **Frequency**: Daily automated backups minimum
- **Retention**: Keep 30 daily, 12 monthly, 7 yearly backups
- **Location**: Store backups off-site or in different storage system
- **Testing**: Test restore procedures monthly
- **Monitoring**: Alert on backup failures

### Security

```bash
# Encrypt backups
gpg --cipher-algo AES256 --compress-algo 1 --symmetric --output backup.db.gpg backup.db

# Decrypt for restore
gpg --decrypt backup.db.gpg > backup.db
```

### Performance

- Use WAL mode for better concurrent access during backups
- Schedule backups during low-usage periods
- Monitor backup duration and size trends
- Use incremental backups for large databases

### Monitoring

```bash
# Check backup age
find /backup -name "*.db.gz" -mtime -1 | wc -l

# Check backup size trends
ls -la /backup/*.db.gz | tail -5
```

## Troubleshooting

### Common Issues

**Database locked during backup**
```bash
# Check for long-running transactions
sqlite3 /var/lib/rackd/rackd.db "PRAGMA wal_checkpoint(FULL);"
```

**Corrupted backup**
```bash
# Verify backup integrity
sqlite3 backup.db "PRAGMA integrity_check;"
```

**Restore fails**
```bash
# Check file permissions
ls -la /var/lib/rackd/rackd.db
chown rackd:rackd /var/lib/rackd/rackd.db
```

**Migration data loss**
```bash
# Compare schemas
sqlite3 source.db ".schema" > source-schema.sql
sqlite3 dest.db ".schema" > dest-schema.sql
diff source-schema.sql dest-schema.sql
```