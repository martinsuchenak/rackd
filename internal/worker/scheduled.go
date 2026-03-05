package worker

import (
	"context"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/audit"
	"github.com/martinsuchenak/rackd/internal/discovery"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/robfig/cron/v3"
)

type ScheduledScanWorker struct {
	scheduledStore   storage.ScheduledScanStorage
	profileStore     storage.ProfileStorage
	discoveryService discovery.AdvancedScanner
	cron             *cron.Cron
	jobs             map[string]cron.EntryID
	mu               sync.Mutex
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewScheduledScanWorker(
	scheduledStore storage.ScheduledScanStorage,
	profileStore storage.ProfileStorage,
	discoveryService discovery.AdvancedScanner,
) *ScheduledScanWorker {
	// Create base context with audit info for scheduler operations
	baseCtx := context.Background()
	auditCtx := audit.WithContext(baseCtx, &audit.Context{
		Source: "scheduler",
	})
	ctx, cancel := context.WithCancel(auditCtx)
	return &ScheduledScanWorker{
		scheduledStore:   scheduledStore,
		profileStore:     profileStore,
		discoveryService: discoveryService,
		cron:             cron.New(),
		jobs:             make(map[string]cron.EntryID),
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (w *ScheduledScanWorker) Start() error {
	scans, err := w.scheduledStore.List("")
	if err != nil {
		return err
	}

	for _, scan := range scans {
		if scan.Enabled {
			w.scheduleJob(&scan)
		}
	}

	w.cron.Start()
	return nil
}

func (w *ScheduledScanWorker) Stop() {
	w.cancel()
	ctx := w.cron.Stop()
	<-ctx.Done()
}

func (w *ScheduledScanWorker) AddSchedule(scan *model.ScheduledScan) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if existingID, ok := w.jobs[scan.ID]; ok {
		w.cron.Remove(existingID)
	}

	if scan.Enabled {
		return w.scheduleJob(scan)
	}
	return nil
}

func (w *ScheduledScanWorker) RemoveSchedule(scanID string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if entryID, ok := w.jobs[scanID]; ok {
		w.cron.Remove(entryID)
		delete(w.jobs, scanID)
	}
}

func (w *ScheduledScanWorker) scheduleJob(scan *model.ScheduledScan) error {
	scanCopy := *scan
	entryID, err := w.cron.AddFunc(scan.CronExpression, func() {
		w.runScheduledScan(&scanCopy)
	})
	if err != nil {
		return err
	}

	w.jobs[scan.ID] = entryID

	entry := w.cron.Entry(entryID)
	nextRun := entry.Next
	scanCopy.NextRunAt = &nextRun
	w.scheduledStore.Update(&scanCopy)

	return nil
}

func (w *ScheduledScanWorker) runScheduledScan(scan *model.ScheduledScan) {
	network, err := w.discoveryService.GetNetwork(w.ctx, scan.NetworkID)
	if err != nil {
		return
	}

	profile, err := w.profileStore.Get(w.ctx, scan.ProfileID)
	if err != nil {
		return
	}

	_, err = w.discoveryService.ScanAdvanced(w.ctx, network, profile, "", "")
	if err != nil {
		return
	}

	now := time.Now()
	scan.LastRunAt = &now

	w.mu.Lock()
	if entryID, ok := w.jobs[scan.ID]; ok {
		entry := w.cron.Entry(entryID)
		nextRun := entry.Next
		scan.NextRunAt = &nextRun
	}
	w.mu.Unlock()

	w.scheduledStore.Update(scan)
}
