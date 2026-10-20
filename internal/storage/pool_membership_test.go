package storage

import (
	"context"
	"errors"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// An address counts toward a pool when its IP is in the pool's range and it
// is bound to the pool, or bound to none and on the pool's network or on no
// network. The web UI's device form sets neither, and those addresses used
// to be invisible to reservations, next-IP and the heatmap (found by a Strix
// run: the same IP ended up both reserved and assigned).
func TestUnboundAddressesCountTowardTheirPool(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	pool := newReservationPool(t, s) // 10.0.0.10-10.0.0.20
	other := &model.Network{Name: "elsewhere", Subnet: "10.0.0.0/24"}
	if err := s.CreateNetwork(ctx, other); err != nil {
		t.Fatal(err)
	}

	device := &model.Device{Name: "host", Addresses: []model.Address{
		{IP: "10.0.0.10", Type: "ipv4"},                            // no pool, no network
		{IP: "10.0.0.11", Type: "ipv4", NetworkID: pool.NetworkID}, // the pool's network
		{IP: "10.0.0.12", Type: "ipv4", NetworkID: other.ID},       // another network
	}}
	if err := s.CreateDevice(ctx, device); err != nil {
		t.Fatal(err)
	}

	for ip, inUse := range map[string]bool{"10.0.0.10": true, "10.0.0.11": true, "10.0.0.12": false} {
		err := s.CreateReservation(ctx, &model.Reservation{PoolID: pool.ID, IPAddress: ip, ReservedBy: "test", Status: model.ReservationStatusActive})
		if inUse && !errors.Is(err, ErrIPConflict) {
			t.Errorf("reserve %s: %v, want ErrIPConflict", ip, err)
		}
		if !inUse && err != nil {
			t.Errorf("reserve %s: %v, want it reserved", ip, err)
		}
	}

	// .10 and .11 are in use and .12 is now reserved, so next is .13.
	if ip, err := s.GetNextAvailableIP(ctx, pool.ID); err != nil || ip != "10.0.0.13" {
		t.Errorf("next IP = %s, %v; want 10.0.0.13", ip, err)
	}

	heatmap, err := s.GetPoolHeatmap(ctx, pool.ID)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, h := range heatmap {
		status[h.IP] = h.Status
	}
	if status["10.0.0.10"] != "used" || status["10.0.0.11"] != "used" || status["10.0.0.12"] != "reserved" {
		t.Errorf("heatmap .10=%s .11=%s .12=%s; want used, used, reserved", status["10.0.0.10"], status["10.0.0.11"], status["10.0.0.12"])
	}
}

// A device taking a reserved address claims the reservation, whether or not
// the address is bound to the pool.
func TestClaimReservationForUnboundAddresses(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	pool := newReservationPool(t, s)
	other := &model.Network{Name: "elsewhere", Subnet: "10.0.0.0/24"}
	if err := s.CreateNetwork(ctx, other); err != nil {
		t.Fatal(err)
	}
	onNetwork := reserve(t, s, pool.ID, "10.0.0.14", nil)
	noNetwork := reserve(t, s, pool.ID, "10.0.0.15", nil)
	elsewhere := reserve(t, s, pool.ID, "10.0.0.16", nil)

	for _, c := range []struct {
		addr  model.Address
		want  bool
		res   *model.Reservation
		after model.ReservationStatus
	}{
		{model.Address{IP: "10.0.0.14", NetworkID: pool.NetworkID}, true, onNetwork, model.ReservationStatusClaimed},
		{model.Address{IP: "10.0.0.15"}, true, noNetwork, model.ReservationStatusClaimed},
		{model.Address{IP: "10.0.0.16", NetworkID: other.ID}, false, elsewhere, model.ReservationStatusActive},
	} {
		claimed, err := s.ClaimReservation(ctx, c.addr)
		if err != nil || claimed != c.want {
			t.Errorf("claim %+v = %v, %v; want %v", c.addr, claimed, err, c.want)
		}
		if got := reservationStatus(t, s, c.res.ID); got != c.after {
			t.Errorf("reservation for %s is %s, want %s", c.addr.IP, got, c.after)
		}
	}

	// With no network, two pools reserving the same IP on different networks
	// are ambiguous, so neither is claimed.
	otherPool := &model.NetworkPool{NetworkID: other.ID, Name: "elsewhere", StartIP: "10.0.0.10", EndIP: "10.0.0.20"}
	if err := s.CreateNetworkPool(ctx, otherPool); err != nil {
		t.Fatal(err)
	}
	a := reserve(t, s, pool.ID, "10.0.0.17", nil)
	b := reserve(t, s, otherPool.ID, "10.0.0.17", nil)
	if claimed, err := s.ClaimReservation(ctx, model.Address{IP: "10.0.0.17"}); err != nil || claimed {
		t.Errorf("ambiguous claim = %v, %v; want false", claimed, err)
	}
	if reservationStatus(t, s, a.ID) != model.ReservationStatusActive || reservationStatus(t, s, b.ID) != model.ReservationStatusActive {
		t.Error("an ambiguous claim changed a reservation")
	}
}
