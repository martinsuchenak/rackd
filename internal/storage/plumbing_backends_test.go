package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/xdal/pkg/migrate"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Plumbing that differs between backends: extension migrations and
// permissions, and matching webhooks to an event. These run on every backend.

func TestExtensionMigrationsRunOnStorage(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	steps := []migrate.Migration{{
		Version: "20260101000000",
		Name:    "create_gadgets",
		Up: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `CREATE TABLE ext_gadgets (id INTEGER PRIMARY KEY)`)
		},
		Down: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `DROP TABLE ext_gadgets`)
		},
	}}
	res, err := s.RunExtensionMigrations(ctx, "gadgets", steps)
	if err != nil {
		t.Fatalf("RunExtensionMigrations: %v", err)
	}
	if len(res.Versions) != 1 {
		t.Errorf("applied %v, want one version", res.Versions)
	}
	if _, err := s.q.ExecContext(ctx, `INSERT INTO ext_gadgets (id) VALUES (?)`, 1); err != nil {
		t.Errorf("extension table not usable: %v", err)
	}
	if res, err := s.RunExtensionMigrations(ctx, "gadgets", steps); err != nil || len(res.Versions) != 0 {
		t.Errorf("second run = %v, %v; want nothing applied", res.Versions, err)
	}
	if _, err := s.RunExtensionMigrations(ctx, SchemaMigrationNamespace, steps); err == nil {
		t.Error("an extension ran in core's namespace")
	}
}

func TestEnsurePermissionsGrantsDefaultRolesOnce(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	grants := []PermissionGrant{{Resource: "gadgets", Action: "zap", DefaultRoles: []string{"admin", "admin", "no-such-role"}}}
	if err := s.EnsurePermissions(ctx, grants); err != nil {
		t.Fatalf("EnsurePermissions: %v", err)
	}
	// Again: an existing permission is left alone.
	if err := s.EnsurePermissions(ctx, grants); err != nil {
		t.Fatalf("second EnsurePermissions: %v", err)
	}

	var n int
	if err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM role_permissions rp
		JOIN permissions p ON p.id = rp.permission_id
		JOIN roles r ON r.id = rp.role_id
		WHERE p.resource = ? AND p.action = ? AND r.name = ?`, "gadgets", "zap", "admin").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("admin grants = %d, want 1", n)
	}
	if err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM permissions WHERE resource = ? AND action = ?`, "gadgets", "zap").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("permissions = %d, want 1", n)
	}
}

func TestWebhooksForEventMatchExactly(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	add := func(name string, active bool, events ...model.EventType) {
		t.Helper()
		if err := s.CreateWebhook(ctx, &model.Webhook{Name: name, URL: "https://example.com/" + name, Active: active, Events: events}); err != nil {
			t.Fatalf("CreateWebhook %s: %v", name, err)
		}
	}
	add("exact", true, "device.created", "device.deleted")
	add("longer-name", true, "device.created_extra")
	add("lookalike", true, "deviceXcreated")
	add("inactive", false, "device.created")

	got, err := s.GetWebhooksForEvent(ctx, "device.created")
	if err != nil {
		t.Fatalf("GetWebhooksForEvent: %v", err)
	}
	if len(got) != 1 || got[0].Name != "exact" {
		names := make([]string, len(got))
		for i, w := range got {
			names[i] = w.Name
		}
		t.Errorf("webhooks = %v, want [exact]", names)
	}
}
