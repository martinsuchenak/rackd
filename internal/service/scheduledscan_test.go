package service

import (
	"errors"
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type scheduledScanStoreStub struct {
	items     map[string]*model.ScheduledScan
	createErr error
	updateErr error
}

func (s *scheduledScanStoreStub) List(networkID string) ([]model.ScheduledScan, error) {
	var items []model.ScheduledScan
	for _, item := range s.items {
		if networkID != "" && item.NetworkID != networkID {
			continue
		}
		items = append(items, *item)
	}
	return items, nil
}

func (s *scheduledScanStoreStub) Create(scan *model.ScheduledScan) error {
	if s.createErr != nil {
		return s.createErr
	}
	if s.items == nil {
		s.items = make(map[string]*model.ScheduledScan)
	}
	cloned := *scan
	s.items[scan.ID] = &cloned
	return nil
}

func (s *scheduledScanStoreStub) Get(id string) (*model.ScheduledScan, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, storage.ErrScheduledScanNotFound
	}
	cloned := *item
	return &cloned, nil
}

func (s *scheduledScanStoreStub) Update(scan *model.ScheduledScan) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	if _, ok := s.items[scan.ID]; !ok {
		return storage.ErrScheduledScanNotFound
	}
	cloned := *scan
	s.items[scan.ID] = &cloned
	return nil
}

func (s *scheduledScanStoreStub) Delete(id string) error {
	if _, ok := s.items[id]; !ok {
		return storage.ErrScheduledScanNotFound
	}
	delete(s.items, id)
	return nil
}

func TestScheduledScanService_CreateUpdateDeleteMapErrors(t *testing.T) {
	store := &scheduledScanStoreStub{items: map[string]*model.ScheduledScan{
		"scan-1": {ID: "scan-1", NetworkID: "net-1"},
	}}
	rbac := newServiceTestStorage()
	rbac.setPermission("user-1", "scheduled-scans", "create", true)
	rbac.setPermission("user-1", "scheduled-scans", "update", true)
	rbac.setPermission("user-1", "scheduled-scans", "delete", true)
	svc := NewScheduledScanService(store, rbac)

	store.createErr = fmt.Errorf("bad schedule")
	err := svc.Create(userContext("user-1"), &model.ScheduledScan{ID: "scan-2"})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation-wrapped create error, got %v", err)
	}

	store.createErr = nil
	store.updateErr = fmt.Errorf("bad update")
	err = svc.Update(userContext("user-1"), "scan-1", &model.ScheduledScan{NetworkID: "net-1"})
	if err == nil || !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation-wrapped update error, got %v", err)
	}

	err = svc.Delete(userContext("user-1"), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found on missing scheduled scan delete, got %v", err)
	}
}
