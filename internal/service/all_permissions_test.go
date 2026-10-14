package service

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// Rackd has more permissions than the default page of 100; every one must be
// assignable to a role and issuable as an OAuth scope.
func TestEveryPermissionIsUsable(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := SystemContext(context.Background(), "test")

	perms, err := store.ListPermissions(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) <= model.DefaultPageSize {
		t.Fatalf("only %d permissions; this test needs more than %d", len(perms), model.DefaultPageSize)
	}

	role := &model.Role{ID: "custom-role", Name: "custom", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatal(err)
	}
	last := perms[len(perms)-1]
	if err := NewRoleService(store).SetPermissions(ctx, role.ID, []string{last.ID}); err != nil {
		t.Errorf("assigning permission #%d (%s:%s): %v", len(perms), last.Resource, last.Action, err)
	}

	catalog, err := NewOAuthService(store, nil, "https://rackd.example.test").scopeCatalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog) != len(perms) {
		t.Errorf("OAuth scope catalog has %d scopes, want %d", len(catalog), len(perms))
	}
}
