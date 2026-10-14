package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestDashboardUtilizationMatchesPerNetwork(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	full := &model.Network{Name: "full", Subnet: "10.1.0.0/29"}
	empty := &model.Network{Name: "empty", Subnet: "10.2.0.0/24"}
	for _, n := range []*model.Network{full, empty} {
		if err := s.CreateNetwork(ctx, n); err != nil {
			t.Fatal(err)
		}
	}
	devices := []*model.Device{
		{Name: "a", Addresses: []model.Address{{IP: "10.1.0.1", NetworkID: full.ID}, {IP: "10.1.0.2", NetworkID: full.ID}}},
		{Name: "b", Addresses: []model.Address{{IP: "10.1.0.1", NetworkID: full.ID}}}, // same IP: counted once
		{Name: "c", Addresses: []model.Address{{IP: "192.0.2.1"}}},                    // no network
	}
	for _, d := range devices {
		if err := s.CreateDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := s.GetDashboardStats(ctx, 30, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.NetworkUtilization) != 2 {
		t.Fatalf("got %d networks, want 2", len(stats.NetworkUtilization))
	}
	for _, got := range stats.NetworkUtilization {
		want, err := s.GetNetworkUtilization(ctx, got.NetworkID)
		if err != nil {
			t.Fatal(err)
		}
		if got.TotalIPs != want.TotalIPs || got.UsedIPs != want.UsedIPs || got.Utilization != want.Utilization {
			t.Errorf("%s: dashboard %d/%d (%.2f%%), per-network %d/%d (%.2f%%)", got.NetworkName,
				got.UsedIPs, got.TotalIPs, got.Utilization, want.UsedIPs, want.TotalIPs, want.Utilization)
		}
	}
	if stats.NetworkUtilization[0].UsedIPs+stats.NetworkUtilization[1].UsedIPs != 2 {
		t.Errorf("used IPs = %+v, want 2 in total", stats.NetworkUtilization)
	}
}

// Utilization counts must read only the (network_id, ip) index, not the table.
func TestUtilizationQueriesUseCoveringIndex(t *testing.T) {
	s := newTestStorage(t)
	for _, query := range []string{
		`SELECT COUNT(DISTINCT ip) FROM addresses WHERE network_id = 'x'`,
		`SELECT network_id, COUNT(DISTINCT ip) FROM addresses WHERE network_id IS NOT NULL GROUP BY network_id`,
	} {
		rows, err := s.db.Query("EXPLAIN QUERY PLAN " + query)
		if err != nil {
			t.Fatal(err)
		}
		var plan []string
		for rows.Next() {
			var id, parent, notused int
			var detail string
			if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
				t.Fatal(err)
			}
			plan = append(plan, detail)
		}
		_ = rows.Close()
		if got := strings.Join(plan, "; "); !strings.Contains(got, "COVERING INDEX idx_addresses_network_ip") {
			t.Errorf("%s\nplan: %s", query, got)
		}
	}
}
