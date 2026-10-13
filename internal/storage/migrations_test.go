package storage

import (
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	_ "modernc.org/sqlite"
)

func openMemDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func openFileDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countCoreRecords(t *testing.T, db *sql.DB) int {
	t.Helper()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM "+migrate.DefaultTable+" WHERE namespace = ?", CoreMigrationNamespace).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count migration records: %v", err)
	}
	return count
}

func hasTable(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	return tableExists(context.Background(), db, name)
}

// legacyFixtureMigrations is the number of migrations applied in the v0.0.21 fixture.
const legacyFixtureMigrations = 35

// extractLegacyFixture writes the rackd v0.0.21 database fixture to a temp file.
func extractLegacyFixture(t *testing.T) string {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "fixtures", "legacy-v0.0.21.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rackd.db")
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, zr); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunMigrations(t *testing.T) {
	db := openMemDB(t)
	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	if got := countCoreRecords(t, db); got != len(migrations) {
		t.Errorf("expected %d migration records, got %d", len(migrations), got)
	}
	for _, table := range []string{
		"datacenters", "networks", "network_pools", "devices", "addresses", "tags", "domains",
		"device_relationships", "discovered_devices", "discovery_scans", "discovery_rules",
	} {
		if !hasTable(t, db, table) {
			t.Errorf("table %s not found", table)
		}
	}
	if hasTable(t, db, legacyMigrationTable) {
		t.Errorf("a fresh database should not get the legacy %s table", legacyMigrationTable)
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("first RunMigrations failed: %v", err)
	}
	res, err := RunMigrationsWithOptions(ctx, db, MigrationOptions{BackupDir: t.TempDir()})
	if err != nil {
		t.Fatalf("second RunMigrations failed: %v", err)
	}
	if len(res.Versions) != 0 || res.BackupPath != "" {
		t.Errorf("second run should be a no-op, got %+v", res)
	}
	if got := countCoreRecords(t, db); got != len(migrations) {
		t.Errorf("expected %d migration records, got %d", len(migrations), got)
	}
}

func TestMigrationStatus(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()

	statuses, err := GetMigrationStatus(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range statuses {
		if s.Applied {
			t.Fatalf("fresh database reports %s as applied", s.Version)
		}
	}

	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	statuses, err = GetMigrationStatus(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != len(migrations) {
		t.Fatalf("expected %d statuses, got %d", len(migrations), len(statuses))
	}
	for _, s := range statuses {
		if !s.Applied || s.Dirty || s.Unknown || s.AppliedAt == "" || s.Name == "" {
			t.Errorf("unexpected status %+v", s)
		}
	}
}

func TestForeignKeyConstraints(t *testing.T) {
	db := openMemDB(t)
	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}
	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled); err != nil {
		t.Fatalf("failed to query foreign_keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Error("foreign keys should be enabled")
	}
	_, err := db.Exec(`INSERT INTO devices (id, name, datacenter_id) VALUES ('test-id', 'test-device', 'non-existent-dc')`)
	if err == nil {
		t.Error("expected the datacenter foreign key to reject a missing datacenter")
	}
}

func TestIndexesCreated(t *testing.T) {
	db := openMemDB(t)
	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}
	for _, idx := range []string{
		"idx_devices_name", "idx_devices_datacenter", "idx_addresses_device", "idx_addresses_ip",
		"idx_networks_datacenter", "idx_discovered_devices_network",
	} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, idx).Scan(&name); err != nil {
			t.Errorf("index %s not found: %v", idx, err)
		}
	}
}

