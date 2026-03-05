package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestBulkCreateDevices(t *testing.T) {
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	devices := []*model.Device{
		{Name: "device1", Hostname: "host1.example.com"},
		{Name: "device2", Hostname: "host2.example.com"},
		{Name: "device3", Hostname: "host3.example.com"},
	}

	result, err := store.BulkCreateDevices(context.Background(), devices)
	if err != nil {
		t.Fatalf("BulkCreateDevices failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if result.Success != 3 {
		t.Errorf("expected success 3, got %d", result.Success)
	}
	if result.Failed != 0 {
		t.Errorf("expected failed 0, got %d", result.Failed)
	}
}

func TestBulkDeleteDevices(t *testing.T) {
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create devices first
	devices := []*model.Device{
		{Name: "device1"},
		{Name: "device2"},
		{Name: "device3"},
	}
	for _, d := range devices {
		if err := store.CreateDevice(context.Background(), d); err != nil {
			t.Fatalf("failed to create device: %v", err)
		}
	}

	ids := []string{devices[0].ID, devices[1].ID, devices[2].ID}
	result, err := store.BulkDeleteDevices(context.Background(), ids)
	if err != nil {
		t.Fatalf("BulkDeleteDevices failed: %v", err)
	}

	if result.Total != 3 {
		t.Errorf("expected total 3, got %d", result.Total)
	}
	if result.Success != 3 {
		t.Errorf("expected success 3, got %d", result.Success)
	}
}

func TestBulkAddTags(t *testing.T) {
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create devices
	devices := []*model.Device{
		{Name: "device1", Tags: []string{"existing"}},
		{Name: "device2"},
	}
	for _, d := range devices {
		if err := store.CreateDevice(context.Background(), d); err != nil {
			t.Fatalf("failed to create device: %v", err)
		}
	}

	ids := []string{devices[0].ID, devices[1].ID}
	tags := []string{"prod", "web"}

	result, err := store.BulkAddTags(context.Background(), ids, tags)
	if err != nil {
		t.Fatalf("BulkAddTags failed: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("expected success 2, got %d", result.Success)
	}

	// Verify tags were added
	d1, _ := store.GetDevice(context.Background(), devices[0].ID)
	if len(d1.Tags) != 3 { // existing + prod + web
		t.Errorf("expected 3 tags, got %d", len(d1.Tags))
	}
}

func TestBulkRemoveTags(t *testing.T) {
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create devices with tags
	devices := []*model.Device{
		{Name: "device1", Tags: []string{"prod", "web", "keep"}},
		{Name: "device2", Tags: []string{"prod", "db"}},
	}
	for _, d := range devices {
		if err := store.CreateDevice(context.Background(), d); err != nil {
			t.Fatalf("failed to create device: %v", err)
		}
	}

	ids := []string{devices[0].ID, devices[1].ID}
	tags := []string{"prod"}

	result, err := store.BulkRemoveTags(context.Background(), ids, tags)
	if err != nil {
		t.Fatalf("BulkRemoveTags failed: %v", err)
	}

	if result.Success != 2 {
		t.Errorf("expected success 2, got %d", result.Success)
	}

	// Verify tag was removed
	d1, _ := store.GetDevice(context.Background(), devices[0].ID)
	if len(d1.Tags) != 2 { // web + keep
		t.Errorf("expected 2 tags, got %d", len(d1.Tags))
	}
}

func TestBulkCreateNetworks(t *testing.T) {
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	networks := []*model.Network{
		{Name: "net1", Subnet: "10.0.1.0/24"},
		{Name: "net2", Subnet: "10.0.2.0/24"},
	}

	result, err := store.BulkCreateNetworks(context.Background(), networks)
	if err != nil {
		t.Fatalf("BulkCreateNetworks failed: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if result.Success != 2 {
		t.Errorf("expected success 2, got %d", result.Success)
	}
}

func TestBulkDeleteNetworks(t *testing.T) {
	store, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create networks first
	networks := []*model.Network{
		{Name: "net1", Subnet: "10.0.1.0/24"},
		{Name: "net2", Subnet: "10.0.2.0/24"},
	}
	for _, n := range networks {
		if err := store.CreateNetwork(context.Background(), n); err != nil {
			t.Fatalf("failed to create network: %v", err)
		}
	}

	ids := []string{networks[0].ID, networks[1].ID}
	result, err := store.BulkDeleteNetworks(context.Background(), ids)
	if err != nil {
		t.Fatalf("BulkDeleteNetworks failed: %v", err)
	}

	if result.Total != 2 {
		t.Errorf("expected total 2, got %d", result.Total)
	}
	if result.Success != 2 {
		t.Errorf("expected success 2, got %d", result.Success)
	}
}
