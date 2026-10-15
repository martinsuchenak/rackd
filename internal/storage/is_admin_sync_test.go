package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Before this migration, users promoted by granting the admin role kept
// is_admin = 0, and users created with is_admin = 1 had no admin rights.
func TestSyncIsAdminMigration(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	admin, err := s.GetRoleByName(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	promoted := &model.User{Username: "promoted", Email: "p@example.test", PasswordHash: "x", IsActive: true}
	flagged := &model.User{Username: "flagged", Email: "f@example.test", PasswordHash: "x", IsActive: true}
	for _, u := range []*model.User{promoted, flagged} {
		if err := s.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	// Recreate the old, inconsistent state directly.
	if _, err := s.db.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id, created_at) VALUES (?, ?, CURRENT_TIMESTAMP)`, promoted.ID, admin.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE users SET is_admin = 1 WHERE id = ?`, flagged.ID); err != nil {
		t.Fatal(err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateSyncIsAdminUp(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		user *model.User
		want bool
	}{{promoted, true}, {flagged, false}} {
		got, err := s.GetUser(ctx, tc.user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.IsAdmin != tc.want {
			t.Errorf("%s: is_admin = %v, want %v", tc.user.Username, got.IsAdmin, tc.want)
		}
	}
}

func TestScrubDeliveryErrorURLsMigration(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	hook := &model.Webhook{Name: "h", URL: "https://hooks.example/x", Active: true, Events: []model.EventType{model.EventTypeDeviceCreated}}
	if err := s.CreateWebhook(ctx, hook); err != nil {
		t.Fatal(err)
	}
	for id, text := range map[string]string{
		"d1": `failed to send request: Post "https://hooks.slack.com/services/T0/B0/secret": dial tcp: connection refused`,
		"d2": `unexpected status 500`,
	} {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO webhook_deliveries (id, webhook_id, event_type, payload, status, error, created_at) VALUES (?, ?, 'device.created', '{}', 'failed', ?, CURRENT_TIMESTAMP)`, id, hook.ID, text); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := migrateScrubDeliveryErrorURLsUp(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var got1, got2 string
	_ = s.db.QueryRowContext(ctx, `SELECT error FROM webhook_deliveries WHERE id = 'd1'`).Scan(&got1)
	_ = s.db.QueryRowContext(ctx, `SELECT error FROM webhook_deliveries WHERE id = 'd2'`).Scan(&got2)
	if got1 != `failed to send request: Post "<webhook URL>": dial tcp: connection refused` {
		t.Errorf("scrubbed error = %q", got1)
	}
	if got2 != "unexpected status 500" {
		t.Errorf("unrelated error changed to %q", got2)
	}
}
