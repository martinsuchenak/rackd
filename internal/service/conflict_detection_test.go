package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestDeviceWritesDetectDuplicateIPs(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := SystemContext(context.Background(), "test")
	svc := NewServices(store, nil, nil)
	captureEvents(t)

	conflictsFor := func(ip string) []model.Conflict {
		t.Helper()
		cs, err := store.GetConflictsByIP(ctx, ip)
		if err != nil {
			t.Fatal(err)
		}
		return cs
	}

	// 150 devices, so the duplicate below is beyond the first page of 100 by
	// name (the old check listed only that page).
	batch := make([]*model.Device, 150)
	for i := range batch {
		batch[i] = &model.Device{Name: fmt.Sprintf("a-%03d", i), Addresses: []model.Address{{IP: fmt.Sprintf("10.1.%d.%d", i/250, i%250+1), Type: "ipv4"}}}
	}
	if _, err := store.BulkCreateDevices(ctx, batch); err != nil {
		t.Fatal(err)
	}
	dupIP := batch[120].Addresses[0].IP
	late := &model.Device{Name: "zz-late", Addresses: []model.Address{{IP: dupIP, Type: "ipv4"}}}
	if err := svc.Devices.Create(ctx, late); err != nil {
		t.Fatal(err)
	}
	cs := conflictsFor(dupIP)
	if len(cs) != 1 || len(cs[0].DeviceIDs) != 2 || cs[0].DeviceNames[0] != "a-120" || cs[0].DeviceNames[1] != "zz-late" {
		t.Fatalf("conflicts for %s = %+v", dupIP, cs)
	}
	// A second write doesn't duplicate the record.
	if err := svc.Devices.Update(ctx, late); err != nil {
		t.Fatal(err)
	}
	if len(conflictsFor(dupIP)) != 1 {
		t.Error("update created a second conflict for the same IP")
	}

	// An update that adds a duplicate is caught too.
	other := &model.Device{Name: "other", Addresses: []model.Address{{IP: "10.200.0.1", Type: "ipv4"}}}
	if err := svc.Devices.Create(ctx, other); err != nil {
		t.Fatal(err)
	}
	other.Addresses = append(other.Addresses, model.Address{IP: batch[5].Addresses[0].IP, Type: "ipv4"})
	if err := svc.Devices.Update(ctx, other); err != nil {
		t.Fatal(err)
	}
	if len(conflictsFor(batch[5].Addresses[0].IP)) != 1 {
		t.Error("update adding a duplicate IP was not detected")
	}

	// The same IP twice on one device is not a conflict.
	twice := &model.Device{Name: "twice", Addresses: []model.Address{{IP: "10.201.0.1", Type: "ipv4"}, {IP: "10.201.0.1", Type: "ipv6"}}}
	if err := svc.Devices.Create(ctx, twice); err != nil {
		t.Fatal(err)
	}
	if cs := conflictsFor("10.201.0.1"); len(cs) != 0 {
		t.Errorf("one device with an IP twice raised %+v", cs)
	}

	// An ignored conflict stays ignored: a new holder doesn't raise another.
	ignored := conflictsFor(dupIP)[0]
	if err := store.UpdateConflictStatus(ctx, ignored.ID, model.ConflictStatusIgnored, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.Devices.Create(ctx, &model.Device{Name: "third", Addresses: []model.Address{{IP: dupIP, Type: "ipv4"}}}); err != nil {
		t.Fatal(err)
	}
	if cs := conflictsFor(dupIP); len(cs) != 1 || cs[0].Status != model.ConflictStatusIgnored {
		t.Errorf("ignored conflict was reopened or duplicated: %+v", cs)
	}
}
