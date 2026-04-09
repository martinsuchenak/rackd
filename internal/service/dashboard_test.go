package service

import (
	"testing"

	"github.com/martinsuchenak/rackd/internal/model"
)

func TestDashboardService_DefaultsStatsAndTrendParameters(t *testing.T) {
	store := newServiceTestStorage()
	store.setPermission("user-1", "dashboard", "read", true)
	svc := NewDashboardService(store)

	if _, err := svc.GetStats(userContext("user-1"), 0, 0); err != nil {
		t.Fatalf("GetStats returned unexpected error: %v", err)
	}
	if store.dashboardStaleDays != 7 || store.dashboardRecentLimit != 10 {
		t.Fatalf("expected default dashboard params 7/10, got %d/%d", store.dashboardStaleDays, store.dashboardRecentLimit)
	}

	if _, err := svc.GetUtilizationTrend(userContext("user-1"), model.SnapshotTypeNetwork, "net-1", 0); err != nil {
		t.Fatalf("GetUtilizationTrend returned unexpected error: %v", err)
	}
	if store.utilTrendDays != 30 {
		t.Fatalf("expected default trend days 30, got %d", store.utilTrendDays)
	}
}
