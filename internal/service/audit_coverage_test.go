package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// Security-relevant changes reach the audit log, with names and IDs but never
// secrets or full webhook URLs.
func TestSecurityChangesAreAudited(t *testing.T) {
	f := newPrivilegeFixture(t)
	admin := as(f.admin)
	ctx := context.Background()

	user, err := f.users.Create(admin, &model.CreateUserRequest{Username: "bob", Email: "bob@example.test", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.users.ResetPassword(admin, user.ID, &model.ResetPasswordRequest{NewPassword: "a-new-password"}); err != nil {
		t.Fatal(err)
	}
	role := f.role("helpdesk")
	if err := f.roles.SetPermissions(admin, role, []string{f.perms["users:list"]}); err != nil {
		t.Fatal(err)
	}
	if err := f.roles.AssignToUser(admin, user.ID, role); err != nil {
		t.Fatal(err)
	}
	if _, err := NewAPIKeyService(f.store).Create(admin, &model.APIKey{Name: "ci"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewWebhookService(f.store).Create(admin, &model.CreateWebhookRequest{
		Name: "slack", URL: "https://hooks.slack.com/services/T0/B0/SECRETPATH", Secret: "SIGNINGSECRET",
		Events: []model.EventType{model.EventTypeDeviceCreated}, Active: true,
	}); err != nil {
		t.Fatal(err)
	}
	authSvc := NewAuthService(f.store, auth.NewSessionManager(time.Hour, nil))
	attempt := audit.WithContext(ctx, &audit.Context{Username: "bob", IPAddress: "203.0.113.7", Source: "api"})
	if _, err := authSvc.Login(attempt, "bob", "wrong"); err == nil {
		t.Fatal("wrong password accepted")
	}
	if _, err := authSvc.Login(attempt, "bob", "a-new-password"); err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"create user": false, "password_reset user": false, "set_permissions role": false,
		"assign_role user": false, "create apikey": false, "create webhook": false,
		"login session failure": false, "login session success": false,
	}
	var logs []model.AuditLog
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		logs, err = f.store.ListAuditLogs(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range logs {
			key := l.Action + " " + l.Resource
			if l.Action == "login" {
				key += " " + l.Status
			}
			if _, ok := want[key]; ok {
				want[key] = true
			}
		}
		done := true
		for _, seen := range want {
			done = done && seen
		}
		if done {
			break
		}
	}
	for key, seen := range want {
		if !seen {
			t.Errorf("no audit entry for %s", key)
		}
	}
	for _, l := range logs {
		for _, secret := range []string{"SECRETPATH", "SIGNINGSECRET", "a-new-password", "password123"} {
			if strings.Contains(l.Changes, secret) {
				t.Errorf("audit entry %s %s holds %q: %s", l.Action, l.Resource, secret, l.Changes)
			}
		}
		if l.Action == "login" && l.Status == "failure" && (l.IPAddress != "203.0.113.7" || l.Error != "wrong password") {
			t.Errorf("failed login entry = %+v", l)
		}
	}
}
