package service

import (
	"context"
	"fmt"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
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
	result, err := s.store.BulkCreateDevices(enrichAuditCtx(ctx), devices)
	if err == nil {
		publishDevices(model.EventTypeDeviceCreated, devices, result.SucceededIDs)
	}
	return result, err
}

func (s *BulkService) UpdateDevices(ctx context.Context, devices []*model.Device) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return nil, err
	}
	if err := validateBulkDevices(ctx, devices); err != nil {
		return nil, err
	}
	result, err := s.store.BulkUpdateDevices(enrichAuditCtx(ctx), devices)
	if err == nil {
		publishDevices(model.EventTypeDeviceUpdated, devices, result.SucceededIDs)
	}
	return result, err
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
	deleted := s.storedDevices(ctx, ids)
	result, err := s.store.BulkDeleteDevices(enrichAuditCtx(ctx), ids)
	if err == nil {
		publishDevices(model.EventTypeDeviceDeleted, deleted, result.SucceededIDs)
	}
	return result, err
}

func (s *BulkService) AddTags(ctx context.Context, deviceIDs []string, tags []string) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return nil, err
	}
	result, err := s.store.BulkAddTags(enrichAuditCtx(ctx), deviceIDs, tags)
	if err == nil {
		publishDevices(model.EventTypeDeviceUpdated, s.storedDevices(ctx, result.SucceededIDs), result.SucceededIDs)
	}
	return result, err
}

func (s *BulkService) RemoveTags(ctx context.Context, deviceIDs []string, tags []string) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return nil, err
	}
	result, err := s.store.BulkRemoveTags(enrichAuditCtx(ctx), deviceIDs, tags)
	if err == nil {
		publishDevices(model.EventTypeDeviceUpdated, s.storedDevices(ctx, result.SucceededIDs), result.SucceededIDs)
	}
	return result, err
}

func (s *BulkService) CreateNetworks(ctx context.Context, networks []*model.Network) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "networks", "create"); err != nil {
		return nil, err
	}
	result, err := s.store.BulkCreateNetworks(enrichAuditCtx(ctx), networks)
	if err == nil {
		publishNetworks(model.EventTypeNetworkCreated, networks, result.SucceededIDs)
	}
	return result, err
}

func (s *BulkService) DeleteNetworks(ctx context.Context, ids []string) (*storage.BulkResult, error) {
	if err := requirePermission(ctx, s.store, "networks", "delete"); err != nil {
		return nil, err
	}
	deleted := make([]*model.Network, 0, len(ids))
	for _, id := range ids {
		if n, err := s.store.GetNetwork(ctx, id); err == nil {
			deleted = append(deleted, n)
		} else {
			deleted = append(deleted, &model.Network{ID: id})
		}
	}
	result, err := s.store.BulkDeleteNetworks(enrichAuditCtx(ctx), ids)
	if err == nil {
		publishNetworks(model.EventTypeNetworkDeleted, deleted, result.SucceededIDs)
	}
	return result, err
}

// storedDevices reads the devices with the given IDs, for event payloads. A
// device that can't be read is represented by its ID alone.
func (s *BulkService) storedDevices(ctx context.Context, ids []string) []*model.Device {
	out := make([]*model.Device, 0, len(ids))
	for _, id := range ids {
		if d, err := s.store.GetDevice(ctx, id); err == nil {
			out = append(out, d)
		} else {
			out = append(out, &model.Device{ID: id})
		}
	}
	return out
}

// publishDevices publishes t for each device whose ID is in succeeded.
func publishDevices(t model.EventType, devices []*model.Device, succeeded []string) {
	ok := map[string]bool{}
	for _, id := range succeeded {
		ok[id] = true
	}
	for _, d := range devices {
		if ok[d.ID] {
			publishEvent(t, deviceSnapshot(d))
		}
	}
}

// publishNetworks publishes t for each network whose ID is in succeeded.
func publishNetworks(t model.EventType, networks []*model.Network, succeeded []string) {
	ok := map[string]bool{}
	for _, id := range succeeded {
		ok[id] = true
	}
	for _, n := range networks {
		if ok[n.ID] {
			publishEvent(t, networkSnapshot(n))
		}
	}
}
