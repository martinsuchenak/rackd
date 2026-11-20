package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/sqlite"
	"github.com/martinsuchenak/xdal/pkg/xdal"

	"github.com/martinsuchenak/rackd/internal/storage/dialecttest"
)

// baselineOnly is a track whose single step builds the baseline whatever the
// dialect, so it can be compared with the legacy track on SQLite.
func baselineOnly() []migrate.Migration {
	return []migrate.Migration{{Version: baselineVersion, Name: "baseline", Up: createBaseline}}
}

// legacyOnlyIndexes are in the legacy schema but deliberately left out of the
// baseline; baselineStatements says why.
var legacyOnlyIndexes = map[string]bool{"idx_cfv_string": true}

// The baseline builds the same schema the legacy track does: tables, columns,
// nullability, defaults, keys, indexes and seed rows. Column types are
// compared by SQLite affinity, with the baseline's BOOLEAN flags matching the
// legacy INTEGER ones.
func TestBaselineMatchesLegacySchema(t *testing.T) {
	ctx := context.Background()

	legacy, err := NewSQLiteStorageWithOptions(t.TempDir(), StorageOptions{SkipPreMigrationBackup: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = legacy.Close() }()

	baseDB, err := openDB(filepath.Join(t.TempDir(), "baseline.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = baseDB.Close() }()
	m, err := migrate.New(sqlite.NewSQLiteDB(baseDB, nil), baselineOnly(), migrate.Options{Namespace: SchemaMigrationNamespace})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("baseline on SQLite: %v", err)
	}

	want := describeSchema(t, legacy.db, true)
	got := describeSchema(t, baseDB, false)
	if diff := diffLines(want, got); diff != "" {
		t.Errorf("baseline schema differs from the legacy schema (- legacy, + baseline):\n%s", diff)
	}

	if diff := diffLines(describeSeed(t, legacy.db), describeSeed(t, baseDB)); diff != "" {
		t.Errorf("baseline seed differs from the legacy seed (- legacy, + baseline):\n%s", diff)
	}
}

// A legacy rollback is refused once portable migrations after the baseline
// are applied, and allowed while only the baseline is.
func TestRollbackStopsAtPortableMigrations(t *testing.T) {
	ctx := context.Background()
	s, err := NewSQLiteStorageWithOptions(t.TempDir(), StorageOptions{SkipPreMigrationBackup: true})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if err := requireOnlyBaseline(ctx, s.db); err != nil {
		t.Fatalf("only the baseline is applied: %v", err)
	}
	q, args, err := xdal.NewQueryBuilder(sqlite.NewDialect()).Insert(migrate.DefaultTable).
		Set("namespace", SchemaMigrationNamespace).Set("version", "29990101000000").Set("name", "later").
		Set("checksum", "-").Set("applied_at", time.Now().UTC()).Set("duration_ms", 0).Set("dirty", false).Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, q, args...); err != nil {
		t.Fatal(err)
	}
	if _, err := RollbackMigrations(ctx, s.db, 1, "", MigrationOptions{}); !errors.Is(err, migrate.ErrInvalid) {
		t.Errorf("rollback under a later schema migration: %v, want ErrInvalid", err)
	}
}

// legacyMigrationCount is the length of the frozen legacy track.
const legacyMigrationCount = 48

// The legacy track is frozen at the baseline: new schema changes go in
// portableMigrations, written with xdal's schema builder.
func TestLegacyTrackIsFrozen(t *testing.T) {
	if len(migrations) != legacyMigrationCount || migrations[len(migrations)-1].Version >= baselineVersion {
		t.Fatalf("the legacy migration track (migrations.go) changed: add schema changes to portableMigrations in baseline.go instead")
	}
}

// The baseline applies on every server backend, seeds the same rows, and
// running the portable track again changes nothing.
func TestDialectBaseline(t *testing.T) {
	dialecttest.Run(t, SchemaMigrationNamespace, portableMigrations(), func(t *testing.T, db *dialecttest.DB) {
		ctx := context.Background()
		count := func(table string) int {
			t.Helper()
			q, args, err := db.QueryBuilder().Select("COUNT(*)").From(table).Build()
			if err != nil {
				t.Fatal(err)
			}
			var n int
			if err := db.DB.QueryRow(ctx, q, args...).Scan(&n); err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			return n
		}
		if db.Name == xdal.DialectSQLite {
			// The legacy track builds SQLite; the baseline only records itself.
			if n := count(migrate.DefaultTable); n != 1 {
				t.Errorf("history rows = %d, want 1", n)
			}
			return
		}
		if got, want := count("permissions"), len(baselinePermissions); got != want {
			t.Errorf("permissions = %d, want %d", got, want)
		}
		if got, want := count("roles"), len(baselineRoles); got != want {
			t.Errorf("roles = %d, want %d", got, want)
		}
		if got := count("scan_profiles"); got != len(baselineScanProfiles) {
			t.Errorf("scan profiles = %d, want %d", got, len(baselineScanProfiles))
		}

		// The partial unique index lets an IP be reserved again once released.
		qb := db.QueryBuilder
		exec := func(b interface {
			Build() (string, []interface{}, error)
		}) error {
			q, args, err := b.Build()
			if err != nil {
				return err
			}
			_, err = db.DB.Exec(ctx, q, args...)
			return err
		}
		must := func(err error) {
			t.Helper()
			if err != nil {
				t.Fatal(err)
			}
		}
		must(exec(qb().Insert("networks").Set("id", "n1").Set("name", "lan").Set("subnet", "10.0.0.0/24")))
		must(exec(qb().Insert("network_pools").Set("id", "p1").Set("network_id", "n1").Set("name", "pool").
			Set("start_ip", "10.0.0.10").Set("end_ip", "10.0.0.20")))
		reserve := func(id, status string) error {
			return exec(qb().Insert("reservations").Set("id", id).Set("pool_id", "p1").Set("ip_address", "10.0.0.11").
				Set("reserved_by", "test").Set("status", status))
		}
		must(reserve("r1", "released"))
		must(reserve("r2", "active"))
		if err := reserve("r3", "active"); err == nil {
			t.Error("a second active reservation of the same IP was accepted")
		}

		m, err := migrate.New(db.DB, portableMigrations(), migrate.Options{Namespace: SchemaMigrationNamespace})
		if err != nil {
			t.Fatal(err)
		}
		if applied, err := m.Up(ctx); err != nil || len(applied) != 0 {
			t.Errorf("second run applied %v, %v", applied, err)
		}
	})
}

// describeSchema renders a SQLite schema as sorted lines, one per column, key
// and index, normalised so the legacy and baseline schemas compare equal.
func describeSchema(t *testing.T, db *sql.DB, legacy bool) []string {
	t.Helper()
	var lines []string
	for _, table := range queryStrings(t, db, `SELECT name FROM sqlite_master WHERE type = 'table'
		AND name NOT LIKE 'sqlite_%' AND name NOT LIKE '%_fts%' AND name != ?`, migrate.DefaultTable) {
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%q)", table))
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var cid, pk int
			var name, typ string
			var notNull bool
			var def sql.NullString
			if err := rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
				t.Fatal(err)
			}
			lines = append(lines, fmt.Sprintf("%s.%s %s notnull=%v pk=%d default=%s",
				table, name, affinity(typ), notNull || pk > 0, pk, normDefault(def)))
		}
		_ = rows.Close()

		rows, err = db.Query(fmt.Sprintf("PRAGMA foreign_key_list(%q)", table))
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var id, seq int
			var ref, from, to, onUpdate, onDelete, match string
			if err := rows.Scan(&id, &seq, &ref, &from, &to, &onUpdate, &onDelete, &match); err != nil {
				t.Fatal(err)
			}
			if onDelete == "RESTRICT" {
				onDelete = "NO ACTION"
			}
			lines = append(lines, fmt.Sprintf("%s.%s -> %s.%s on delete %s", table, from, ref, to, onDelete))
		}
		_ = rows.Close()

		type index struct {
			name            string
			unique, partial bool
		}
		var indexes []index
		rows, err = db.Query(fmt.Sprintf("PRAGMA index_list(%q)", table))
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var seq int
			var ix index
			var origin string
			if err := rows.Scan(&seq, &ix.name, &ix.unique, &origin, &ix.partial); err != nil {
				t.Fatal(err)
			}
			if origin == "pk" || (legacy && legacyOnlyIndexes[ix.name]) {
				continue
			}
			indexes = append(indexes, ix)
		}
		_ = rows.Close()
		for _, ix := range indexes {
			cols := queryStrings(t, db, fmt.Sprintf("SELECT name FROM pragma_index_info(%q) ORDER BY seqno", ix.name))
			name := ix.name
			if strings.HasPrefix(name, "sqlite_autoindex_") {
				name = "(constraint)" // named by SQLite, differently for each schema
			}
			lines = append(lines, fmt.Sprintf("%s index %s (%s) unique=%v partial=%v",
				table, name, strings.Join(cols, ", "), ix.unique, ix.partial))
		}
	}
	sort.Strings(lines)
	return lines
}

