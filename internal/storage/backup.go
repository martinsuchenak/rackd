package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/martinsuchenak/xdal/pkg/migrate"
)

// ErrDatabaseInUse is returned by RestoreDatabase when another process has the database open.
var ErrDatabaseInUse = errors.New("database is in use; stop the rackd server before restoring")

// BackupInfo describes a verified backup file.
type BackupInfo struct {
	Path              string
	SizeBytes         int64
	Migrations        int
	LatestMigration   string
	UnknownMigrations []string
}

// BackupDatabase writes a consistent snapshot of the open database to dst with
// VACUUM INTO, which is safe while the server is writing, then verifies the copy
// and returns what the verification found. dst must not exist; it is created
// with mode 0600.
func BackupDatabase(ctx context.Context, db *sql.DB, dst string) (*BackupInfo, error) {
	if _, err := os.Stat(dst); err == nil {
		return nil, fmt.Errorf("backup file already exists: %s", dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	if err := vacuumInto(ctx, db, dst); err != nil {
		return nil, err
	}
	info, err := VerifyBackup(ctx, dst)
	if err != nil {
		_ = os.Remove(dst)
		return nil, err
	}
	return info, nil
}

// BackupFile opens the database file at dbPath read-only and snapshots it to dst.
// Use it from tools that do not already hold the database open.
func BackupFile(ctx context.Context, dbPath, dst string) (*BackupInfo, error) {
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database not found at %s: %w", dbPath, err)
	}
	db, err := openReadOnly(dbPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()
	return BackupDatabase(ctx, db, dst)
}

// vacuumInto writes a single-file snapshot of db to dst with mode 0600.
func vacuumInto(ctx context.Context, db *sql.DB, dst string) error {
	if _, err := db.ExecContext(ctx, "VACUUM INTO ?", dst); err != nil {
		_ = os.Remove(dst)
		return fmt.Errorf("failed to write snapshot: %w", err)
	}
	if err := os.Chmod(dst, 0o600); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}

// sqliteURI builds a file: URI for path with the given query, escaping characters
// such as '?' and '#' that would otherwise be read as URI syntax.
func sqliteURI(path, query string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs), RawQuery: query}).String(), nil
}

// openReadOnly opens a database file read-only. SQLite still reads (but never
// checkpoints) a -wal file next to it.
func openReadOnly(path string) (*sql.DB, error) {
	uri, err := sqliteURI(path, "mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", uri)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

// VerifyBackup opens a backup read-only, runs an integrity check and reads its
// migration history. UnknownMigrations lists applied versions this binary does
// not define, which means the backup came from a newer rackd.
func VerifyBackup(ctx context.Context, path string) (*BackupInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("backup not found: %w", err)
	}
	db, err := openReadOnly(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = db.Close() }()

	var result string
	if err := db.QueryRowContext(ctx, "PRAGMA integrity_check").Scan(&result); err != nil {
		return nil, fmt.Errorf("backup %s is not a readable SQLite database: %w", path, err)
	}
	if result != "ok" {
		return nil, fmt.Errorf("backup %s failed the integrity check: %s", path, result)
	}

	versions, err := appliedVersions(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("backup %s has no readable migration history: %w", path, err)
	}
	known := make(map[string]bool, len(migrations))
	for _, m := range migrations {
		known[m.Version] = true
	}
	info := &BackupInfo{Path: path, SizeBytes: fi.Size(), Migrations: len(versions)}
	for _, v := range versions {
		if v > info.LatestMigration {
			info.LatestMigration = v
		}
		if !known[v] {
			info.UnknownMigrations = append(info.UnknownMigrations, v)
		}
	}
	return info, nil
}