func TestAdoptLegacyHistoryFromV0021(t *testing.T) {
	path := extractLegacyFixture(t)
	db := openFileDB(t, path)
	ctx := context.Background()

	backups := t.TempDir()
	res, err := RunMigrationsWithOptions(ctx, db, MigrationOptions{BackupDir: backups})
	if err != nil {
		t.Fatalf("upgrade from v0.0.21 failed: %v", err)
	}
	newer := migrations[legacyFixtureMigrations:]
	if len(res.Versions) != len(newer) || res.Versions[0] != newer[0].Version {
		t.Errorf("expected the migrations added after v0.0.21 to run, got %+v", res.Versions)
	}
	if res.BackupPath == "" {
		t.Error("upgrading an existing database should take a backup")
	} else if info, err := VerifyBackup(ctx, res.BackupPath); err != nil || info.Migrations != legacyFixtureMigrations {
		t.Errorf("pre-upgrade backup = %+v, %v", info, err)
	}
	if got := countCoreRecords(t, db); got != len(migrations) {
		t.Errorf("recorded %d migrations, want %d", got, len(migrations))
	}

	var legacy int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + legacyMigrationTable).Scan(&legacy); err != nil || legacy != legacyFixtureMigrations {
		t.Errorf("legacy table should be left intact: %d rows, %v", legacy, err)
	}
	var name string
	if err := db.QueryRow("SELECT name FROM devices WHERE id = 'dev-1'").Scan(&name); err != nil || name != "web-01" {
		t.Errorf("fixture data lost: %q, %v", name, err)
	}

	if _, err := RunMigrationsWithOptions(ctx, db, MigrationOptions{}); err != nil {
		t.Fatalf("second run after adoption failed: %v", err)
	}
	if got := countCoreRecords(t, db); got != len(migrations) {
		t.Errorf("adoption ran twice: %d records", got)
	}
}

func TestLegacyFixtureOpensWithStorage(t *testing.T) {
	path := extractLegacyFixture(t)
	s, err := NewSQLiteStorageWithPath(path)
	if err != nil {
		t.Fatalf("NewSQLiteStorageWithPath on a v0.0.21 database: %v", err)
	}
	defer func() { _ = s.Close() }()
	var n int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM datacenters WHERE id = 'dc-syd'").Scan(&n); err != nil || n != 1 {
		t.Errorf("fixture datacenter missing: %d, %v", n, err)
	}
}

func TestEveryDownMigrationReverts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rackd.db")
	db := openFileDB(t, path)
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}

	res, err := RollbackMigrations(ctx, db, 0, migrations[0].Version, MigrationOptions{})
	if err != nil {
		t.Fatalf("rolling back to the initial schema failed: %v", err)
	}
	if len(res.Versions) != len(migrations)-1 {
		t.Errorf("reverted %d migrations, want %d", len(res.Versions), len(migrations)-1)
	}
	if _, err := RollbackMigrations(ctx, db, 1, "", MigrationOptions{}); err != nil {
		t.Fatalf("reverting the initial schema failed: %v", err)
	}
	if hasTable(t, db, "devices") {
		t.Error("devices table should be gone after a full rollback")
	}

	res, err = RunMigrationsWithOptions(ctx, db, MigrationOptions{})
	if err != nil {
		t.Fatalf("re-applying after a full rollback failed: %v", err)
	}
	if len(res.Versions) != len(migrations) {
		t.Errorf("re-applied %d migrations, want %d", len(res.Versions), len(migrations))
	}
}

