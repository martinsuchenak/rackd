package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type UserService struct {
	store    storage.ExtendedStorage
	sessions SessionInvalidator
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
	}
}

func (s *UserService) List(ctx context.Context, filter *model.UserFilter) ([]model.User, error) {
	if err := requirePermission(ctx, s.store, "users", "list"); err != nil {
		return nil, err
	}

	users, err := s.store.ListUsers(filter)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (s *UserService) Create(ctx context.Context, req *model.CreateUserRequest) (*model.UserResponse, error) {
	if err := requirePermission(ctx, s.store, "users", "create"); err != nil {
		return nil, err
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

	if existing, _ := s.store.GetUserByUsername(req.Username); existing != nil {
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
		IsAdmin:      req.IsAdmin,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.store.CreateUser(enrichAuditCtx(ctx), user); err != nil {
		return nil, err
	}

	if req.RoleID != "" {
		_ = s.store.AssignRoleToUser(ctx, user.ID, req.RoleID)
	}

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

	user, err := s.store.GetUser(id)
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

	user, err := s.store.GetUser(id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FullName != "" {
		user.FullName = req.FullName
	}
	// Privileged fields require users:update permission
	if req.IsActive != nil && !isSelf {
		user.IsActive = *req.IsActive
	}
	if req.IsAdmin != nil && !isSelf {
		user.IsAdmin = *req.IsAdmin
	}
	user.UpdatedAt = time.Now()

	if err := s.store.UpdateUser(enrichAuditCtx(ctx), user); err != nil {
		return nil, err
	}

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

	caller := CallerFrom(ctx)
	if caller != nil && caller.UserID == id {
		return ErrSelfDelete
	}

	if err := s.store.DeleteUser(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrNotFound
		}
		return err
	}

	if s.sessions != nil {
		s.sessions.InvalidateUserSessions(id)
	}

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

	user, err := s.store.GetUser(id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrNotFound
		}
		return err
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.OldPassword); err != nil {
		return ErrValidation
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()
	return s.store.UpdateUser(enrichAuditCtx(ctx), user)
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

	user, err := s.store.GetUser(id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return ErrNotFound
		}
		return err
	}

	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()
	return s.store.UpdateUser(enrichAuditCtx(ctx), user)
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

	user, err := s.store.GetUser(caller.UserID)
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
