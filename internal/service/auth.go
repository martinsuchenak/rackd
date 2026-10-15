package service

import (
	"context"
	"errors"
	"time"

	"github.com/martinsuchenak/rackd/internal/audit"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type AuthService struct {
	store          storage.ExtendedStorage
	sessionManager *auth.SessionManager
	throttle       *loginThrottle
}

type LoginResult struct {
	User    model.UserResponse
	Session *auth.Session
}

func NewAuthService(store storage.ExtendedStorage, sessionManager *auth.SessionManager) *AuthService {
	return &AuthService{
		store:          store,
		sessionManager: sessionManager,
		throttle:       newLoginThrottle(defaultMaxLoginFailures, defaultLoginFailureWindow),
	}
}

// SetLoginThrottle sets how many failed sign-ins an account may have within
// window before further attempts are refused until the window passes; max 0
// turns the limit off.
func (s *AuthService) SetLoginThrottle(max int, window time.Duration) {
	s.throttle = newLoginThrottle(max, window)
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	// Refuse before checking the password, so guessing past the limit also
	// stops costing bcrypt work. The limit applies to unknown usernames too,
	// so it doesn't reveal which exist.
	// Refusals while locked are logged but not audited: the lock itself was
	// (see failed), and auditing each refusal would let anyone fill the log.
	if s.throttle.blocked(username) {
		log.Warn("Login refused: too many failed sign-ins", "username", username)
		return nil, ErrTooManyLoginAttempts
	}

	user, err := s.store.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			// Burn the same bcrypt work as the password-verify path so login
			// timing does not reveal whether the username exists.
			auth.VerifyDummyPassword(password)
			// The typed username isn't recorded: it is often a password
			// entered in the wrong field.
			s.failed(ctx, username, nil, "unknown user")
			return nil, ErrUnauthenticated
		}
		log.Error("Failed to get user for login", "error", err, "username", username)
		return nil, ErrUnauthenticated
	}

	if !user.IsActive {
		// Equalize timing for deactivated accounts too.
		auth.VerifyDummyPassword(password)
		s.failed(ctx, username, user, "account is deactivated")
		return nil, ErrUnauthenticated
	}

	if err := auth.VerifyPassword(user.PasswordHash, password); err != nil {
		s.failed(ctx, username, user, "wrong password")
		return nil, ErrUnauthenticated
	}
	s.throttle.reset(username)

	isAdmin, _ := userCan(ctx, s.store, user.ID, "users", "create")
	session, err := s.sessionManager.CreateSession(user.ID, user.Username, isAdmin)
	if err != nil {
		log.Error("Failed to create session", "error", err, "user_id", user.ID)
		return nil, err
	}

	recordAudit(audit.WithContext(ctx, signedIn(ctx, user)), s.store, "login", "session", user.ID, nil)

	now := time.Now()
	if err := s.store.UpdateUserLastLogin(ctx, user.ID, now); err != nil {
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
	if session, err := s.sessionManager.GetSession(token); err == nil {
		recordAudit(ctx, s.store, "logout", "session", session.UserID, nil)
	}
	return s.sessionManager.InvalidateSession(token)
}

// failed counts and audits a failed sign-in. user is nil for an unknown
// username, which is then left out of the entry.
func (s *AuthService) failed(ctx context.Context, username string, user *model.User, reason string) {
	locked := s.throttle.fail(username)
	actor := &audit.Context{IPAddress: attemptAddress(ctx), Source: "api"}
	id := ""
	if user != nil {
		actor.UserID, actor.Username, id = user.ID, user.Username, user.ID
	}
	actx := audit.WithContext(ctx, actor)
	recordAuditFailure(actx, s.store, "login", "session", id, nil, reason)
	if locked {
		recordAuditFailure(actx, s.store, "login", "session", id, nil, "too many failed sign-ins: further attempts refused for "+s.throttle.window.String())
	}
}

// signedIn is the audit actor for a successful sign-in: the user, from the
// address the attempt came from.
func signedIn(ctx context.Context, user *model.User) *audit.Context {
	return &audit.Context{UserID: user.ID, Username: user.Username, IPAddress: attemptAddress(ctx), Source: "api"}
}

// attemptAddress is the client address the handler attached for the audit log.
func attemptAddress(ctx context.Context) string {
	if a, ok := audit.FromContext(ctx); ok {
		return a.IPAddress
	}
	return ""
}

func (s *AuthService) GetCurrentUser(ctx context.Context) (*model.UserResponse, error) {
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

	resp := user.ToResponse()
	if roles, err := s.store.GetUserRoles(ctx, caller.UserID); err == nil {
		resp.Roles = roles
	}

	return &resp, nil
}

func (s *AuthService) GetCurrentUserWithPermissions(ctx context.Context) (*model.CurrentUserResponse, error) {
	caller := CallerFrom(ctx)
	if caller == nil || caller.UserID == "" {
		return nil, ErrUnauthenticated
	}

	return s.GetCurrentUserWithPermissionsByID(ctx, caller.UserID)
}

func (s *AuthService) GetCurrentUserWithPermissionsByID(ctx context.Context, userID string) (*model.CurrentUserResponse, error) {
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	roles, err := s.store.GetUserRoles(ctx, userID)
	if err != nil {
		log.Warn("Failed to get user roles", "error", err, "user_id", userID)
		roles = []model.Role{}
	}

	permissions, err := s.store.GetUserPermissions(ctx, userID)
	if err != nil {
		log.Warn("Failed to get user permissions", "error", err, "user_id", userID)
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
