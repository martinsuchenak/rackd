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

	if caller == nil || caller.UserID == "" {
		log.Debug("RBAC: unauthenticated caller", "resource", resource, "action", action)
		return ErrUnauthenticated
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
