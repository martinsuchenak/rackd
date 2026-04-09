package worker

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type mockAdvancedScanner struct {
	scanCount int
	networks  map[string]*model.Network
}

func (m *mockAdvancedScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
	return &model.DiscoveryScan{ID: "scan", NetworkID: network.ID, Status: model.ScanStatusCompleted, ScanType: scanType}, nil
}

func (m *mockAdvancedScanner) GetScanStatus(ctx context.Context, scanID string) (*model.DiscoveryScan, error) {
	return &model.DiscoveryScan{ID: scanID, Status: model.ScanStatusCompleted}, nil
}

func (m *mockAdvancedScanner) CancelScan(ctx context.Context, scanID string) error { return nil }

func (m *mockAdvancedScanner) GetNetwork(ctx context.Context, id string) (*model.Network, error) {
	return m.networks[id], nil
}

func (m *mockAdvancedScanner) ScanAdvanced(ctx context.Context, network *model.Network, profile *model.ScanProfile, snmpCredID, sshCredID string) (*model.DiscoveryScan, error) {
	m.scanCount++
	return &model.DiscoveryScan{ID: "advanced", NetworkID: network.ID, Status: model.ScanStatusCompleted, ScanType: profile.ScanType}, nil
}

func init() {
	log.Init("text", "error", io.Discard)
}

func TestDNSWorkerRunOnceAndStartStop(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer store.Close()

	services := service.NewServices(store, nil, nil)
	services.SetDNSService(store, nil)

	cfg := &config.Config{DNSSyncInterval: 10 * time.Millisecond}
	worker := NewDNSWorker(services.DNS, cfg)

	if err := worker.RunOnce(); err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}

	worker.Start()
	time.Sleep(25 * time.Millisecond)
	worker.Stop()
}

func TestSnapshotWorkerRunOnceAndCleanup(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	network := &model.Network{Name: "Snapshot Network", Subnet: "10.0.0.0/24"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}
	pool := &model.NetworkPool{Name: "Pool", NetworkID: network.ID, StartIP: "10.0.0.10", EndIP: "10.0.0.20"}
	if err := store.CreateNetworkPool(ctx, pool); err != nil {
		t.Fatalf("CreateNetworkPool failed: %v", err)
	}

	cfg := &config.Config{
		SnapshotInterval:      10 * time.Millisecond,
		SnapshotRetentionDays: 1,
	}
	worker := NewSnapshotWorker(store, cfg)

	if err := worker.RunOnce(); err != nil {
		t.Fatalf("RunOnce failed: %v", err)
	}
	snapshots, err := store.ListSnapshots(ctx, &model.SnapshotFilter{})
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}
	if len(snapshots) == 0 {
		t.Fatal("expected snapshots to be created")
	}

	worker.Start()
	time.Sleep(25 * time.Millisecond)
	worker.Stop()
}

func TestScheduledScanWorkerLifecycle(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	profiles, err := storage.NewSQLiteProfileStorage(store.DB())
	if err != nil {
		t.Fatalf("NewSQLiteProfileStorage failed: %v", err)
	}
	scheduledStore, err := storage.NewSQLiteScheduledScanStorage(store.DB())
	if err != nil {
		t.Fatalf("NewSQLiteScheduledScanStorage failed: %v", err)
	}

	network := &model.Network{ID: "net-scheduled", Name: "Scheduled", Subnet: "10.10.0.0/24"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}
	profile := &model.ScanProfile{
		Name:       "Scheduled Profile",
		ScanType:   "quick",
		TimeoutSec: 10,
		MaxWorkers: 5,
	}
	if err := profiles.Create(ctx, profile); err != nil {
		t.Fatalf("Create profile failed: %v", err)
	}

	scan := &model.ScheduledScan{
		NetworkID:      network.ID,
		ProfileID:      profile.ID,
		Name:           "Every hour",
		CronExpression: "0 * * * *",
		Enabled:        true,
	}
	if err := scheduledStore.Create(scan); err != nil {
		t.Fatalf("Create scheduled scan failed: %v", err)
	}

	mockScanner := &mockAdvancedScanner{
		networks: map[string]*model.Network{network.ID: network},
	}
	worker := NewScheduledScanWorker(scheduledStore, profiles, mockScanner)

	if err := worker.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer worker.Stop()

	if len(worker.jobs) == 0 {
		t.Fatal("expected scheduled cron job to be registered")
	}

	worker.runScheduledScan(scan)
	if mockScanner.scanCount != 1 {
		t.Fatalf("expected scheduled scan execution, got %d", mockScanner.scanCount)
	}

	got, err := scheduledStore.Get(scan.ID)
	if err != nil {
		t.Fatalf("Get scheduled scan failed: %v", err)
	}
	if got.LastRunAt == nil || got.NextRunAt == nil {
		t.Fatalf("expected run timestamps to be updated, got %+v", got)
	}

	worker.RemoveSchedule(scan.ID)
	if len(worker.jobs) != 0 {
		t.Fatal("expected scheduled job to be removed")
	}
}

func TestSchedulerCleanupRunsWithoutRules(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStorage failed: %v", err)
	}
	defer store.Close()

	cfg := &config.Config{
		DiscoveryInterval:      time.Hour,
		DiscoveryCleanupDays:   1,
		DiscoveryScanOnStartup: false,
	}
	scheduler := NewScheduler(store, &mockScanner{}, cfg)
	scheduler.runScheduledScans()
}

var _ discovery.AdvancedScanner = (*mockAdvancedScanner)(nil)
