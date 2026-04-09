package service

import (
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func TestBulkService_DelegatesOperationsAfterPermissionChecks(t *testing.T) {
	store := newServiceTestStorage()
	store.bulkResult = &storage.BulkResult{Total: 1, Success: 1}
	store.setPermission("user-1", "devices", "create", true)
	store.setPermission("user-1", "devices", "update", true)
	store.setPermission("user-1", "devices", "delete", true)
	store.setPermission("user-1", "networks", "create", true)
	store.setPermission("user-1", "networks", "delete", true)
	svc := NewBulkService(store)

	if _, err := svc.CreateDevices(userContext("user-1"), []*model.Device{{Name: "r1"}}); err != nil || store.lastBulkOp != "create-devices" {
		t.Fatalf("expected bulk create devices to delegate, op=%q err=%v", store.lastBulkOp, err)
	}
	if _, err := svc.AddTags(userContext("user-1"), []string{"dev-1"}, []string{"core"}); err != nil || store.lastBulkOp != "add-tags" {
		t.Fatalf("expected bulk add tags to delegate, op=%q err=%v", store.lastBulkOp, err)
	}
	if _, err := svc.CreateNetworks(userContext("user-1"), []*model.Network{{Name: "net", Subnet: "10.0.0.0/24"}}); err != nil || store.lastBulkOp != "create-networks" {
		t.Fatalf("expected bulk create networks to delegate, op=%q err=%v", store.lastBulkOp, err)
	}
}
