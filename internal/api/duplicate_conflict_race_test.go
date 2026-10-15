package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// Concurrent device writes for one address recorded several active
// duplicate-IP conflicts (found by the Strix run).
func TestConcurrentDeviceWritesRecordOneConflict(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			b, _ := json.Marshal(map[string]any{
				"name":      "dup-" + string(rune('a'+i)),
				"addresses": []map[string]any{{"ip": "10.0.0.7", "type": "ipv4"}},
			})
			req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			if w := performRequest(mux, authReq(req)); w.Code != http.StatusCreated {
				t.Errorf("create device: %d %s", w.Code, w.Body.String())
			}
		}(i)
	}
	wg.Wait()

	conflicts, err := store.ListConflicts(context.Background(), &model.ConflictFilter{Status: model.ConflictStatusActive})
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("%d active conflicts for 10.0.0.7, want 1", len(conflicts))
	}
}
