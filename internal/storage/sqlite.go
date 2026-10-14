package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	rbac "github.com/martinsuchenak/go-rbac"
	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/pkg/model"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements ExtendedStorage using SQLite
type SQLiteStorage struct {
	db        *sql.DB
	auditChan chan *model.AuditLog
	auditDone chan struct{}
	auditWG   sync.WaitGroup
	closeOnce sync.Once

	rbacManager *rbac.Manager

	// webhookEncryptor, when set, transparently encrypts webhook signing
	// secrets at rest (write) and decrypts them on read. Legacy plaintext
	// values are returned as-is when decryption fails.
	webhookEncryptor WebhookEncryptor

	// migrateOpts are the backup settings core migrations ran with, reused for
	// extension migrations.
	migrateOpts MigrationOptions
}

// WebhookEncryptor is the subset of the credentials encryptor used by the
// storage layer for webhook secret encryption at rest.
type WebhookEncryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// SetWebhookEncryptor enables webhook secret encryption at rest.
func (s *SQLiteStorage) SetWebhookEncryptor(e WebhookEncryptor) {
	s.webhookEncryptor = e
}

// NewSQLiteStorage creates a new SQLite storage instance
// StorageOptions configures NewSQLiteStorageWithOptions.
type StorageOptions struct {
	// SkipPreMigrationBackup disables the snapshot taken before pending migrations run.
	SkipPreMigrationBackup bool
}

// NewSQLiteStorage opens the database in dataDir and runs pending migrations,
// backing up an existing database first.
func NewSQLiteStorage(dataDir string) (*SQLiteStorage, error) {
	return NewSQLiteStorageWithOptions(dataDir, StorageOptions{})
}

// NewSQLiteStorageWithOptions is NewSQLiteStorage with options.
func NewSQLiteStorageWithOptions(dataDir string, opts StorageOptions) (*SQLiteStorage, error) {
	var dbPath string

	if dataDir == ":memory:" {
		dbPath = ":memory:"
	} else {
		// Ensure data directory exists
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
		dbPath = filepath.Join(dataDir, "rackd.db")
	}

	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &SQLiteStorage{
		db:        db,
		auditChan: make(chan *model.AuditLog, 1000),
		auditDone: make(chan struct{}),
	}

	// Start audit log worker
	s.auditWG.Add(1)
	go s.auditWorker()

	// Run migrations, snapshotting an existing database first
	ctx := context.Background()
	var migrateOpts MigrationOptions
	if dataDir != ":memory:" && !opts.SkipPreMigrationBackup {
		migrateOpts.BackupDir = filepath.Join(dataDir, "backups")
	}
	s.migrateOpts = migrateOpts
	if err := s.runMigrations(ctx, migrateOpts); err != nil {
		_ = s.Close()
		return nil, err
	}

	// Create default datacenter if none exists
	if err := s.ensureDefaultDatacenter(ctx); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("failed to ensure default datacenter: %w", err)
	}

	// Build in-memory RBAC manager from seeded role/permission data
	if err := s.initRBAC(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStorage) runMigrations(ctx context.Context, opts MigrationOptions) error {
	res, err := RunMigrationsWithOptions(ctx, s.db, opts)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	if res.BackupPath != "" {
		// One backup covers this start-up; extension migrations reuse it.
		s.migrateOpts.BackupDir = ""
	}
	if len(res.Versions) > 0 {
		log.Info("Applied database migrations", "count", len(res.Versions), "latest", res.Versions[len(res.Versions)-1], "backup", res.BackupPath)
	}
	return nil
}

// maxOpenConns is the connection pool size for file databases. In WAL mode
// SQLite runs readers concurrently with each other and with one writer.
const maxOpenConns = 8

// openDB opens a SQLite database with rackd's connection settings.
//
// Each connection waits up to busy_timeout for a lock instead of failing, and
// transactions start with BEGIN IMMEDIATE (_txlock), so a transaction that
// writes takes the write lock up front rather than failing to upgrade a read
// lock while another connection writes. An in-memory database is private to a
// connection, so it keeps a single one.
func openDB(dbPath string) (*sql.DB, error) {
	dsn := dbPath + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	conns := maxOpenConns
	if dbPath == ":memory:" {
		conns = 1
	}
	db.SetMaxOpenConns(conns)
	db.SetMaxIdleConns(conns)
	db.SetConnMaxLifetime(time.Hour)
	return db, nil
}

