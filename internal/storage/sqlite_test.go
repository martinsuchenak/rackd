package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/martinsuchenak/xdal/pkg/postgres"
	"github.com/martinsuchenak/xdal/pkg/xdal"
	"github.com/martinsuchenak/xdal/pkg/xdaltest"
)

func intPtr(i int) *int { return &i }

func TestNewSQLiteStorage(t *testing.T) {
	// Test with in-memory database
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer func() { _ = storage.Close() }()

	// Verify database connection is valid
	if storage.db == nil {
		t.Error("database connection should not be nil")
	}

	// Verify we can ping the database
	if err := storage.db.Ping(); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestNewSQLiteStorageWithDataDir(t *testing.T) {
	// Use temp directory
	tmpDir := t.TempDir()

	storage, err := NewSQLiteStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer func() { _ = storage.Close() }()

	// Verify database connection is valid
	if storage.db == nil {
		t.Error("database connection should not be nil")
	}

	// Verify we can ping the database
	if err := storage.db.Ping(); err != nil {
		t.Errorf("failed to ping database: %v", err)
	}
}

func TestSQLiteStorageRunsMigrations(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer func() { _ = storage.Close() }()

	// Verify migrations were run by checking for tables
	tables := []string{"devices", "networks", "datacenters"}
	for _, table := range tables {
		var name string
		err := storage.db.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='table' AND name=?
		`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestSQLiteStorageClose(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}

	// Close the storage
	if err := storage.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify database is closed (ping should fail)
	if err := storage.db.Ping(); err == nil {
		t.Error("expected ping to fail after close")
	}
}

func TestNewUUID(t *testing.T) {
	// Generate multiple UUIDs and verify they're unique
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := newUUID()
		if id == "" {
			t.Error("UUID should not be empty")
		}
		if seen[id] {
			t.Errorf("duplicate UUID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestFactoryFunctions(t *testing.T) {
	t.Run("NewStorage", func(t *testing.T) {
		storage, err := NewStorage(":memory:")
		if err != nil {
			t.Fatalf("NewStorage failed: %v", err)
		}

		// Verify it implements Storage interface
		var _ = storage

		// Close
		if s, ok := storage.(*SQLiteStorage); ok {
			_ = s.Close()
		}
	})

	t.Run("NewExtendedStorage", func(t *testing.T) {
		storage, err := NewExtendedStorage(":memory:")
		if err != nil {
			t.Fatalf("NewExtendedStorage failed: %v", err)
		}

		// Verify it implements ExtendedStorage interface
		var _ = storage

		// Close
		_ = storage.Close()
	})
}

func TestSQLiteStorageImplementsInterfaces(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer func() { _ = storage.Close() }()

	// Verify SQLiteStorage implements all interfaces
	var _ DeviceStorage = storage
	var _ DatacenterStorage = storage
	var _ NetworkStorage = storage
	var _ NetworkPoolStorage = storage
	var _ RelationshipStorage = storage
	var _ DiscoveryStorage = storage
	var _ ExtendedStorage = storage
}

func TestWALModeEnabled(t *testing.T) {
	// Use temp directory to test file-based database
	tmpDir := t.TempDir()

	storage, err := NewSQLiteStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer func() { _ = storage.Close() }()

	// Check journal mode is WAL
	var journalMode string
	err = storage.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("expected journal_mode to be 'wal', got '%s'", journalMode)
	}
}

// Migrating a fresh database takes about half a second under the race
// detector, and the tests open hundreds, so they copy one migrated once
// (storagetest does the same for other packages; it can't be imported here).
var (
	templateOnce sync.Once
	templateData []byte
	templateErr  error
)

func migratedTemplate() ([]byte, error) {
	templateOnce.Do(func() {
		var dir string
		if dir, templateErr = os.MkdirTemp("", "rackd-test-template-*"); templateErr != nil {
			return
		}
		defer func() { _ = os.RemoveAll(dir) }()
		path := filepath.Join(dir, "template.db")
		var s *SQLiteStorage
		if s, templateErr = NewSQLiteStorageWithPath(path); templateErr != nil {
			return
		}
		// Fold the WAL into the main file, so its bytes are the whole database.
		if _, templateErr = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); templateErr != nil {
			_ = s.Close()
			return
		}
		if templateErr = s.Close(); templateErr != nil {
			return
		}
		templateData, templateErr = os.ReadFile(path)
	})
	return templateData, templateErr
}

// testBackendEnv picks the database newTestStorage opens: sqlite (the
// default), postgres or mysql. The server backends need the xdaltest DSN
// variables; `task test:backends` sets everything up.
const testBackendEnv = "RACKD_TEST_BACKEND"

// portableTestFiles are the test files whose storage area has been made
// portable; on a server backend newTestStorage skips tests in other files.
// The list goes away once every area is converted.
var portableTestFiles = map[string]bool{
	"search_backends_test.go":       true,
	"database_test.go":              true,
	"copy_test.go":                  true,
	"plumbing_backends_test.go":     true,
	"audit_test.go":                 true,
	"audit_backpressure_test.go":    true,
	"webhook_sqlite_test.go":        true,
	"identity_backends_test.go":     true,
	"user_test.go":                  true,
	"apikey_test.go":                true,
	"session_sqlite_test.go":        true,
	"rbac_sqlite_test.go":           true,
	"user_roles_batch_test.go":      true,
	"oauth_sqlite_test.go":          true,
	"datacenter_sqlite_test.go":     true,
	"network_sqlite_test.go":        true,
	"pool_sqlite_test.go":           true,
	"pool_membership_test.go":       true,
	"reservation_sqlite_test.go":    true,
	"reservation_lifecycle_test.go": true,
	"conflict_sqlite_test.go":       true,
	"overlap_test.go":               true,
	"device_sqlite_test.go":         true,
	"device_relations_test.go":      true,
	"relationship_sqlite_test.go":   true,
	"custom_field_sqlite_test.go":   true,
	"circuit_sqlite_test.go":        true,
	"nat_sqlite_test.go":            true,
	"constraint_errors_test.go":     true,
	"bulk_test.go":                  true,
	"bulk_tags_test.go":             true,
	"bulk_savepoint_test.go":        true,
	"dns_sqlite_test.go":            true,
	"discovery_sqlite_test.go":      true,
	"discovery_match_test.go":       true,
	"profiles_test.go":              true,
	"scheduled_test.go":             true,
	"snapshot_sqlite_test.go":       true,
	"dashboard_utilization_test.go": true,
}

// reopenPostgres opens the test's PostgreSQL schema again through
// openPostgresPool, the way rackd opens PostgreSQL, rather than xdaltest's
// plain pool.
func reopenPostgres(t *testing.T, db *xdaltest.DB) (*sql.DB, xdal.DBInterface) {
	t.Helper()
	var schema string
	if err := db.SQL.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgx.ParseConfig(os.Getenv(xdaltest.EnvPostgresDSN))
	if err != nil {
		t.Fatal(err)
	}
	cfg.RuntimeParams["search_path"] = schema
	pool := openPostgresPool(cfg)
	t.Cleanup(func() { _ = pool.Close() })
	return pool, postgres.NewPostgresDB(pool, nil)
}

// skipUnlessSQLite skips a test that checks something only SQLite has.
func skipUnlessSQLite(t *testing.T, why string) {
	t.Helper()
	if b := testBackend(); b != xdal.DialectSQLite {
		t.Skipf("%s: SQLite only: %s", b, why)
	}
}

// callerIsPortable reports whether the running test function is in a
// portable test file. It walks up the stack to the Test function itself, so
// storage opened through a helper in another file is judged by the test.
func callerIsPortable() bool {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs)])
	for {
		f, more := frames.Next()
		name := f.Function[strings.LastIndex(f.Function, ".")+1:]
		if strings.HasPrefix(name, "Test") && strings.HasSuffix(f.File, "_test.go") {
			return portableTestFiles[filepath.Base(f.File)]
		}
		if !more {
			return false
		}
	}
}

func testBackend() string {
	if b := os.Getenv(testBackendEnv); b != "" {
		return b
	}
	return xdal.DialectSQLite
}

// Helper function to create a test storage: a private copy of a migrated
// database, so each test starts from the same fresh state.
//
// With RACKD_TEST_BACKEND set to postgres or mysql, it opens storage on a
// fresh database of that backend instead (see testBackend).
func newTestStorage(t *testing.T) *SQLiteStorage {
	t.Helper()
	if b := testBackend(); b != xdal.DialectSQLite {
		if !callerIsPortable() {
			t.Skipf("%s: this area's storage isn't portable yet", b)
		}
		db := xdaltest.Open(t, b)
		sqlDB, x := db.SQL, db.DB
		if b == xdal.DialectPostgres {
			sqlDB, x = reopenPostgres(t, db)
		}
		s, err := openStorageOn(context.Background(), sqlDB, x)
		if err != nil {
			t.Fatalf("open storage on %s: %v", b, err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
	data, err := migratedTemplate()
	if err != nil {
		t.Fatalf("failed to create template database: %v", err)
	}
	// Not t.TempDir: a background write finishing after the test can recreate
	// the WAL file while the directory is removed, which TempDir reports as a
	// failure.
	dir, err := os.MkdirTemp("", "rackd-test-*")
	if err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	dbPath := filepath.Join(dir, "rackd.db")
	if err := os.WriteFile(dbPath, data, 0600); err != nil {
		t.Fatalf("failed to copy template database: %v", err)
	}

	storage, err := NewSQLiteStorageWithPath(dbPath)
	if err != nil {
		t.Fatalf("failed to create test storage: %v", err)
	}
	return storage
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestNullString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"value", true},
	}

	for _, tt := range tests {
		result := nullString(tt.input)
		if result.Valid != tt.expected {
			t.Errorf("nullString(%q).Valid = %v, expected %v", tt.input, result.Valid, tt.expected)
		}
		if tt.expected && result.String != tt.input {
			t.Errorf("nullString(%q).String = %q, expected %q", tt.input, result.String, tt.input)
		}
	}
}

func TestNullInt(t *testing.T) {
	tests := []struct {
		input    int
		expected bool
	}{
		{0, false},
		{1, true},
		{-1, true},
	}

	for _, tt := range tests {
		result := nullInt(tt.input)
		if result.Valid != tt.expected {
			t.Errorf("nullInt(%d).Valid = %v, expected %v", tt.input, result.Valid, tt.expected)
		}
		if tt.expected && result.Int64 != int64(tt.input) {
			t.Errorf("nullInt(%d).Int64 = %d, expected %d", tt.input, result.Int64, tt.input)
		}
	}
}

func TestDBMethod(t *testing.T) {
	storage := newTestStorage(t)
	defer func() { _ = storage.Close() }()

	db := storage.DB()
	if db == nil {
		t.Error("DB() should return non-nil database")
	}

	// Verify it's the same connection
	if err := db.Ping(); err != nil {
		t.Errorf("DB() returned invalid connection: %v", err)
	}
}
