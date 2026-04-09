package service

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestPoolService_CreateRejectsMissingNetwork(t *testing.T) {
	store := newServiceTestStorage()
	svc := NewPoolService(store)
	ctx := SystemContext(context.Background(), "test")

	err := svc.Create(ctx, &model.NetworkPool{
		Name:      "pool-a",
		NetworkID: "missing-network",
		StartIP:   "10.0.0.10",
		EndIP:     "10.0.0.20",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected network not found, got %v", err)
	}
}

func TestPoolService_GetNextIPMapsPoolAndAvailabilityErrors(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "pools", "read", true)
	svc := NewPoolService(store)

	_, err := svc.GetNextIP(userContext("user-1"), "missing-pool")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for missing pool, got %v", err)
	}

	store.pools["pool-1"] = true
	_, err = svc.GetNextIP(userContext("user-1"), "pool-1")
	if !errors.Is(err, ErrIPNotAvailable) {
		t.Fatalf("expected ErrIPNotAvailable, got %v", err)
	}
}

func TestPoolService_ListByNetworkAndGetHeatmapMapMissingNetworkOrPool(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "pools", "list", true)
	store.setPermission("user-1", "pools", "read", true)
	svc := NewPoolService(store)

	_, err := svc.ListByNetwork(userContext("user-1"), "missing-network")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected network not found, got %v", err)
	}

	_, err = svc.GetHeatmap(userContext("user-1"), "missing-pool")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected pool not found, got %v", err)
	}
}
