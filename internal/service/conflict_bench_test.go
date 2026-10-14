package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// BenchmarkCreateDeviceWithAddresses creates a 10-address device in an
// inventory of 10,000 devices, which runs the duplicate-IP check per address.
func BenchmarkCreateDeviceWithAddresses(b *testing.B) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(b.TempDir(), "rackd.db"))
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	ctx := SystemContext(context.Background(), "bench")

	const inventory = 10000
	for start := 0; start < inventory; start += 1000 {
		batch := make([]*model.Device, 0, 1000)
		for i := start; i < start+1000; i++ {
			batch = append(batch, &model.Device{Name: fmt.Sprintf("dev-%05d", i),
				Addresses: []model.Address{{IP: fmt.Sprintf("10.%d.%d.%d", i>>16&255, i>>8&255, i&255), Type: "ipv4"}}})
		}
		if _, err := store.BulkCreateDevices(ctx, batch); err != nil {
			b.Fatal(err)
		}
	}
	svc := NewServices(store, nil, nil)
	prev := publishEvent
	publishEvent = func(model.EventType, any) {}
	defer func() { publishEvent = prev }()

	n := 0
	b.ResetTimer()
	for b.Loop() {
		n++
		addresses := make([]model.Address, 10)
		for j := range addresses {
			addresses[j] = model.Address{IP: fmt.Sprintf("172.%d.%d.%d", n>>8&255, n&255, j+1), Type: "ipv4"}
		}
		if err := svc.Devices.Create(ctx, &model.Device{Name: fmt.Sprintf("new-%d", n), Addresses: addresses}); err != nil {
			b.Fatal(err)
		}
	}
}
