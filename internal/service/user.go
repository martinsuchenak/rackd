package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type UserService struct {
	store    storage.ExtendedStorage
	sessions SessionInvalidator
	guard    privilegeGuard
}

type SessionInvalidator interface {
	InvalidateUserSessions(userID string)
}

func NewUserService(store storage.ExtendedStorage, sessionManager *auth.SessionManager) *UserService {
	var sessions SessionInvalidator
	if sessionManager != nil {
		sessions = sessionManager
	}
	return &UserService{
		store:    store,
		sessions: sessions,
		guard:    privilegeGuard{store: store},
	}
}

func (s *UserService) List(ctx context.Context, filter *model.UserFilter) ([]model.User, error) {
	if err := requirePermission(ctx, s.store, "users", "list"); err != nil {
		return nil, err
	}

	users, err := s.store.ListUsers(ctx, filter)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *UserService) Create(ctx context.Context, req *model.CreateUserRequest) (*model.UserResponse, error) {
	if err := requirePermission(ctx, s.store, "users", "create"); err != nil {
		return nil, err
	}

	// Only admins can create admin users
	if req.IsAdmin && !s.isAdmin(ctx) {
		return nil, &PrivilegeError{Reason: "Only admins can create admins"}
	}

	var errs ValidationErrors
	if req.Username == "" {
		errs = append(errs, ValidationError{Field: "username", Message: "Username is required"})
	}
	if len(req.Password) < 8 {
		errs = append(errs, ValidationError{Field: "password", Message: "Password must be at least 8 characters"})
	}
	if req.Email == "" {
		errs = append(errs, ValidationError{Field: "email", Message: "Email is required"})
	}
	if len(errs) > 0 {
		return nil, errs
	}

	if existing, _ := s.store.GetUserByUsername(ctx, req.Username); existing != nil {
		return nil, fmt.Errorf("username: %w", ErrAlreadyExists)
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.Must(uuid.NewV7()).String(),
		Username:     req.Username,
		Email:        req.Email,
		FullName:     req.FullName,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	// The creator may only give the new user a role they could assign.
	roleIDs := []string{}
	if req.RoleID != "" {
		if _, err := s.guard.requireRole(ctx, req.RoleID); err != nil {
			return nil, err
		}
		roleIDs = append(roleIDs, req.RoleID)
	}
	// is_admin follows the admin role, so an admin user gets the role.
	if req.IsAdmin {
		adminRole, err := s.store.GetRoleByName(ctx, "admin")
		if err != nil {
			return nil, fmt.Errorf("finding the admin role: %w", err)
		}
		roleIDs = append(roleIDs, adminRole.ID)
	}

	if err := s.store.CreateUser(enrichAuditCtx(ctx), user); err != nil {
		return nil, NormalizeError(err)
	}

	for _, roleID := range roleIDs {
		if err := s.store.AssignRoleToUser(ctx, user.ID, roleID); err != nil {
			// The user exists but lacks the role; surface this rather than
			// silently creating an unprivileged orphan.
			log.Error("Failed to assign role to new user", "user_id", user.ID, "role_id", roleID, "error", err)
		}
	}
	if updated, err := s.store.GetUser(ctx, user.ID); err == nil {
		user = updated
	}
	recordAudit(ctx, s.store, "create", "user", user.ID, map[string]any{
		"username": user.Username, "email": user.Email, "is_admin": user.IsAdmin, "role_ids": roleIDs,
	})

	resp := user.ToResponse()
	if roles, err := s.store.GetUserRoles(ctx, user.ID); err == nil {
		resp.Roles = roles
	}

	return &resp, nil
}

func (s *UserService) Get(ctx context.Context, id string) (*model.UserResponse, error) {
	if err := requirePermission(ctx, s.store, "users", "read"); err != nil {
		return nil, err
	}

	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	resp := user.ToResponse()
	if roles, err := s.store.GetUserRoles(ctx, id); err == nil {
		resp.Roles = roles
	}

	return &resp, nil
}

func (s *UserService) Update(ctx context.Context, id string, req *model.UpdateUserRequest) (*model.UserResponse, error) {
	caller := CallerFrom(ctx)
	isSelf := caller != nil && caller.UserID == id

	if !isSelf {
		if err := requirePermission(ctx, s.store, "users", "update"); err != nil {
			return nil, err
		}
	}

	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if !isSelf {
		if err := s.guard.requireOutranks(ctx, id); err != nil {
			return nil, err
		}
	}

	if req.Username != "" && req.Username != user.Username {
		if existing, _ := s.store.GetUserByUsername(ctx, req.Username); existing != nil && existing.ID != id {
			return nil, ValidationErrors{{Field: "username", Message: "Username already exists"}}
		}
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	// Privileged fields require users:update permission
	deactivate := false
	if req.IsActive != nil && !isSelf {
		if !*req.IsActive && user.IsActive {
			if err := s.guard.requireAnotherAdmin(ctx, id); err != nil {
				return nil, err
			}
			deactivate = true
		}
		user.IsActive = *req.IsActive
	}
	// Only admins can elevate others to admin or remove admin status, which
	// grants or removes the admin role.
	var adminRoleChange func() error
	if req.IsAdmin != nil && !isSelf && *req.IsAdmin != user.IsAdmin {
		if !s.isAdmin(ctx) {
			return nil, &PrivilegeError{Reason: "Only admins can make or remove admins"}
		}
		adminRole, err := s.store.GetRoleByName(ctx, "admin")
		if err != nil {
			return nil, fmt.Errorf("finding the admin role: %w", err)
		}
		if *req.IsAdmin {
			adminRoleChange = func() error { return s.store.AssignRoleToUser(ctx, id, adminRole.ID) }
		} else {
			if err := s.guard.requireAnotherAdmin(ctx, id); err != nil {
				return nil, err
			}
			adminRoleChange = func() error { return s.store.RemoveRoleFromUser(ctx, id, adminRole.ID) }
		}
	}
	user.UpdatedAt = time.Now().UTC()

	if err := s.store.UpdateUser(enrichAuditCtx(ctx), user); err != nil {
		return nil, err
	}
	// Deactivating a user must terminate their live sessions immediately;
	// otherwise a suspended account keeps access until cookie expiry.
	if deactivate {
		s.endAccess(ctx, id)
	}
	if adminRoleChange != nil {
		if err := adminRoleChange(); err != nil {
			return nil, err
		}
		if updated, err := s.store.GetUser(ctx, id); err == nil {
			user = updated
		}
	}
	recordAudit(ctx, s.store, "update", "user", id, map[string]any{
		"username": user.Username, "email": user.Email, "full_name": user.FullName,
		"is_active": user.IsActive, "is_admin": user.IsAdmin,
	})

	resp := user.ToResponse()
	if roles, err := s.store.GetUserRoles(ctx, id); err == nil {
		resp.Roles = roles
	}

	return &resp, nil
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "users", "delete"); err != nil {
		return err
	}

	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrNotFound
		}
		return err
	}

	caller := CallerFrom(ctx)
	if caller != nil && caller.UserID == id {
		return ErrSelfDelete
	}
	if err := s.guard.requireOutranks(ctx, user.ID); err != nil {
		return err
	}
	if err := s.guard.requireAnotherAdmin(ctx, user.ID); err != nil {
		return err
	}

	if err := s.store.DeleteUser(enrichAuditCtx(ctx), id); err != nil {
		return err
	}
	recordAudit(ctx, s.store, "delete", "user", id, map[string]string{"username": user.Username})

	s.endAccess(ctx, id)

	return nil
}

