package service

import (
	"context"

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
	return s.store.BulkCreateDevices(enrichAuditCtx(ctx), devices)
}

func (s *BulkService) UpdateDevices(ctx context.Context, devices []*model.Device) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return nil, err
	}
	return s.store.BulkUpdateDevices(enrichAuditCtx(ctx), devices)
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
