package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func migratedFileDB(t *testing.T) (string, func()) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rackd.db")
	db := openFileDB(t, path)
	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return path, func() { _ = db.Close() }
}

func TestBackupWhileWriting(t *testing.T) {
	path, _ := migratedFileDB(t)
	db := openFileDB(t, path)
	ctx := context.Background()

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
			}
			_, _ = db.Exec("INSERT INTO datacenters (id, name) VALUES (?, ?)", fmt.Sprintf("dc-%d", i), "dc")
		}
	}()

	dst := filepath.Join(t.TempDir(), "backups", "snap.db")
	_, err := BackupFile(ctx, path, dst)
	close(stop)
	wg.Wait()
	if err != nil {
		t.Fatalf("backup while writing: %v", err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(dst + suffix); err == nil {
			t.Errorf("backup should be a single file, found %s", suffix)
		}
	}
	fi, _ := os.Stat(dst)
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("backup mode = %v, want 0600", fi.Mode().Perm())
	}
	info, err := VerifyBackup(ctx, dst)
	if err != nil || info.Migrations != len(migrations) {
		t.Errorf("verify = %+v, %v", info, err)
	}
	if _, err := BackupFile(ctx, path, dst); err == nil {
		t.Error("backup should refuse to overwrite an existing file")
	}
}

