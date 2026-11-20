package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martinsuchenak/xdal/pkg/xdal"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// openStorageOn opens storage on a database of any backend that xdal
// supports, brings its schema up to date with the portable migration track,
// and loads what storage needs at start-up.
//
// It is how tests run storage on PostgreSQL and MySQL while the conversion is
// in progress; the backend setting will build on it. db is the pool x wraps;
// Close closes it.
func openStorageOn(ctx context.Context, db *sql.DB, x xdal.DBInterface) (*SQLiteStorage, error) {
	if x.Dialect().Name() == xdal.DialectSQLite {
		return nil, fmt.Errorf("openStorageOn: SQLite databases open with NewSQLiteStorage, which runs the legacy migrations first")
	}
	sm, err := newSchemaMigrator(x)
	if err != nil {
		return nil, err
	}
	if _, err := sm.Up(ctx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	s := &SQLiteStorage{
		db:        db,
		q:         portableDB{x: x},
		auditChan: make(chan *model.AuditLog, 1000),
		auditDone: make(chan struct{}),
	}
	s.auditWG.Add(1)
	go s.auditWorker()

	if err := s.ensureDefaultDatacenter(ctx); err != nil {
		_ = s.Close()
		return nil, fmt.Errorf("failed to ensure default datacenter: %w", err)
	}
	if err := s.initRBAC(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}
