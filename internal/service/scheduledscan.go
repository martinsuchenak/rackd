package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type ScheduledScanService struct {
	store storage.ScheduledScanStorage
	rbac  PermissionChecker
}

func NewScheduledScanService(store storage.ScheduledScanStorage, rbac PermissionChecker) *ScheduledScanService {
	return &ScheduledScanService{store: store, rbac: rbac}
}

func (s *ScheduledScanService) List(ctx context.Context, networkID string) ([]model.ScheduledScan, error) {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "list"); err != nil {
		return nil, err
	}

	return s.store.List(networkID)
}

func (s *ScheduledScanService) Create(ctx context.Context, scan *model.ScheduledScan) error {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "create"); err != nil {
		return err
	}

	if err := s.store.Create(scan); err != nil {
		return ValidationErrors{{Field: "scheduled_scan", Message: err.Error()}}
	}

	return nil
}

func (s *ScheduledScanService) Get(ctx context.Context, id string) (*model.ScheduledScan, error) {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "read"); err != nil {
		return nil, err
	}

	scan, err := s.store.Get(id)
	if errors.Is(err, storage.ErrScheduledScanNotFound) {
		return nil, ErrNotFound
	}
	return scan, err
}

func (s *ScheduledScanService) Update(ctx context.Context, id string, scan *model.ScheduledScan) error {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "update"); err != nil {
		return err
	}

	scan.ID = id
	if err := s.store.Update(scan); err != nil {
		if errors.Is(err, storage.ErrScheduledScanNotFound) {
			return ErrNotFound
		}
		return ValidationErrors{{Field: "scheduled_scan", Message: err.Error()}}
	}

	return nil
}

func (s *ScheduledScanService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "delete"); err != nil {
		return err
	}

	err := s.store.Delete(id)
	if errors.Is(err, storage.ErrScheduledScanNotFound) {
		return ErrNotFound
	}
	return err
}
