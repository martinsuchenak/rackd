package auth

import (
	"context"
	"fmt"

	"github.com/martinsuchenak/rackd/internal/model"
)

type Checker interface {
	HasPermission(ctx context.Context, userID, resource, action string) (bool, error)
	GetUserPermissions(ctx context.Context, userID string) ([]model.Permission, error)
	GetUserRoles(ctx context.Context, userID string) ([]model.Role, error)
}

func HasAnyPermission(ctx context.Context, checker Checker, userID string, permissions []string) (bool, error) {
	for _, permName := range permissions {
		resource, action, err := parsePermissionName(permName)
		if err != nil {
			return false, err
		}

		has, err := checker.HasPermission(ctx, userID, resource, action)
		if err != nil {
			return false, err
		}
		if has {
			return true, nil
		}
	}
	return false, nil
}

func HasAllPermissions(ctx context.Context, checker Checker, userID string, permissions []string) (bool, error) {
	for _, permName := range permissions {
		resource, action, err := parsePermissionName(permName)
		if err != nil {
			return false, err
		}

		has, err := checker.HasPermission(ctx, userID, resource, action)
		if err != nil {
			return false, err
		}
		if !has {
			return false, nil
		}
	}
	return true, nil
}

func IsAdmin(ctx context.Context, checker Checker, userID string) (bool, error) {
	roles, err := checker.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role.Name == "admin" {
			return true, nil
		}
	}
	return false, nil
}

func parsePermissionName(name string) (resource, action string, err error) {
	for i, r := range name {
		if r == ':' {
			if i == 0 {
				return "", "", fmt.Errorf("invalid permission name: %s", name)
			}
			return name[:i], name[i+1:], nil
		}
	}
	return "", "", fmt.Errorf("invalid permission name: %s", name)
}
