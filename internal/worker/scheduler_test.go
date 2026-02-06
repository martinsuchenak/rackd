package worker

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func init() {
	log.Init("text", "error", io.Discard)
}

type mockScanner struct {
	scanCalled bool
	scanCount  int
}

func (m *mockScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
	m.scanCalled = true
	m.scanCount++
	return &model.DiscoveryScan{
		ID:        "scan-1",
		NetworkID: network.ID,
		Status:    model.ScanStatusCompleted,
		ScanType:  scanType,
	}, nil
}

func (m *mockScanner) GetScanStatus(scanID string) (*model.DiscoveryScan, error) {
	return &model.DiscoveryScan{ID: scanID, Status: model.ScanStatusCompleted}, nil
}

func (m *mockScanner) CancelScan(scanID string) error {
	return nil
}

func newTestScheduler(t *testing.T) (*Scheduler, storage.ExtendedStorage, *mockScanner) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	cfg := &config.Config{
		DiscoveryInterval:      100 * time.Millisecond,
		DiscoveryCleanupDays:   30,
		DiscoveryScanOnStartup: false,
	}

	scanner := &mockScanner{}
	scheduler := NewScheduler(store, scanner, cfg)

	return scheduler, store, scanner
}

func TestNewScheduler(t *testing.T) {
	scheduler, store, _ := newTestScheduler(t)
	defer store.Close()

	if scheduler == nil {
		t.Fatal("Expected scheduler to be created")
	}
	if scheduler.storage == nil {
		t.Error("Expected storage to be set")
	}
	if scheduler.scanner == nil {
		t.Error("Expected scanner to be set")
	}
	if scheduler.config == nil {
		t.Error("Expected config to be set")
	}
	if scheduler.ctx == nil {
		t.Error("Expected context to be set")
	}
	if scheduler.cancel == nil {
		t.Error("Expected cancel func to be set")
	}
}

func TestScheduler_StartStop(t *testing.T) {
	scheduler, store, _ := newTestScheduler(t)
	defer store.Close()

	scheduler.Start()

	scheduler.mu.Lock()
	running := scheduler.running
	scheduler.mu.Unlock()

	if !running {
		t.Error("Expected scheduler to be running")
	}

	scheduler.Stop()

	scheduler.mu.Lock()
	running = scheduler.running
	scheduler.mu.Unlock()

	if running {
		t.Error("Expected scheduler to be stopped")
	}
}

func TestScheduler_DoubleStart(t *testing.T) {
	scheduler, store, _ := newTestScheduler(t)
	defer store.Close()

	scheduler.Start()
	scheduler.Start() // Should be no-op

	scheduler.mu.Lock()
	running := scheduler.running
	scheduler.mu.Unlock()

	if !running {
		t.Error("Expected scheduler to be running")
	}

	scheduler.Stop()
}

func TestScheduler_DoubleStop(t *testing.T) {
	scheduler, store, _ := newTestScheduler(t)
	defer store.Close()

	scheduler.Start()
	scheduler.Stop()
	scheduler.Stop() // Should be no-op

	scheduler.mu.Lock()
	running := scheduler.running
	scheduler.mu.Unlock()

	if running {
		t.Error("Expected scheduler to be stopped")
	}
}

func TestScheduler_RunsScheduledScans(t *testing.T) {
	scheduler, store, scanner := newTestScheduler(t)
	defer store.Close()
	ctx := context.Background()

	// Create a network and rule
	network := &model.Network{
		ID:     "net-1",
		Name:   "Test Network",
		Subnet: "192.168.1.0/24",
	}
	store.CreateNetwork(ctx, network)

	rule := &model.DiscoveryRule{
		ID:            "rule-1",
		NetworkID:     "net-1",
		Enabled:       true,
		ScanType:      model.ScanTypeQuick,
		IntervalHours: 24,
	}
	store.SaveDiscoveryRule(ctx, rule)

	scheduler.runScheduledScans()

	if !scanner.scanCalled {
		t.Error("Expected scan to be called")
	}
}

func TestScheduler_SkipsDisabledRules(t *testing.T) {
	scheduler, store, scanner := newTestScheduler(t)
	defer store.Close()
	ctx := context.Background()

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test Network",
		Subnet: "192.168.1.0/24",
	}
	store.CreateNetwork(ctx, network)

	rule := &model.DiscoveryRule{
		ID:            "rule-1",
		NetworkID:     "net-1",
		Enabled:       false, // Disabled
		ScanType:      model.ScanTypeQuick,
		IntervalHours: 24,
	}
	store.SaveDiscoveryRule(ctx, rule)

	scheduler.runScheduledScans()

	if scanner.scanCalled {
		t.Error("Expected scan NOT to be called for disabled rule")
	}
}

func TestScheduler_HandlesNetworkNotFound(t *testing.T) {
	scheduler, store, scanner := newTestScheduler(t)
	defer store.Close()
	ctx := context.Background()

	// Rule without corresponding network
	rule := &model.DiscoveryRule{
		ID:            "rule-1",
		NetworkID:     "nonexistent",
		Enabled:       true,
		ScanType:      model.ScanTypeQuick,
		IntervalHours: 24,
	}
	store.SaveDiscoveryRule(ctx, rule)

	// Should not panic
	scheduler.runScheduledScans()

	if scanner.scanCalled {
		t.Error("Expected scan NOT to be called when network not found")
	}
}

func TestScheduler_ScanOnStartup(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	cfg := &config.Config{
		DiscoveryInterval:      1 * time.Hour, // Long interval
		DiscoveryCleanupDays:   30,
		DiscoveryScanOnStartup: true,
	}

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test Network",
		Subnet: "192.168.1.0/24",
	}
	store.CreateNetwork(ctx, network)

	rule := &model.DiscoveryRule{
		ID:            "rule-1",
		NetworkID:     "net-1",
		Enabled:       true,
		ScanType:      model.ScanTypeQuick,
		IntervalHours: 24,
	}
	store.SaveDiscoveryRule(ctx, rule)

	scanner := &mockScanner{}
	scheduler := NewScheduler(store, scanner, cfg)

	scheduler.Start()
	time.Sleep(50 * time.Millisecond) // Give time for startup scan
	scheduler.Stop()

	if !scanner.scanCalled {
		t.Error("Expected scan to be called on startup")
	}
}

func TestScheduler_TickerTriggersScans(t *testing.T) {
	store, err := storage.NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()
	ctx := context.Background()

	cfg := &config.Config{
		DiscoveryInterval:      50 * time.Millisecond, // Short interval for testing
		DiscoveryCleanupDays:   30,
		DiscoveryScanOnStartup: false,
	}

	network := &model.Network{
		ID:     "net-1",
		Name:   "Test Network",
		Subnet: "192.168.1.0/24",
	}
	store.CreateNetwork(ctx, network)

	rule := &model.DiscoveryRule{
		ID:            "rule-1",
		NetworkID:     "net-1",
		Enabled:       true,
		ScanType:      model.ScanTypeQuick,
		IntervalHours: 24,
	}
	store.SaveDiscoveryRule(ctx, rule)

	scanner := &mockScanner{}
	scheduler := NewScheduler(store, scanner, cfg)

	scheduler.Start()
	time.Sleep(150 * time.Millisecond) // Wait for at least 2 ticks
	scheduler.Stop()

	if scanner.scanCount < 2 {
		t.Errorf("Expected at least 2 scans, got %d", scanner.scanCount)
	}
}
