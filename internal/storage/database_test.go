package storage

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/xdal"
	"github.com/martinsuchenak/xdal/pkg/xdaltest"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestDatabaseBackend(t *testing.T) {
	for url, want := range map[string]string{
		"":                             xdal.DialectSQLite,
		"postgres://u:p@db/rackd":      xdal.DialectPostgres,
		"postgresql://u:p@db/rackd":    xdal.DialectPostgres,
		"MySQL://u:p@db:3306/rackd":    xdal.DialectMySQL,
		"sqlserver://u:p@db/rackd":     "",
		"u:p@tcp(db:3306)/rackd":       "",
		"/var/lib/rackd/rackd.db":      "",
		"sqlite:///var/lib/rackd/x.db": "",
	} {
		got, err := DatabaseBackend(url)
		if want == "" {
			if err == nil {
				t.Errorf("DatabaseBackend(%q) = %q, want an error", url, got)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("DatabaseBackend(%q) = %q, %v; want %q", url, got, err, want)
		}
	}
}

func TestMySQLConfigFromURL(t *testing.T) {
	cfg, err := mysqlConfig("mysql://rackd:p%40ss%2Fw0rd@db.internal/inventory?tls=skip-verify&timeout=5s")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.User != "rackd" || cfg.Passwd != "p@ss/w0rd" {
		t.Errorf("user, password = %q, %q", cfg.User, cfg.Passwd)
	}
	if cfg.Net != "tcp" || cfg.Addr != "db.internal:3306" || cfg.DBName != "inventory" {
		t.Errorf("net, addr, db = %q, %q, %q", cfg.Net, cfg.Addr, cfg.DBName)
	}
	if cfg.TLSConfig != "skip-verify" || cfg.Timeout != 5*time.Second {
		t.Errorf("driver parameters not kept: tls %q, timeout %v", cfg.TLSConfig, cfg.Timeout)
	}
	if !cfg.ParseTime || cfg.Loc != time.UTC || cfg.Params["time_zone"] != "'+00:00'" {
		t.Errorf("times not read and written as UTC: parseTime %v, loc %v, params %v", cfg.ParseTime, cfg.Loc, cfg.Params)
	}

	if _, err := mysqlConfig("mysql://rackd:secret@db:3306/"); err == nil {
		t.Error("a URL without a database was accepted")
	}
	_, err = mysqlConfig("mysql://rackd:secret@db:bad port/rackd")
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Errorf("bad URL error = %v; want an error without the password", err)
	}
}

// databaseURLFor returns a DATABASE_URL for the test database db.
func databaseURLFor(t *testing.T, db *xdaltest.DB) string {
	t.Helper()
	switch db.Name {
	case xdal.DialectPostgres:
		var schema string
		if err := db.SQL.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(os.Getenv(xdaltest.EnvPostgresDSN))
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		return u.String()
	case xdal.DialectMySQL:
		var name string
		if err := db.SQL.QueryRow("SELECT DATABASE()").Scan(&name); err != nil {
			t.Fatal(err)
		}
		cfg, err := gomysql.ParseDSN(os.Getenv(xdaltest.EnvMySQLDSN))
		if err != nil {
			t.Fatal(err)
		}
		u := url.URL{Scheme: "mysql", User: url.UserPassword(cfg.User, cfg.Passwd), Host: cfg.Addr, Path: "/" + name}
		return u.String()
	}
	t.Fatalf("no DATABASE_URL for %s", db.Name)
	return ""
}

