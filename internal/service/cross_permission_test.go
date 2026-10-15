package service

import (
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Resolving a duplicate-IP conflict takes the IP off other devices, so it
// needs devices:update as well as conflicts:resolve.
func TestResolvingADuplicateIPNeedsDevicePermission(t *testing.T) {
	f := newPrivilegeFixture(t)
	devices := NewDeviceService(f.store)
	keep := &model.Device{Name: "a", Addresses: []model.Address{{IP: "10.1.1.1", Type: "ipv4"}}}
	other := &model.Device{Name: "b", Addresses: []model.Address{{IP: "10.1.1.1", Type: "ipv4"}}}
	for _, d := range []*model.Device{keep, other} {
		if err := devices.Create(f.sys(), d); err != nil {
			t.Fatal(err)
		}
	}
	conflicts := NewConflictService(f.store)
	found, err := conflicts.DetectDuplicateIPs(f.sys())
	if err != nil || len(found) == 0 {
		t.Fatalf("detect: %v, %d conflicts", err, len(found))
	}
	recorded, err := f.store.GetConflictsByIP(f.sys(), "10.1.1.1")
	if err != nil || len(recorded) == 0 {
		t.Fatalf("no recorded conflict: %v", err)
	}
	resolution := &model.ConflictResolution{ConflictID: recorded[0].ID, KeepDeviceID: keep.ID}

	resolver := f.user("resolver", f.role("resolver", "conflicts:resolve"))
	wantForbidden(t, "resolve without devices:update", conflicts.Resolve(as(resolver), resolution))

	both := f.user("both", f.role("both", "conflicts:resolve", "devices:update"))
	if err := conflicts.Resolve(as(both), resolution); err != nil {
		t.Errorf("resolve with both permissions: %v", err)
	}
}
