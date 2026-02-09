package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type RoleService struct {
	store storage.ExtendedStorage
}

func NewRoleService(store storage.ExtendedStorage) *RoleService {
	return &RoleService{store: store}
}

func (s *RoleService) List(ctx context.Context, filter *model.RoleFilter) ([]model.Role, error) {
	if err := requirePermission(ctx, s.store, "roles", "list"); err != nil {
		return nil, err
	}
	return s.store.ListRoles(ctx, filter)
}

func (s *RoleService) Create(ctx context.Context, role *model.Role) error {
	if err := requirePermission(ctx, s.store, "roles", "create"); err != nil {
		return err
	}

	if role.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	if role.IsSystem {
		log.Warn("Attempt to create system role", "name", role.Name)
		return ErrSystemRole
	}

	role.ID = uuid.Must(uuid.NewV7()).String()
	role.CreatedAt = time.Now()
	role.UpdatedAt = time.Now()

	return s.store.CreateRole(enrichAuditCtx(ctx), role)
}

func (s *RoleService) Get(ctx context.Context, id string) (*model.Role, error) {
	if err := requirePermission(ctx, s.store, "roles", "read"); err != nil {
		return nil, err
	}

	role, err := s.store.GetRole(ctx, id)
	if err != nil {
		return nil, err
	}

	return role, nil
}

func (s *RoleService) Update(ctx context.Context, id string, role *model.Role) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	existing, err := s.store.GetRole(ctx, id)
	if err != nil {
		return err
	}

	if existing.IsSystem && role.IsSystem != existing.IsSystem {
		return ErrSystemRole
	}

	role.ID = id
	role.UpdatedAt = time.Now()

	return s.store.UpdateRole(ctx, role)
}

func (s *RoleService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "roles", "delete"); err != nil {
		return err
	}

	role, err := s.store.GetRole(ctx, id)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	if err := s.store.DeleteRole(ctx, id); err != nil {
		return err
	}

	return nil
}

func (s *RoleService) GetPermissions(ctx context.Context, id string) ([]model.Permission, error) {
	if err := requirePermission(ctx, s.store, "roles", "read"); err != nil {
		return nil, err
	}

	return s.store.GetRolePermissions(ctx, id)
}

func (s *RoleService) GrantPermission(ctx context.Context, roleID, permissionID string) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	return s.store.AddRolePermission(ctx, roleID, permissionID)
}

func (s *RoleService) RevokePermission(ctx context.Context, roleID, permissionID string) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	return s.store.RemoveRolePermission(ctx, roleID, permissionID)
}

func (s *RoleService) AssignToUser(ctx context.Context, userID, roleID string) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	return s.store.AssignRoleToUser(ctx, userID, roleID)
}

func (s *RoleService) RevokeFromUser(ctx context.Context, userID, roleID string) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	return s.store.RemoveRoleFromUser(ctx, userID, roleID)
}

func (s *RoleService) ListPermissions(ctx context.Context, filter *model.PermissionFilter) ([]model.Permission, error) {
	if err := requirePermission(ctx, s.store, "roles", "list"); err != nil {
		return nil, err
	}

	return s.store.ListPermissions(ctx, filter)
}
