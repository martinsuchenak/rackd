package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type ScanProfileService struct {
	store storage.ProfileStorage
	rbac  auditedPermissionStore
}

func NewScanProfileService(store storage.ProfileStorage, rbac auditedPermissionStore) *ScanProfileService {
	return &ScanProfileService{store: store, rbac: rbac}
}

func (s *ScanProfileService) List(ctx context.Context) ([]model.ScanProfile, error) {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "list"); err != nil {
		return nil, err
	}

	return s.store.List(ctx)
}

func (s *ScanProfileService) Create(ctx context.Context, profile *model.ScanProfile) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "create"); err != nil {
		return err
	}

	if err := s.store.Create(ctx, profile); err != nil {
		return ValidationErrors{{Field: "profile", Message: err.Error()}}
	}

	recordAudit(ctx, s.rbac, "create", "scan_profile", profile.ID, profile)
	return nil
}

func (s *ScanProfileService) Get(ctx context.Context, id string) (*model.ScanProfile, error) {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "read"); err != nil {
		return nil, err
	}

	profile, err := s.store.Get(ctx, id)
	if errors.Is(err, storage.ErrProfileNotFound) {
		return nil, ErrNotFound
	}
	return profile, err
}

func (s *ScanProfileService) Update(ctx context.Context, id string, profile *model.ScanProfile) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "update"); err != nil {
		return err
	}

	profile.ID = id
	if err := s.store.Update(ctx, profile); err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			return ErrNotFound
		}
		return ValidationErrors{{Field: "profile", Message: err.Error()}}
	}

	recordAudit(ctx, s.rbac, "update", "scan_profile", id, profile)
	return nil
}

func (s *ScanProfileService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "delete"); err != nil {
		return err
	}

	err := s.store.Delete(ctx, id)
	if errors.Is(err, storage.ErrProfileNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	recordAudit(ctx, s.rbac, "delete", "scan_profile", id, nil)
	return nil
}
