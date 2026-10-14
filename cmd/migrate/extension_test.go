package migrate

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/paularlott/cli"
)

type gadgets struct{}

func (gadgets) Manifest() extension.Manifest { return extension.Manifest{Name: "gadgets"} }

func (gadgets) Migrations() []extension.Migration {
	return []extension.Migration{{
		Version: "20260101000000",
		Name:    "create_gadgets",
		Up: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `CREATE TABLE gadgets (id INTEGER PRIMARY KEY)`)
		},
		Down: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `DROP TABLE gadgets`)
		},
	}}
}

func execute(t *testing.T, features []extension.Feature, args ...string) error {
	t.Helper()
	old := os.Args
	defer func() { os.Args = old }()
	os.Args = append([]string{"rackd", "migrate"}, args...)
	root := &cli.Command{Name: "rackd", Commands: []*cli.Command{NewCommand(features)}}
	return root.Execute(context.Background())
}

func gadgetsTable(t *testing.T, dataDir string) bool {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'gadgets'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

func TestMigrateCommandsCoverExtensions(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewSQLiteStorage(dir) // core migrations
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	features := []extension.Feature{gadgets{}}

	if err := execute(t, features, "run", "--data-dir", dir); err != nil {
		t.Fatal(err)
	}
	if !gadgetsTable(t, dir) {
		t.Fatal("migrate run did not apply the extension's migration")
	}
	if err := execute(t, features, "status", "--data-dir", dir); err != nil {
		t.Fatal(err)
	}

	// A binary without the extension refuses to revert it.
	if err := execute(t, nil, "rollback", "--namespace", "gadgets", "--yes", "--data-dir", dir); err == nil || !strings.Contains(err.Error(), "not in this binary") {
		t.Errorf("rollback without the extension = %v", err)
	}
	// Without --yes nothing changes.
	if err := execute(t, features, "rollback", "--namespace", "gadgets", "--data-dir", dir); err != nil || !gadgetsTable(t, dir) {
		t.Errorf("rollback without --yes: %v, table present %v", err, gadgetsTable(t, dir))
	}
	if err := execute(t, features, "rollback", "--namespace", "gadgets", "--yes", "--data-dir", dir); err != nil {
		t.Fatal(err)
	}
	if gadgetsTable(t, dir) {
		t.Error("rollback --namespace gadgets did not revert the extension's migration")
	}
	status, err := storage.ExtensionMigrationStatus(context.Background(), openTestDB(t, dir), "gadgets", gadgets{}.Migrations())
	if err != nil || len(status) != 1 || status[0].Applied {
		t.Errorf("status after rollback = %+v, %v", status, err)
	}
	// Core was not touched.
	core, err := storage.GetMigrationStatus(context.Background(), openTestDB(t, dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range core {
		if !s.Applied {
			t.Errorf("core migration %s reverted", s.Version)
		}
	}
}

func openTestDB(t *testing.T, dir string) *sql.DB {
	t.Helper()
	db, err := openDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
