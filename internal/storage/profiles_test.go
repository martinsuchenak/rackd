package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestSQLiteProfileStorageCRUD(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	profiles, err := NewSQLiteProfileStorage(storage.DB())
	if err != nil {
		t.Fatalf("NewSQLiteProfileStorage failed: %v", err)
	}

	initial, err := profiles.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(initial) < 3 {
		t.Fatalf("expected seeded default profiles, got %d", len(initial))
	}

	profile := &model.ScanProfile{
		Name:        "Custom Profile",
		ScanType:    "custom",
		Ports:       []int{22, 8443},
		EnableSNMP:  true,
		EnableSSH:   true,
		TimeoutSec:  15,
		MaxWorkers:  8,
		Description: "custom profile for tests",
	}
	if err := profiles.Create(context.Background(), profile); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := profiles.Get(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != profile.Name || len(got.Ports) != 2 {
		t.Fatalf("unexpected profile after create: %+v", got)
	}

	profile.Description = "updated"
	profile.MaxWorkers = 12
	if err := profiles.Update(context.Background(), profile); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err = profiles.Get(context.Background(), profile.ID)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if got.Description != "updated" || got.MaxWorkers != 12 {
		t.Fatalf("unexpected profile after update: %+v", got)
	}

	if err := profiles.Delete(context.Background(), profile.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := profiles.Get(context.Background(), profile.ID); err != ErrProfileNotFound {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestSQLiteProfileStorageErrors(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	profiles, err := NewSQLiteProfileStorage(storage.DB())
	if err != nil {
		t.Fatalf("NewSQLiteProfileStorage failed: %v", err)
	}

	if err := profiles.Create(context.Background(), &model.ScanProfile{
		Name:       "Invalid",
		ScanType:   "bogus",
		TimeoutSec: 10,
		MaxWorkers: 1,
	}); err == nil {
		t.Fatal("expected invalid profile create to fail")
	}

	if err := profiles.Update(context.Background(), &model.ScanProfile{
		ID:         "missing",
		Name:       "Missing",
		ScanType:   "quick",
		TimeoutSec: 10,
		MaxWorkers: 1,
	}); err != ErrProfileNotFound {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}

	if err := profiles.Delete(context.Background(), "missing"); err != ErrProfileNotFound {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}
