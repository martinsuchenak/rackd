# Backup & Restore

This guide covers backup strategies, restore procedures, and upgrades for Rackd.

## Backing Up

`rackd backup` writes a consistent snapshot of the database with SQLite's `VACUUM INTO`. It is safe while the server is running, produces a single file (no `-wal`/`-shm` companions), sets the file mode to `0600`, and verifies the copy before reporting success.

```bash
# Snapshot to rackd-backup-<timestamp>.db in the current directory
rackd backup --data-dir /var/lib/rackd

# Snapshot to a specific file (the command refuses to overwrite an existing file)
rackd backup --data-dir /var/lib/rackd --output /backup/rackd-$(date +%Y%m%d-%H%M%S).db

# Replace a fixed file; the old one is kept until the new backup is verified
rackd backup --data-dir /var/lib/rackd --output /backup/rackd-latest.db --overwrite
```

The command reads the database through SQLite. While the server is stopped, the user running it needs write access to the data directory, because SQLite creates a `-shm` file there.

Do not copy `rackd.db` with `cp` while the server runs: in WAL mode the main file and the `-wal` file change independently, and a plain copy can be inconsistent.

### Verifying a Backup

```bash
rackd backup verify --file /backup/rackd-20260101-020000.db
```

This opens the file read-only, runs `PRAGMA integrity_check`, and reports how many migrations it contains. It also warns when the backup was made by a newer Rackd version, which this binary cannot restore.

### Automatic Backups Before Migrations

When the server starts (or `rackd migrate run` runs) and there are pending migrations on an existing database, Rackd first writes a verified snapshot to `<data-dir>/backups/rackd-pre-migrate-<timestamp>-<target>.db`. `rackd migrate rollback` does the same before reverting. The five newest pre-migration snapshots are kept. If the snapshot fails, no migration runs.

## Automated Backups

### Scheduled Backups in the Server

The server can back itself up on a schedule:

```bash
BACKUP_INTERVAL=24h   # how often; 0 (the default) turns scheduled backups off
BACKUP_KEEP=7         # how many scheduled backups to keep
BACKUP_DIR=/backups   # where; defaults to <data-dir>/backups
```

Each backup is the same verified snapshot as `rackd backup`, written as `rackd-scheduled-<timestamp>.db`. Once a new one is taken, the oldest scheduled backups beyond `BACKUP_KEEP` are deleted; manual and pre-migration backups in the same directory are never touched.

- The next backup is due `BACKUP_INTERVAL` after the newest one in the directory, so restarting the server doesn't take an extra one. An overdue backup runs a minute after startup.
- A failed backup is logged as an error and retried after the shorter of `BACKUP_INTERVAL` and an hour.
- Each backup is a full copy of the database, so the interval must be at least 5 minutes. On network storage such as NFS, a longer interval, or a `BACKUP_DIR` on another volume, keeps the extra writes down.
- A backup on the same disk as the database doesn't survive losing that disk. Copy the directory elsewhere, or point `BACKUP_DIR` at other storage.

### Cron

```bash
#!/bin/bash
# /usr/local/bin/rackd-backup.sh
set -euo pipefail
BACKUP_DIR="/backup/rackd"
RETENTION_DAYS=30

mkdir -p "$BACKUP_DIR"
BACKUP_FILE="$BACKUP_DIR/rackd-$(date +%Y%m%d-%H%M%S).db"
rackd backup --data-dir /var/lib/rackd --output "$BACKUP_FILE"
gzip "$BACKUP_FILE"
find "$BACKUP_DIR" -name "rackd-*.db.gz" -mtime +$RETENTION_DAYS -delete
```

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

### Restoring a Backup

Stop the server first; `restore` refuses to run while another process has the database open.

```bash
sudo systemctl stop rackd

# Preview, then restore
rackd backup restore --data-dir /var/lib/rackd --file /backup/rackd-20260101-020000.db
rackd backup restore --data-dir /var/lib/rackd --file /backup/rackd-20260101-020000.db --yes

sudo chown rackd:rackd /var/lib/rackd/rackd.db
sudo systemctl start rackd
```

`restore` verifies the backup, folds the current database's WAL into its main file, moves it aside as `rackd.db.pre-restore-<timestamp>` (nothing is deleted), and copies the backup into place with mode `0600`. Restoring an older backup is fine: pending migrations run on the next start. A backup made by a newer Rackd version is refused.

### Restore from Compressed Backup

```bash
gunzip -k /backup/rackd-20260101-020000.db.gz
rackd backup restore --data-dir /var/lib/rackd --file /backup/rackd-20260101-020000.db --yes
```

### Verify Restore

```bash
rackd migrate status --data-dir /var/lib/rackd
curl -fsS http://localhost:8080/healthz
curl -fsS http://localhost:8080/readyz
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
   VERSION=x.y.z   # the release to install, without the leading v
   curl -LO https://github.com/martinsuchenak/rackd/releases/download/v${VERSION}/rackd_${VERSION}_linux_amd64.tar.gz
   tar -xzf rackd_${VERSION}_linux_amd64.tar.gz rackd
   sudo install -m 0755 rackd /usr/local/bin/rackd
   
   # Create data directory
   mkdir -p /var/lib/rackd
   ```

3. **Restore Database**
   ```bash
   # Check the backup, then restore it into the new data directory
   rackd backup verify --file /backup/rackd-latest.db
   rackd backup restore --data-dir /var/lib/rackd --file /backup/rackd-latest.db --yes
   ```

4. **Start Services**
   ```bash
   # Start Rackd
   rackd server
   
   # Verify functionality
   curl http://localhost:8080/readyz
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

# Check the backup and its migrations
./rackd backup verify --file "$TEST_DIR/rackd.db"
./rackd migrate status --data-dir "$TEST_DIR"

echo "Recovery test completed"
rm -rf "$TEST_DIR"
```

## Data Export/Import

### Export Data

Export specific data types using the CLI:

```bash
# Export all devices
rackd export devices --output devices.json

# Export networks
rackd export networks --output networks.json

# Export datacenters
rackd export datacenters --output datacenters.json
```

### Import Data

```bash
# Import datacenters first, then networks and devices that refer to them
rackd import datacenters --file datacenters.json
rackd import networks --file networks.json
rackd import devices --file devices.json   # add --dry-run to validate first
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

### Upgrading Rackd

Upgrades run in place: replace the binary and restart. On start, Rackd applies pending migrations after taking a verified snapshot in `<data-dir>/backups/`.

```bash
rackd backup --data-dir /var/lib/rackd --output /backup/rackd-pre-upgrade.db   # optional extra copy
sudo systemctl stop rackd
sudo install -m 0755 rackd /usr/local/bin/rackd   # extracted from rackd_<version>_linux_amd64.tar.gz
rackd migrate status --data-dir /var/lib/rackd    # shows the pending migrations
sudo systemctl start rackd
```

To undo an upgrade, either restore the pre-migration snapshot with `rackd backup restore` and reinstall the old binary, or revert migrations with `rackd migrate rollback --steps N --yes` before downgrading. Rollback also snapshots first; some Down steps drop tables and data.

Rackd versions after v0.0.21 record migrations in the `xdal_migrations` table. On first start they copy the history from the older `schema_migrations` table, which is left in place so an older binary can still open the database.

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