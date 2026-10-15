package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type RoleService struct {
	store storage.ExtendedStorage
	guard privilegeGuard
}

func NewRoleService(store storage.ExtendedStorage) *RoleService {
	return &RoleService{store: store, guard: privilegeGuard{store: store}}
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
		if errors.Is(err, storage.ErrRoleNotFound) {
			return nil, ErrNotFound
		}
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
		if errors.Is(err, storage.ErrRoleNotFound) {
			return ErrNotFound
		}
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
		if errors.Is(err, storage.ErrRoleNotFound) {
			return ErrNotFound
		}
		return err
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	// Deleting a role takes its permissions from everyone who has it.
	perms, err := s.store.GetRolePermissions(ctx, id)
	if err != nil {
		return err
	}
	if err := s.guard.requireHolds(ctx, perms); err != nil {
		return err
	}

	return s.store.DeleteRole(ctx, id)
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

	perm, err := s.findPermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if err := s.guard.requireHolds(ctx, []model.Permission{*perm}); err != nil {
		return err
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

	perm, err := s.findPermission(ctx, permissionID)
	if err != nil {
		return err
	}
	if err := s.guard.requireHolds(ctx, []model.Permission{*perm}); err != nil {
		return err
	}

	return s.store.RemoveRolePermission(ctx, roleID, permissionID)
}

// SetPermissions atomically replaces a role's permission set. Permission IDs
// are validated to exist; because role_permissions has no FK, dangling IDs
// would otherwise grant nothing while reporting success.
func (s *RoleService) SetPermissions(ctx context.Context, roleID string, permissionIDs []string) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	role, err := s.store.GetRole(ctx, roleID)
	if err != nil {
		if errors.Is(err, storage.ErrRoleNotFound) {
			return ErrNotFound
		}
		return err
	}

	if role.IsSystem {
		return ErrSystemRole
	}

	current, err := s.store.GetRolePermissions(ctx, roleID)
	if err != nil {
		return err
	}
	if err := s.checkPermissionChange(ctx, current, permissionIDs); err != nil {
		return err
	}

	return s.store.SetRolePermissions(ctx, roleID, permissionIDs)
}

// CreateWithPermissions creates a role holding permissionIDs. It checks the
// permissions before creating anything, so a refusal leaves no empty role.
func (s *RoleService) CreateWithPermissions(ctx context.Context, role *model.Role, permissionIDs []string) error {
	if len(permissionIDs) > 0 {
		if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
			return err
		}
		if err := s.checkPermissionChange(ctx, nil, permissionIDs); err != nil {
			return err
		}
	}
	if err := s.Create(ctx, role); err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}
	return s.store.SetRolePermissions(ctx, role.ID, permissionIDs)
}

// checkPermissionChange fails if any of permissionIDs is unknown, or if the
// caller doesn't hold every permission that changing a role from current to
// permissionIDs adds or removes.
func (s *RoleService) checkPermissionChange(ctx context.Context, current []model.Permission, permissionIDs []string) error {
	perms, err := s.store.ListPermissions(ctx, nil)
	if err != nil {
		return err
	}
	byID := make(map[string]model.Permission, len(perms))
	for _, p := range perms {
		byID[p.ID] = p
	}
	wanted := make(map[string]bool, len(permissionIDs))
	for _, id := range permissionIDs {
		if _, ok := byID[id]; !ok {
			return ValidationErrors{{Field: "permissions", Message: "Unknown permission ID: " + id}}
		}
		wanted[id] = true
	}

	had := make(map[string]bool, len(current))
	var changed []model.Permission
	for _, p := range current {
		had[p.ID] = true
		if !wanted[p.ID] {
			changed = append(changed, p)
		}
	}
	for id := range wanted {
		if !had[id] {
			changed = append(changed, byID[id])
		}
	}
	return s.guard.requireHolds(ctx, changed)
}

func (s *RoleService) findPermission(ctx context.Context, id string) (*model.Permission, error) {
	perms, err := s.store.ListPermissions(ctx, nil)
	if err != nil {
		return nil, err
	}
	for i := range perms {
		if perms[i].ID == id {
			return &perms[i], nil
		}
	}
	return nil, ValidationErrors{{Field: "permission_id", Message: "Unknown permission ID: " + id}}
}

func (s *RoleService) AssignToUser(ctx context.Context, userID, roleID string) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	if _, err := s.guard.requireRole(ctx, roleID); err != nil {
		return err
	}

	return s.store.AssignRoleToUser(ctx, userID, roleID)
}

func (s *RoleService) RevokeFromUser(ctx context.Context, userID, roleID string) error {
	if err := requirePermission(ctx, s.store, "roles", "update"); err != nil {
		return err
	}

	role, err := s.guard.requireRole(ctx, roleID)
	if err != nil {
		return err
	}
	// Taking a role away from someone more privileged is acting on them.
	if err := s.guard.requireOutranks(ctx, userID); err != nil {
		return err
	}
	if role.Name == "admin" {
		if err := s.guard.requireAnotherAdmin(ctx, userID); err != nil {
			return err
		}
	}

	return s.store.RemoveRoleFromUser(ctx, userID, roleID)
}

func (s *RoleService) ListPermissions(ctx context.Context, filter *model.PermissionFilter) ([]model.Permission, error) {
	if err := requirePermission(ctx, s.store, "roles", "list"); err != nil {
		return nil, err
	}

	return s.store.ListPermissions(ctx, filter)
}
