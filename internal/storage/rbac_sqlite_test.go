package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestRBACStoragePermissionsRolesAndAssignments(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()
	ctx := context.Background()

	user := &model.User{
		Username:     "rbac-user",
		Email:        "rbac@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	if err := storage.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	perm := &model.Permission{
		Name:     "devices:list:test",
		Resource: "devices",
		Action:   "list",
	}
	if err := storage.CreatePermission(ctx, perm); err != nil {
		t.Fatalf("CreatePermission failed: %v", err)
	}

	gotPerm, err := storage.GetPermission(ctx, perm.ID)
	if err != nil {
		t.Fatalf("GetPermission failed: %v", err)
	}
	if gotPerm.Name != perm.Name {
		t.Fatalf("unexpected permission: %+v", gotPerm)
	}

	gotPerm, err = storage.GetPermissionByName(ctx, perm.Name)
	if err != nil {
		t.Fatalf("GetPermissionByName failed: %v", err)
	}
	if gotPerm.ID != perm.ID {
		t.Fatalf("permission lookup by name returned wrong ID: %+v", gotPerm)
	}

	perms, err := storage.ListPermissions(ctx, &model.PermissionFilter{Resource: "devices", Action: "list"})
	if err != nil {
		t.Fatalf("ListPermissions failed: %v", err)
	}
	if len(perms) == 0 {
		t.Fatal("expected filtered permissions to include created permission")
	}

	role := &model.Role{Name: "test-rbac-role", Description: "role for tests"}
	if err := storage.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	gotRole, err := storage.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}
	if gotRole.Name != role.Name {
		t.Fatalf("unexpected role: %+v", gotRole)
	}

	gotRole, err = storage.GetRoleByName(ctx, role.Name)
	if err != nil {
		t.Fatalf("GetRoleByName failed: %v", err)
	}
	if gotRole.ID != role.ID {
		t.Fatalf("role lookup by name returned wrong ID: %+v", gotRole)
	}

	roles, err := storage.ListRoles(ctx, &model.RoleFilter{Name: "test-rbac"})
	if err != nil {
		t.Fatalf("ListRoles failed: %v", err)
	}
	if len(roles) == 0 {
		t.Fatal("expected filtered roles to include created role")
	}

	if err := storage.SetRolePermissions(ctx, role.ID, []string{perm.ID}); err != nil {
		t.Fatalf("SetRolePermissions failed: %v", err)
	}
	rolePerms, err := storage.GetRolePermissions(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRolePermissions failed: %v", err)
	}
	if len(rolePerms) != 1 || rolePerms[0].ID != perm.ID {
		t.Fatalf("unexpected role permissions: %+v", rolePerms)
	}

	if err := storage.RemoveRolePermission(ctx, role.ID, perm.ID); err != nil {
		t.Fatalf("RemoveRolePermission failed: %v", err)
	}
	rolePerms, err = storage.GetRolePermissions(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRolePermissions after remove failed: %v", err)
	}
	if len(rolePerms) != 0 {
		t.Fatalf("expected permissions to be removed, got %+v", rolePerms)
	}

	if err := storage.AddRolePermission(ctx, role.ID, perm.ID); err != nil {
		t.Fatalf("AddRolePermission failed: %v", err)
	}
	if err := storage.AssignRoleToUser(ctx, user.ID, role.ID); err != nil {
		t.Fatalf("AssignRoleToUser failed: %v", err)
	}

	userRoles, err := storage.GetUserRoles(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserRoles failed: %v", err)
	}
	if len(userRoles) == 0 {
		t.Fatal("expected assigned role for user")
	}

	userPerms, err := storage.GetUserPermissions(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetUserPermissions failed: %v", err)
	}
	if len(userPerms) == 0 {
		t.Fatal("expected permissions inherited from role")
	}

	allowed, err := storage.HasPermission(ctx, user.ID, "devices", "list")
	if err != nil {
		t.Fatalf("HasPermission failed: %v", err)
	}
	if !allowed {
		t.Fatal("expected permission check to pass")
	}

	if err := storage.RemoveRoleFromUser(ctx, user.ID, role.ID); err != nil {
		t.Fatalf("RemoveRoleFromUser failed: %v", err)
	}
	allowed, err = storage.HasPermission(ctx, user.ID, "devices", "list")
	if err != nil {
		t.Fatalf("HasPermission after remove failed: %v", err)
	}
	if allowed {
		t.Fatal("expected permission check to fail after removing role")
	}

	role.Description = "updated description"
	if err := storage.UpdateRole(ctx, role); err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}

	if err := storage.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	if _, err := storage.GetRole(ctx, role.ID); err != ErrRoleNotFound {
		t.Fatalf("expected ErrRoleNotFound, got %v", err)
	}

	if err := storage.DeletePermission(ctx, perm.ID); err != nil {
		t.Fatalf("DeletePermission failed: %v", err)
	}
	if _, err := storage.GetPermission(ctx, perm.ID); err != ErrPermissionNotFound {
		t.Fatalf("expected ErrPermissionNotFound, got %v", err)
	}
}

func TestRBACStorageSystemRoleDeleteBlocked(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	ctx := context.Background()
	role := &model.Role{Name: "system-delete-test", Description: "system", IsSystem: true}
	if err := storage.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	if err := storage.DeleteRole(ctx, role.ID); err == nil {
		t.Fatal("expected system role delete to fail")
	}
}
