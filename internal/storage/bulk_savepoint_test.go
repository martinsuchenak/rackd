package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// A bulk create that skips a failed device keeps none of that device's rows,
// and still creates the others, on every backend.
func TestBulkCreateDropsAFailedDeviceWhole(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	good := &model.Device{Name: "good", Tags: []string{"a"}, Domains: []string{}, Addresses: []model.Address{}}
	// The device row is written before its address, whose pool doesn't exist.
	bad := &model.Device{Name: "bad", Tags: []string{"b"}, Domains: []string{},
		Addresses: []model.Address{{IP: "10.0.0.1", Type: "ipv4", PoolID: "no-such-pool"}}}
	after := &model.Device{Name: "after", Tags: []string{}, Domains: []string{}, Addresses: []model.Address{}}

	res, err := s.BulkCreateDevices(ctx, []*model.Device{good, bad, after})
	if err != nil {
		t.Fatal(err)
	}
	if res.Success != 2 || res.Failed != 1 {
		t.Fatalf("result = %+v, want 2 created and 1 failed", res)
	}
	devices, err := s.ListDevices(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, d := range devices {
		names[d.Name] = true
	}
	if !names["good"] || !names["after"] || names["bad"] {
		t.Errorf("devices = %v, want good and after only", names)
	}
	var tags int
	if err := s.q.QueryRowContext(ctx, `SELECT COUNT(*) FROM tags WHERE tag = ?`, "b").Scan(&tags); err != nil || tags != 0 {
		t.Errorf("the failed device's tags: %d, %v", tags, err)
	}
}
