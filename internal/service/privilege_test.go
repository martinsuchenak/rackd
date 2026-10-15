package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type privilegeFixture struct {
	t     *testing.T
	store *storage.SQLiteStorage
	users *UserService
	roles *RoleService
	perms map[string]string // "resource:action" -> permission ID
	admin string            // the bootstrap admin's ID
}

func newPrivilegeFixture(t *testing.T) *privilegeFixture {
	t.Helper()
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	ctx := SystemContext(context.Background(), "test")
	if err := store.CreateInitialAdmin(ctx, "root", "root@example.test", "Root", "password123"); err != nil {
		t.Fatal(err)
	}
	root, err := store.GetUserByUsername(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	all, err := store.ListPermissions(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	perms := map[string]string{}
	for _, p := range all {
		perms[p.Resource+":"+p.Action] = p.ID
	}
	return &privilegeFixture{t: t, store: store, users: NewUserService(store, nil), roles: NewRoleService(store), perms: perms, admin: root.ID}
}

func (f *privilegeFixture) sys() context.Context { return SystemContext(context.Background(), "test") }

// role creates a custom role holding perms.
func (f *privilegeFixture) role(name string, perms ...string) string {
	f.t.Helper()
	r := &model.Role{ID: uuid.NewString(), Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := f.store.CreateRole(f.sys(), r); err != nil {
		f.t.Fatal(err)
	}
	ids := make([]string, len(perms))
	for i, p := range perms {
		id, ok := f.perms[p]
		if !ok {
			f.t.Fatalf("unknown permission %s", p)
		}
		ids[i] = id
	}
	if err := f.store.SetRolePermissions(f.sys(), r.ID, ids); err != nil {
		f.t.Fatal(err)
	}
	return r.ID
}

// user creates an active user holding roles.
func (f *privilegeFixture) user(name string, roles ...string) string {
	f.t.Helper()
	u := &model.User{ID: uuid.NewString(), Username: name, Email: name + "@example.test", PasswordHash: "x", IsActive: true}
	if err := f.store.CreateUser(f.sys(), u); err != nil {
		f.t.Fatal(err)
	}
	for _, r := range roles {
		if err := f.store.AssignRoleToUser(f.sys(), u.ID, r); err != nil {
			f.t.Fatal(err)
		}
	}
	return u.ID
}

func (f *privilegeFixture) adminRole() string {
	r, err := f.store.GetRoleByName(f.sys(), "admin")
	if err != nil {
		f.t.Fatal(err)
	}
	return r.ID
}

func as(userID string) context.Context { return userContext(userID) }

func wantForbidden(t *testing.T, what string, err error) {
	t.Helper()
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("%s: got %v, want forbidden", what, err)
	}
}

func TestDelegatedUserManagementCannotTakeOverMorePrivilegedUsers(t *testing.T) {
	f := newPrivilegeFixture(t)
	helpdesk := f.user("helpdesk", f.role("helpdesk", "users:list", "users:read", "users:update", "users:delete"))
	// An admin promoted the usual way, by granting the admin role.
	promoted := f.user("promoted", f.adminRole())
	// A non-admin who holds a permission the helpdesk lacks.
	manager := f.user("manager", f.role("role-manager", "roles:update"))
	// A user whose permissions the helpdesk also holds.
	peer := f.user("peer", f.role("reader", "users:list"))

	reset := &model.ResetPasswordRequest{NewPassword: "taken-over-1"}
	wantForbidden(t, "reset a promoted admin's password", f.users.ResetPassword(as(helpdesk), promoted, reset))
	wantForbidden(t, "reset the bootstrap admin's password", f.users.ResetPassword(as(helpdesk), f.admin, reset))
	wantForbidden(t, "reset a more privileged user's password", f.users.ResetPassword(as(helpdesk), manager, reset))
	inactive := false
	_, err := f.users.Update(as(helpdesk), promoted, &model.UpdateUserRequest{IsActive: &inactive})
	wantForbidden(t, "deactivate a promoted admin", err)
	wantForbidden(t, "delete a promoted admin", f.users.Delete(as(helpdesk), promoted))

	if err := f.users.ResetPassword(as(helpdesk), peer, reset); err != nil {
		t.Errorf("reset a less privileged user's password: %v", err)
	}
}

func TestRoleManagersCanOnlyGrantPermissionsTheyHold(t *testing.T) {
	f := newPrivilegeFixture(t)
	ownRole := f.role("role-manager", "roles:list", "roles:read", "roles:update")
	manager := f.user("manager", ownRole)
	other := f.role("other")

	// Giving their own role everything.
	all := make([]string, 0, len(f.perms))
	for _, id := range f.perms {
		all = append(all, id)
	}
	wantForbidden(t, "grant every permission to own role", f.roles.SetPermissions(as(manager), ownRole, all))
	wantForbidden(t, "grant users:update to own role", f.roles.GrantPermission(as(manager), ownRole, f.perms["users:update"]))
	wantForbidden(t, "grant users:update to another role", f.roles.SetPermissions(as(manager), other, []string{f.perms["users:update"]}))

	// Stripping a permission they don't hold from a role is also refused.
	stocked := f.role("stocked", "devices:delete")
	wantForbidden(t, "remove devices:delete from a role", f.roles.SetPermissions(as(manager), stocked, nil))

	// Permissions they hold can be granted and removed.
	if err := f.roles.SetPermissions(as(manager), other, []string{f.perms["roles:list"]}); err != nil {
		t.Errorf("grant a held permission: %v", err)
	}
	if err := f.roles.SetPermissions(as(manager), other, nil); err != nil {
		t.Errorf("remove a held permission: %v", err)
	}

	// A role carrying permissions they lack can't be assigned, to themselves
	// or anyone, and a system role needs an admin.
	powerful := f.role("powerful", "users:update")
	wantForbidden(t, "assign a more powerful role to self", f.roles.AssignToUser(as(manager), manager, powerful))
	operator, err := f.store.GetRoleByName(f.sys(), "operator")
	if err != nil {
		t.Fatal(err)
	}
	wantForbidden(t, "assign the operator system role", f.roles.AssignToUser(as(manager), manager, operator.ID))
}

func TestUserCreatorsCanOnlyAssignRolesTheyCouldGrant(t *testing.T) {
	f := newPrivilegeFixture(t)
	creator := f.user("creator", f.role("creator", "users:create", "users:list"))
	powerful := f.role("powerful", "roles:update")
	operator, err := f.store.GetRoleByName(f.sys(), "operator")
	if err != nil {
		t.Fatal(err)
	}

	create := func(name, roleID string) error {
		_, err := f.users.Create(as(creator), &model.CreateUserRequest{Username: name, Email: name + "@example.test", Password: "password123", RoleID: roleID})
		return err
	}
	wantForbidden(t, "create a user with a more powerful custom role", create("u1", powerful))
	wantForbidden(t, "create a user with a system role", create("u2", operator.ID))
	if _, err := f.store.GetUserByUsername(f.sys(), "u1"); err == nil {
		t.Error("the refused user was created anyway")
	}
	if err := create("u3", f.role("lesser", "users:list")); err != nil {
		t.Errorf("create a user with a lesser role: %v", err)
	}
	if err := create("u4", "no-such-role"); !errors.Is(err, ErrValidation) {
		t.Errorf("create a user with an unknown role: got %v, want a validation error", err)
	}
}

func TestIsAdminFollowsTheAdminRole(t *testing.T) {
	f := newPrivilegeFixture(t)
	isAdmin := func(id string) bool {
		u, err := f.store.GetUser(f.sys(), id)
		if err != nil {
			t.Fatal(err)
		}
		return u.IsAdmin
	}

	u := f.user("u")
	if err := f.roles.AssignToUser(as(f.admin), u, f.adminRole()); err != nil {
		t.Fatal(err)
	}
	if !isAdmin(u) {
		t.Error("granting the admin role did not set is_admin")
	}
	if err := f.roles.RevokeFromUser(as(f.admin), u, f.adminRole()); err != nil {
		t.Fatal(err)
	}
	if isAdmin(u) {
		t.Error("revoking the admin role did not clear is_admin")
	}

	// is_admin on create and update grants and removes the role.
	created, err := f.users.Create(as(f.admin), &model.CreateUserRequest{Username: "a2", Email: "a2@example.test", Password: "password123", IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	if !created.IsAdmin || !isAdmin(created.ID) {
		t.Error("creating with is_admin did not make an admin")
	}
	no := false
	if _, err := f.users.Update(as(f.admin), created.ID, &model.UpdateUserRequest{IsAdmin: &no}); err != nil {
		t.Fatal(err)
	}
	if isAdmin(created.ID) {
		t.Error("updating is_admin to false left the admin role")
	}
}

func TestTheLastAdminCannotBeRemoved(t *testing.T) {
	f := newPrivilegeFixture(t)
	second := f.user("second", f.adminRole())

	// Two admins: one can demote the other.
	if err := f.roles.RevokeFromUser(as(f.admin), second, f.adminRole()); err != nil {
		t.Fatalf("demote one of two admins: %v", err)
	}
	// Now the bootstrap admin is the only one.
	if err := f.roles.RevokeFromUser(as(f.admin), f.admin, f.adminRole()); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("demote the last admin: got %v, want ErrLastAdmin", err)
	}
	inactive := false
	if _, err := f.users.Update(f.sys(), f.admin, &model.UpdateUserRequest{IsActive: &inactive}); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("deactivate the last admin: got %v, want ErrLastAdmin", err)
	}
	if err := f.users.Delete(f.sys(), f.admin); !errors.Is(err, ErrLastAdmin) {
		t.Errorf("delete the last admin: got %v, want ErrLastAdmin", err)
	}
}

func TestSystemRolesFollowTheSubsetRule(t *testing.T) {
	f := newPrivilegeFixture(t)
	viewer, err := f.store.GetRoleByName(f.sys(), "viewer")
	if err != nil {
		t.Fatal(err)
	}
	viewerPerms, err := f.store.GetRolePermissions(f.sys(), viewer.ID)
	if err != nil {
		t.Fatal(err)
	}
	// A helpdesk that holds everything a viewer can do may create viewers.
	held := []string{"users:create"}
	for _, p := range viewerPerms {
		held = append(held, p.Resource+":"+p.Action)
	}
	helpdesk := f.user("helpdesk", f.role("helpdesk", held...))
	if _, err := f.users.Create(as(helpdesk), &model.CreateUserRequest{Username: "v", Email: "v@example.test", Password: "password123", RoleID: viewer.ID}); err != nil {
		t.Errorf("create a viewer: %v", err)
	}
	// The admin role stays admin-only, whatever the caller holds.
	wantForbidden(t, "assign the admin role", f.roles.AssignToUser(as(helpdesk), helpdesk, f.adminRole()))
}

func TestARefusedRoleIsNotCreated(t *testing.T) {
	f := newPrivilegeFixture(t)
	manager := f.user("manager", f.role("role-manager", "roles:create", "roles:update", "roles:list"))
	role := &model.Role{Name: "sneaky"}
	wantForbidden(t, "create a role with users:update", f.roles.CreateWithPermissions(as(manager), role, []string{f.perms["users:update"]}))
	if _, err := f.store.GetRoleByName(f.sys(), "sneaky"); err == nil {
		t.Error("the refused role was created anyway")
	}
	if err := f.roles.CreateWithPermissions(as(manager), &model.Role{Name: "fine"}, []string{f.perms["roles:list"]}); err != nil {
		t.Errorf("create a role with a held permission: %v", err)
	}
}

func TestAnOAuthTokenGrantsOnlyWithinItsScope(t *testing.T) {
	f := newPrivilegeFixture(t)
	manager := f.user("manager", f.role("role-manager", "roles:update", "users:update"))
	other := f.role("other")
	scoped := WithCaller(context.Background(), &Caller{Type: CallerTypeUser, UserID: manager, Scopes: []string{"roles:update"}})
	// The user holds users:update, but the token doesn't.
	wantForbidden(t, "grant a permission outside the token's scope", f.roles.GrantPermission(scoped, other, f.perms["users:update"]))
	if err := f.roles.GrantPermission(scoped, other, f.perms["roles:update"]); err != nil {
		t.Errorf("grant a permission within the scope: %v", err)
	}
}
