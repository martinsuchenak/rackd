package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// scheduledBackupPrefix names the backups the server takes on a schedule
// (BACKUP_INTERVAL). Pruning only touches files with this prefix, so manual
// and pre-migration backups in the same directory are left alone.
const scheduledBackupPrefix = "rackd-scheduled-"

// ScheduledBackup takes a verified backup of db into dir, named by the time
// it was taken, and deletes the oldest scheduled backups beyond keep.
func ScheduledBackup(ctx context.Context, db *sql.DB, dir string, keep int, now time.Time) (*BackupInfo, error) {
	path := filepath.Join(dir, scheduledBackupPrefix+now.UTC().Format(backupTimestampFormat)+".db")
	info, err := BackupDatabase(ctx, db, path)
	if err != nil {
		return nil, err
	}
	if keep > 0 {
		if err := pruneBackups(dir, scheduledBackupPrefix, keep); err != nil {
			return info, fmt.Errorf("backup taken, but pruning old ones failed: %w", err)
		}
	}
	return info, nil
}

// LatestScheduledBackup returns when the newest scheduled backup in dir was
// taken, read from its name, so the schedule carries on across restarts.
func LatestScheduledBackup(dir string) (time.Time, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return time.Time{}, false
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), scheduledBackupPrefix) && strings.HasSuffix(e.Name(), ".db") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for i := len(names) - 1; i >= 0; i-- {
		stamp := strings.TrimSuffix(strings.TrimPrefix(names[i], scheduledBackupPrefix), ".db")
		if t, err := time.Parse(backupTimestampFormat, stamp); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
