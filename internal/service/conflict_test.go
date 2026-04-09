package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestConflictHelpersAndResolveValidation(t *testing.T) {
	if !sameNetworks([]string{"a", "b"}, []string{"b", "a"}) {
		t.Fatal("expected sameNetworks to treat slices as sets")
	}
	if sameNetworks([]string{"a"}, []string{"a", "b"}) {
		t.Fatal("expected sameNetworks to reject different sets")
	}

	store := newServiceTestStorage()
	store.setPermission("user-1", "conflicts", "resolve", true)
	svc := NewConflictService(store)

	err := svc.Resolve(userContext("user-1"), &model.ConflictResolution{})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing conflict ID, got %v", err)
	}
}