func TestRollbackAndUpgradeTakeBackups(t *testing.T) {
	dir := t.TempDir()
	db := openFileDB(t, filepath.Join(dir, "rackd.db"))
	ctx := context.Background()
	backups := filepath.Join(dir, "backups")
	opts := MigrationOptions{BackupDir: backups}

	res, err := RunMigrationsWithOptions(ctx, db, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.BackupPath != "" {
		t.Errorf("a fresh database should not be backed up, got %s", res.BackupPath)
	}

	latest := migrations[len(migrations)-1].Version
	res, err = RollbackMigrations(ctx, db, 1, "", opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Versions) != 1 || res.Versions[0] != latest || res.BackupPath == "" {
		t.Fatalf("rollback result %+v", res)
	}
	info, err := VerifyBackup(ctx, res.BackupPath)
	if err != nil {
		t.Fatalf("pre-rollback backup does not verify: %v", err)
	}
	if info.Migrations != len(migrations) || info.LatestMigration != latest {
		t.Errorf("pre-rollback backup should hold the full history: %+v", info)
	}

	res, err = RunMigrationsWithOptions(ctx, db, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Versions) != 1 || res.BackupPath == "" {
		t.Errorf("upgrade result %+v", res)
	}
	entries, _ := os.ReadDir(backups)
	if len(entries) != 2 {
		t.Errorf("expected 2 backups, found %d", len(entries))
	}
}

func TestNewerDatabaseIsRefused(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec("INSERT INTO "+migrate.DefaultTable+" (namespace, version, name, checksum, applied_at, duration_ms, dirty) VALUES (?, '29990101000000', 'from_the_future', 'x', CURRENT_TIMESTAMP, 0, 0)", CoreMigrationNamespace)
	if err != nil {
		t.Fatal(err)
	}
	err = RunMigrations(ctx, db)
	if !errors.Is(err, migrate.ErrUnknownApplied) || !strings.Contains(err.Error(), "newer rackd") {
		t.Errorf("err = %v", err)
	}
}

func TestPruneBackupsKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	names := []string{"20260101T000000Z", "20260102T000000Z", "20260103T000000Z", "20260104T000000Z"}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, preMigrateBackupPrefix+n+"-up-1.db"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "manual.db"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := pruneBackups(dir, preMigrateBackupPrefix, 2); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{"manual.db", preMigrateBackupPrefix + names[2] + "-up-1.db", preMigrateBackupPrefix + names[3] + "-up-1.db"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("after prune: %v, want %v", got, want)
	}
}

// schemaSnapshot describes every table (columns), index and trigger, plus row
// counts, so a migration's Down step can be checked against the state before its Up.
func schemaSnapshot(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query(`SELECT type, name, tbl_name FROM sqlite_master
		WHERE name NOT LIKE 'sqlite_%' AND name <> ? ORDER BY type, name`, migrate.DefaultTable)
	if err != nil {
		t.Fatal(err)
	}
	type object struct{ typ, name, table string }
	var objects []object
	for rows.Next() {
		var o object
		if err := rows.Scan(&o.typ, &o.name, &o.table); err != nil {
			t.Fatal(err)
		}
		objects = append(objects, o)
	}
	_ = rows.Close()

	var b strings.Builder
	for _, o := range objects {
		fmt.Fprintf(&b, "%s %s on %s\n", o.typ, o.name, o.table)
		switch o.typ {
		case "table":
			cols, err := db.Query(`SELECT name, type, "notnull", COALESCE(dflt_value, ''), pk FROM pragma_table_info(?) ORDER BY cid`, o.name)
			if err != nil {
				t.Fatal(err)
			}
			for cols.Next() {
				var name, typ, def string
				var notNull, pk int
				if err := cols.Scan(&name, &typ, &notNull, &def, &pk); err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(&b, "  col %s %s notnull=%d default=%s pk=%d\n", name, typ, notNull, def, pk)
			}
			_ = cols.Close()
			fks, err := db.Query(`SELECT "table", "from", "to", on_delete FROM pragma_foreign_key_list(?) ORDER BY "from"`, o.name)
			if err != nil {
				t.Fatal(err)
			}
			for fks.Next() {
				var tbl, from, to, onDelete string
				var toNull sql.NullString
				if err := fks.Scan(&tbl, &from, &toNull, &onDelete); err != nil {
					t.Fatal(err)
				}
				to = toNull.String
				fmt.Fprintf(&b, "  fk %s -> %s.%s on delete %s\n", from, tbl, to, onDelete)
			}
			_ = fks.Close()
			if !strings.Contains(o.name, "_fts") {
				var n int
				if err := db.QueryRow(`SELECT COUNT(*) FROM "` + o.name + `"`).Scan(&n); err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(&b, "  rows %d\n", n)
			}
		case "index":
			cols, err := db.Query(`SELECT COALESCE(name, '') FROM pragma_index_info(?) ORDER BY seqno`, o.name)
			if err != nil {
				t.Fatal(err)
			}
			for cols.Next() {
				var name string
				if err := cols.Scan(&name); err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(&b, "  on %s\n", name)
			}
			_ = cols.Close()
		}
	}
	return b.String()
}