func (s *UserService) ChangePassword(ctx context.Context, id string, req *model.ChangePasswordRequest) error {
	caller := CallerFrom(ctx)
	if caller == nil || caller.UserID != id {
		if err := requirePermission(ctx, s.store, "users", "update"); err != nil {
			return err
		}
	}

	if req.OldPassword == "" {
		return ValidationErrors{{Field: "old_password", Message: "Old password is required"}}
	}
	if len(req.NewPassword) < 8 {
		return ValidationErrors{{Field: "new_password", Message: "New password must be at least 8 characters"}}
	}

	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrNotFound
		}
		return err
	}

	if caller == nil || caller.UserID != id {
		if err := s.guard.requireOutranks(ctx, id); err != nil {
			return err
		}
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.OldPassword); err != nil {
		return ValidationErrors{{Field: "old_password", Message: "Current password is incorrect"}}
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(enrichAuditCtx(ctx), user); err != nil {
		return err
	}

	// Invalidate all sessions for this user to prevent old sessions from remaining active
	recordAudit(ctx, s.store, "password_change", "user", id, map[string]string{"username": user.Username})
	s.endAccess(ctx, id)

	return nil
}

// ResetPassword allows admins to reset a user's password without knowing the old one
func (s *UserService) ResetPassword(ctx context.Context, id string, req *model.ResetPasswordRequest) error {
	// Only users with users:update permission can reset passwords
	if err := requirePermission(ctx, s.store, "users", "update"); err != nil {
		return err
	}

	if len(req.NewPassword) < 8 {
		return ValidationErrors{{Field: "new_password", Message: "New password must be at least 8 characters"}}
	}

	user, err := s.store.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrNotFound
		}
		return err
	}

	if err := s.guard.requireOutranks(ctx, id); err != nil {
		return err
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateUser(enrichAuditCtx(ctx), user); err != nil {
		return err
	}

	// Invalidate all sessions for this user to prevent old sessions from remaining active
	recordAudit(ctx, s.store, "password_reset", "user", id, map[string]string{"username": user.Username})
	s.endAccess(ctx, id)

	return nil
}