// NewSQLiteStorageWithPath creates a new SQLite storage instance with a specific database file path
func NewSQLiteStorageWithPath(dbPath string) (*SQLiteStorage, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &SQLiteStorage{
		db:        db,
		auditChan: make(chan *model.AuditLog, 1000),
		auditDone: make(chan struct{}),
	}

	// Start audit log worker
	s.auditWG.Add(1)
	go s.auditWorker()

	// Run migrations, snapshotting an existing database first
	ctx := context.Background()
	if err := s.runMigrations(ctx, MigrationOptions{BackupDir: filepath.Join(filepath.Dir(dbPath), "backups")}); err != nil {
		_ = s.Close()
		return nil, err
	}

	// Create default datacenter if none exists
	if err := s.ensureDefaultDatacenter(ctx); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("failed to ensure default datacenter: %w", err)
	}

	// Build in-memory RBAC manager from seeded role/permission data
	if err := s.initRBAC(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}

	return s, nil
}

// Close closes the database connection
// Close stops the audit worker (after it drains queued entries) and closes
// the database. Safe to call more than once.
func (s *SQLiteStorage) Close() error {
	if s.auditDone != nil {
		s.closeOnce.Do(func() { close(s.auditDone) })
		s.auditWG.Wait()
	}
	return s.db.Close()
}

// DB returns the underlying database connection for testing
func (s *SQLiteStorage) DB() *sql.DB {
	return s.db
}

// newUUID generates a new UUIDv7
func newUUID() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fall back to v4 if v7 generation fails
		return uuid.New().String()
	}
	return id.String()
}

// nowUTC returns the current time in UTC.
// All storage methods MUST use this instead of time.Now() directly.
func nowUTC() time.Time {
	return time.Now().UTC()
}

// nullString returns a sql.NullString for empty strings
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullInt returns a sql.NullInt64 for zero values
func nullInt(i int) sql.NullInt64 {
	if i == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(i), Valid: true}
}

// nullIntPtr returns a sql.NullInt64 for nil pointer values
func nullIntPtr(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// auditWorker processes audit logs from the queue until Close signals
// auditDone, then persists any entries still buffered before exiting so
// shutdown does not lose queued audit records.
func (s *SQLiteStorage) auditWorker() {
	defer s.auditWG.Done()
	for {
		select {
		case logEntry := <-s.auditChan:
			s.persistAuditEntry(logEntry)
		case <-s.auditDone:
			for {
				select {
				case logEntry := <-s.auditChan:
					s.persistAuditEntry(logEntry)
				default:
					return
				}
			}
		}
	}
}

func (s *SQLiteStorage) persistAuditEntry(logEntry *model.AuditLog) {
	if err := s.CreateAuditLog(context.Background(), logEntry); err != nil {
		log.Error("Failed to create audit log", "error", err)
	}
}

// auditLog creates an audit log entry asynchronously using a worker pool
func (s *SQLiteStorage) auditLog(ctx context.Context, action, resource, resourceID string, changes any) {
	auditCtx, ok := audit.FromContext(ctx)
	if !ok {
		return
	}

	var changesStr string
	if changes != nil {
		if str, ok := changes.(string); ok {
			changesStr = str
		} else {
			changesBytes, err := json.Marshal(changes)
			if err == nil {
				changesStr = string(changesBytes)
			}
		}
	}

	entry := &model.AuditLog{
		Timestamp:  nowUTC(),
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		UserID:     auditCtx.UserID,
		Username:   auditCtx.Username,
		IPAddress:  auditCtx.IPAddress,
		Changes:    changesStr,
		Source:     auditCtx.Source,
		Status:     "success",
	}

	if s.auditChan == nil {
		// Fallback for tests
		_ = s.CreateAuditLog(context.Background(), entry)
		return
	}

	select {
	case s.auditChan <- entry:
	default:
		log.Error("Audit log channel full, dropping log entry", "action", action)
	}
}
