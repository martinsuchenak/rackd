package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type profileStoreStub struct {
	items     map[string]*model.ScanProfile
	createErr error
	updateErr error
}

func (s *profileStoreStub) List(_ context.Context) ([]model.ScanProfile, error) {
	var items []model.ScanProfile
	for _, item := range s.items {
		items = append(items, *item)
	}
	return items, nil
}

func (s *profileStoreStub) Create(_ context.Context, profile *model.ScanProfile) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.items == nil {
		s.items = make(map[string]*model.ScanProfile)
	}
	cloned := *profile
	s.items[profile.ID] = &cloned
	return nil
}

func (s *profileStoreStub) Get(_ context.Context, id string) (*model.ScanProfile, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, storage.ErrProfileNotFound
	}
	cloned := *item
	return &cloned, nil
}

func (s *profileStoreStub) Update(_ context.Context, profile *model.ScanProfile) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if _, ok := s.items[profile.ID]; !ok {
		return storage.ErrProfileNotFound
	}
	cloned := *profile
	s.items[profile.ID] = &cloned
	return nil
}

func (s *profileStoreStub) Delete(_ context.Context, id string) error {
	if _, ok := s.items[id]; !ok {
		return storage.ErrProfileNotFound
	}
	delete(s.items, id)
	return nil
}

func TestScanProfileService_UpdateAndDeleteMapErrors(t *testing.T) {
	store := &profileStoreStub{items: map[string]*model.ScanProfile{
		"profile-1": {ID: "profile-1", Name: "basic"},
	}}
	rbac := newServiceTestStorage()
	rbac.setPermission("user-1", "scan-profiles", "update", true)
	rbac.setPermission("user-1", "scan-profiles", "delete", true)
	svc := NewScanProfileService(store, rbac)

	store.updateErr = fmt.Errorf("bad profile")
	err := svc.Update(userContext("user-1"), "profile-1", &model.ScanProfile{Name: "updated"})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation-wrapped update error, got %v", err)
	}

	err = svc.Delete(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found on missing profile delete, got %v", err)
	}
}
