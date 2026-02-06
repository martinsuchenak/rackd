package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	_ "modernc.org/sqlite"
)

// SQLiteStorage implements ExtendedStorage using SQLite
type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(dataDir string) (*SQLiteStorage, error) {
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

	// Open database with SQLite pragma settings
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &SQLiteStorage{db: db}

	// Run migrations
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create default datacenter if none exists
	if err := s.ensureDefaultDatacenter(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure default datacenter: %w", err)
	}

	return s, nil
}

// NewSQLiteStorageWithPath creates a new SQLite storage instance with a specific database file path
func NewSQLiteStorageWithPath(dbPath string) (*SQLiteStorage, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open database with SQLite pragma settings
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	s := &SQLiteStorage{db: db}

	// Run migrations
	ctx := context.Background()
	if err := RunMigrations(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create default datacenter if none exists
	if err := s.ensureDefaultDatacenter(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ensure default datacenter: %w", err)
	}

	return s, nil
}

// Close closes the database connection
func (s *SQLiteStorage) Close() error {
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

// auditLog creates an audit log entry asynchronously
func (s *SQLiteStorage) auditLog(ctx context.Context, action, resource, resourceID string, changes any) {
	auditCtx, ok := audit.FromContext(ctx)
	if !ok {
		return
	}

	go func() {
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

		auditLog := &model.AuditLog{
			Timestamp:  time.Now(),
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

		if err := s.CreateAuditLog(auditLog); err != nil {
			log.Error("Failed to create audit log", "error", err)
		}
	}()
}
