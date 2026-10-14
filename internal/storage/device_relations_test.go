package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestLoadDeviceRelationsAcrossChunks(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	const n = relationChunk + 20 // spans two IN (...) chunks
	devices := make([]*model.Device, n)
	for i := range devices {
		devices[i] = &model.Device{Name: fmt.Sprintf("dev-%04d", i), Tags: []string{}, Domains: []string{}, Addresses: []model.Address{}}
		if i%2 == 0 { // odd devices have no relations
			devices[i].Tags = []string{"a", fmt.Sprintf("t%d", i)}
			devices[i].Domains = []string{fmt.Sprintf("d%d.example.test", i)}
			devices[i].Addresses = []model.Address{
				{IP: fmt.Sprintf("10.%d.%d.1", i/256, i%256), Type: "ipv4", Label: "eth0", Port: intPtr(22)},
				{IP: fmt.Sprintf("10.%d.%d.2", i/256, i%256), Type: "ipv4", Label: "eth1"},
			}
		}
	}
	if _, err := s.BulkCreateDevices(ctx, devices); err != nil {
		t.Fatal(err)
	}

	listed, err := s.ListDevices(ctx, &model.DeviceFilter{Pagination: model.Pagination{Limit: model.MaxPageSize}})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != n {
		t.Fatalf("listed %d devices, want %d", len(listed), n)
	}
	for _, d := range listed {
		var i int
		if _, err := fmt.Sscanf(d.Name, "dev-%04d", &i); err != nil {
			t.Fatal(err)
		}
		if d.Tags == nil || d.Domains == nil || d.Addresses == nil {
			t.Fatalf("%s: relations must be empty slices, not nil", d.Name)
		}
		want := 0
		if i%2 == 0 {
			want = 2
		}
		if len(d.Addresses) != want || len(d.Tags) != want || len(d.Domains) != want/2 {
			t.Errorf("%s: %d addresses, %d tags, %d domains", d.Name, len(d.Addresses), len(d.Tags), len(d.Domains))
		}
		if want > 0 && (d.Addresses[0].Port == nil || *d.Addresses[0].Port != 22 || d.Addresses[0].Label != "eth0") {
			t.Errorf("%s: first address = %+v", d.Name, d.Addresses[0])
		}
	}

	// The same device twice in one slice gets its relations in both places.
	twice := []model.Device{{ID: devices[0].ID}, {ID: devices[0].ID}}
	if err := s.loadDeviceRelations(ctx, twice); err != nil {
		t.Fatal(err)
	}
	if len(twice[0].Addresses) != 2 || len(twice[1].Addresses) != 2 {
		t.Errorf("duplicate entries: %d and %d addresses", len(twice[0].Addresses), len(twice[1].Addresses))
	}

	got, err := s.GetDevice(ctx, devices[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Addresses) != 2 || len(got.Tags) != 2 || len(got.Domains) != 1 {
		t.Errorf("GetDevice: %d addresses, %d tags, %d domains", len(got.Addresses), len(got.Tags), len(got.Domains))
	}
}
