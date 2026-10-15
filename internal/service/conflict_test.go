package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestConflictHelpersAndResolveValidation(t *testing.T) {
	if networkPairKey([]string{"a", "b"}) != networkPairKey([]string{"b", "a"}) {
		t.Fatal("expected networkPairKey to ignore order")
	}
	if networkPairKey([]string{"a"}) == networkPairKey([]string{"a", "b"}) {
		t.Fatal("expected networkPairKey to tell different sets apart")
	}

	store := newServiceTestStorage()
	store.setPermission("user-1", "conflicts", "resolve", true)
	svc := NewConflictService(store)

	err := svc.Resolve(userContext("user-1"), &model.ConflictResolution{})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for missing conflict ID, got %v", err)
	}
}
