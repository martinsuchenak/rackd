package worker

import (
	"context"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// SnapshotWorker periodically captures utilization snapshots
type SnapshotWorker struct {
	storage storage.ExtendedStorage
	config  *config.Config
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

// NewSnapshotWorker creates a new snapshot worker
func NewSnapshotWorker(store storage.ExtendedStorage, cfg *config.Config) *SnapshotWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &SnapshotWorker{
		storage: store,
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start begins the snapshot worker
func (w *SnapshotWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	log.Info("Snapshot worker started", "interval", w.config.SnapshotInterval)
}

// Stop halts the snapshot worker
func (w *SnapshotWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	w.mu.Unlock()

	w.cancel()
	w.wg.Wait()
	log.Info("Snapshot worker stopped")
}

// RunOnce triggers an immediate snapshot (useful for testing or manual triggers)
func (w *SnapshotWorker) RunOnce() error {
	return w.takeSnapshots()
}

func (w *SnapshotWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.SnapshotInterval)
	defer ticker.Stop()

	// Take initial snapshot on startup
	if err := w.takeSnapshots(); err != nil {
		log.Error("Failed to take initial snapshots", "error", err)
	}

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.takeSnapshots(); err != nil {
				log.Error("Failed to take snapshots", "error", err)
			}
			w.cleanupOldSnapshots()
		}
	}
}

func (w *SnapshotWorker) takeSnapshots() error {
	log.Debug("Taking utilization snapshots")
	now := time.Now().UTC()

	// Snapshot all networks
	networks, err := w.storage.ListNetworks(w.ctx, nil)
	if err != nil {
		log.Error("Failed to list networks for snapshot", "error", err)
		return err
	}

	for _, network := range networks {
		util, err := w.storage.GetNetworkUtilization(w.ctx, network.ID)
		if err != nil {
			log.Error("Failed to get network utilization", "network_id", network.ID, "error", err)
			continue
		}

		snapshot := &model.UtilizationSnapshot{
			Type:         model.SnapshotTypeNetwork,
			ResourceID:   network.ID,
			ResourceName: network.Name,
			TotalIPs:     util.TotalIPs,
			UsedIPs:      util.UsedIPs,
			Utilization:  util.Utilization,
			Timestamp:    now,
		}

		if err := w.storage.CreateSnapshot(w.ctx, snapshot); err != nil {
			log.Error("Failed to create network snapshot", "network_id", network.ID, "error", err)
		}
	}

	// Snapshot all pools
	pools, err := w.storage.ListNetworkPools(w.ctx, nil)
	if err != nil {
		log.Error("Failed to list pools for snapshot", "error", err)
		return err
	}
	// The previous snapshots say which pools were already over the alert
	// threshold, so the event fires once per crossing, across restarts too.
	previous := map[string]float64{}
	if latest, err := w.storage.GetLatestSnapshots(w.ctx, model.SnapshotTypePool); err == nil {
		for _, s := range latest {
			previous[s.ResourceID] = s.Utilization
		}
	} else {
		log.Warn("Failed to read previous pool snapshots", "error", err)
	}

	for _, pool := range pools {
		heatmap, err := w.storage.GetPoolHeatmap(w.ctx, pool.ID)
		if err != nil {
			log.Error("Failed to get pool heatmap", "pool_id", pool.ID, "error", err)
			continue
		}

		usedIPs := 0
		for _, ip := range heatmap {
			if ip.Status != "available" {
				usedIPs++
			}
		}

		totalIPs := len(heatmap)
		var utilization float64
		if totalIPs > 0 {
			utilization = float64(usedIPs) / float64(totalIPs) * 100
		}

		snapshot := &model.UtilizationSnapshot{
			Type:         model.SnapshotTypePool,
			ResourceID:   pool.ID,
			ResourceName: pool.Name,
			TotalIPs:     totalIPs,
			UsedIPs:      usedIPs,
			Utilization:  utilization,
			Timestamp:    now,
		}

		if err := w.storage.CreateSnapshot(w.ctx, snapshot); err != nil {
			log.Error("Failed to create pool snapshot", "pool_id", pool.ID, "error", err)
		}

		threshold := w.config.PoolUtilizationAlertPercent
		if threshold > 0 && totalIPs > 0 && utilization >= threshold {
			if before, seen := previous[pool.ID]; !seen || before < threshold {
				webhook.Publish(model.EventTypePoolUtilization, map[string]any{
					"pool_id":     pool.ID,
					"pool_name":   pool.Name,
					"network_id":  pool.NetworkID,
					"used_ips":    usedIPs,
					"total_ips":   totalIPs,
					"utilization": utilization,
					"threshold":   threshold,
				})
			}
		}
	}

	log.Info("Snapshots completed", "networks", len(networks), "pools", len(pools))
	return nil
}

func (w *SnapshotWorker) cleanupOldSnapshots() {
	if w.config.SnapshotRetentionDays > 0 {
		if err := w.storage.DeleteOldSnapshots(w.ctx, w.config.SnapshotRetentionDays); err != nil {
			log.Error("Failed to cleanup old snapshots", "error", err)
		}
	}
}
