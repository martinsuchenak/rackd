package storage

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Identity storage that differs between backends: upserts, boolean flags and
// case-insensitive name filters. These run on every backend.

func TestSSHHostKeySaveReplaces(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	if got, err := s.GetSSHHostKey(ctx, "switch1:22"); err != nil || got != nil {
		t.Fatalf("unknown host = %v, %v; want nil, nil", got, err)
	}
	if err := s.SaveSSHHostKey(ctx, "switch1:22", []byte("first")); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := s.SaveSSHHostKey(ctx, "switch1:22", []byte("second")); err != nil {
		t.Fatalf("second save: %v", err)
	}
	got, err := s.GetSSHHostKey(ctx, "switch1:22")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Equal(got, []byte("second")) {
		t.Errorf("key = %q, want the second save", got)
	}
}

func TestCleanupExpiredCodesRemovesUsedAndExpired(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	createTestUserAndClient(t, s)

	code := func(hash string, expires time.Time) {
		t.Helper()
		if err := s.CreateAuthorizationCode(ctx, &model.OAuthAuthorizationCode{
			CodeHash: hash, ClientID: "client1", UserID: "user1", RedirectURI: "http://localhost/cb",
			CodeChallenge: "c", CodeChallengeMethod: "S256", ExpiresAt: expires, CreatedAt: nowUTC(),
		}); err != nil {
			t.Fatalf("create %s: %v", hash, err)
		}
	}
	code("live", nowUTC().Add(time.Hour))
	code("used", nowUTC().Add(time.Hour))
	code("expired", nowUTC().Add(-time.Hour))
	if err := s.MarkAuthorizationCodeUsed(ctx, "used"); err != nil {
		t.Fatalf("mark used: %v", err)
	}
	if err := s.MarkAuthorizationCodeUsed(ctx, "used"); err != ErrOAuthCodeUsed {
		t.Fatalf("second mark used = %v, want ErrOAuthCodeUsed", err)
	}

	if err := s.CleanupExpiredCodes(ctx); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	var left []string
	rows, err := s.q.QueryContext(ctx, `SELECT code_hash FROM oauth_authorization_codes ORDER BY code_hash`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			t.Fatal(err)
		}
		left = append(left, h)
	}
	if len(left) != 1 || left[0] != "live" {
		t.Errorf("codes left = %v, want [live]", left)
	}
}

func TestIdentityFiltersIgnoreCase(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	if _, err := createUser(t, s, "AliceAdmin", "Alice@Example.com"); err != nil {
		t.Fatal(err)
	}
	users, err := s.ListUsers(ctx, &model.UserFilter{Username: "aliceadmin", Email: "alice@example"})
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("users = %d, want 1", len(users))
	}

	if err := s.CreateAPIKey(ctx, &model.APIKey{Name: "Deploy Bot", Key: "k-deploy"}); err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	keys, err := s.ListAPIKeys(ctx, &model.APIKeyFilter{Name: "deploy"})
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("API keys = %d, want 1", len(keys))
	}

	if err := s.CreateRole(ctx, &model.Role{Name: "NetOps"}); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	roles, err := s.ListRoles(ctx, &model.RoleFilter{Name: "netops"})
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(roles) != 1 {
		t.Errorf("roles = %d, want 1", len(roles))
	}
}

func TestAssignRoleTwiceKeepsOneAssignment(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	user, err := createUser(t, s, "twice", "twice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	admin, err := s.GetRoleByName(ctx, "admin")
	if err != nil {
		t.Fatalf("admin role: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := s.AssignRoleToUser(ctx, user.ID, admin.ID); err != nil {
			t.Fatalf("assign %d: %v", i+1, err)
		}
	}
	roles, err := s.GetUserRoles(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 1 {
		t.Errorf("roles = %d, want 1", len(roles))
	}
	got, err := s.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsAdmin {
		t.Error("is_admin not set after assigning the admin role")
	}
}
