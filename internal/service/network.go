package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
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
	return s.store.ListNetworks(filter)
}

func (s *NetworkService) Create(ctx context.Context, network *model.Network) error {
	if err := requirePermission(ctx, s.store, "networks", "create"); err != nil {
		return err
	}

	if network.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	if network.Subnet == "" {
		return ValidationErrors{{Field: "subnet", Message: "Subnet is required"}}
	}

	return s.store.CreateNetwork(enrichAuditCtx(ctx), network)
}

func (s *NetworkService) Get(ctx context.Context, id string) (*model.Network, error) {
	if err := requirePermission(ctx, s.store, "networks", "read"); err != nil {
		return nil, err
	}

	network, err := s.store.GetNetwork(id)
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

	if network.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	if network.Subnet == "" {
		return ValidationErrors{{Field: "subnet", Message: "Subnet is required"}}
	}

	return s.store.UpdateNetwork(enrichAuditCtx(ctx), network)
}

func (s *NetworkService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "networks", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteNetwork(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *NetworkService) GetDevices(ctx context.Context, networkID string) ([]model.Device, error) {
	if err := requirePermission(ctx, s.store, "networks", "read"); err != nil {
		return nil, err
	}

	return s.store.GetNetworkDevices(networkID)
}

func (s *NetworkService) GetUtilization(ctx context.Context, networkID string) (*model.NetworkUtilization, error) {
	if err := requirePermission(ctx, s.store, "networks", "read"); err != nil {
		return nil, err
	}

	return s.store.GetNetworkUtilization(networkID)
}

func (s *NetworkService) Search(ctx context.Context, query string) ([]model.Network, error) {
	if err := requirePermission(ctx, s.store, "networks", "search"); err != nil {
		return nil, err
	}

	return s.store.SearchNetworks(query)
}