// describeSeed renders the seeded rows without their generated ids and times.
func describeSeed(t *testing.T, db *sql.DB) []string {
	t.Helper()
	var lines []string
	lines = append(lines, queryStrings(t, db, `SELECT 'permission ' || name || ' ' || resource || ' ' || action FROM permissions`)...)
	lines = append(lines, queryStrings(t, db, `SELECT 'role ' || name || ': ' || COALESCE(description, '') || ' system=' || is_system FROM roles`)...)
	lines = append(lines, queryStrings(t, db, `SELECT 'grant ' || r.name || ' ' || p.name FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id JOIN permissions p ON p.id = rp.permission_id`)...)
	lines = append(lines, queryStrings(t, db, `SELECT 'scan profile ' || id || ' ' || name || ' ' || scan_type || ' ' || ports ||
		' snmp=' || enable_snmp || ' ssh=' || enable_ssh || ' timeout=' || timeout_sec || ' workers=' || max_workers || ' ' || COALESCE(description, '')
		FROM scan_profiles`)...)
	// The default datacenter is created at startup, not by a migration.
	for _, table := range []string{"users", "devices", "networks", "api_keys", "webhooks"} {
		lines = append(lines, queryStrings(t, db, `SELECT ? || ' rows: ' || COUNT(*) FROM `+table, table)...)
	}
	sort.Strings(lines)
	return lines
}

func affinity(typ string) string {
	switch typ = strings.ToUpper(typ); {
	case typ == "BOOLEAN":
		return "INTEGER"
	case typ == "DATETIME":
		return "TIMESTAMP"
	case strings.HasPrefix(typ, "VARCHAR"):
		return "TEXT"
	default:
		return typ
	}
}

func normDefault(d sql.NullString) string {
	if !d.Valid {
		return "none"
	}
	switch v := strings.Trim(d.String, "()"); strings.ToUpper(v) {
	case "TRUE":
		return "1"
	case "FALSE":
		return "0"
	default:
		return v
	}
}

func queryStrings(t *testing.T, db *sql.DB, q string, args ...any) []string {
	t.Helper()
	rows, err := db.Query(q, args...)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// diffLines lists the lines only in want (-) or only in got (+).
func diffLines(want, got []string) string {
	var b strings.Builder
	for _, l := range want {
		if !slices.Contains(got, l) {
			b.WriteString("- " + l + "\n")
		}
	}
	for _, l := range got {
		if !slices.Contains(want, l) {
			b.WriteString("+ " + l + "\n")
		}
	}
	return b.String()
}
