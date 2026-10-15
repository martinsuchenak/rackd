package storage

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func overlapPairs(conflicts []model.Conflict) []string {
	var pairs []string
	for _, c := range conflicts {
		ids := append([]string(nil), c.NetworkIDs...)
		sort.Strings(ids)
		pairs = append(pairs, strings.Join(ids, "|"))
	}
	sort.Strings(pairs)
	return pairs
}

// The sweep must find exactly the pairs a brute-force comparison finds.
func TestFindOverlappingSubnetsMatchesBruteForce(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	r := rand.New(rand.NewSource(1))
	var nets []*model.Network
	for i := 0; i < 120; i++ {
		var subnet string
		if i%10 == 9 {
			subnet = fmt.Sprintf("2001:db8:%x::/%d", r.Intn(4), 46+r.Intn(4))
		} else {
			subnet = fmt.Sprintf("10.%d.%d.0/%d", r.Intn(3), r.Intn(8)*32, 16+r.Intn(9))
		}
		_, ipNet, _ := net.ParseCIDR(subnet)
		n := &model.Network{Name: fmt.Sprintf("n%d", i), Subnet: ipNet.String()}
		if err := s.CreateNetwork(ctx, n); err != nil {
			t.Fatal(err)
		}
		nets = append(nets, n)
	}

	var want []string
	for i, a := range nets {
		_, na, _ := net.ParseCIDR(a.Subnet)
		for _, b := range nets[i+1:] {
			_, nb, _ := net.ParseCIDR(b.Subnet)
			if na.Contains(nb.IP) || nb.Contains(na.IP) {
				ids := []string{a.ID, b.ID}
				sort.Strings(ids)
				want = append(want, strings.Join(ids, "|"))
			}
		}
	}
	sort.Strings(want)
	if len(want) == 0 || len(want) >= MaxOverlapConflicts {
		t.Fatalf("test data gives %d overlaps; want some, under the cap", len(want))
	}

	got, err := s.FindOverlappingSubnets(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if g := overlapPairs(got); strings.Join(g, ",") != strings.Join(want, ",") {
		t.Errorf("found %d overlaps, want %d", len(g), len(want))
	}
}

// n identical subnets make n²/2 pairs; one run reports at most the cap.
func TestFindOverlappingSubnetsIsCapped(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	for i := 0; i < 60; i++ { // 1770 pairs
		if err := s.CreateNetwork(ctx, &model.Network{Name: fmt.Sprintf("n%d", i), Subnet: "10.0.0.0/24"}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.FindOverlappingSubnets(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != MaxOverlapConflicts {
		t.Errorf("got %d conflicts, want the cap of %d", len(got), MaxOverlapConflicts)
	}

	// Pairs already recorded are skipped and don't count, so the next run
	// finds the rest.
	known := map[string]bool{}
	for _, p := range overlapPairs(got) {
		known[p] = true
	}
	rest, err := s.FindOverlappingSubnets(ctx, func(ids []string) bool {
		return known[strings.Join(overlapPairs([]model.Conflict{{NetworkIDs: ids}}), "")]
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rest) != 1770-MaxOverlapConflicts {
		t.Errorf("second run found %d, want the remaining %d", len(rest), 1770-MaxOverlapConflicts)
	}
}
