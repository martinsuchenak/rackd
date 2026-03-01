package worker

import (
	"context"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/service"
)

// DNSWorker periodically syncs DNS zones with auto_sync enabled
type DNSWorker struct {
	dns     *service.DNSService
	config  *config.Config
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// NewDNSWorker creates a new DNS sync worker
func NewDNSWorker(dnsSvc *service.DNSService, cfg *config.Config) *DNSWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DNSWorker{
		dns:    dnsSvc,
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins the DNS sync worker
func (w *DNSWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	log.Info("DNS sync worker started", "interval", w.config.DNSSyncInterval)
}

// Stop halts the DNS sync worker
func (w *DNSWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.cancel()
	w.wg.Wait()
	log.Info("DNS sync worker stopped")
}

// RunOnce triggers an immediate sync (useful for testing or manual triggers)
func (w *DNSWorker) RunOnce() error {
	return w.syncZones()
}

func (w *DNSWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.DNSSyncInterval)
	defer ticker.Stop()

	// Run initial sync on startup
	if err := w.syncZones(); err != nil {
		log.Error("Failed to take initial DNS sync", "error", err)
	}

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.syncZones(); err != nil {
				log.Error("Failed to sync DNS zones", "error", err)
			}
		}
	}
}

func (w *DNSWorker) syncZones() error {
	log.Debug("Syncing DNS zones with auto_sync enabled")

	// Create system context to bypass RBAC for internal operations
	sysCtx := service.SystemContext(w.ctx, "dns-worker")

	// List all zones
	zones, err := w.dns.ListZones(sysCtx, nil)
	if err != nil {
		log.Error("Failed to list DNS zones for sync", "error", err)
		return err
	}

	synced := 0
	failed := 0

	for _, zone := range zones {
		if zone.AutoSync {
			log.Info("Syncing DNS zone", "zone", zone.Name, "zone_id", zone.ID)
			if _, err := w.dns.SyncZone(sysCtx, zone.ID); err != nil {
				log.Error("Failed to sync DNS zone", "zone", zone.Name, "error", err)
				failed++
			} else {
				synced++
			}
		}
	}

	if synced > 0 || failed > 0 {
		log.Info("DNS sync completed", "zones_synced", synced, "zones_failed", failed)
	}

	return nil
}
