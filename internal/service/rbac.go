package service

import (
	"context"
	"fmt"

	"github.com/martinsuchenak/rackd/internal/log"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, resource, action string) (bool, error)
}

func requirePermission(ctx context.Context, checker PermissionChecker, resource, action string) error {
	caller := CallerFrom(ctx)
	if caller != nil && caller.IsSystem() {
		return nil
	}

	// Legacy API keys (no user_id) still use CallerTypeAPIKey and bypass RBAC.
	// New per-user keys are resolved to CallerTypeUser in auth middleware,
	// so they go through normal RBAC below.
	if caller != nil && caller.Type == CallerTypeAPIKey {
		return nil
	}

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

	has, err := checker.HasPermission(ctx, caller.UserID, resource, action)
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
