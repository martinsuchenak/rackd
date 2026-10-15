package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type ScheduledScanService struct {
	store storage.ScheduledScanStorage
	rbac  auditedPermissionStore
}

func NewScheduledScanService(store storage.ScheduledScanStorage, rbac auditedPermissionStore) *ScheduledScanService {
	return &ScheduledScanService{store: store, rbac: rbac}
}

func (s *ScheduledScanService) List(ctx context.Context, networkID string) ([]model.ScheduledScan, error) {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "list"); err != nil {
		return nil, err
	}

	return s.store.List(networkID)
}

func (s *ScheduledScanService) Create(ctx context.Context, scan *model.ScheduledScan) error {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "create"); err != nil {
		return err
	}
	if errs := validateScheduledScan(scan); len(errs) > 0 {
		return errs
	}

	if err := s.store.Create(scan); err != nil {
		return fromStoreError(err)
	}

	recordAudit(ctx, s.rbac, "create", "scheduled_scan", scan.ID, scan)
	return nil
}

func (s *ScheduledScanService) Get(ctx context.Context, id string) (*model.ScheduledScan, error) {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "read"); err != nil {
		return nil, err
	}

	scan, err := s.store.Get(id)
	if errors.Is(err, storage.ErrScheduledScanNotFound) {
		return nil, ErrNotFound
	}
	return scan, err
}

func (s *ScheduledScanService) Update(ctx context.Context, id string, scan *model.ScheduledScan) error {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "update"); err != nil {
		return err
	}

	scan.ID = id
	if errs := validateScheduledScan(scan); len(errs) > 0 {
		return errs
	}
	if err := s.store.Update(scan); err != nil {
		if errors.Is(err, storage.ErrScheduledScanNotFound) {
			return ErrNotFound
		}
		return fromStoreError(err)
	}

	recordAudit(ctx, s.rbac, "update", "scheduled_scan", id, scan)
	return nil
}

func (s *ScheduledScanService) Delete(ctx context.Context, id string) error {
	if err := requirePermission(ctx, s.rbac, "scheduled-scans", "delete"); err != nil {
		return err
	}

	err := s.store.Delete(id)
	if errors.Is(err, storage.ErrScheduledScanNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	recordAudit(ctx, s.rbac, "delete", "scheduled_scan", id, nil)
	return nil
}

// validateScheduledScan checks a schedule before it is stored. The cron
// expression is parsed as the scheduler parses it, so a bad one is refused
// here rather than failing silently when the worker loads it.
func validateScheduledScan(scan *model.ScheduledScan) ValidationErrors {
	var errs ValidationErrors
	if strings.TrimSpace(scan.Name) == "" {
		errs = append(errs, ValidationError{Field: "name", Message: "name is required"})
	}
	if scan.NetworkID == "" {
		errs = append(errs, ValidationError{Field: "network_id", Message: "network_id is required"})
	}
	if schedule, err := cron.ParseStandard(scan.CronExpression); err != nil {
		errs = append(errs, ValidationError{Field: "cron_expression", Message: "invalid cron expression: " + err.Error()})
	} else if gap := shortestGap(schedule); gap < minScanInterval {
		errs = append(errs, ValidationError{Field: "cron_expression", Message: fmt.Sprintf("runs every %s; scheduled scans must be at least %s apart", gap, minScanInterval)})
	}
	return errs
}

// minScanInterval is the shortest time allowed between runs of a schedule: a
// scan can probe up to 65,536 hosts, and only a few run at once.
const minScanInterval = 5 * time.Minute

// shortestGap is the shortest time between a schedule's next several runs, so
// irregular expressions like "0,1 * * * *" are caught too.
func shortestGap(schedule cron.Schedule) time.Duration {
	shortest := time.Duration(1<<63 - 1)
	prev := schedule.Next(time.Now())
	for i := 0; i < 50; i++ {
		next := schedule.Next(prev)
		if next.IsZero() {
			break
		}
		if gap := next.Sub(prev); gap < shortest {
			shortest = gap
		}
		prev = next
	}
	return shortest
}
