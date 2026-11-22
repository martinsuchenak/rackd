package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/xdal"
	"github.com/martinsuchenak/xdal/pkg/xdaltest"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestConvertValue(t *testing.T) {
	want := time.Date(2026, 3, 4, 5, 6, 7, 123456000, time.UTC)
	for _, s := range []string{
		"2026-03-04 13:06:07.123456789+08:00",
		"2026-03-04T05:06:07.123456Z",
		"2026-03-04 05:06:07.123456 +0000 UTC m=+0.001",
		"2026-03-04 05:06:07.123456",
	} {
		got, err := convertValue(s, kindTime)
		if err != nil || !got.(time.Time).Equal(want) {
			t.Errorf("time %q = %v, %v; want %v", s, got, err, want)
		}
	}
	if _, err := convertValue("yesterday", kindTime); err == nil {
		t.Error("an unreadable time was accepted")
	}
	for v, want := range map[any]any{int64(1): true, int64(0): false, "true": true} {
		if got, err := convertValue(v, kindBool); err != nil || got != want {
			t.Errorf("bool %v = %v, %v", v, got, err)
		}
	}
	if got, _ := convertValue(int64(42), kindText); got != "42" {
		t.Errorf("an integer in a text column = %v", got)
	}
	if got, _ := convertValue(nil, kindTime); got != nil {
		t.Errorf("NULL = %v", got)
	}
}

func TestOrderByForeignKeys(t *testing.T) {
	got := orderByForeignKeys([]string{"tags", "devices", "addresses", "networks", "datacenters"}, []foreignKey{
		{"tags", "device_id", "devices", "id"},
		{"addresses", "device_id", "devices", "id"},
		{"addresses", "network_id", "networks", "id"},
		{"devices", "datacenter_id", "datacenters", "id"},
		{"networks", "datacenter_id", "datacenters", "id"},
	})
	want := []string{"datacenters", "devices", "networks", "addresses", "tags"}
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

// copyEndpoints returns a migrated SQLite source and an empty target on the
// test's server backend.
func copyEndpoints(t *testing.T) (*SQLiteStorage, *sql.DB) {
	t.Helper()
	b := testBackend()
	if b == xdal.DialectSQLite {
		t.Skip("copies into PostgreSQL or MySQL")
	}
	src, err := NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = src.Close() })
	dst := xdaltest.Open(t, b).SQL
	if b == xdal.DialectPostgres {
		dst, _ = reopenPostgres(t, xdaltest.Open(t, b))
	}
	return src, dst
}