func TestBackupOverwrite(t *testing.T) {
	path, closeDB := migratedFileDB(t)
	defer closeDB()
	ctx := context.Background()
	dir := t.TempDir()
	dst := filepath.Join(dir, "latest.db")

	if err := os.WriteFile(dst, []byte("previous"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BackupFileOverwrite(ctx, filepath.Join(dir, "missing.db"), dst); err == nil {
		t.Fatal("backup of a missing database should fail")
	}
	if got, _ := os.ReadFile(dst); string(got) != "previous" {
		t.Error("a failed backup must leave the previous file in place")
	}

	info, err := BackupFileOverwrite(ctx, path, dst)
	if err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	if info.Path != dst || info.Migrations != len(migrations) {
		t.Errorf("info = %+v", info)
	}
	if _, err := VerifyBackup(ctx, dst); err != nil {
		t.Errorf("replaced backup does not verify: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temporary files left behind: %v", entries)
	}

	if _, err := BackupFileOverwrite(ctx, path, dir); err == nil {
		t.Error("overwrite should refuse a directory")
	}

	link := filepath.Join(t.TempDir(), "link.db")
	if err := os.Symlink(dst, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := BackupFileOverwrite(ctx, path, link); err != nil {
		t.Fatalf("overwrite through symlink: %v", err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("overwrite should keep the symlink and replace its target")
	}
}

func TestVerifyRejectsBadFiles(t *testing.T) {
	ctx := context.Background()
	junk := filepath.Join(t.TempDir(), "junk.db")
	if err := os.WriteFile(junk, []byte(strings.Repeat("not a database ", 100)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBackup(ctx, junk); err == nil {
		t.Error("junk file verified")
	}
	if _, err := VerifyBackup(ctx, filepath.Join(t.TempDir(), "missing.db")); err == nil {
		t.Error("missing file verified")
	}
	empty := filepath.Join(t.TempDir(), "empty.db")
	db := openFileDB(t, empty)
	if _, err := db.Exec("CREATE TABLE x (id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBackup(ctx, empty); err == nil {
		t.Error("database without migration history verified")
	}
}

func TestVerifyReadsLegacyHistory(t *testing.T) {
	info, err := VerifyBackup(context.Background(), extractLegacyFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if info.Migrations != legacyFixtureMigrations || len(info.UnknownMigrations) != 0 {
		t.Errorf("legacy fixture verify = %+v", info)
	}
}

func TestRestoreReplacesDatabase(t *testing.T) {
	path, closeDB := migratedFileDB(t)
	ctx := context.Background()
	db := openFileDB(t, path)
	if _, err := db.Exec("INSERT INTO datacenters (id, name) VALUES ('before', 'kept in backup')"); err != nil {
		t.Fatal(err)
	}
	snap := filepath.Join(t.TempDir(), "snap.db")
	if _, err := BackupDatabase(ctx, db, snap); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO datacenters (id, name) VALUES ('after', 'only in current')"); err != nil {
		t.Fatal(err)
	}

	if _, err := RestoreDatabase(ctx, snap, path); !errors.Is(err, ErrDatabaseInUse) {
		t.Fatalf("restore with the database open: err = %v", err)
	}

	_ = db.Close()
	closeDB()
	movedTo, err := RestoreDatabase(ctx, snap, path)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if movedTo == "" {
		t.Fatal("previous database was not kept")
	}

	restored := openFileDB(t, path)
	var n int
	_ = restored.QueryRow("SELECT COUNT(*) FROM datacenters WHERE id IN ('before', 'after')").Scan(&n)
	if n != 1 {
		t.Errorf("restored database should hold only the backed-up row, found %d", n)
	}
	previous := openFileDB(t, movedTo)
	_ = previous.QueryRow("SELECT COUNT(*) FROM datacenters WHERE id = 'after'").Scan(&n)
	if n != 1 {
		t.Error("the previous database lost its newest row when moved aside")
	}
}

func TestRestoreRefusesNewerBackup(t *testing.T) {
	path, closeDB := migratedFileDB(t)
	ctx := context.Background()
	db := openFileDB(t, path)
	if _, err := db.Exec("INSERT INTO xdal_migrations (namespace, version, name, checksum, applied_at, duration_ms, dirty) VALUES ('core', '29990101000000', 'future', 'x', CURRENT_TIMESTAMP, 0, 0)"); err != nil {
		t.Fatal(err)
	}
	snap := filepath.Join(t.TempDir(), "snap.db")
	if _, err := BackupDatabase(ctx, db, snap); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	closeDB()
	if _, err := RestoreDatabase(ctx, snap, filepath.Join(t.TempDir(), "rackd.db")); err == nil || !strings.Contains(err.Error(), "newer rackd") {
		t.Errorf("err = %v", err)
	}
}

func journalMode(t *testing.T, path string) string {
	t.Helper()
	db, err := openReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	return mode
}

func TestRestoreIncludesBackupWAL(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "live.db")
	db := openFileDB(t, src)
	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA wal_autocheckpoint=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO datacenters (id, name) VALUES ('only-in-wal', 'x')"); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(dir, "copied.db")
	for _, suffix := range []string{"", "-wal"} {
		data, err := os.ReadFile(src + suffix)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(backup+suffix, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	target := filepath.Join(t.TempDir(), "rackd.db")
	if _, err := RestoreDatabase(context.Background(), backup, target); err != nil {
		t.Fatalf("restore: %v", err)
	}
	restored := openFileDB(t, target)
	var n int
	_ = restored.QueryRow("SELECT COUNT(*) FROM datacenters WHERE id = 'only-in-wal'").Scan(&n)
	if n != 1 {
		t.Error("restore dropped data that was only in the backup's -wal file")
	}
}

func TestRestoredDatabaseIsInWALMode(t *testing.T) {
	path, closeDB := migratedFileDB(t)
	db := openFileDB(t, path)
	snap := filepath.Join(t.TempDir(), "snap.db")
	if _, err := BackupDatabase(context.Background(), db, snap); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	closeDB()
	if _, err := RestoreDatabase(context.Background(), snap, path); err != nil {
		t.Fatal(err)
	}
	if mode := journalMode(t, path); mode != "wal" {
		t.Errorf("restored database journal_mode = %s, want wal", mode)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if strings.Contains(e.Name(), ".restore-") {
			t.Errorf("staging file left behind: %s", e.Name())
		}
	}
}

func TestVerifyHandlesUnusualPathsWithoutWriting(t *testing.T) {
	path, closeDB := migratedFileDB(t)
	db := openFileDB(t, path)
	snap := filepath.Join(t.TempDir(), "odd ?name #1.db")
	if _, err := BackupDatabase(context.Background(), db, snap); err != nil {
		t.Fatalf("backup to an unusual path: %v", err)
	}
	_ = db.Close()
	closeDB()
	before, _ := os.ReadDir(filepath.Dir(snap))
	if _, err := VerifyBackup(context.Background(), snap); err != nil {
		t.Fatalf("verify: %v", err)
	}
	after, _ := os.ReadDir(filepath.Dir(snap))
	if len(after) != len(before) {
		t.Errorf("verify created files next to the backup: %d before, %d after", len(before), len(after))
	}
}
