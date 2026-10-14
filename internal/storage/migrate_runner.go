package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/sqlite"
	"github.com/martinsuchenak/xdal/pkg/xdal"

	"github.com/martinsuchenak/rackd/internal/log"
)

// CoreMigrationNamespace is the migration history namespace for rackd's own schema.
// Extensions use their own namespaces in the same tracking table.
const CoreMigrationNamespace = "core"

// legacyMigrationTable is the tracking table written by rackd v0.0.21 and earlier.
const legacyMigrationTable = "schema_migrations"

// defaultKeepBackups is how many pre-migration snapshots RunMigrationsWithOptions keeps.
const defaultKeepBackups = 5

// preMigrateBackupPrefix names the snapshots taken before migrations run.
const preMigrateBackupPrefix = "rackd-pre-migrate-"

// backupTimestampFormat has fixed-width nanoseconds, so names sort chronologically
// and two backups in the same second do not collide.
const backupTimestampFormat = "20060102T150405.000000000Z"

// legacyAdoptionNamespace holds the marker row written when schema_migrations was adopted.
const legacyAdoptionNamespace = "rackd-legacy-adoption"

// sqlTx is the part of a transaction that migration steps use.
type sqlTx interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// migrationTx adapts the runner's transaction to sqlTx.
type migrationTx struct {
	tx *migrate.Tx
}

func (t migrationTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.tx.Exec(ctx, query, args...)
}

func (t migrationTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.tx.Query(ctx, query, args...)
}

func (t migrationTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRow(ctx, query, args...)
}

// MigrationOptions configures RunMigrationsWithOptions and RollbackMigrations.
type MigrationOptions struct {
	// BackupDir receives a snapshot of an existing database before migrations change it.
	// Empty disables the snapshot.
	BackupDir string
	// KeepBackups is how many pre-migration snapshots to keep in BackupDir. Defaults to 5.
	KeepBackups int
}

// MigrationResult reports what a migration run changed.
type MigrationResult struct {
	// Versions lists the migrations applied or reverted, in the order they ran.
	Versions []string
	// BackupPath is the snapshot taken before the run, if any.
	BackupPath string
}

// MigrationStatus describes the state of a single migration.
type MigrationStatus struct {
	Version   string
	Name      string
	Applied   bool
	Dirty     bool
	Unknown   bool
	AppliedAt string // empty if not applied
}

// migrationLogger forwards migration runner messages to the rackd log.
type migrationLogger struct{}

func (migrationLogger) Trace(msg string, kv ...any) { log.Trace(msg, kv...) }
func (migrationLogger) Debug(msg string, kv ...any) { log.Debug(msg, kv...) }
func (migrationLogger) Info(msg string, kv ...any)  { log.Debug(msg, kv...) }
func (migrationLogger) Warn(msg string, kv ...any)  { log.Warn(msg, kv...) }
func (migrationLogger) Error(msg string, kv ...any) { log.Error(msg, kv...) }
func (migrationLogger) Fatal(msg string, kv ...any) { log.Error(msg, kv...) }

// legacyChecksum reproduces the checksum stored by the pre-xdal runner, so adopted
// history validates against the same migration definitions.
func legacyChecksum(m *Migration) string {
	hash := sha256.Sum256([]byte(m.Version + m.Name))
	return hex.EncodeToString(hash[:8])
}

func adaptStep(step func(context.Context, sqlTx) error) func(context.Context, *migrate.Tx) error {
	if step == nil {
		return nil
	}
	return func(ctx context.Context, tx *migrate.Tx) error {
		return step(ctx, migrationTx{tx: tx})
	}
}

func coreMigrations() []migrate.Migration {
	out := make([]migrate.Migration, len(migrations))
	for i, m := range migrations {
		out[i] = migrate.Migration{
			Version:  m.Version,
			Name:     m.Name,
			Checksum: legacyChecksum(m),
			Up:       adaptStep(m.Up),
			Down:     adaptStep(m.Down),
		}
	}
	return out
}

