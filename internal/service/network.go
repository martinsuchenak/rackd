package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type NetworkService struct {
	store storage.ExtendedStorage
}

func NewNetworkService(store storage.ExtendedStorage) *NetworkService {
	return &NetworkService{store: store}
}

func (s *NetworkService) List(ctx context.Context, filter *model.NetworkFilter) ([]model.Network, error) {
	if err := requirePermission(ctx, s.store, "networks", "list"); err != nil {
		return nil, err
	}
	return s.store.ListNetworks(ctx, filter)
}

func (s *NetworkService) Create(ctx context.Context, network *model.Network) error {
	if err := requirePermission(ctx, s.store, "networks", "create"); err != nil {
		return err
	}

	if errs := ValidateNetwork(network); len(errs) > 0 {
		return errs
	}

	if err := s.store.CreateNetwork(enrichAuditCtx(ctx), network); err != nil {
		return err
	}
	publishEvent(model.EventTypeNetworkCreated, networkSnapshot(network))
	return nil
}

func (s *NetworkService) Get(ctx context.Context, id string) (*model.Network, error) {
	if err := requirePermission(ctx, s.store, "networks", "read"); err != nil {
		return nil, err
	}

	network, err := s.store.GetNetwork(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return network, nil
}

func (s *NetworkService) Update(ctx context.Context, network *model.Network) error {
	if err := requirePermission(ctx, s.store, "networks", "update"); err != nil {
		return err
	}

	if network.ID == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if errs := ValidateNetwork(network); len(errs) > 0 {
		return errs
	}

	if err := s.store.UpdateNetwork(enrichAuditCtx(ctx), network); err != nil {
		return err
	}
	publishEvent(model.EventTypeNetworkUpdated, networkSnapshot(network))
	return nil
}

func (s *NetworkService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "networks", "delete"); err != nil {
		return err
	}

	deleted, err := s.store.GetNetwork(ctx, id)
	if err != nil {
		deleted = &model.Network{ID: id}
	}
	if err := s.store.DeleteNetwork(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			return ErrNotFound
		}
		return err
	}
	publishEvent(model.EventTypeNetworkDeleted, deleted)
	return nil
}

func (s *NetworkService) GetDevices(ctx context.Context, networkID string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "networks", "read"); err != nil {
		return nil, err
	}

	// Verify network exists before listing devices
	if _, err := s.store.GetNetwork(ctx, networkID); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return s.store.GetNetworkDevices(ctx, networkID)
}

func (s *NetworkService) GetUtilization(ctx context.Context, networkID string) (*model.NetworkUtilization, error) {
	if err := requirePermission(ctx, s.store, "networks", "read"); err != nil {
		return nil, err
	}

	utilization, err := s.store.GetNetworkUtilization(ctx, networkID)
	if err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return utilization, nil
}

func (s *NetworkService) Search(ctx context.Context, query string) ([]model.Network, error) {
	if err := requirePermission(ctx, s.store, "networks", "list"); err != nil {
		return nil, err
	}

	return s.store.SearchNetworks(ctx, query)
}
