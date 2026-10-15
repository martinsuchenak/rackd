package worker

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestReservationExpiryWorkerExpiresOnStart(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := context.Background()
	network := &model.Network{Name: "n", Subnet: "10.0.0.0/24"}
	if err := store.CreateNetwork(ctx, network); err != nil {
		t.Fatal(err)
	}
	pool := &model.NetworkPool{NetworkID: network.ID, Name: "p", StartIP: "10.0.0.10", EndIP: "10.0.0.20"}
	if err := store.CreateNetworkPool(ctx, pool); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	r := &model.Reservation{PoolID: pool.ID, IPAddress: "10.0.0.10", ReservedBy: "t", Status: model.ReservationStatusActive, ExpiresAt: &past}
	if err := store.CreateReservation(ctx, r); err != nil {
		t.Fatal(err)
	}

	w := NewReservationExpiryWorker(store)
	w.Start()
	defer w.Stop()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got, err := store.GetReservation(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status == model.ReservationStatusExpired {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("reservation still %s after the worker started", got.Status)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
