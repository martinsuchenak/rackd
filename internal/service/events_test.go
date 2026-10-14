package service

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type capturedEvent struct {
	Type    model.EventType
	Payload any
}

// captureEvents replaces publishEvent for the test and returns the events
// published so far.
func captureEvents(t *testing.T) func() []capturedEvent {
	t.Helper()
	var mu sync.Mutex
	var events []capturedEvent
	prev := publishEvent
	publishEvent = func(et model.EventType, payload any) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, capturedEvent{et, payload})
	}
	t.Cleanup(func() { publishEvent = prev })
	return func() []capturedEvent {
		mu.Lock()
		defer mu.Unlock()
		out := append([]capturedEvent(nil), events...)
		events = nil
		return out
	}
}

func eventTypes(events []capturedEvent) []model.EventType {
	out := make([]model.EventType, len(events))
	for i, e := range events {
		out[i] = e.Type
	}
	return out
}

func TestServicesPublishEvents(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	svc := NewServices(store, nil, nil)
	ctx := SystemContext(context.Background(), "test")
	events := captureEvents(t)

	expect := func(step string, want ...model.EventType) []capturedEvent {
		t.Helper()
		got := events()
		if types := eventTypes(got); len(types) != len(want) || !equalTypes(types, want) {
			t.Fatalf("%s published %v, want %v", step, types, want)
		}
		return got
	}

	// Devices: created, updated, deleted (with the device as it was).
	device := &model.Device{Name: "srv01", Hostname: "srv01.example.test", Status: model.DeviceStatusActive}
	if err := svc.Devices.Create(ctx, device); err != nil {
		t.Fatal(err)
	}
	got := expect("create device", model.EventTypeDeviceCreated)
	created := got[0].Payload.(*model.Device)
	if created.ID != device.ID || created == device {
		t.Errorf("created payload = %+v (must be a copy of the stored device)", created)
	}
	device.Name = "srv01-renamed"
	if err := svc.Devices.Update(ctx, device); err != nil {
		t.Fatal(err)
	}
	if p := expect("update device", model.EventTypeDeviceUpdated)[0].Payload.(*model.Device); p.Name != "srv01-renamed" {
		t.Errorf("updated payload name = %s", p.Name)
	}
	if err := svc.Devices.Delete(ctx, device.ID); err != nil {
		t.Fatal(err)
	}
	if p := expect("delete device", model.EventTypeDeviceDeleted)[0].Payload.(*model.Device); p.Hostname != "srv01.example.test" {
		t.Errorf("deleted payload lost the device's data: %+v", p)
	}
	if err := svc.Devices.Delete(ctx, device.ID); err == nil {
		t.Error("deleting a missing device succeeded")
	}
	expect("failed delete")

	// Networks.
	network := &model.Network{Name: "lan", Subnet: "10.9.0.0/24"}
	if err := svc.Networks.Create(ctx, network); err != nil {
		t.Fatal(err)
	}
	expect("create network", model.EventTypeNetworkCreated)
	if err := svc.Networks.Update(ctx, network); err != nil {
		t.Fatal(err)
	}
	expect("update network", model.EventTypeNetworkUpdated)
	if err := svc.Networks.Delete(ctx, network.ID); err != nil {
		t.Fatal(err)
	}
	if p := expect("delete network", model.EventTypeNetworkDeleted)[0].Payload.(*model.Network); p.Subnet != "10.9.0.0/24" {
		t.Errorf("deleted network payload = %+v", p)
	}

	// Bulk: one event per device that succeeded.
	bulk := []*model.Device{{Name: "b1"}, {Name: "b2"}}
	if _, err := svc.Bulk.CreateDevices(ctx, bulk); err != nil {
		t.Fatal(err)
	}
	expect("bulk create", model.EventTypeDeviceCreated, model.EventTypeDeviceCreated)
	if _, err := svc.Bulk.AddTags(ctx, []string{bulk[0].ID}, []string{"rack-1"}); err != nil {
		t.Fatal(err)
	}
	if p := expect("bulk add tags", model.EventTypeDeviceUpdated)[0].Payload.(*model.Device); len(p.Tags) != 1 {
		t.Errorf("tagged payload = %+v", p)
	}
	res, err := svc.Bulk.DeleteDevices(ctx, []string{bulk[0].ID, "missing"})
	if err != nil {
		t.Fatal(err)
	}
	got = expect("bulk delete", model.EventTypeDeviceDeleted)
	if res.Success != 1 || got[0].Payload.(*model.Device).Name != "b1" {
		t.Errorf("bulk delete result %+v, payload %+v", res, got[0].Payload)
	}

	// IP conflicts: detection publishes new conflicts once; resolving publishes too.
	for _, name := range []string{"a", "b"} {
		if err := svc.Devices.Create(ctx, &model.Device{Name: name, Addresses: []model.Address{{IP: "10.9.9.9", Type: "ipv4"}}}); err != nil {
			t.Fatal(err)
		}
	}
	expect("devices sharing an IP", model.EventTypeDeviceCreated, model.EventTypeDeviceCreated)
	if _, err := svc.Conflicts.DetectDuplicateIPs(ctx); err != nil {
		t.Fatal(err)
	}
	conflict := expect("detect", model.EventTypeConflictDetected)[0].Payload.(model.Conflict)
	if _, err := svc.Conflicts.DetectDuplicateIPs(ctx); err != nil {
		t.Fatal(err)
	}
	expect("detect again")
	if err := svc.Conflicts.Resolve(ctx, &model.ConflictResolution{ConflictID: conflict.ID}); err != nil {
		t.Fatal(err)
	}
	if p := expect("resolve conflict", model.EventTypeConflictResolved)[0].Payload.(*model.Conflict); p.Status != model.ConflictStatusResolved {
		t.Errorf("resolved payload status = %s", p.Status)
	}
}

func equalTypes(a, b []model.EventType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
