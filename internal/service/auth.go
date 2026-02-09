package service

import (
	"context"
	"errors"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type AuthService struct {
	store          storage.ExtendedStorage
	sessionManager *auth.SessionManager
}

type LoginResult struct {
	User    model.UserResponse
	Session *auth.Session
}

func NewAuthService(store storage.ExtendedStorage, sessionManager *auth.SessionManager) *AuthService {
	return &AuthService{
		store:          store,
		sessionManager: sessionManager,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	user, err := s.store.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrUnauthenticated
		}
		log.Error("Failed to get user for login", "error", err, "username", username)
		return nil, ErrUnauthenticated
	}

	if !user.IsActive {
		return nil, ErrUnauthenticated
	}

	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		return nil, ErrUnauthenticated
	}

	isAdmin, _ := s.store.HasPermission(ctx, user.ID, "users", "create")
	session, err := s.sessionManager.CreateSession(user.ID, user.Username, isAdmin)
	if err != nil {
		log.Error("Failed to create session", "error", err, "user_id", user.ID)
		return nil, err
	}

	now := time.Now()
	if err := s.store.UpdateUserLastLogin(user.ID, now); err != nil {
		log.Warn("Failed to update last login", "error", err, "user_id", user.ID)
	}

	resp := user.ToResponse()
	if roles, err := s.store.GetUserRoles(ctx, user.ID); err == nil {
		resp.Roles = roles
	}

	return &LoginResult{
		User:    resp,
		Session: session,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessionManager.InvalidateSession(token)
}

func (s *AuthService) GetCurrentUser(ctx context.Context) (*model.UserResponse, error) {
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

	resp := user.ToResponse()
	if roles, err := s.store.GetUserRoles(ctx, caller.UserID); err == nil {
		resp.Roles = roles
	}

	return &resp, nil
}
