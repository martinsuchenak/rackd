package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestNetworkService_SearchUsesListPermission(t *testing.T) {
	store := newServiceTestStorage()
	store.networks = []model.Network{{ID: "net-1", Name: "prod-net"}}
	store.setPermission("user-1", "networks", "list", true)
	svc := NewNetworkService(store)

	results, err := svc.Search(userContext("user-1"), "prod-net")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "net-1" {
		t.Fatalf("expected search result for prod-net, got %#v", results)
	}
}

func TestNetworkService_CRUDDevicesAndUtilizationPaths(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "networks", "list", true)
	store.setPermission("user-1", "networks", "create", true)
	store.setPermission("user-1", "networks", "read", true)
	store.setPermission("user-1", "networks", "update", true)
	store.setPermission("user-1", "networks", "delete", true)
	store.networks = []model.Network{{ID: "net-1", Name: "prod-net", Subnet: "10.0.0.0/24"}}
	store.networkDevices["net-1"] = []model.Device{{ID: "dev-1", Name: "router"}}
	store.networkUtilization = &model.NetworkUtilization{NetworkID: "net-1"}
	svc := NewNetworkService(store)

	if _, err := svc.List(userContext("user-1"), nil); err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if err := svc.Create(userContext("user-1"), &model.Network{ID: "net-2", Name: "lab", Subnet: "10.1.0.0/24"}); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if _, err := svc.Get(userContext("user-1"), "net-1"); err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if err := svc.Update(userContext("user-1"), &model.Network{ID: "net-1", Name: "prod-updated", Subnet: "10.0.0.0/24"}); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}
	if _, err := svc.GetDevices(userContext("user-1"), "net-1"); err != nil {
		t.Fatalf("GetDevices returned unexpected error: %v", err)
	}
	if _, err := svc.GetUtilization(userContext("user-1"), "net-1"); err != nil {
		t.Fatalf("GetUtilization returned unexpected error: %v", err)
	}
	if err := svc.Delete(userContext("user-1"), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found on delete, got %v", err)
	}
}
