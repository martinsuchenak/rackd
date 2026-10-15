package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// The database holds password hashes, webhook secrets and audit history, so
// it, its WAL and shared-memory files, and a new data directory are private.
func TestDatabaseFilesArePrivate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "data")
	s, err := NewSQLiteStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	// Force a WAL write so the -wal and -shm files exist.
	if _, err := s.DB().Exec(`CREATE TABLE IF NOT EXISTS file_mode_probe (x INTEGER)`); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0700 {
		t.Errorf("data directory mode = %v, %v; want 0700", info.Mode().Perm(), err)
	}
	for _, name := range []string{"rackd.db", "rackd.db-wal", "rackd.db-shm"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if perm := info.Mode().Perm(); perm&0077 != 0 {
			t.Errorf("%s mode = %v; want no group or other access", name, perm)
		}
	}
}

// An existing world-readable database is tightened on open.
func TestExistingDatabaseIsTightened(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rackd.db")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	s, err := NewSQLiteStorageWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	if info, _ := os.Stat(path); info.Mode().Perm() != 0600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
}
