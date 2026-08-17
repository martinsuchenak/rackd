package service

import (
	"context"
	"fmt"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type BulkService struct {
	store storage.ExtendedStorage
}

func NewBulkService(store storage.ExtendedStorage) *BulkService {
	return &BulkService{store: store}
}

func (s *BulkService) CreateDevices(ctx context.Context, devices []*model.Device) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "create"); err != nil {
		return nil, err
	}
	if err := validateBulkDevices(ctx, devices); err != nil {
		return nil, err
	}
	return s.store.BulkCreateDevices(enrichAuditCtx(ctx), devices)
}

func (s *BulkService) UpdateDevices(ctx context.Context, devices []*model.Device) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return nil, err
	}
	if err := validateBulkDevices(ctx, devices); err != nil {
		return nil, err
	}
	return s.store.BulkUpdateDevices(enrichAuditCtx(ctx), devices)
}

// validateBulkDevices applies the same status validation and audit attribution
// as the single-device path, so bulk operations cannot submit invalid statuses
// or forge status_changed_by.
func validateBulkDevices(ctx context.Context, devices []*model.Device) error {
	for i, device := range devices {
		if err := validateStatus(device.Status); err != nil {
			if verrs, ok := err.(ValidationErrors); ok && len(verrs) > 0 {
				verrs[0].Field = fmt.Sprintf("devices[%d].%s", i, verrs[0].Field)
			}
			return err
		}
		setStatusChangedBy(ctx, device)
	}
	return nil
}

func (s *BulkService) DeleteDevices(ctx context.Context, ids []string) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "delete"); err != nil {
		return nil, err
	}
	return s.store.BulkDeleteDevices(enrichAuditCtx(ctx), ids)
}

func (s *BulkService) AddTags(ctx context.Context, deviceIDs []string, tags []string) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return nil, err
	}
	return s.store.BulkAddTags(enrichAuditCtx(ctx), deviceIDs, tags)
}

func (s *BulkService) RemoveTags(ctx context.Context, deviceIDs []string, tags []string) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return nil, err
	}
	return s.store.BulkRemoveTags(enrichAuditCtx(ctx), deviceIDs, tags)
}

func (s *BulkService) CreateNetworks(ctx context.Context, networks []*model.Network) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "networks", "create"); err != nil {
		return nil, err
	}
	return s.store.BulkCreateNetworks(enrichAuditCtx(ctx), networks)
}

func (s *BulkService) DeleteNetworks(ctx context.Context, ids []string) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "networks", "delete"); err != nil {
		return nil, err
	}
	return s.store.BulkDeleteNetworks(enrichAuditCtx(ctx), ids)
}
