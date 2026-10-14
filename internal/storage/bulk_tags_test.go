package storage

import (
	"context"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestBulkTagsCountEachDeviceOnce(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()
	device := &model.Device{Name: "tagged"}
	if err := s.CreateDevice(ctx, device); err != nil {
		t.Fatal(err)
	}

	for name, run := range map[string]func() (*BulkResult, error){
		"add":    func() (*BulkResult, error) { return s.BulkAddTags(ctx, []string{device.ID, "missing"}, []string{"a"}) },
		"remove": func() (*BulkResult, error) { return s.BulkRemoveTags(ctx, []string{device.ID}, []string{"a"}) },
	} {
		res, err := run()
		if err != nil {
			t.Fatal(err)
		}
		if res.Success+res.Failed != res.Total {
			t.Errorf("%s: success %d + failed %d != total %d", name, res.Success, res.Failed, res.Total)
		}
		if len(res.SucceededIDs) != res.Success || (res.Success > 0 && res.SucceededIDs[0] != device.ID) {
			t.Errorf("%s: succeeded IDs %v, success %d", name, res.SucceededIDs, res.Success)
		}
	}
}
