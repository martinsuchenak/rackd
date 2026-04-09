package discovery

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestUnifiedScannerGetNetworkAndScanAdvanced(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()
	ctx := context.Background()

	network := &model.Network{ID: "net-advanced", Name: "Advanced", Subnet: "127.0.0.0/30"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	got, err := scanner.GetNetwork(ctx, network.ID)
	if err != nil {
		t.Fatalf("GetNetwork failed: %v", err)
	}
	if got.ID != network.ID {
		t.Fatalf("unexpected network: %+v", got)
	}

	profile := &model.ScanProfile{
		Name:       "Profile",
		ScanType:   model.ScanTypeQuick,
		Ports:      []int{22, 443},
		TimeoutSec: 10,
		MaxWorkers: 5,
	}
	scan, err := scanner.ScanAdvanced(ctx, network, profile, "", "")
	if err != nil {
		t.Fatalf("ScanAdvanced failed: %v", err)
	}
	if scan.ScanType != model.ScanTypeQuick {
		t.Fatalf("expected scan type from profile, got %+v", scan)
	}
}

func TestUnifiedScannerCancelScanPaths(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()
	ctx := context.Background()

	if err := scanner.CancelScan(ctx, "missing"); err != ErrScanNotFound {
		t.Fatalf("expected ErrScanNotFound, got %v", err)
	}

	completed := &model.DiscoveryScan{ID: "completed", Status: model.ScanStatusCompleted}
	scanner.mu.Lock()
	scanner.scans[completed.ID] = completed
	scanner.cancelFuncs[completed.ID] = func() {}
	scanner.mu.Unlock()
	if err := scanner.CancelScan(ctx, completed.ID); err != ErrScanNotRunning {
		t.Fatalf("expected ErrScanNotRunning, got %v", err)
	}

	network := &model.Network{ID: "net-cancel", Name: "Cancel", Subnet: "192.168.1.0/24"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}
	scan := &model.DiscoveryScan{
		ID:        "pending-scan",
		NetworkID: network.ID,
		Status:    model.ScanStatusPending,
		ScanType:  model.ScanTypeQuick,
	}
	if err := store.CreateDiscoveryScan(ctx, scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}

	cancelled := false
	scanner.mu.Lock()
	scanner.scans[scan.ID] = scan
	scanner.cancelFuncs[scan.ID] = func() { cancelled = true }
	scanner.mu.Unlock()

	if err := scanner.CancelScan(ctx, scan.ID); err != nil {
		t.Fatalf("CancelScan failed: %v", err)
	}
	if !cancelled {
		t.Fatal("expected cancel function to be invoked")
	}

	got, err := store.GetDiscoveryScan(ctx, scan.ID)
	if err != nil {
		t.Fatalf("GetDiscoveryScan failed: %v", err)
	}
	if got.Status != model.ScanStatusFailed || got.ErrorMessage != "scan cancelled" || got.CompletedAt == nil {
		t.Fatalf("unexpected scan after cancel: %+v", got)
	}
}

func TestUnifiedScannerQuickNetworkScansAndHelpers(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()

	results := scanner.runNetworkScans(context.Background(), "192.168.1.0/24", model.ScanTypeQuick)
	if len(results.netbios) != 0 || len(results.mdns) != 0 || len(results.lldp) != 0 {
		t.Fatalf("expected quick scan to skip broadcast discovery, got %+v", results)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	open := scanner.scanPorts("127.0.0.1", []int{port}, 200*time.Millisecond)
	if len(open) != 1 || open[0] != port {
		t.Fatalf("expected open port %d, got %v", port, open)
	}

	opts := &ScanOptions{ScanType: model.ScanTypeQuick}
	ports := opts.getPorts()
	if len(ports) == 0 || ports[0] != 22 {
		t.Fatalf("unexpected default quick ports: %v", ports)
	}
	opts.Profile = &model.ScanProfile{Ports: []int{161, 443}}
	ports = opts.getPorts()
	if len(ports) != 2 || ports[0] != 161 {
		t.Fatalf("expected profile ports to take precedence, got %v", ports)
	}
}

func TestUnifiedScannerDiscoverHostHonorsCancelledContext(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	device := scanner.discoverHostWithOptions(ctx, "127.0.0.1", "net-1", &ScanOptions{ScanType: model.ScanTypeQuick}, 50*time.Millisecond, nil)
	if device != nil {
		t.Fatalf("expected cancelled context to return nil device, got %+v", device)
	}
}

func TestUnifiedScannerDiscoverHostWithOpenPort(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	device := scanner.discoverHostWithOptions(
		context.Background(),
		"127.0.0.1",
		"net-1",
		&ScanOptions{
			ScanType: model.ScanTypeQuick,
			Profile: &model.ScanProfile{
				Ports:      []int{port},
				TimeoutSec: 5,
				MaxWorkers: 1,
			},
		},
		200*time.Millisecond,
		nil,
	)
	if device == nil {
		t.Fatal("expected discovered device for localhost listener")
	}
	if device.IP != "127.0.0.1" || len(device.OpenPorts) != 1 || device.OpenPorts[0] != port {
		t.Fatalf("unexpected discovered device: %+v", device)
	}
}

func TestUnifiedScannerRunScanWithOptionsCompletesAndPersists(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer store.Close()
	ctx := context.Background()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	network := &model.Network{ID: "net-run", Name: "Loopback", Subnet: "127.0.0.1/32"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatalf("CreateNetwork failed: %v", err)
	}

	scan := &model.DiscoveryScan{
		ID:        "scan-run",
		NetworkID: network.ID,
		Status:    model.ScanStatusPending,
		ScanType:  model.ScanTypeQuick,
	}
	if err := store.CreateDiscoveryScan(ctx, scan); err != nil {
		t.Fatalf("CreateDiscoveryScan failed: %v", err)
	}

	_, ipNet, err := net.ParseCIDR(network.Subnet)
	if err != nil {
		t.Fatalf("ParseCIDR failed: %v", err)
	}

	scanner.runScanWithOptions(ctx, scan, network, ipNet, &ScanOptions{
		NetworkID: network.ID,
		ScanType:  model.ScanTypeQuick,
		Profile: &model.ScanProfile{
			Ports:      []int{port},
			TimeoutSec: 5,
			MaxWorkers: 1,
		},
	})

	gotScan, err := store.GetDiscoveryScan(ctx, scan.ID)
	if err != nil {
		t.Fatalf("GetDiscoveryScan failed: %v", err)
	}
	if gotScan.Status != model.ScanStatusCompleted || gotScan.CompletedAt == nil {
		t.Fatalf("expected completed persisted scan, got %+v", gotScan)
	}

	devices, err := store.ListDiscoveredDevices(ctx, network.ID)
	if err != nil {
		t.Fatalf("ListDiscoveredDevices failed: %v", err)
	}
	if len(devices) != 1 || devices[0].IP != "127.0.0.1" {
		t.Fatalf("expected one persisted discovered device, got %+v", devices)
	}
}
