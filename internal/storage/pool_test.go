package storage

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
)

func TestOpenDBPoolSizes(t *testing.T) {
	file, err := openDB(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if got := file.Stats().MaxOpenConnections; got != maxOpenConns {
		t.Errorf("file database pool = %d, want %d", got, maxOpenConns)
	}

	mem, err := openDB(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mem.Close() }()
	if got := mem.Stats().MaxOpenConnections; got != 1 {
		t.Errorf("in-memory database pool = %d, want 1", got)
	}
}

// Transactions that read and then write must not fail to upgrade their lock
// ("database is locked") or lose updates when they run on several connections.
func TestOpenDBConcurrentReadModifyWrite(t *testing.T) {
	ctx := context.Background()
	db, err := openDB(filepath.Join(t.TempDir(), "pool.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("CREATE TABLE counter (n INTEGER NOT NULL); INSERT INTO counter VALUES (0)"); err != nil {
		t.Fatal(err)
	}

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				errs <- err
				return
			}
			defer func() { _ = tx.Rollback() }()
			var n int
			if err := tx.QueryRowContext(ctx, "SELECT n FROM counter").Scan(&n); err != nil {
				errs <- err
				return
			}
			if _, err := tx.ExecContext(ctx, "UPDATE counter SET n = ?", n+1); err != nil {
				errs <- err
				return
			}
			errs <- tx.Commit()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("transaction: %v", err)
		}
	}
	var n int
	if err := db.QueryRow("SELECT n FROM counter").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != workers {
		t.Errorf("counter = %d, want %d (lost updates)", n, workers)
	}
}
