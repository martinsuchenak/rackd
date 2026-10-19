// Package storagetest gives tests a fresh, migrated SQLite store quickly.
//
// Migrating a new database runs every migration, which takes about half a
// second under the race detector; test suites open hundreds. New migrates one
// database per test binary, keeps its bytes, and gives each test a private
// copy of them.
package storagetest

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage"
)

var (
	once         sync.Once
	templateData []byte
	templateErr  error
)

func migrated() ([]byte, error) {
	once.Do(func() {
		var dir string
		if dir, templateErr = os.MkdirTemp("", "rackd-storagetest-*"); templateErr != nil {
			return
		}
		defer func() { _ = os.RemoveAll(dir) }()
		path := filepath.Join(dir, "template.db")
		var s *storage.SQLiteStorage
		if s, templateErr = storage.NewSQLiteStorageWithPath(path); templateErr != nil {
			return
		}
		// Fold the WAL into the main file, so its bytes are the whole database.
		if _, templateErr = s.DB().Exec("PRAGMA wal_checkpoint(TRUNCATE)"); templateErr != nil {
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

// New returns a store on a private copy of a freshly migrated database,
// closed when the test ends.
func New(tb testing.TB) *storage.SQLiteStorage {
	tb.Helper()
	data, err := migrated()
	if err != nil {
		tb.Fatalf("storagetest: migrating the template database: %v", err)
	}
	// Not tb.TempDir: a background write still finishing after the test (an
	// API key's last-used time, a session cleanup) can recreate the WAL file
	// while the directory is removed, and TempDir fails the test for that.
	dir, err := os.MkdirTemp("", "rackd-storagetest-*")
	if err != nil {
		tb.Fatalf("storagetest: creating a directory: %v", err)
	}
	path := filepath.Join(dir, "rackd.db")
	if err := os.WriteFile(path, data, 0600); err != nil {
		tb.Fatalf("storagetest: copying the template database: %v", err)
	}
	s, err := storage.NewSQLiteStorageWithPath(path)
	if err != nil {
		tb.Fatalf("storagetest: opening the database: %v", err)
	}
	tb.Cleanup(func() {
		_ = s.Close()
		_ = os.RemoveAll(dir)
	})
	return s
}