// newCoreMigrator wraps db for the migration runner.
func newCoreMigrator(db *sql.DB) (*migrate.Migrator, error) {
	return migrate.New(sqlite.NewSQLiteDB(db, nil), coreMigrations(), migrate.Options{
		Namespace: CoreMigrationNamespace,
		Logger:    migrationLogger{},
	})
}

// openCoreMigrator wraps db for the migration runner and adopts any legacy history.
func openCoreMigrator(ctx context.Context, db *sql.DB) (*migrate.Migrator, error) {
	m, err := newCoreMigrator(db)
	if err != nil {
		return nil, err
	}
	if err := adoptLegacyHistory(ctx, db, m); err != nil {
		return nil, fmt.Errorf("failed to adopt migration history from %s: %w", legacyMigrationTable, err)
	}
	return m, nil
}

type legacyRecord struct {
	version, name, checksum string
	appliedAt               time.Time
	durationMS              int64
}

// legacyAdoptionNeeded reports whether schema_migrations exists and its history
// has not been copied yet. Adoption is recorded by a marker row in its own
// namespace, so rolling back every core migration later does not trigger it again.
func legacyAdoptionNeeded(ctx context.Context, db *sql.DB) (bool, error) {
	if !tableExists(ctx, db, legacyMigrationTable) {
		return false, nil
	}
	if !tableExists(ctx, db, migrate.DefaultTable) {
		return true, nil
	}
	var n int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+migrate.DefaultTable+" WHERE namespace = ?", legacyAdoptionNamespace).Scan(&n)
	return n == 0, err
}

