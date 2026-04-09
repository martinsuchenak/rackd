package service

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestAuthService_GetCurrentUserRequiresCallerAndMapsMissingUser(t *testing.T) {
	store := newServiceTestStorage()
	svc := NewAuthService(store, nil)

	_, err := svc.GetCurrentUser(context.Background())
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated error, got %v", err)
	}

	_, err = svc.GetCurrentUser(userContext("missing"))
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for missing current user, got %v", err)
	}
}

func TestAuthService_GetCurrentUserWithPermissionsByIDBuildsResponse(t *testing.T) {
	store := newServiceTestStorage()
	store.users["user-1"] = &model.User{ID: "user-1", Username: "alice", Email: "alice@example.com", IsActive: true}
	store.userRoles["user-1"] = []model.Role{{ID: "viewer", Name: "viewer"}}
	svc := NewAuthService(store, nil)

	resp, err := svc.GetCurrentUserWithPermissionsByID(userContext("user-1"), "user-1")
	if err != nil {
		t.Fatalf("GetCurrentUserWithPermissionsByID returned unexpected error: %v", err)
	}
	if resp.ID != "user-1" || resp.Username != "alice" || len(resp.Roles) != 1 {
		t.Fatalf("unexpected current-user response %#v", resp)
	}
}
