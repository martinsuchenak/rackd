package worker

import (
	"context"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type Scheduler struct {
	storage storage.ExtendedStorage
	scanner discovery.Scanner
	config  *config.Config
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

func NewScheduler(store storage.ExtendedStorage, scanner discovery.Scanner, cfg *config.Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		storage: store,
		scanner: scanner,
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run()

	log.Info("Discovery scheduler started", "interval", s.config.DiscoveryInterval)
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.cancel()
	s.wg.Wait()
	log.Info("Discovery scheduler stopped")
}

func (s *Scheduler) run() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.DiscoveryInterval)
	defer ticker.Stop()

	if s.config.DiscoveryScanOnStartup {
		s.runScheduledScans()
	}

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.runScheduledScans()
		}
	}
}

func (s *Scheduler) runScheduledScans() {
	log.Debug("Running scheduled discovery scans")
	
	rules, err := s.storage.ListDiscoveryRules()
	if err != nil {
		log.Error("Failed to list discovery rules", "error", err)
		return
	}

	log.Debug("Found discovery rules", "count", len(rules))

	for _, rule := range rules {
		if !rule.Enabled {
			log.Trace("Skipping disabled rule", "network_id", rule.NetworkID)
			continue
		}

		network, err := s.storage.GetNetwork(rule.NetworkID)
		if err != nil {
			log.Error("Failed to get network for discovery", "network_id", rule.NetworkID, "error", err)
			continue
		}

		log.Info("Starting scheduled discovery scan", "network", network.Name, "subnet", network.Subnet, "scan_type", rule.ScanType)

		_, err = s.scanner.Scan(s.ctx, network, rule.ScanType)
		if err != nil {
			log.Error("Scheduled scan failed", "network", network.Name, "error", err)
		} else {
			log.Info("Scheduled scan completed", "network", network.Name)
		}
	}

	if s.config.DiscoveryCleanupDays > 0 {
		log.Debug("Cleaning up old discoveries", "days", s.config.DiscoveryCleanupDays)
		if err := s.storage.CleanupOldDiscoveries(s.config.DiscoveryCleanupDays); err != nil {
			log.Error("Failed to cleanup old discoveries", "error", err)
		}
	}
}
