package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestListsReturnEveryRowUnlessAPageIsAsked(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	const n = 150 // more than the default page of 100
	for i := range n {
		if err := s.CreateNetwork(ctx, &model.Network{Name: fmt.Sprintf("net-%03d", i), Subnet: fmt.Sprintf("10.%d.%d.0/24", i/250, i%250)}); err != nil {
			t.Fatal(err)
		}
	}
	count := func(filter *model.NetworkFilter) int {
		t.Helper()
		networks, err := s.ListNetworks(ctx, filter)
		if err != nil {
			t.Fatal(err)
		}
		return len(networks)
	}
	for name, tc := range map[string]struct {
		filter *model.NetworkFilter
		want   int
	}{
		"nil filter":          {nil, n},
		"filter with no page": {&model.NetworkFilter{}, n},
		"limit":               {&model.NetworkFilter{Pagination: model.Pagination{Limit: 10}}, 10},
		"limit and offset":    {&model.NetworkFilter{Pagination: model.Pagination{Limit: 10, Offset: 145}}, 5},
		"offset alone":        {&model.NetworkFilter{Pagination: model.Pagination{Offset: 20}}, model.DefaultPageSize},
		"limit above the cap": {&model.NetworkFilter{Pagination: model.Pagination{Limit: 5000}}, n},
		"negative limit":      {&model.NetworkFilter{Pagination: model.Pagination{Limit: -1}}, n},
	} {
		if got := count(tc.filter); got != tc.want {
			t.Errorf("%s: got %d networks, want %d", name, got, tc.want)
		}
	}
}

func TestCountInventory(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	devices := make([]*model.Device, 120)
	for i := range devices {
		devices[i] = &model.Device{Name: fmt.Sprintf("d-%03d", i)}
	}
	if _, err := s.BulkCreateDevices(ctx, devices); err != nil {
		t.Fatal(err)
	}
	c, err := s.CountInventory(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dcs, _ := s.ListDatacenters(ctx, nil)
	if c.Devices != 120 || c.Networks != 0 || c.Datacenters != int64(len(dcs)) {
		t.Errorf("counts = %+v", c)
	}
}