func TestEachMigrationIsReversible(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)
	m, err := openCoreMigrator(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, mig := range migrations {
		before := schemaSnapshot(t, db)
		if _, err := m.UpTo(ctx, mig.Version); err != nil {
			t.Fatalf("up %s: %v", mig.Version, err)
		}
		if _, err := m.Down(ctx, 1); err != nil {
			t.Fatalf("down %s %s: %v", mig.Version, mig.Name, err)
		}
		if after := schemaSnapshot(t, db); after != before {
			t.Errorf("%s %s is not reversed by its Down step:\n%s", mig.Version, mig.Name, snapshotDiff(before, after))
		}
		if _, err := m.UpTo(ctx, mig.Version); err != nil {
			t.Fatalf("re-apply %s: %v", mig.Version, err)
		}
	}
}

func snapshotDiff(before, after string) string {
	b := strings.Split(before, "\n")
	a := strings.Split(after, "\n")
	inB := map[string]bool{}
	for _, l := range b {
		inB[l] = true
	}
	inA := map[string]bool{}
	for _, l := range a {
		inA[l] = true
	}
	var out []string
	for _, l := range b {
		if !inA[l] {
			out = append(out, "- "+l)
		}
	}
	for _, l := range a {
		if !inB[l] {
			out = append(out, "+ "+l)
		}
	}
	return strings.Join(out, "\n")
}

