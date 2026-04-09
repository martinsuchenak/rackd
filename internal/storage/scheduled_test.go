package storage

import (
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestSQLiteScheduledScanStorageCRUD(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	scheduled, err := NewSQLiteScheduledScanStorage(storage.DB())
	if err != nil {
		t.Fatalf("NewSQLiteScheduledScanStorage failed: %v", err)
	}

	nextRun := time.Now().Add(15 * time.Minute).UTC()
	scan := &model.ScheduledScan{
		NetworkID:      "network-1",
		ProfileID:      "profile-1",
		Name:           "Nightly scan",
		CronExpression: "0 * * * *",
		Enabled:        true,
		Description:    "nightly inventory",
		NextRunAt:      &nextRun,
	}
	if err := scheduled.Create(scan); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := scheduled.Get(scan.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != scan.Name || got.NextRunAt == nil {
		t.Fatalf("unexpected scheduled scan after create: %+v", got)
	}

	lastRun := time.Now().UTC()
	scan.Enabled = false
	scan.Description = "updated"
	scan.LastRunAt = &lastRun
	if err := scheduled.Update(scan); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err = scheduled.Get(scan.ID)
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if got.Enabled || got.Description != "updated" || got.LastRunAt == nil {
		t.Fatalf("unexpected scheduled scan after update: %+v", got)
	}

	all, err := scheduled.List("")
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 scheduled scan, got %d", len(all))
	}

	filtered, err := scheduled.List("network-1")
	if err != nil {
		t.Fatalf("List by network failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 scheduled scan for network, got %d", len(filtered))
	}

	if err := scheduled.Delete(scan.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := scheduled.Get(scan.ID); err != ErrScheduledScanNotFound {
		t.Fatalf("expected ErrScheduledScanNotFound, got %v", err)
	}
}

func TestSQLiteScheduledScanStorageErrors(t *testing.T) {
	storage := newTestStorage(t)
	defer storage.Close()

	scheduled, err := NewSQLiteScheduledScanStorage(storage.DB())
	if err != nil {
		t.Fatalf("NewSQLiteScheduledScanStorage failed: %v", err)
	}

	if err := scheduled.Create(&model.ScheduledScan{
		NetworkID:      "network-1",
		ProfileID:      "profile-1",
		Name:           "Too fast",
		CronExpression: "* * * * *",
	}); err == nil {
		t.Fatal("expected invalid scheduled scan create to fail")
	}

	if err := scheduled.Update(&model.ScheduledScan{ID: "missing"}); err != ErrScheduledScanNotFound {
		t.Fatalf("expected ErrScheduledScanNotFound, got %v", err)
	}
	if err := scheduled.Delete("missing"); err != ErrScheduledScanNotFound {
		t.Fatalf("expected ErrScheduledScanNotFound, got %v", err)
	}
}
