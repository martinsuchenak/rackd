package storage

import (
	"context"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// The reservation lifecycle the Strix run found broken: auto-assign offering
// reserved addresses, expiry never taking effect, claims never happening.

func newReservationPool(t *testing.T, s *SQLiteStorage) *model.NetworkPool {
	t.Helper()
	ctx := context.Background()
	network := &model.Network{Name: "lifecycle", Subnet: "10.0.0.0/24"}
	if err := s.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}
	pool := &model.NetworkPool{NetworkID: network.ID, Name: "lifecycle", StartIP: "10.0.0.10", EndIP: "10.0.0.20"}
	if err := s.CreateNetworkPool(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func reserve(t *testing.T, s *SQLiteStorage, poolID, ip string, expiresAt *time.Time) *model.Reservation {
	t.Helper()
	r := &model.Reservation{PoolID: poolID, IPAddress: ip, ReservedBy: "test", Status: model.ReservationStatusActive, ExpiresAt: expiresAt}
	if err := s.CreateReservation(context.Background(), r); err != nil {
		t.Fatalf("reserve %s: %v", ip, err)
	}
	return r
}

func reservationStatus(t *testing.T, s *SQLiteStorage, id string) model.ReservationStatus {
	t.Helper()
	r, err := s.GetReservation(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return r.Status
}

func TestNextAvailableIPSkipsReservedAddresses(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	pool := newReservationPool(t, s)

	reserve(t, s, pool.ID, "10.0.0.10", nil)
	future := time.Now().Add(time.Hour)
	reserve(t, s, pool.ID, "10.0.0.11", &future)
	past := time.Now().Add(-time.Hour)
	lapsed := reserve(t, s, pool.ID, "10.0.0.12", nil)
	// Expire it in place, as if its expiry passed before the sweep ran.
	lapsed.ExpiresAt = &past
	if err := s.UpdateReservation(ctx, lapsed); err != nil {
		t.Fatal(err)
	}

	ip, err := s.GetNextAvailableIP(ctx, pool.ID)
	if err != nil {
		t.Fatal(err)
	}
	if ip != "10.0.0.12" {
		t.Fatalf("next IP = %s, want 10.0.0.12 (.10 and .11 reserved, .12's reservation lapsed)", ip)
	}

	// The lapsed reservation doesn't block reserving its address either.
	reserve(t, s, pool.ID, "10.0.0.12", nil)
	if got := reservationStatus(t, s, lapsed.ID); got != model.ReservationStatusExpired {
		t.Fatalf("lapsed reservation is %s, want expired", got)
	}
}

func TestExpireReservationsComparesTimesNotText(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	pool := newReservationPool(t, s)

	// Half an hour ago, written in UTC+8: as text it sorts after "now" in
	// UTC, so a SQL string comparison would never expire it.
	perth := time.FixedZone("AWST", 8*60*60)
	lapsedAt := time.Now().Add(-30 * time.Minute).In(perth)
	lapsed := reserve(t, s, pool.ID, "10.0.0.10", &lapsedAt)
	// And one an hour ahead, written in UTC-10, which sorts before "now".
	honolulu := time.FixedZone("HST", -10*60*60)
	validUntil := time.Now().Add(time.Hour).In(honolulu)
	valid := reserve(t, s, pool.ID, "10.0.0.11", &validUntil)

	n, err := s.ExpireReservations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expired %d reservations, want 1", n)
	}
	if got := reservationStatus(t, s, lapsed.ID); got != model.ReservationStatusExpired {
		t.Errorf("lapsed reservation is %s, want expired", got)
	}
	if got := reservationStatus(t, s, valid.ID); got != model.ReservationStatusActive {
		t.Errorf("unexpired reservation is %s, want active", got)
	}
}

func TestClaimReservation(t *testing.T) {
	s := newTestStorage(t)
	defer func() { _ = s.Close() }()
	ctx := context.Background()
	pool := newReservationPool(t, s)

	r := reserve(t, s, pool.ID, "10.0.0.10", nil)
	claimed, err := s.ClaimReservation(ctx, pool.ID, "10.0.0.10")
	if err != nil || !claimed {
		t.Fatalf("ClaimReservation = %v, %v; want true", claimed, err)
	}
	if got := reservationStatus(t, s, r.ID); got != model.ReservationStatusClaimed {
		t.Fatalf("reservation is %s, want claimed", got)
	}

	// Nothing to claim: no reservation, or one already claimed.
	for _, ip := range []string{"10.0.0.11", "10.0.0.10"} {
		if claimed, err := s.ClaimReservation(ctx, pool.ID, ip); err != nil || claimed {
			t.Errorf("ClaimReservation(%s) = %v, %v; want false", ip, claimed, err)
		}
	}

	// A lapsed reservation isn't claimed.
	past := time.Now().Add(-time.Minute)
	lapsed := reserve(t, s, pool.ID, "10.0.0.12", nil)
	lapsed.ExpiresAt = &past
	if err := s.UpdateReservation(ctx, lapsed); err != nil {
		t.Fatal(err)
	}
	if claimed, err := s.ClaimReservation(ctx, pool.ID, "10.0.0.12"); err != nil || claimed {
		t.Fatalf("lapsed reservation claimed = %v, %v; want false", claimed, err)
	}
}
