package service

import (
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestDatacenterService_SearchUsesListPermission(t *testing.T) {
	store := newServiceTestStorage()
	store.datacenters = []model.Datacenter{{ID: "dc-1", Name: "perth-dc"}}
	store.setPermission("user-1", "datacenters", "list", true)
	svc := NewDatacenterService(store)

	results, err := svc.Search(userContext("user-1"), "perth-dc")
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "dc-1" {
		t.Fatalf("expected search result for perth-dc, got %#v", results)
	}
}

func TestDatacenterService_CRUDAndGetDevicesPaths(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "datacenters", "list", true)
	store.setPermission("user-1", "datacenters", "create", true)
	store.setPermission("user-1", "datacenters", "read", true)
	store.setPermission("user-1", "datacenters", "update", true)
	store.setPermission("user-1", "datacenters", "delete", true)
	store.datacenters = []model.Datacenter{{ID: "dc-1", Name: "perth"}}
	store.datacenterDevices["dc-1"] = []model.Device{{ID: "dev-1", Name: "router"}}
	svc := NewDatacenterService(store)

	if _, err := svc.List(userContext("user-1"), nil); err != nil {
		t.Fatalf("List returned unexpected error: %v", err)
	}
	if err := svc.Create(userContext("user-1"), &model.Datacenter{ID: "dc-2", Name: "sydney"}); err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}
	if _, err := svc.Get(userContext("user-1"), "dc-1"); err != nil {
		t.Fatalf("Get returned unexpected error: %v", err)
	}
	if err := svc.Update(userContext("user-1"), &model.Datacenter{ID: "dc-1", Name: "perth-updated"}); err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}
	devices, err := svc.GetDevices(userContext("user-1"), "dc-1")
	if err != nil || len(devices) != 1 {
		t.Fatalf("expected datacenter devices, got %#v err=%v", devices, err)
	}
	if err := svc.Delete(userContext("user-1"), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found on delete, got %v", err)
	}
}
