package service

import (
	"context"
	"errors"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type PoolService struct {
	store storage.ExtendedStorage
}

func NewPoolService(store storage.ExtendedStorage) *PoolService {
	return &PoolService{store: store}
}

func (s *PoolService) List(ctx context.Context, filter *model.NetworkPoolFilter) ([]model.NetworkPool, error) {
	if err := requirePermission(ctx, s.store, "pools", "list"); err != nil {
		return nil, err
	}
	return s.store.ListNetworkPools(ctx, filter)
}

// ListByNetwork lists pools for a network, returning ErrNotFound if the network doesn't exist.
func (s *PoolService) ListByNetwork(ctx context.Context, networkID string) ([]model.NetworkPool, error) {
	if err := requirePermission(ctx, s.store, "pools", "list"); err != nil {
		return nil, err
	}

	// Verify network exists
	if _, err := s.store.GetNetwork(ctx, networkID); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return s.store.ListNetworkPools(ctx, &model.NetworkPoolFilter{NetworkID: networkID})
}

func (s *PoolService) Create(ctx context.Context, pool *model.NetworkPool) error {
	if err := requirePermission(ctx, s.store, "pools", "create"); err != nil {
		return err
	}

	if pool.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	if pool.NetworkID == "" {
		return ValidationErrors{{Field: "network_id", Message: "Network ID is required"}}
	}

	// Verify network exists
	if _, err := s.store.GetNetwork(ctx, pool.NetworkID); err != nil {
		if errors.Is(err, storage.ErrNetworkNotFound) {
			return ErrNotFound
		}
		return err
	}

	if pool.StartIP == "" {
		return ValidationErrors{{Field: "start_ip", Message: "Start IP is required"}}
	}

	if pool.EndIP == "" {
		return ValidationErrors{{Field: "end_ip", Message: "End IP is required"}}
	}

	return s.store.CreateNetworkPool(enrichAuditCtx(ctx), pool)
}

func (s *PoolService) Get(ctx context.Context, id string) (*model.NetworkPool, error) {
	if err := requirePermission(ctx, s.store, "pools", "read"); err != nil {
		return nil, err
	}

	pool, err := s.store.GetNetworkPool(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return pool, nil
}

func (s *PoolService) Update(ctx context.Context, pool *model.NetworkPool) error {
	if err := requirePermission(ctx, s.store, "pools", "update"); err != nil {
		return err
	}

	if pool.ID == "" {
		return ValidationErrors{{Field: "id", Message: "ID is required"}}
	}

	if pool.Name == "" {
		return ValidationErrors{{Field: "name", Message: "Name is required"}}
	}

	return s.store.UpdateNetworkPool(enrichAuditCtx(ctx), pool)
}

func (s *PoolService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.store, "pools", "delete"); err != nil {
		return err
	}

	if err := s.store.DeleteNetworkPool(enrichAuditCtx(ctx), id); err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *PoolService) GetNextIP(ctx context.Context, poolID string) (string, error) {
	if err := requirePermission(ctx, s.store, "pools", "read"); err != nil {
		return "", err
	}

	ip, err := s.store.GetNextAvailableIP(ctx, poolID)
	if err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			return "", ErrNotFound
		}
		if errors.Is(err, storage.ErrIPNotAvailable) {
			return "", ErrIPNotAvailable
		}
		return "", err
	}
	return ip, nil
}

func (s *PoolService) ValidateIPInPool(ctx context.Context, poolID, ip string) (bool, error) {
	if err := requirePermission(ctx, s.store, "pools", "read"); err != nil {
		return false, err
	}

	return s.store.ValidateIPInPool(ctx, poolID, ip)
}

func (s *PoolService) GetHeatmap(ctx context.Context, poolID string) ([]storage.IPStatus, error) {
	if err := requirePermission(ctx, s.store, "pools", "read"); err != nil {
		return nil, err
	}

	heatmap, err := s.store.GetPoolHeatmap(ctx, poolID)
	if err != nil {
		if errors.Is(err, storage.ErrPoolNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return heatmap, nil
}
