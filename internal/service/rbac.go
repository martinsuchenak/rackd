package service

import (
	"context"
	"fmt"

	rbac "github.com/martinsuchenak/go-rbac"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type permissionStore interface {
	GetUserRoles(ctx context.Context, userID string) ([]model.Role, error)
	RBACManager() *rbac.Manager
}

// userCan reports whether userID's roles grant resource:action.
func userCan(ctx context.Context, store permissionStore, userID, resource, action string) (bool, error) {
	roles, err := store.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	return store.RBACManager().Can(storage.RBACRoles(roles), rbac.Permission(resource+":"+action)), nil
}

func requirePermission(ctx context.Context, store permissionStore, resource, action string) error {
	caller := CallerFrom(ctx)
	if caller != nil && caller.IsSystem() {
		return nil
	}

	// All API keys must go through RBAC. Legacy API keys (without user association)
	// are no longer supported and will be denied access. Users should migrate to
	// user-associated API keys which enforce RBAC based on the owner's roles.
	if caller == nil || caller.UserID == "" {
		log.Debug("RBAC: unauthenticated caller", "resource", resource, "action", action)
		return ErrUnauthenticated
	}

	// Check OAuth scope restriction: if scopes are set, the requested
	// resource:action must be in the token's scope list.
	if caller.Scopes != nil {
		scopeKey := resource + ":" + action
		found := false
		for _, s := range caller.Scopes {
			if s == scopeKey || s == "*" {
				found = true
				break
			}
		}
		if !found {
			log.Debug("RBAC: scope denied", "user_id", caller.UserID, "scope_key", scopeKey, "scopes", caller.Scopes)
			return ErrForbidden
		}
	}

	has, err := userCan(ctx, store, caller.UserID, resource, action)
	if err != nil {
		log.Error("RBAC: permission check error", "error", err, "user_id", caller.UserID, "resource", resource, "action", action)
		return fmt.Errorf("checking permission %s:%s: %w", resource, action, err)
	}

	if !has {
		log.Debug("RBAC: permission denied", "user_id", caller.UserID, "resource", resource, "action", action)
		return ErrForbidden
	}

	return nil
}
