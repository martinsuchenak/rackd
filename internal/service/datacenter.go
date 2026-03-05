package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type DatacenterService struct {
	store storage.ExtendedStorage
}

func NewDatacenterService(store storage.ExtendedStorage) *DatacenterService {
	return &DatacenterService{store: store}
}

func (s *DatacenterService) List(ctx context.Context, filter *model.DatacenterFilter) ([]model.Datacenter, error) {
	if err := requirePermission(ctx, s.store, "datacenters", "list"); err != nil {
		return nil, err
	}
	return s.store.ListDatacenters(ctx, filter)
}

func (s *DatacenterService) Create(ctx context.Context, dc *model.Datacenter) error {
	if err := requirePermission(ctx, s.store, "datacenters", "create"); err != nil {
		return err
	}

	if dc.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	return s.store.CreateDatacenter(enrichAuditCtx(ctx), dc)
}

func (s *DatacenterService) Get(ctx context.Context, id string) (*model.Datacenter, error) {
	if err := requirePermission(ctx, s.store, "datacenters", "read"); err != nil {
		return nil, err
	}

	dc, err := s.store.GetDatacenter(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return dc, nil
}

func (s *DatacenterService) Update(ctx context.Context, dc *model.Datacenter) error {
	if err := requirePermission(ctx, s.store, "datacenters", "update"); err != nil {
		return err
	}

	if dc.ID == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if dc.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	return s.store.UpdateDatacenter(enrichAuditCtx(ctx), dc)
}

func (s *DatacenterService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "datacenters", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteDatacenter(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *DatacenterService) GetDevices(ctx context.Context, datacenterID string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "datacenters", "read"); err != nil {
		return nil, err
	}

	// Verify datacenter exists before listing devices
	if _, err := s.store.GetDatacenter(ctx, datacenterID); err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return s.store.GetDatacenterDevices(ctx, datacenterID)
}

func (s *DatacenterService) Search(ctx context.Context, query string) ([]model.Datacenter, error) {
	if err := requirePermission(ctx, s.store, "datacenters", "search"); err != nil {
		return nil, err
	}

	return s.store.SearchDatacenters(ctx, query)
}
