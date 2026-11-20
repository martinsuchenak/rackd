package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Search input that a substring match must treat as plain text. Runs on every
// backend.
func TestSearchTreatsInputLiterally(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	for _, d := range []*model.Device{
		{Name: "Core-Router", Description: "uplink at 100% load"},
		{Name: "edge_switch", Description: "access layer"},
		{Name: "edgeXswitch", Description: "spare"},
	} {
		if err := s.CreateDevice(ctx, d); err != nil {
			t.Fatalf("CreateDevice %s: %v", d.Name, err)
		}
	}
	if err := s.CreateNetwork(ctx, &model.Network{Name: "Office LAN", Subnet: "10.20.0.0/16"}); err != nil {
		t.Fatalf("CreateNetwork: %v", err)
	}
	if err := s.CreateDatacenter(ctx, &model.Datacenter{Name: "Perth DC", Location: "Malaga"}); err != nil {
		t.Fatalf("CreateDatacenter: %v", err)
	}

	devices := func(q string) []string {
		t.Helper()
		got, err := s.SearchDevices(ctx, q)
		if err != nil {
			t.Fatalf("SearchDevices(%q): %v", q, err)
		}
		names := make([]string, len(got))
		for i, d := range got {
			names[i] = d.Name
		}
		return names
	}
	if got := devices("core-rout"); len(got) != 1 || got[0] != "Core-Router" {
		t.Errorf("core-rout = %v, want [Core-Router]", got)
	}
	if got := devices("edge_switch"); len(got) != 1 || got[0] != "edge_switch" {
		t.Errorf("edge_switch = %v, want only edge_switch", got)
	}
	if got := devices("100%"); len(got) != 1 || got[0] != "Core-Router" {
		t.Errorf("100%% = %v, want [Core-Router]", got)
	}
	if _, err := s.SearchDevices(ctx, "core\x00router"); err != nil {
		t.Errorf("query with NUL: %v", err)
	}

	nets, err := s.SearchNetworks(ctx, "office")
	if err != nil || len(nets) != 1 {
		t.Errorf("SearchNetworks(office) = %d, %v; want 1", len(nets), err)
	}
	dcs, err := s.SearchDatacenters(ctx, "malaga")
	if err != nil || len(dcs) != 1 {
		t.Errorf("SearchDatacenters(malaga) = %d, %v; want 1", len(dcs), err)
	}
}