// Opening storage through DATABASE_URL, as the server and commands do.
func TestOpenWithDatabaseURL(t *testing.T) {
	b := testBackend()
	if b == xdal.DialectSQLite {
		t.Skip("DATABASE_URL selects PostgreSQL or MySQL")
	}
	ctx := context.Background()
	databaseURL := databaseURLFor(t, xdaltest.Open(t, b))
	opts := StorageOptions{DatabaseURL: databaseURL}

	s, err := NewSQLiteStorageWithOptions("", opts)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if s.Backend() != b {
		t.Errorf("Backend() = %q, want %q", s.Backend(), b)
	}
	if err := s.CreateDevice(ctx, &model.Device{Name: "kept"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	// Again: nothing left to migrate, and the data is there.
	s, err = NewSQLiteStorageWithOptions("", opts)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = s.Close() }()
	if got, err := s.SearchDevices(ctx, "kept"); err != nil || len(got) != 1 {
		t.Errorf("device after reopening = %d, %v; want 1", len(got), err)
	}

	// The commands' view of the same database.
	db, err := OpenDatabase(ctx, "", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if res, err := RunMigrationsWithOptions(ctx, db, MigrationOptions{BackupDir: t.TempDir()}); err != nil || len(res.Versions) != 0 || res.BackupPath != "" {
		t.Errorf("migrate run = %+v, %v; want nothing applied and no backup", res, err)
	}
	if core, err := GetMigrationStatus(ctx, db); err != nil || len(core) != 0 {
		t.Errorf("core status = %v, %v; want none on %s", core, err, b)
	}
	schema, err := GetSchemaMigrationStatus(ctx, db)
	if err != nil || len(schema) == 0 {
		t.Fatalf("schema status = %v, %v", schema, err)
	}
	for _, st := range schema {
		if !st.Applied {
			t.Errorf("schema migration %s not applied", st.Version)
		}
	}
	if _, err := RollbackMigrations(ctx, db, 1, "", MigrationOptions{}); err == nil {
		t.Error("core rollback ran on a database server")
	}
}

func TestOpenRefusesDatabaseWithOtherTables(t *testing.T) {
	b := testBackend()
	if b == xdal.DialectSQLite {
		t.Skip("DATABASE_URL selects PostgreSQL or MySQL")
	}
	db := xdaltest.Open(t, b)
	if _, err := db.SQL.Exec(`CREATE TABLE devices (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	_, err := NewSQLiteStorageWithOptions("", StorageOptions{DatabaseURL: databaseURLFor(t, db)})
	if err == nil || !strings.Contains(err.Error(), "no rackd migration history") {
		t.Fatalf("open = %v; want a refusal", err)
	}
	if tableExists(context.Background(), db.SQL, migrate.DefaultTable) {
		t.Error("migration history was written to a database that isn't rackd's")
	}
}

// Extension migrations from the migrate command, on any backend.
func TestExtensionMigrationsFromCommandOnAnyBackend(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	steps := []migrate.Migration{{
		Version: "20260101000000",
		Name:    "create_widgets",
		Up: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `CREATE TABLE ext_widgets (id INTEGER PRIMARY KEY)`)
		},
		Down: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `DROP TABLE ext_widgets`)
		},
	}}
	res, err := RunExtensionMigrationsOn(ctx, s.DB(), "widgets", steps, MigrationOptions{})
	if err != nil || len(res.Versions) != 1 {
		t.Fatalf("run = %+v, %v", res, err)
	}
	if ns, err := RecordedExtensionNamespaces(ctx, s.DB()); err != nil || len(ns) != 1 || ns[0] != "widgets" {
		t.Errorf("recorded namespaces = %v, %v", ns, err)
	}
	if res, err := RollbackExtensionMigrations(ctx, s.DB(), "widgets", steps, 1, "", MigrationOptions{}); err != nil || len(res.Versions) != 1 {
		t.Fatalf("rollback = %+v, %v", res, err)
	}
	st, err := ExtensionMigrationStatus(ctx, s.DB(), "widgets", steps)
	if err != nil || len(st) != 1 || st[0].Applied {
		t.Errorf("status after rollback = %+v, %v", st, err)
	}
	if tableExists(ctx, s.DB(), "ext_widgets") {
		t.Error("rollback left the extension's table")
	}
}
