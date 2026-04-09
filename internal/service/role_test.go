package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestRoleService_AssignToUserRejectsAdminRoleForNonAdmin(t *testing.T) {
	store := newServiceTestStorage()
	store.roles["role-admin"] = &model.Role{ID: "role-admin", Name: "admin"}
	store.setPermission("user-1", "roles", "update", true)
	store.userRoles["user-1"] = []model.Role{{ID: "role-viewer", Name: "viewer"}}

	svc := NewRoleService(store)

	err := svc.AssignToUser(userContext("user-1"), "user-2", "role-admin")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden error, got %v", err)
	}
	if store.assignedRoleID != "" {
		t.Fatalf("expected AssignRoleToUser not to be called, got %q", store.assignedRoleID)
	}
}

func TestRoleService_DeleteRejectsSystemRole(t *testing.T) {
	store := newServiceTestStorage()
	store.roles["role-system"] = &model.Role{ID: "role-system", Name: "system", IsSystem: true}
	store.setPermission("user-1", "roles", "delete", true)

	svc := NewRoleService(store)

	err := svc.Delete(userContext("user-1"), "role-system")
	if !errors.Is(err, ErrSystemRole) {
		t.Fatalf("expected system role error, got %v", err)
	}
}
