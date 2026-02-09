package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type DeviceService struct {
	store storage.ExtendedStorage
}

func NewDeviceService(store storage.ExtendedStorage) *DeviceService {
	return &DeviceService{store: store}
}

func (s *DeviceService) List(ctx context.Context, filter *model.DeviceFilter) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "list"); err != nil {
		return nil, err
	}
	return s.store.ListDevices(filter)
}

func (s *DeviceService) Create(ctx context.Context, device *model.Device) error {
	if err := requirePermission(ctx, s.store, "devices", "create"); err != nil {
		return err
	}

	if device.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	return s.store.CreateDevice(enrichAuditCtx(ctx), device)
}

func (s *DeviceService) Get(ctx context.Context, id string) (*model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "read"); err != nil {
		return nil, err
	}

	device, err := s.store.GetDevice(id)
	if err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return device, nil
}

func (s *DeviceService) Update(ctx context.Context, device *model.Device) error {
	if err := requirePermission(ctx, s.store, "devices", "update"); err != nil {
		return err
	}

	if device.ID == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if device.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	return s.store.UpdateDevice(enrichAuditCtx(ctx), device)
}

func (s *DeviceService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "devices", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteDevice(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *DeviceService) Search(ctx context.Context, query string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "devices", "search"); err != nil {
		return nil, err
	}

	return s.store.SearchDevices(query)
}