func (s *UserService) GetRoles(ctx context.Context, userID string) ([]model.Role, error) {
	if err := requirePermission(ctx, s.store, "users", "read"); err != nil {
		return nil, err
	}

	return s.store.GetUserRoles(ctx, userID)
}

func (s *UserService) GetPermissions(ctx context.Context, userID string) ([]model.Permission, error) {
	if err := requirePermission(ctx, s.store, "users", "read"); err != nil {
		return nil, err
	}

	return s.store.GetUserPermissions(ctx, userID)
}

func (s *UserService) GetCurrentUserWithPermissions(ctx context.Context) (*model.CurrentUserResponse, error) {
	caller := CallerFrom(ctx)
	if caller == nil || caller.UserID == "" {
		return nil, ErrUnauthenticated
	}

	user, err := s.store.GetUser(ctx, caller.UserID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	roles, err := s.store.GetUserRoles(ctx, caller.UserID)
	if err != nil {
		log.Warn("Failed to get user roles", "error", err, "user_id", caller.UserID)
		roles = []model.Role{}
	}

	permissions, err := s.store.GetUserPermissions(ctx, caller.UserID)
	if err != nil {
		log.Warn("Failed to get user permissions", "error", err, "user_id", caller.UserID)
		permissions = []model.Permission{}
	}

	return &model.CurrentUserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		FullName:    user.FullName,
		IsActive:    user.IsActive,
		IsAdmin:     user.IsAdmin,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		LastLoginAt: user.LastLoginAt,
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

// endAccess signs the user out everywhere: their sessions end and their
// OAuth tokens are revoked, so a stolen session or token can't outlive a
// password change. API keys are kept; their owners manage them.
func (s *UserService) endAccess(ctx context.Context, userID string) {
	if s.sessions != nil {
		s.sessions.InvalidateUserSessions(userID)
	}
	if err := s.store.RevokeOAuthTokensByUser(ctx, userID); err != nil {
		log.Error("Failed to revoke OAuth tokens", "user_id", userID, "error", err)
	}
}

func (s *UserService) isAdmin(ctx context.Context) bool {
	admin, _ := s.guard.callerIsAdmin(ctx)
	return admin
}
