package service

import (
	"context"
	"errors"
	"strings"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// Delegated user and role management must not let anyone gain rights they
// don't have. Two rules apply to every caller except holders of the admin
// role and the system:
//
//   - A caller can only grant, revoke or assign permissions they hold, so
//     managing roles can't raise anyone, including the caller, above them.
//   - A caller can only act on another user (update, reset or change their
//     password, delete them, change their roles) whose permissions are a
//     subset of the caller's, so nobody can take over a more privileged
//     account.
//
// Admins are exempt because they hold the system's highest role, which only
// admins can assign; the last active admin can't be removed.

// ErrLastAdmin is returned when a change would leave no active admin.
var ErrLastAdmin = errors.New("this would leave no active admin")

// PrivilegeError is a refusal under these rules. It matches ErrForbidden, and
// its message says why, so the API can show it.
type PrivilegeError struct {
	Reason string
}

func (e *PrivilegeError) Error() string        { return e.Reason }
func (e *PrivilegeError) Is(target error) bool { return target == ErrForbidden }

type permissionSet map[string]bool

func newPermissionSet(perms []model.Permission) permissionSet {
	set := make(permissionSet, len(perms))
	for _, p := range perms {
		set[p.Resource+":"+p.Action] = true
	}
	return set
}

// missing returns the permissions in want that the set lacks.
func (s permissionSet) missing(want []model.Permission) []string {
	var out []string
	for _, p := range want {
		if key := p.Resource + ":" + p.Action; !s[key] {
			out = append(out, key)
		}
	}
	return out
}

// privilegeGuard applies the rules above. It reads permissions from storage
// on each call, like requirePermission, so a role change applies at once.
type privilegeGuard struct {
	store storage.ExtendedStorage
}

// callerIsAdmin reports whether the caller is the system or holds the admin role.
func (g privilegeGuard) callerIsAdmin(ctx context.Context) (bool, error) {
	caller := CallerFrom(ctx)
	if caller == nil {
		return false, nil
	}
	if caller.IsSystem() {
		return true, nil
	}
	return auth.IsAdmin(ctx, g.store, caller.UserID)
}

func (g privilegeGuard) callerPermissions(ctx context.Context) (permissionSet, error) {
	caller := CallerFrom(ctx)
	if caller == nil || caller.UserID == "" {
		return nil, ErrUnauthenticated
	}
	perms, err := g.store.GetUserPermissions(ctx, caller.UserID)
	if err != nil {
		return nil, err
	}
	held := newPermissionSet(perms)
	// An OAuth token holds only the permissions in its scope.
	if caller.Scopes != nil {
		scoped := permissionSet{}
		for _, s := range caller.Scopes {
			if s == "*" {
				return held, nil
			}
			if held[s] {
				scoped[s] = true
			}
		}
		held = scoped
	}
	return held, nil
}

// requireHolds fails with ErrForbidden unless the caller holds every
// permission in perms.
func (g privilegeGuard) requireHolds(ctx context.Context, perms []model.Permission) error {
	if admin, err := g.callerIsAdmin(ctx); err != nil || admin {
		return err
	}
	held, err := g.callerPermissions(ctx)
	if err != nil {
		return err
	}
	if missing := held.missing(perms); len(missing) > 0 {
		return &PrivilegeError{Reason: "You can only grant permissions you hold; you don't hold " + strings.Join(missing, ", ")}
	}
	return nil
}

// requireRole fails unless the caller may give or take away roleID: they
// hold all its permissions, and only admins handle the admin role. The other
// system roles follow the same rule as custom ones.
func (g privilegeGuard) requireRole(ctx context.Context, roleID string) (*model.Role, error) {
	role, err := g.store.GetRole(ctx, roleID)
	if err != nil {
		if errors.Is(err, storage.ErrRoleNotFound) {
			return nil, ValidationErrors{{Field: "role_id", Message: "Unknown role"}}
		}
		return nil, err
	}
	admin, err := g.callerIsAdmin(ctx)
	if err != nil {
		return nil, err
	}
	if admin {
		return role, nil
	}
	if role.Name == "admin" {
		return nil, &PrivilegeError{Reason: "Only admins can assign or remove the admin role"}
	}
	perms, err := g.store.GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return role, g.requireHolds(ctx, perms)
}

// requireOutranks fails unless the caller may act on the user targetID:
// the caller is an admin, or holds every permission the target holds.
func (g privilegeGuard) requireOutranks(ctx context.Context, targetID string) error {
	if admin, err := g.callerIsAdmin(ctx); err != nil || admin {
		return err
	}
	if targetAdmin, err := auth.IsAdmin(ctx, g.store, targetID); err != nil {
		return err
	} else if targetAdmin {
		return &PrivilegeError{Reason: "Only admins can manage admins"}
	}
	perms, err := g.store.GetUserPermissions(ctx, targetID)
	if err != nil {
		return err
	}
	if err := g.requireHolds(ctx, perms); err != nil {
		if errors.Is(err, ErrForbidden) {
			return &PrivilegeError{Reason: "You can only manage users whose permissions you also hold"}
		}
		return err
	}
	return nil
}

// requireAnotherAdmin fails with ErrLastAdmin if userID is an admin and no
// other active user is. Call it before deactivating, deleting or demoting
// userID.
func (g privilegeGuard) requireAnotherAdmin(ctx context.Context, userID string) error {
	isAdmin, err := auth.IsAdmin(ctx, g.store, userID)
	if err != nil || !isAdmin {
		return err
	}
	admin := true
	admins, err := g.store.ListUsers(ctx, &model.UserFilter{IsAdmin: &admin})
	if err != nil {
		return err
	}
	for _, u := range admins {
		if u.ID != userID && u.IsActive {
			return nil
		}
	}
	return ErrLastAdmin
}
