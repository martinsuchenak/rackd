package storage

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Check-then-insert races the Strix run found: duplicate-IP conflicts and
// discovered hosts could be recorded twice by concurrent writers.

func TestOneActiveConflictPerDuplicateIP(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()

	var wg sync.WaitGroup
	errs := make([]error, 12)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = s.CreateConflict(ctx, &model.Conflict{Type: model.ConflictTypeDuplicateIP, Status: model.ConflictStatusActive, IPAddress: "10.0.0.7"})
		}(i)
	}
	wg.Wait()
	created := 0
	for _, err := range errs {
		switch {
		case err == nil:
			created++
		case !errors.Is(err, ErrConflictRecorded):
			t.Fatalf("CreateConflict: %v", err)
		}
	}
	if created != 1 {
		t.Fatalf("%d concurrent creates succeeded, want 1", created)
	}

	// Once resolved, a new duplicate on the address can be recorded.
	active, _ := s.ListConflicts(ctx, &model.ConflictFilter{Status: model.ConflictStatusActive})
	if err := s.UpdateConflictStatus(ctx, active[0].ID, model.ConflictStatusResolved, "test", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateConflict(ctx, &model.Conflict{Type: model.ConflictTypeDuplicateIP, Status: model.ConflictStatusActive, IPAddress: "10.0.0.7"}); err != nil {
		t.Fatalf("recording a conflict after the last was resolved: %v", err)
	}
}

func TestSaveDiscoveredDeviceOnceUnderConcurrentScans(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	network := &model.Network{Name: "scan", Subnet: "10.0.0.0/24"}
	if err := s.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	created := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := s.SaveDiscoveredDevice(ctx, &model.DiscoveredDevice{IP: "10.0.0.9", NetworkID: network.ID, Status: "online"})
			if err != nil {
				t.Errorf("SaveDiscoveredDevice: %v", err)
				return
			}
			if ok {
				mu.Lock()
				created++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	devices, err := s.ListDiscoveredDevices(ctx, network.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || created != 1 {
		t.Fatalf("%d records, %d reported created; want 1 and 1", len(devices), created)
	}
}

func TestDuplicateRecordMigrationsDedupe(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	network := &model.Network{Name: "dedupe", Subnet: "10.0.0.0/24"}
	if err := s.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}

	// Duplicates as they could exist before the indexes.
	for _, idx := range []string{"idx_conflicts_active_duplicate_ip", "idx_discovered_devices_network_ip"} {
		if _, err := s.db.ExecContext(ctx, "DROP INDEX "+idx); err != nil {
			t.Fatal(err)
		}
	}
	first := &model.Conflict{Type: model.ConflictTypeDuplicateIP, Status: model.ConflictStatusActive, IPAddress: "10.0.0.7", DetectedAt: time.Now().Add(-time.Hour).UTC()}
	if err := s.CreateConflict(ctx, first); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := s.CreateConflict(ctx, &model.Conflict{Type: model.ConflictTypeDuplicateIP, Status: model.ConflictStatusActive, IPAddress: "10.0.0.7"}); err != nil {
			t.Fatal(err)
		}
	}
	old := &model.DiscoveredDevice{IP: "10.0.0.9", NetworkID: network.ID, Hostname: "old"}
	if err := s.CreateDiscoveredDevice(ctx, old); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)
	recent := &model.DiscoveredDevice{IP: "10.0.0.9", NetworkID: network.ID, Hostname: "recent"}
	if err := s.CreateDiscoveredDevice(ctx, recent); err != nil {
		t.Fatal(err)
	}

	if err := migrateUniqueActiveDuplicateIPConflictUp(ctx, s.db); err != nil {
		t.Fatal(err)
	}
	if err := migrateUniqueDiscoveredDeviceUp(ctx, s.db); err != nil {
		t.Fatal(err)
	}

	active, _ := s.ListConflicts(ctx, &model.ConflictFilter{Status: model.ConflictStatusActive})
	if len(active) != 1 || active[0].ID != first.ID {
		t.Errorf("active conflicts after dedupe: %+v, want only the first detected", active)
	}
	resolved, _ := s.ListConflicts(ctx, &model.ConflictFilter{Status: model.ConflictStatusResolved})
	if len(resolved) != 2 {
		t.Errorf("%d duplicates resolved, want 2", len(resolved))
	}
	devices, _ := s.ListDiscoveredDevices(ctx, network.ID)
	if len(devices) != 1 || devices[0].ID != recent.ID {
		t.Errorf("discovered devices after dedupe: %+v, want only the one seen last", devices)
	}
}
