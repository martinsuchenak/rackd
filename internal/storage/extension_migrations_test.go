package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/martinsuchenak/xdal/pkg/migrate"
)

func widgetMigrations() []migrate.Migration {
	step := func(version, table string) migrate.Migration {
		return migrate.Migration{
			Version: version,
			Name:    "create_" + table,
			Up: func(ctx context.Context, tx *migrate.Tx) error {
				return tx.ExecSQL(ctx, `CREATE TABLE `+table+` (id INTEGER PRIMARY KEY)`)
			},
			Down: func(ctx context.Context, tx *migrate.Tx) error {
				return tx.ExecSQL(ctx, `DROP TABLE `+table)
			},
		}
	}
	return []migrate.Migration{step("20260101000000", "widgets_a"), step("20260102000000", "widgets_b")}
}

func TestExtensionMigrationLifecycle(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := NewSQLiteStorageWithPath(filepath.Join(dir, "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	steps := widgetMigrations()
	opts := MigrationOptions{BackupDir: filepath.Join(dir, "backups")}

	status, err := ExtensionMigrationStatus(ctx, s.db, "widgets", steps)
	if err != nil || len(status) != 2 || status[0].Applied {
		t.Fatalf("status before = %+v, %v", status, err)
	}
	if res, err := RunExtensionMigrationsOn(ctx, s.db, "widgets", steps, opts); err != nil || len(res.Versions) != 2 || res.BackupPath != "" {
		t.Fatalf("run = %+v, %v (a first run needs no backup)", res, err)
	}
	if ns, err := RecordedExtensionNamespaces(ctx, s.db); err != nil || len(ns) != 1 || ns[0] != "widgets" {
		t.Errorf("namespaces = %v, %v", ns, err)
	}

	// Rolling back one step backs up first and drops the newest table.
	res, err := RollbackExtensionMigrations(ctx, s.db, "widgets", steps, 1, "", opts)
	if err != nil || len(res.Versions) != 1 || res.Versions[0] != "20260102000000" {
		t.Fatalf("rollback = %+v, %v", res, err)
	}
	if _, err := os.Stat(res.BackupPath); err != nil {
		t.Errorf("no backup at %q: %v", res.BackupPath, err)
	}
	if tableExists(ctx, s.db, "widgets_b") || !tableExists(ctx, s.db, "widgets_a") {
		t.Error("rollback dropped the wrong tables")
	}
	if _, err := RollbackExtensionMigrations(ctx, s.db, "widgets", steps, 0, "20991231000000", opts); err == nil {
		t.Error("rolling back to an unknown version succeeded")
	}

	// Core's history is untouched, and an extension can't use core's namespace.
	coreStatus, err := GetMigrationStatus(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range coreStatus {
		if !st.Applied || st.Unknown {
			t.Errorf("core migration %s is %+v", st.Version, st)
		}
	}
	if _, err := RunExtensionMigrationsOn(ctx, s.db, CoreMigrationNamespace, steps, opts); err == nil {
		t.Error("an extension ran migrations in core's namespace")
	}

	// A namespace whose extension isn't in the binary reports its records as unknown.
	foreign, err := ExtensionMigrationStatus(ctx, s.db, "widgets", nil)
	if err != nil || len(foreign) != 1 || !foreign[0].Unknown {
		t.Errorf("foreign status = %+v, %v", foreign, err)
	}
}
