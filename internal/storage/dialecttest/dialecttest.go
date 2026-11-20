// Package dialecttest runs storage tests written on xdal against every
// database backend rackd aims to support.
//
// Storage code written with xdal's query and schema builders, and migrations
// written with the schema builder, are portable. Run checks that they behave
// the same everywhere: it applies a migration track to a fresh database of
// each kind, then runs the test on it. SQLite always runs; PostgreSQL and
// MySQL run when XDAL_TEST_POSTGRES_DSN and XDAL_TEST_MYSQL_DSN are set (CI
// sets both and XDAL_TEST_REQUIRE). See xdaltest for the details.
//
// Core's own tables come from SQLite-only migrations, so a portable track
// can't reference them yet: give it the tables it needs.
//
// Name tests TestDialect* so `go test -run Dialect` picks them out; CI runs
// them that way against every backend.
package dialecttest

import (
	"context"
	"testing"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/xdal"
	"github.com/martinsuchenak/xdal/pkg/xdaltest"
)

// Backends are the databases Run tests. SQL Server is left out: the plan
// supports SQLite, PostgreSQL and MySQL 8.
var Backends = []string{xdal.DialectSQLite, xdal.DialectPostgres, xdal.DialectMySQL}

// DB is a fresh database with the migration track applied.
type DB = xdaltest.DB

// Run applies steps, as migration namespace ns, to a fresh database of each
// backend and runs fn on it, each as a subtest named after the backend.
func Run(t *testing.T, ns string, steps []migrate.Migration, fn func(t *testing.T, db *DB)) {
	t.Helper()
	for _, name := range Backends {
		t.Run(name, func(t *testing.T) {
			db := xdaltest.Open(t, name)
			if len(steps) > 0 {
				Migrate(t, db, ns, steps)
			}
			fn(t, db)
		})
	}
}

// Migrate applies steps to db as namespace ns, failing the test on error.
func Migrate(t testing.TB, db *DB, ns string, steps []migrate.Migration) {
	t.Helper()
	m, err := migrate.New(db.DB, steps, migrate.Options{Namespace: ns})
	if err != nil {
		t.Fatalf("dialecttest: %s migrations: %v", ns, err)
	}
	if _, err := m.Up(context.Background()); err != nil {
		t.Fatalf("dialecttest: %s migrations on %s: %v", ns, db.Name, err)
	}
}