func loadLegacyRecords(ctx context.Context, db *sql.DB) ([]legacyRecord, error) {
	rows, err := db.QueryContext(ctx, "SELECT version, name, applied_at, checksum, COALESCE(execution_time_ms, 0) FROM "+legacyMigrationTable+" WHERE success = 1 ORDER BY version")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var records []legacyRecord
	for rows.Next() {
		var r legacyRecord
		if err := rows.Scan(&r.version, &r.name, &r.appliedAt, &r.checksum, &r.durationMS); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// adoptLegacyHistory copies applied versions from schema_migrations into the xdal
// tracking table the first time the new runner opens an existing database, in one
// transaction together with the adoption marker. The legacy table is left in place
// so an older rackd binary can still read it.
func adoptLegacyHistory(ctx context.Context, db *sql.DB, m *migrate.Migrator) error {
	need, err := legacyAdoptionNeeded(ctx, db)
	if err != nil || !need {
		return err
	}
	records, err := loadLegacyRecords(ctx, db)
	if err != nil {
		return err
	}
	if err := m.EnsureTable(ctx); err != nil {
		return err
	}

	xdb := sqlite.NewSQLiteDB(db, nil)
	qb := xdb.NewQueryBuilder()
	insert := func(tx *xdal.Tx, namespace, version, name, checksum string, appliedAt time.Time, durationMS int64) error {
		q, args, err := qb.Insert(migrate.DefaultTable).
			Set("namespace", namespace).
			Set("version", version).
			Set("name", name).
			Set("checksum", checksum).
			Set("applied_at", appliedAt.UTC()).
			Set("duration_ms", durationMS).
			Set("dirty", false).
			OnConflict("namespace", "version").DoNothing().
			Build()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, q, args...)
		return err
	}
	err = xdb.WithTx(ctx, nil, func(tx *xdal.Tx) error {
		for _, r := range records {
			if err := insert(tx, CoreMigrationNamespace, r.version, r.name, r.checksum, r.appliedAt, r.durationMS); err != nil {
				return err
			}
		}
		return insert(tx, legacyAdoptionNamespace, legacyMigrationTable, "adopted", "-", time.Now(), 0)
	})
	if err != nil {
		return err
	}
	log.Info("Adopted migration history", "from", legacyMigrationTable, "to", migrate.DefaultTable, "migrations", len(records))
	return nil
}

// RunMigrations applies all pending core migrations.
func RunMigrations(ctx context.Context, db *sql.DB) error {
	_, err := RunMigrationsWithOptions(ctx, db, MigrationOptions{})
	return err
}

// RunMigrationsWithOptions applies all pending core migrations. When BackupDir is
// set and the database already has applied migrations, a verified snapshot is
// written there first.
func RunMigrationsWithOptions(ctx context.Context, db *sql.DB, opts MigrationOptions) (MigrationResult, error) {
	var res MigrationResult
	m, err := openCoreMigrator(ctx, db)
	if err != nil {
		return res, err
	}
	pending, err := m.Pending(ctx)
	if err != nil {
		return res, migrationError(err, "")
	}
	if len(pending) == 0 {
		return res, nil
	}

	if len(pending) < len(migrations) {
		res.BackupPath, err = preMigrationBackup(ctx, db, opts, "up-"+pending[len(pending)-1].Version)
		if err != nil {
			return res, err
		}
	}

	res.Versions, err = m.Up(ctx)
	if err != nil {
		return res, migrationError(err, res.BackupPath)
	}
	return res, nil
}

// RollbackMigrations reverts core migrations newest first: everything newer than
// toVersion when it is set, otherwise the latest steps migrations. The target is
// validated, and a snapshot is written to BackupDir only when something will be
// reverted. Down steps can drop data; restore the snapshot to undo.
func RollbackMigrations(ctx context.Context, db *sql.DB, steps int, toVersion string, opts MigrationOptions) (MigrationResult, error) {
	if toVersion != "" && !knownMigration(toVersion) {
		return MigrationResult{}, fmt.Errorf("%w: %s", migrate.ErrUnknownVersion, toVersion)
	}
	if steps < 0 {
		return MigrationResult{}, fmt.Errorf("%w: negative step count %d", migrate.ErrInvalid, steps)
	}
	m, err := openCoreMigrator(ctx, db)
	if err != nil {
		return MigrationResult{}, err
	}
	return rollback(ctx, db, m, steps, toVersion, opts, "")
}

// rollback reverts the latest steps migrations of m, or every one newer than
// toVersion, after a backup. labelPrefix names the namespace in the backup file.
func rollback(ctx context.Context, db *sql.DB, m *migrate.Migrator, steps int, toVersion string, opts MigrationOptions, labelPrefix string) (MigrationResult, error) {
	var res MigrationResult
	statuses, err := m.Status(ctx)
	if err != nil {
		return res, err
	}
	var applied []string
	for _, s := range statuses {
		switch {
		case s.Dirty:
			return res, fmt.Errorf("%w: %s %s", migrate.ErrDirty, s.Version, s.Name)
		case s.Unknown:
			return res, migrationError(fmt.Errorf("%w: %s %s", migrate.ErrUnknownApplied, s.Version, s.Name), "")
		case s.Applied:
			applied = append(applied, s.Version)
		}
	}
	count := 0
	if toVersion != "" {
		for _, v := range applied {
			if v > toVersion {
				count++
			}
		}
	} else {
		count = min(steps, len(applied))
	}
	if count == 0 {
		return res, nil
	}

	label := fmt.Sprintf("%sdown-%d", labelPrefix, count)
	if toVersion != "" {
		label = labelPrefix + "down-to-" + toVersion
	}
	res.BackupPath, err = preMigrationBackup(ctx, db, opts, label)
	if err != nil {
		return res, err
	}

	if toVersion != "" {
		res.Versions, err = m.DownTo(ctx, toVersion)
	} else {
		res.Versions, err = m.Down(ctx, count)
	}
	if err != nil {
		return res, migrationError(err, res.BackupPath)
	}
	return res, nil
}

func knownMigration(version string) bool {
	for _, m := range migrations {
		if m.Version == version {
			return true
		}
	}
	return false
}

// GetMigrationStatus returns the status of every core migration, plus any applied
// migration this binary does not know about. It does not write to the database:
// history that has not been adopted yet is read from schema_migrations directly.
func GetMigrationStatus(ctx context.Context, db *sql.DB) ([]MigrationStatus, error) {
	need, err := legacyAdoptionNeeded(ctx, db)
	if err != nil {
		return nil, err
	}
	if need || !tableExists(ctx, db, migrate.DefaultTable) {
		var records []legacyRecord
		if need {
			if records, err = loadLegacyRecords(ctx, db); err != nil {
				return nil, err
			}
		}
		return statusFromLegacy(records), nil
	}

	m, err := newCoreMigrator(db)
	if err != nil {
		return nil, err
	}
	return migratorStatus(ctx, m)
}

func migratorStatus(ctx context.Context, m *migrate.Migrator) ([]MigrationStatus, error) {
	statuses, err := m.Status(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]MigrationStatus, 0, len(statuses))
	for _, s := range statuses {
		ms := MigrationStatus{Version: s.Version, Name: s.Name, Applied: s.Applied, Dirty: s.Dirty, Unknown: s.Unknown}
		if !s.AppliedAt.IsZero() {
			ms.AppliedAt = s.AppliedAt.UTC().Format("2006-01-02 15:04:05")
		}
		out = append(out, ms)
	}
	return out, nil
}

