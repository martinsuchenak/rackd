package discovery

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// slowScanStore counts scan progress writes and how many run at once, with
// each one slow, as on network storage.
type slowScanStore struct {
	storage.DiscoveryStorage
	updates, inFlight, maxInFlight atomic.Int32
}

func (s *slowScanStore) UpdateDiscoveryScan(ctx context.Context, scan *model.DiscoveryScan) error {
	n := s.inFlight.Add(1)
	defer s.inFlight.Add(-1)
	for {
		m := s.maxInFlight.Load()
		if n <= m || s.maxInFlight.CompareAndSwap(m, n) {
			break
		}
	}
	s.updates.Add(1)
	time.Sleep(20 * time.Millisecond)
	return s.DiscoveryStorage.UpdateDiscoveryScan(ctx, scan)
}

// A scan wrote progress after every host, from every worker at once, which
// took the whole connection pool and starved the API (found in production on
// NFS). Progress is now written by one goroutine, every few seconds.
func TestScanWritesProgressFromOneGoroutine(t *testing.T) {
	scanner, store := newTestUnifiedScanner(t)
	defer func() { _ = store.Close() }()
	slow := &slowScanStore{DiscoveryStorage: store}
	scanner.storage = slow
	ctx := context.Background()

	network := &model.Network{ID: "net-pressure", Name: "Pressure", Subnet: "192.0.2.0/26"} // TEST-NET-1: nothing answers
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}
	scan := &model.DiscoveryScan{ID: "pressure", NetworkID: network.ID, Status: model.ScanStatusPending, ScanType: model.ScanTypeQuick}
	if err := store.CreateDiscoveryScan(ctx, scan); err != nil {
		t.Fatal(err)
	}
	_, ipNet, _ := net.ParseCIDR(network.Subnet)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner.runScanWithOptions(ctx, scan, network, ipNet, &ScanOptions{ScanType: model.ScanTypeQuick})
	}()
	wg.Wait()

	if scan.Status != model.ScanStatusCompleted || scan.ScannedHosts != scan.TotalHosts {
		t.Fatalf("scan ended %s with %d of %d hosts", scan.Status, scan.ScannedHosts, scan.TotalHosts)
	}
	if got := slow.maxInFlight.Load(); got > 1 {
		t.Errorf("%d progress writes ran at once, want 1", got)
	}
	// Start, a few ticks, and the end; not one per host.
	if got := int(slow.updates.Load()); got >= scan.TotalHosts/2 {
		t.Errorf("%d progress writes for %d hosts, want a handful", got, scan.TotalHosts)
	}
}
