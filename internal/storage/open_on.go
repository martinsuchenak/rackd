package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/martinsuchenak/xdal/pkg/xdal"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// openStorageOn opens storage on a PostgreSQL or MySQL database, brings its
// schema up to date with the portable migration track, and loads what storage
// needs at start-up. db is the pool x wraps; Close closes it.
func openStorageOn(ctx context.Context, db *sql.DB, x xdal.DBInterface) (*SQLiteStorage, error) {
	if x.Dialect().Name() == xdal.DialectSQLite {
		return nil, fmt.Errorf("openStorageOn: SQLite databases open with NewSQLiteStorage, which runs the legacy migrations first")
	}
	applied, err := runServerMigrations(ctx, db)
	if err != nil {
		return nil, err
	}
	if len(applied) > 0 {
		log.Info("Applied database migrations", "count", len(applied), "latest", applied[len(applied)-1])
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
