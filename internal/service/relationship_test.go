package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func TestRelationshipService_AddValidatesTypeAndPersistsValidInput(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "relationships", "create", true)
	svc := NewRelationshipService(store)

	err := svc.Add(userContext("user-1"), "parent-1", "child-1", "unsupported", "bad")
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for unsupported relationship type, got %v", err)
	}

	err = svc.Add(userContext("user-1"), "parent-1", "child-1", model.RelationshipContains, "rack unit")
	if err != nil {
		t.Fatalf("expected valid relationship add to succeed, got %v", err)
	}
	if store.addedParentID != "parent-1" || store.addedChildID != "child-1" || store.addedType != model.RelationshipContains {
		t.Fatalf("unexpected persisted relationship %#v", store)
	}
}

func TestRelationshipService_RemoveMapsMissingDeviceToNotFound(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "relationships", "delete", true)
	store.removeErr = storage.ErrDeviceNotFound
	svc := NewRelationshipService(store)

	err := svc.Remove(userContext("user-1"), "parent-1", "child-1", model.RelationshipContains)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
