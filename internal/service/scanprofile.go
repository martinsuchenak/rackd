package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type ScanProfileService struct {
	store storage.ProfileStorage
	rbac  PermissionChecker
}

func NewScanProfileService(store storage.ProfileStorage, rbac PermissionChecker) *ScanProfileService {
	return &ScanProfileService{store: store, rbac: rbac}
}

func (s *ScanProfileService) List(ctx context.Context) ([]model.ScanProfile, error) {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "list"); err != nil {
		return nil, err
	}

	return s.store.List()
}

func (s *ScanProfileService) Create(ctx context.Context, profile *model.ScanProfile) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "create"); err != nil {
		return err
	}

	if err := s.store.Create(profile); err != nil {
		return ValidationErrors{{Field: "profile", Message: err.Error()}}
	}

	return nil
}

func (s *ScanProfileService) Get(ctx context.Context, id string) (*model.ScanProfile, error) {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "read"); err != nil {
		return nil, err
	}

	profile, err := s.store.Get(id)
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
	if err := s.store.Update(profile); err != nil {
		if errors.Is(err, storage.ErrProfileNotFound) {
			return ErrNotFound
		}
		return ValidationErrors{{Field: "profile", Message: err.Error()}}
	}

	return nil
}

func (s *ScanProfileService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.rbac, "scan-profiles", "delete"); err != nil {
		return err
	}

	err := s.store.Delete(id)
	if errors.Is(err, storage.ErrProfileNotFound) {
		return ErrNotFound
	}
	return err
}
