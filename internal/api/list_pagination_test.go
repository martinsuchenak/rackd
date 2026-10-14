package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func TestListEndpointsReturnEverythingUnlessPaged(t *testing.T) {
	h, store := setupTestHandler(t)
	defer func() { _ = store.Close() }()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	for i := range 150 {
		if err := store.CreateNetwork(context.Background(), &model.Network{Name: fmt.Sprintf("net-%03d", i), Subnet: fmt.Sprintf("10.%d.%d.0/24", i/250, i%250)}); err != nil {
			t.Fatal(err)
		}
	}

	for path, want := range map[string]int{
		"/api/v1/networks":                     150,
		"/api/v1/networks?limit=10":            10,
		"/api/v1/networks?limit=10&offset=145": 5,
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, authReq(httptest.NewRequest("GET", path, nil)))
		var networks []model.Network
		if err := json.Unmarshal(rec.Body.Bytes(), &networks); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
		if len(networks) != want {
			t.Errorf("%s returned %d networks, want %d", path, len(networks), want)
		}
	}

	// The permission catalog is longer than one default page.
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, authReq(httptest.NewRequest("GET", "/api/v1/permissions", nil)))
	var perms []model.Permission
	_ = json.Unmarshal(rec.Body.Bytes(), &perms)
	all, _ := store.ListPermissions(context.Background(), nil)
	if len(perms) != len(all) || len(perms) <= model.DefaultPageSize {
		t.Errorf("GET /permissions returned %d of %d permissions", len(perms), len(all))
	}
}