func appliedVersions(ctx context.Context, db *sql.DB) ([]string, error) {
	var query string
	switch {
	case tableExists(ctx, db, migrate.DefaultTable):
		query = "SELECT version FROM " + migrate.DefaultTable + " WHERE namespace = '" + CoreMigrationNamespace + "' AND dirty = 0"
	case tableExists(ctx, db, legacyMigrationTable):
		query = "SELECT version FROM " + legacyMigrationTable + " WHERE success = 1"
	default:
		return nil, errors.New("no migration table")
	}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func tableExists(ctx context.Context, db *sql.DB, name string) bool {
	var n int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&n)
	return err == nil && n > 0
}

// RestoreDatabase replaces the database at dbPath with a verified backup. The
// current database is moved aside to <dbPath>.pre-restore-<timestamp> and that
// path is returned. The server must be stopped: the restore fails with
// ErrDatabaseInUse if another connection has the database open. Migrations run
// as usual on the next start, so an older backup is brought up to date.
func RestoreDatabase(ctx context.Context, backupPath, dbPath string) (string, error) {
	info, err := VerifyBackup(ctx, backupPath)
	if err != nil {
		return "", err
	}
	if len(info.UnknownMigrations) > 0 {
		return "", fmt.Errorf("backup was made by a newer rackd version (unknown migrations %s); upgrade rackd before restoring it",
			strings.Join(info.UnknownMigrations, ", "))
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return "", err
	}
	staged, err := stageRestore(ctx, backupPath, dbPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = os.Remove(staged) }()

	var movedTo string
	if _, err := os.Stat(dbPath); err == nil {
		if err := checkpointAndClose(ctx, dbPath); err != nil {
			return "", err
		}
		movedTo = fmt.Sprintf("%s.pre-restore-%s", dbPath, time.Now().UTC().Format(backupTimestampFormat))
		if err := moveDatabase(dbPath, movedTo); err != nil {
			return "", fmt.Errorf("failed to move the current database aside: %w", err)
		}
	}

	if err := os.Rename(staged, dbPath); err != nil {
		if movedTo != "" {
			_ = moveDatabase(movedTo, dbPath)
		}
		return "", fmt.Errorf("failed to put the backup in place: %w", err)
	}
	return movedTo, nil
}

// stageRestore writes the backup, including anything still in a -wal file next to
// it, as one file beside dbPath, verifies it, and switches it to WAL mode so the
// in-use check works the next time a restore runs.
func stageRestore(ctx context.Context, backupPath, dbPath string) (string, error) {
	staged := fmt.Sprintf("%s.restore-%s", dbPath, time.Now().UTC().Format(backupTimestampFormat))
	src, err := openReadOnly(backupPath)
	if err != nil {
		return "", err
	}
	err = vacuumInto(ctx, src, staged)
	_ = src.Close()
	if err != nil {
		return "", err
	}
	if _, err := VerifyBackup(ctx, staged); err != nil {
		_ = os.Remove(staged)
		return "", err
	}
	db, err := sql.Open("sqlite", staged)
	if err != nil {
		_ = os.Remove(staged)
		return "", err
	}
	defer func() { _ = db.Close() }()
	var mode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode=WAL").Scan(&mode); err != nil || mode != "wal" {
		_ = os.Remove(staged)
		return "", fmt.Errorf("failed to enable WAL mode on the restored database: %v", err)
	}
	return staged, nil
}

// moveDatabase renames a database file together with its -wal and -shm files.
func moveDatabase(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(from + suffix); err == nil {
			if err := os.Rename(from+suffix, to+suffix); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkpointAndClose folds the WAL into the main file by leaving WAL mode. SQLite
// refuses to leave WAL mode while another connection is open, which makes this
// the in-use check as well. Rackd always keeps its database in WAL mode (the
// server, the CLI and restore all set it); a database switched to rollback-journal
// mode by another tool cannot be checked for idle connections.
func checkpointAndClose(ctx context.Context, dbPath string) error {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(0)")
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	var mode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode=DELETE").Scan(&mode); err != nil {
		return fmt.Errorf("%w: %v", ErrDatabaseInUse, err)
	}
	if mode != "delete" {
		return ErrDatabaseInUse
	}
	return nil
}
