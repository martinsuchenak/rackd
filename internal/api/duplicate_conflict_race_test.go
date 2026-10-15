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

// A duplicate that comes back after its conflict was resolved is recorded
// again; one whose conflict was ignored stays ignored.
func TestDuplicateAfterResolvedConflictIsRecorded(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	ctx := context.Background()

	create := func(name, ip string) {
		t.Helper()
		b, _ := json.Marshal(map[string]any{"name": name, "addresses": []map[string]any{{"ip": ip, "type": "ipv4"}}})
		req := httptest.NewRequest("POST", "/api/v1/devices", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		if w := performRequest(mux, authReq(req)); w.Code != http.StatusCreated {
			t.Fatalf("create %s: %d %s", name, w.Code, w.Body.String())
		}
	}
	active := func(ip string) int {
		t.Helper()
		cs, err := store.GetConflictsByIP(ctx, ip)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, c := range cs {
			if c.Status == model.ConflictStatusActive {
				n++
			}
		}
		return n
	}
	settle := func(ip string, status model.ConflictStatus) {
		t.Helper()
		cs, _ := store.GetConflictsByIP(ctx, ip)
		for _, c := range cs {
			if c.Status == model.ConflictStatusActive {
				if err := store.UpdateConflictStatus(ctx, c.ID, status, "test", ""); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	create("a1", "10.0.0.8")
	create("a2", "10.0.0.8")
	settle("10.0.0.8", model.ConflictStatusResolved)
	create("a3", "10.0.0.8")
	if n := active("10.0.0.8"); n != 1 {
		t.Errorf("after a resolved conflict, a new duplicate has %d active conflicts, want 1", n)
	}

	create("b1", "10.0.0.9")
	create("b2", "10.0.0.9")
	settle("10.0.0.9", model.ConflictStatusIgnored)
	create("b3", "10.0.0.9")
	if n := active("10.0.0.9"); n != 0 {
		t.Errorf("after an ignored conflict, %d active conflicts, want 0", n)
	}
}
