package service

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestAPIKeyService_ListScopesNonAdminToOwnKeys(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "apikeys", "list", true)
	store.apiKeys["key-1"] = &model.APIKey{ID: "key-1", Name: "mine", UserID: "user-1"}
	store.apiKeys["key-2"] = &model.APIKey{ID: "key-2", Name: "other", UserID: "user-2"}
	store.userRoles["user-1"] = []model.Role{{ID: "viewer", Name: "viewer"}}
	svc := NewAPIKeyService(store)

	keys, err := svc.List(userContext("user-1"), nil)
	if err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if store.apiKeyFilter == nil || store.apiKeyFilter.UserID != "user-1" {
		t.Fatalf("expected list filter to be scoped to caller, got %#v", store.apiKeyFilter)
	}
	if len(keys) != 1 || keys[0].UserID != "user-1" {
		t.Fatalf("expected only caller-owned keys, got %#v", keys)
	}
}

func TestAPIKeyService_DeleteRejectsNonOwnerAndMapsUnauthenticated(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "apikeys", "delete", true)
	store.apiKeys["key-1"] = &model.APIKey{ID: "key-1", Name: "other", UserID: "user-2"}
	store.userRoles["user-1"] = []model.Role{{ID: "viewer", Name: "viewer"}}
	svc := NewAPIKeyService(store)

	err := svc.Delete(userContext("user-1"), "key-1")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected forbidden for non-owner delete, got %v", err)
	}

	if err := svc.requireOwnership(context.Background(), &model.APIKey{UserID: "user-1"}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expected unauthenticated ownership check failure, got %v", err)
	}
}

func TestAPIKeyService_CreateRequiresNameAndAssignsCallerOwnership(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "apikeys", "create", true)
	svc := NewAPIKeyService(store)

	_, err := svc.Create(userContext("user-1"), &model.APIKey{})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing name, got %v", err)
	}

	plain, err := svc.Create(userContext("user-1"), &model.APIKey{Name: "ci-key"})
	if err != nil {
		t.Fatalf("expected API key create to succeed, got %v", err)
	}
	if plain == "" {
		t.Fatal("expected plaintext key to be returned")
	}
	foundOwner := false
	for _, key := range store.apiKeys {
		if key.Name == "ci-key" && key.UserID == "user-1" && key.Key != "" {
			foundOwner = true
		}
	}
	if !foundOwner {
		t.Fatalf("expected created API key to be assigned to caller, got %#v", store.apiKeys)
	}
}

func TestAPIKeyService_GetAndDeleteMapNotFound(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "apikeys", "read", true)
	store.setPermission("user-1", "apikeys", "delete", true)
	svc := NewAPIKeyService(store)

	if _, err := svc.Get(userContext("user-1"), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found from Get, got %v", err)
	}

	if err := svc.Delete(userContext("user-1"), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found from Delete, got %v", err)
	}
}
