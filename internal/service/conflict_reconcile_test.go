package service

import (
	"context"
	"strings"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage/storagetest"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// Recorded conflicts follow the inventory: deleting or changing the devices
// and networks involved resolves or updates them the next time conflicts are
// read, however the change was made. They used to stay listed, naming
// devices and networks that no longer existed.
func TestConflictsFollowTheInventory(t *testing.T) {
	store := storagetest.New(t)
	svc := NewConflictService(store)
	ctx := SystemContext(context.Background(), "test")

	device := func(name string, ips ...string) *model.Device {
		d := &model.Device{Name: name}
		for _, ip := range ips {
			d.Addresses = append(d.Addresses, model.Address{IP: ip, Type: "ipv4"})
		}
		if err := store.CreateDevice(ctx, d); err != nil {
			t.Fatal(err)
		}
		return d
	}
	a := device("a", "10.1.0.5")
	b := device("b", "10.1.0.5", "10.1.0.6")
	c := device("c", "10.1.0.5")
	device("twice", "10.1.0.9", "10.1.0.9") // one device listing an IP twice isn't a duplicate
	d := device("d", "10.1.0.6")

	detected, err := svc.DetectDuplicateIPs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byIP := map[string]model.Conflict{}
	for _, cf := range detected {
		byIP[cf.IPAddress] = cf
	}
	if len(detected) != 2 || len(byIP["10.1.0.5"].DeviceIDs) != 3 || len(byIP["10.1.0.6"].DeviceIDs) != 2 {
		t.Fatalf("detected %+v, want .5 on 3 devices and .6 on 2", detected)
	}
	// The response carries the recorded conflicts.
	if got, err := store.GetConflict(ctx, byIP["10.1.0.5"].ID); err != nil || got.IPAddress != "10.1.0.5" {
		t.Fatalf("detected conflict isn't the recorded one: %v %v", got, err)
	}

	// Note the .6 conflict, then delete d: nothing else holds .6.
	if err := store.UpdateConflictStatus(ctx, byIP["10.1.0.6"].ID, model.ConflictStatusIgnored, "", "known, lab gear"); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteDevice(ctx, d.ID); err != nil {
		t.Fatal(err)
	}
	// c drops off .5, which a and b still share.
	if err := store.DeleteDevice(ctx, c.ID); err != nil {
		t.Fatal(err)
	}

	open, err := svc.List(ctx, &model.ConflictFilter{Status: model.ConflictStatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0].IPAddress != "10.1.0.5" || !sameSet(open[0].DeviceIDs, a.ID, b.ID) {
		t.Fatalf("active conflicts %+v, want .5 on a and b", open)
	}
	if !strings.Contains(open[0].Description, "2 devices") {
		t.Errorf("description %q, want it to count 2 devices", open[0].Description)
	}
	gone, err := store.GetConflict(ctx, byIP["10.1.0.6"].ID)
	if err != nil {
		t.Fatal(err)
	}
	if gone.Status != model.ConflictStatusResolved || !strings.HasPrefix(gone.Notes, "known, lab gear") {
		t.Errorf("stale .6 conflict is %s with notes %q, want resolved keeping its notes", gone.Status, gone.Notes)
	}

	summary, err := svc.GetSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary["duplicate_ip"] != 1 {
		t.Errorf("summary %v, want one duplicate IP", summary)
	}
}

func TestOverlapConflictsFollowTheirNetworks(t *testing.T) {
	store := storagetest.New(t)
	svc := NewConflictService(store)
	ctx := SystemContext(context.Background(), "test")
	network := func(name, subnet string) *model.Network {
		n := &model.Network{Name: name, Subnet: subnet}
		if err := store.CreateNetwork(ctx, n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	wide := network("wide", "10.2.0.0/16")
	lab := network("lab", "10.2.5.0/24")
	office := network("office", "10.2.6.0/24")

	if _, err := svc.DetectOverlappingSubnets(ctx); err != nil {
		t.Fatal(err)
	}
	active := func() []model.Conflict {
		cs, err := svc.List(ctx, &model.ConflictFilter{Type: model.ConflictTypeOverlappingSubnet, Status: model.ConflictStatusActive})
		if err != nil {
			t.Fatal(err)
		}
		return cs
	}
	if got := active(); len(got) != 2 {
		t.Fatalf("overlaps %+v, want wide/lab and wide/office", got)
	}

	// Moving lab out of wide, and deleting office, leaves no overlap.
	lab.Subnet = "10.3.5.0/24"
	if err := store.UpdateNetwork(ctx, lab); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteNetwork(ctx, office.ID); err != nil {
		t.Fatal(err)
	}
	if got := active(); len(got) != 0 {
		t.Errorf("overlaps after the change %+v, want none", got)
	}

	// A rename that keeps the overlap updates the recorded name.
	lab.Subnet = "10.2.5.0/24"
	if err := store.UpdateNetwork(ctx, lab); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.DetectOverlappingSubnets(ctx); err != nil {
		t.Fatal(err)
	}
	wide.Name = "campus"
	if err := store.UpdateNetwork(ctx, wide); err != nil {
		t.Fatal(err)
	}
	got := active()
	if len(got) != 1 || !sameSet(got[0].NetworkNames, "campus", "lab") {
		t.Errorf("overlaps %+v, want campus/lab", got)
	}
}

func sameSet(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := map[string]int{}
	for _, g := range got {
		seen[g]++
	}
	for _, w := range want {
		if seen[w] == 0 {
			return false
		}
		seen[w]--
	}
	return true
}
