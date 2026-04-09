package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
)

func TestUserService_UpdateSelfAllowsUsernameChangeWithoutUsersUpdatePermission(t *testing.T) {
	store := newServiceTestStorage()
	store.users["user-1"] = &model.User{
		ID:       "user-1",
		Username: "before",
		Email:    "before@example.com",
		IsActive: true,
	}

	svc := &UserService{store: store}

	resp, err := svc.Update(userContext("user-1"), "user-1", &model.UpdateUserRequest{
		Username: "after",
		Email:    "after@example.com",
	})
	if err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}
	if resp.Username != "after" {
		t.Fatalf("expected updated username %q, got %q", "after", resp.Username)
	}
	if store.updatedUser == nil || store.updatedUser.Username != "after" {
		t.Fatalf("expected UpdateUser to persist updated username, got %#v", store.updatedUser)
	}
}

func TestUserService_UpdateRejectsDuplicateUsername(t *testing.T) {
	store := newServiceTestStorage()
	store.users["user-1"] = &model.User{ID: "user-1", Username: "alice", Email: "alice@example.com", IsActive: true}
	store.users["user-2"] = &model.User{ID: "user-2", Username: "bob", Email: "bob@example.com", IsActive: true}

	svc := &UserService{store: store}

	_, err := svc.Update(userContext("user-1"), "user-1", &model.UpdateUserRequest{Username: "bob"})
	if err == nil {
		t.Fatal("expected duplicate username validation error, got nil")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
	var validationErrs ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(validationErrs) != 1 || validationErrs[0].Field != "username" {
		t.Fatalf("expected username validation error, got %#v", validationErrs)
	}
}

func TestUserService_ChangePasswordInvalidatesSessions(t *testing.T) {
	store := newServiceTestStorage()
	passwordHash, err := auth.HashPassword("old-password")
	if err != nil {
		t.Fatalf("HashPassword returned unexpected error: %v", err)
	}
	store.users["user-1"] = &model.User{
		ID:           "user-1",
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: passwordHash,
		IsActive:     true,
	}
	sessions := &stubSessionInvalidator{}
	svc := &UserService{store: store, sessions: sessions}

	err = svc.ChangePassword(userContext("user-1"), "user-1", &model.ChangePasswordRequest{
		OldPassword: "old-password",
		NewPassword: "new-password",
	})
	if err != nil {
		t.Fatalf("ChangePassword returned unexpected error: %v", err)
	}
	if len(sessions.invalidated) != 1 || sessions.invalidated[0] != "user-1" {
		t.Fatalf("expected sessions to be invalidated for user-1, got %#v", sessions.invalidated)
	}
	if verifyErr := auth.VerifyPassword(store.users["user-1"].PasswordHash, "new-password"); verifyErr != nil {
		t.Fatalf("expected stored password hash to match new password: %v", verifyErr)
	}
}
