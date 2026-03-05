package service

import (
	"context"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type DashboardService struct {
	store storage.ExtendedStorage
}

func NewDashboardService(store storage.ExtendedStorage) *DashboardService {
	return &DashboardService{store: store}
}

// GetStats retrieves aggregated dashboard statistics
func (s *DashboardService) GetStats(ctx context.Context, staleDays, recentLimit int) (*model.DashboardStats, error) {
	if err := requirePermission(ctx, s.store, "dashboard", "read"); err != nil {
		return nil, err
	}

	if staleDays <= 0 {
		staleDays = 7 // Default to 7 days
	}
	if recentLimit <= 0 {
		recentLimit = 10 // Default to 10 recent discoveries
	}

	return s.store.GetDashboardStats(ctx, staleDays, recentLimit)
}

// GetUtilizationTrend retrieves utilization trend data for charts
func (s *DashboardService) GetUtilizationTrend(ctx context.Context, resourceType model.SnapshotType, resourceID string, days int) ([]model.UtilizationTrendPoint, error) {
	if err := requirePermission(ctx, s.store, "dashboard", "read"); err != nil {
		return nil, err
	}

	if days <= 0 {
		days = 30 // Default to 30 days
	}

	return s.store.GetUtilizationTrend(ctx, resourceType, resourceID, days)
}