func statusFromLegacy(records []legacyRecord) []MigrationStatus {
	applied := make(map[string]legacyRecord, len(records))
	for _, r := range records {
		applied[r.version] = r
	}
	out := make([]MigrationStatus, 0, len(migrations))
	for _, m := range migrations {
		s := MigrationStatus{Version: m.Version, Name: m.Name}
		if r, ok := applied[m.Version]; ok {
			s.Applied = true
			s.AppliedAt = r.appliedAt.UTC().Format("2006-01-02 15:04:05")
			delete(applied, m.Version)
		}
		out = append(out, s)
	}
	for _, r := range applied {
		out = append(out, MigrationStatus{Version: r.version, Name: r.name, Applied: true, Unknown: true,
			AppliedAt: r.appliedAt.UTC().Format("2006-01-02 15:04:05")})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out
}

func migrationError(err error, backupPath string) error {
	if errors.Is(err, migrate.ErrUnknownApplied) {
		err = fmt.Errorf("%w (the database was migrated by a newer rackd version)", err)
	}
	if backupPath != "" {
		return fmt.Errorf("%w; the database before this run is saved at %s", err, backupPath)
	}
	return err
}

func preMigrationBackup(ctx context.Context, db *sql.DB, opts MigrationOptions, label string) (string, error) {
	if opts.BackupDir == "" {
		return "", nil
	}
	path := filepath.Join(opts.BackupDir, fmt.Sprintf("%s%s-%s.db", preMigrateBackupPrefix, time.Now().UTC().Format(backupTimestampFormat), label))
	if _, err := BackupDatabase(ctx, db, path); err != nil {
		return "", fmt.Errorf("pre-migration backup failed, no migrations were run (set PRE_MIGRATION_BACKUP=false to skip it): %w", err)
	}
	log.Info("Database backed up before migration", "path", path)
	keep := opts.KeepBackups
	if keep <= 0 {
		keep = defaultKeepBackups
	}
	if err := pruneBackups(opts.BackupDir, preMigrateBackupPrefix, keep); err != nil {
		log.Warn("Failed to prune old pre-migration backups", "dir", opts.BackupDir, "error", err)
	}
	return path, nil
}

// pruneBackups deletes the oldest files named prefix* in dir, keeping the newest keep.
// Names embed a UTC timestamp, so lexical order is chronological.
func pruneBackups(dir, prefix string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".db") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for len(names) > keep {
		if err := os.Remove(filepath.Join(dir, names[0])); err != nil {
			return err
		}
		names = names[1:]
	}
	return nil
}

// RequireTables reports a clear error when a store is opened before migrations ran.
func RequireTables(db *sql.DB, names ...string) error {
	for _, name := range names {
		if !tableExists(context.Background(), db, name) {
			return fmt.Errorf("table %s not found; run the database migrations first", name)
		}
	}
	return nil
}