func TestAdoptKeepsExistingDiscoveryTables(t *testing.T) {
	path := extractLegacyFixture(t)
	db := openFileDB(t, path)
	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE scan_profiles (id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, scan_type TEXT NOT NULL, ports TEXT,
			enable_snmp INTEGER DEFAULT 0, enable_ssh INTEGER DEFAULT 0, timeout_sec INTEGER DEFAULT 30, max_workers INTEGER DEFAULT 10,
			description TEXT, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
		`INSERT INTO scan_profiles (id, name, scan_type, ports) VALUES ('custom', 'My Scan', 'quick', '[22]')`,
		`CREATE TABLE credentials (id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, snmp_community TEXT, snmp_v3_user TEXT,
			snmp_v3_auth TEXT, snmp_v3_priv TEXT, ssh_username TEXT, ssh_key_id TEXT, datacenter_id TEXT, description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`,
		`INSERT INTO credentials (id, name, type) VALUES ('cred-1', 'core switches', 'snmp_v2c')`,
	}
	for _, q := range stmts {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := RunMigrationsWithOptions(ctx, db, MigrationOptions{}); err != nil {
		t.Fatalf("upgrade with existing discovery tables: %v", err)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM scan_profiles`).Scan(&n)
	if n != 4 {
		t.Errorf("expected the custom profile plus 3 defaults, found %d profiles", n)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM credentials WHERE id = 'cred-1'`).Scan(&n)
	if n != 1 {
		t.Error("existing credential lost")
	}
	if !hasTable(t, db, "scheduled_scans") {
		t.Error("scheduled_scans not created")
	}
}

func TestStoresRequireMigrations(t *testing.T) {
	db := openMemDB(t)
	if _, err := NewSQLiteProfileStorage(db); err == nil || !strings.Contains(err.Error(), "migrations") {
		t.Errorf("profile store on an unmigrated database: %v", err)
	}
	if _, err := NewSQLiteScheduledScanStorage(db); err == nil || !strings.Contains(err.Error(), "migrations") {
		t.Errorf("scheduled scan store on an unmigrated database: %v", err)
	}
}

func TestRebuildDownStepsKeepRows(t *testing.T) {
	ctx := context.Background()
	db := openMemDB(t)
	m, err := openCoreMigrator(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	seed := []string{
		`INSERT INTO api_keys (id, name, key, created_at) VALUES ('k1', 'ci', 'hash', CURRENT_TIMESTAMP)`,
		`INSERT INTO dns_provider_configs (id, name, type, endpoint, token, created_at, updated_at) VALUES ('p1', 'tech', 'technitium', 'http://dns', 'x', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		`INSERT INTO dns_zones (id, name, provider_id, created_at, updated_at) VALUES ('z1', 'example.com', 'p1', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		`INSERT INTO dns_records (id, zone_id, name, type, value, ttl, created_at, updated_at) VALUES ('r1', 'z1', 'www', 'A', '10.0.0.1', 300, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
	}
	for _, q := range seed {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := m.DownTo(ctx, "20260302000000"); err != nil {
		t.Fatalf("reverting dns_records.address_id: %v", err)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM dns_records WHERE id = 'r1' AND value = '10.0.0.1'`).Scan(&n)
	if n != 1 {
		t.Error("dns_records rebuild lost a row")
	}

	if _, err := m.DownTo(ctx, "20260207110000"); err != nil {
		t.Fatalf("reverting api_keys.user_id: %v", err)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE id = 'k1' AND key = 'hash'`).Scan(&n)
	if n != 1 {
		t.Error("api_keys rebuild lost a row")
	}
	if _, err := db.Exec(`INSERT INTO api_keys (id, name, key, created_at) VALUES ('k2', 'ci', 'other', CURRENT_TIMESTAMP)`); err == nil {
		t.Error("api_keys lost its UNIQUE(name) constraint in the rebuild")
	}
}

func TestFullRollbackDoesNotReadoptLegacyHistory(t *testing.T) {
	path := extractLegacyFixture(t)
	db := openFileDB(t, path)
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := RollbackMigrations(ctx, db, len(migrations), "", MigrationOptions{}); err != nil {
		t.Fatalf("full rollback: %v", err)
	}
	statuses, err := GetMigrationStatus(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range statuses {
		if s.Applied {
			t.Fatalf("%s reported as applied after a full rollback", s.Version)
		}
	}
	res, err := RunMigrationsWithOptions(ctx, db, MigrationOptions{})
	if err != nil {
		t.Fatalf("re-applying after a full rollback: %v", err)
	}
	if len(res.Versions) != len(migrations) || !hasTable(t, db, "devices") {
		t.Errorf("expected the whole schema to be rebuilt, applied %d", len(res.Versions))
	}
}

func TestMigrationStatusDoesNotWrite(t *testing.T) {
	path := extractLegacyFixture(t)
	db := openFileDB(t, path)
	ctx := context.Background()
	statuses, err := GetMigrationStatus(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if hasTable(t, db, migrate.DefaultTable) {
		t.Error("status created the xdal tracking table")
	}
	applied := 0
	for _, s := range statuses {
		if s.Applied {
			applied++
		}
	}
	if applied != legacyFixtureMigrations || len(statuses) != len(migrations) {
		t.Errorf("status from legacy history: %d applied of %d listed", applied, len(statuses))
	}
}

func TestRollbackValidatesBeforeBackup(t *testing.T) {
	dir := t.TempDir()
	db := openFileDB(t, filepath.Join(dir, "rackd.db"))
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	backups := filepath.Join(dir, "backups")
	opts := MigrationOptions{BackupDir: backups}

	if _, err := RollbackMigrations(ctx, db, 0, "2026typo", opts); !errors.Is(err, migrate.ErrUnknownVersion) {
		t.Errorf("unknown --to: err = %v", err)
	}
	res, err := RollbackMigrations(ctx, db, 0, migrations[len(migrations)-1].Version, opts)
	if err != nil || len(res.Versions) != 0 || res.BackupPath != "" {
		t.Errorf("rollback to the latest version should do nothing: %+v, %v", res, err)
	}
	if _, err := os.Stat(backups); !os.IsNotExist(err) {
		t.Error("a rollback that reverts nothing must not write a backup")
	}
}

func TestPreMigrationBackupNamesAreUnique(t *testing.T) {
	db := openMemDB(t)
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	opts := MigrationOptions{BackupDir: t.TempDir()}
	a, err := preMigrationBackup(ctx, db, opts, "down-1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := preMigrationBackup(ctx, db, opts, "down-1")
	if err != nil {
		t.Fatalf("second backup in the same second: %v", err)
	}
	if a == b {
		t.Errorf("both backups named %s", a)
	}
}

func TestSkipPreMigrationBackup(t *testing.T) {
	path := extractLegacyFixture(t)
	s, err := NewSQLiteStorageWithOptions(filepath.Dir(path), StorageOptions{SkipPreMigrationBackup: true})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "backups")); !os.IsNotExist(err) {
		t.Error("backup written although SkipPreMigrationBackup was set")
	}
}
