package auth

import (
	"context"

	"github.com/martinsuchenak/rackd/pkg/model"
)

type Checker interface {
	GetUserRoles(ctx context.Context, userID string) ([]model.Role, error)
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