func migrateTarget(t *testing.T, dst *sql.DB) {
	t.Helper()
	if _, err := RunMigrationsWithOptions(context.Background(), dst, MigrationOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestCopyDatabase(t *testing.T) {
	src, dst := copyEndpoints(t)
	ctx := context.Background()

	dc := &model.Datacenter{Name: "Perth"}
	if err := src.CreateDatacenter(ctx, dc); err != nil {
		t.Fatal(err)
	}
	net := &model.Network{Name: "LAN", Subnet: "10.0.0.0/24", DatacenterID: dc.ID}
	if err := src.CreateNetwork(ctx, net); err != nil {
		t.Fatal(err)
	}
	dev := &model.Device{Name: "core-router", DatacenterID: dc.ID, Tags: []string{"core", "edge"}, Domains: []string{"r1.example.com"},
		Addresses: []model.Address{{IP: "10.0.0.1", Type: "ipv4", NetworkID: net.ID}}}
	if err := src.CreateDevice(ctx, dev); err != nil {
		t.Fatal(err)
	}
	child := &model.Device{Name: "access-sw"}
	if err := src.CreateDevice(ctx, child); err != nil {
		t.Fatal(err)
	}
	if err := src.AddRelationship(ctx, dev.ID, child.ID, "contains", ""); err != nil {
		t.Fatal(err)
	}
	user, err := createUser(t, src, "alice", "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := src.GetRoleByName(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.AssignRoleToUser(ctx, user.ID, admin.ID); err != nil {
		t.Fatal(err)
	}
	hook := &model.Webhook{Name: "ops", URL: "https://example.com/hook", Active: true, Events: []model.EventType{"device.created"}}
	if err := src.CreateWebhook(ctx, hook); err != nil {
		t.Fatal(err)
	}
	srcDev, err := src.GetDevice(ctx, dev.ID)
	if err != nil {
		t.Fatal(err)
	}

	migrateTarget(t, dst)
	var tables []string
	copied, err := CopyDatabase(ctx, src.DB(), dst, func(table string, _ int) { tables = append(tables, table) })
	if err != nil {
		t.Fatalf("copy: %v", err)
	}
	if len(tables) != len(copied) {
		t.Errorf("progress reported %d tables, copy %d", len(tables), len(copied))
	}
	for _, c := range copied {
		var n int
		if err := src.DB().QueryRow(`SELECT COUNT(*) FROM ` + sqliteIdent(c.Name)).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != c.Rows {
			t.Errorf("%s: copied %d rows, SQLite has %d", c.Name, c.Rows, n)
		}
	}

	// The copy reads back through storage like any database.
	s, err := openStorageOn(ctx, dst, xdalFor(dst))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetDevice(ctx, dev.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "core-router" || len(got.Addresses) != 1 || got.Addresses[0].IP != "10.0.0.1" ||
		!slices.Equal(got.Tags, srcDev.Tags) || !slices.Equal(got.Domains, srcDev.Domains) {
		t.Errorf("device = %+v, want %+v", got, srcDev)
	}
	if !got.CreatedAt.Equal(srcDev.CreatedAt.UTC().Truncate(time.Microsecond)) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, srcDev.CreatedAt)
	}
	if rels, err := s.GetRelationships(ctx, dev.ID); err != nil || len(rels) != 1 {
		t.Errorf("relationships = %v, %v", rels, err)
	}
	u, err := s.GetUser(ctx, user.ID)
	if err != nil || !u.IsAdmin {
		t.Errorf("user = %+v, %v; want an admin", u, err)
	}
	h, err := s.GetWebhook(ctx, hook.ID)
	if err != nil || !h.Active {
		t.Errorf("webhook = %+v, %v; want active", h, err)
	}
	if found, err := s.SearchDevices(ctx, "router"); err != nil || len(found) != 1 {
		t.Errorf("search = %d, %v", len(found), err)
	}
	// The default datacenter isn't added a second time.
	srcDCs, _ := src.ListDatacenters(ctx, nil)
	if dcs, err := s.ListDatacenters(ctx, nil); err != nil || len(dcs) != len(srcDCs) {
		t.Errorf("datacenters = %d, %v; want the %d copied", len(dcs), err, len(srcDCs))
	}
	_ = s.Close()
}

func TestCopyDatabaseChecksFirst(t *testing.T) {
	src, dst := copyEndpoints(t)
	ctx := context.Background()

	long := &model.Device{Name: strings.Repeat("x", keySize+1)}
	if err := src.CreateDevice(ctx, long); err != nil {
		t.Fatal(err)
	}
	if _, err := src.DB().Exec(`UPDATE devices SET created_at = 'last tuesday' WHERE id = ?`, long.ID); err != nil {
		t.Fatal(err)
	}
	// A tag whose device is gone, from before foreign keys were enforced.
	conn, err := src.DB().Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO tags (device_id, tag) VALUES ('gone', 'orphan')`); err != nil {
		t.Fatal(err)
	}
	_, _ = conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
	_ = conn.Close()

	migrateTarget(t, dst)
	_, err = CopyDatabase(ctx, src.DB(), dst, nil)
	var check *CopyCheckError
	if !errors.As(err, &check) {
		t.Fatalf("copy = %v; want a CopyCheckError", err)
	}
	msg := err.Error()
	for _, want := range []string{"devices.name of " + long.ID, "256 characters, the column holds 255", "devices.created_at", "last tuesday", "tags.device_id", "gone"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error doesn't mention %q:\n%s", want, msg)
		}
	}
	// Nothing was written, so the copy can run again once the data is fixed.
	if err := CheckCopyTarget(ctx, dst); err != nil {
		t.Errorf("target after a refused copy: %v", err)
	}
}

func TestCopyDatabaseRefusesNULOnPostgres(t *testing.T) {
	if testBackend() != xdal.DialectPostgres {
		t.Skip("PostgreSQL refuses NUL in text")
	}
	src, dst := copyEndpoints(t)
	ctx := context.Background()
	if err := src.CreateDevice(ctx, &model.Device{Name: "nul", Description: "a\x00b"}); err != nil {
		t.Fatal(err)
	}
	migrateTarget(t, dst)
	if _, err := CopyDatabase(ctx, src.DB(), dst, nil); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Errorf("copy = %v; want the NUL reported", err)
	}
}

func TestCopyDatabaseRefusesTargetWithData(t *testing.T) {
	src, dst := copyEndpoints(t)
	ctx := context.Background()
	s, err := openStorageOn(ctx, dst, xdalFor(dst)) // adds the default datacenter
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, err := CopyDatabase(ctx, src.DB(), dst, nil); err == nil || !strings.Contains(err.Error(), "datacenters (1 rows)") {
		t.Errorf("copy = %v; want a refusal naming datacenters", err)
	}

	other := xdaltest.Open(t, testBackend()).SQL
	if _, err := other.ExecContext(ctx, `CREATE TABLE stray (id INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if err := CheckCopyTarget(ctx, other); err == nil || !strings.Contains(err.Error(), "aren't rackd's") {
		t.Errorf("target with other tables = %v", err)
	}
}

func TestCopyDatabaseRefusesDifferentVersions(t *testing.T) {
	src, dst := copyEndpoints(t)
	ctx := context.Background()
	migrateTarget(t, dst)
	steps := []migrate.Migration{{
		Version: "20260101000000",
		Name:    "create_gadgets",
		Up: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `CREATE TABLE ext_gadgets (id INTEGER PRIMARY KEY)`)
		},
	}}
	if _, err := RunExtensionMigrationsOn(ctx, src.DB(), "gadgets", steps, MigrationOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := CopyDatabase(ctx, src.DB(), dst, nil); err == nil || !strings.Contains(err.Error(), "gadgets") {
		t.Errorf("copy = %v; want the extension migration named", err)
	}
}

// MySQL compares text without case, so values SQLite keeps apart collide on
// a unique index; the copy stops with MySQL's error and writes nothing.
func TestCopyDatabaseCaseCollisionOnMySQL(t *testing.T) {
	if testBackend() != xdal.DialectMySQL {
		t.Skip("MySQL compares text without case")
	}
	src, dst := copyEndpoints(t)
	ctx := context.Background()
	if err := src.CreateDevice(ctx, &model.Device{Name: "sw", Tags: []string{"Lab", "lab"}}); err != nil {
		t.Fatal(err)
	}
	migrateTarget(t, dst)
	_, err := CopyDatabase(ctx, src.DB(), dst, nil)
	if err == nil || !strings.Contains(err.Error(), "writing tags") || !strings.Contains(err.Error(), "Duplicate entry") {
		t.Fatalf("copy = %v; want MySQL's duplicate-entry error for tags", err)
	}
	if err := CheckCopyTarget(ctx, dst); err != nil {
		t.Errorf("target after a failed copy: %v", err)
	}
}
