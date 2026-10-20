package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// API keys carried their owner's full permissions, and a "permissions" field
// at creation was silently ignored (found by a Strix run): a key meant for
// one read-only integration could reset passwords.
func TestAPIKeyScopes(t *testing.T) {
	ts := newTestServer(t)
	adminID := ts.createAdminUser(t, "admin", "securepassword123")
	adminToken := ts.createAPIKeyForUser(t, adminID)

	create := func(token string, body map[string]any) (int, map[string]any) {
		t.Helper()
		w := ts.doRequest(t, http.MethodPost, "/api/v1/keys", body, token)
		var resp map[string]any
		parseJSON(t, w, &resp)
		return w.Code, resp
	}

	code, resp := create(adminToken, map[string]any{"name": "inventory-reader", "scopes": []string{"devices:list", "devices:list", "devices:read"}})
	if code != http.StatusCreated {
		t.Fatalf("create scoped key: %d %v", code, resp)
	}
	if scopes, _ := resp["scopes"].([]any); len(scopes) != 2 {
		t.Errorf("scopes %v, want devices:list and devices:read once each", resp["scopes"])
	}
	scoped := resp["key"].(string)

	for _, c := range []struct {
		method, path string
		body         any
		want         int
	}{
		{http.MethodGet, "/api/v1/devices", nil, http.StatusOK},
		{http.MethodGet, "/api/v1/users", nil, http.StatusForbidden},
		{http.MethodPost, "/api/v1/users", map[string]any{"username": "pwned", "password": "securepassword999", "email": "p@k.test"}, http.StatusForbidden},
		{http.MethodPost, "/api/v1/devices", map[string]any{"name": "nope"}, http.StatusForbidden},
		{http.MethodGet, "/api/v1/roles", nil, http.StatusForbidden},
	} {
		if w := ts.doRequest(t, c.method, c.path, c.body, scoped); w.Code != c.want {
			t.Errorf("scoped key %s %s: %d, want %d", c.method, c.path, w.Code, c.want)
		}
	}

	// A key can't hold scopes its creator lacks, or that its creating key
	// lacks.
	ctx := service.SystemContext(context.Background(), "test")
	role := &model.Role{Name: "key-maker"}
	perms, _ := ts.store.ListPermissions(ctx, nil)
	var permIDs []string
	for _, p := range perms {
		if (p.Resource == "apikeys" && p.Action == "create") || (p.Resource == "devices" && p.Action == "list") {
			permIDs = append(permIDs, p.ID)
		}
	}
	if err := ts.svc.Roles.CreateWithPermissions(ctx, role, permIDs); err != nil {
		t.Fatal(err)
	}
	maker, err := ts.svc.Users.Create(ctx, &model.CreateUserRequest{Username: "maker", Password: "securepassword321", Email: "maker@test.local", RoleID: role.ID})
	if err != nil {
		t.Fatal(err)
	}
	makerToken := ts.createAPIKeyForUser(t, maker.ID)
	if code, resp := create(makerToken, map[string]any{"name": "too-much", "scopes": []string{"devices:list", "users:list"}}); code != http.StatusBadRequest {
		t.Errorf("scope the creator lacks: %d %v, want 400", code, resp)
	}
	if code, resp := create(scoped, map[string]any{"name": "from-scoped"}); code != http.StatusForbidden {
		t.Errorf("a devices-only key creating a key: %d %v, want 403", code, resp)
	}

	// A scoped key that may create keys can't mint one wider than itself:
	// asking for no scopes gives the new key its own.
	code, resp = create(adminToken, map[string]any{"name": "key-minter", "scopes": []string{"apikeys:create", "devices:list"}})
	if code != http.StatusCreated {
		t.Fatalf("create minter key: %d %v", code, resp)
	}
	minter := resp["key"].(string)
	code, resp = create(minter, map[string]any{"name": "minted"})
	if code != http.StatusCreated {
		t.Fatalf("minter creating a key: %d %v", code, resp)
	}
	if scopes, _ := resp["scopes"].([]any); len(scopes) != 2 {
		t.Errorf("a key minted by a scoped key has scopes %v, want the minter's two", resp["scopes"])
	}
	if w := ts.doRequest(t, http.MethodGet, "/api/v1/users", nil, resp["key"].(string)); w.Code != http.StatusForbidden {
		t.Errorf("minted key GET /users: %d, want 403", w.Code)
	}
	if code, resp := create(minter, map[string]any{"name": "wider", "scopes": []string{"users:list"}}); code != http.StatusBadRequest {
		t.Errorf("minter asking for a scope it lacks: %d %v, want 400", code, resp)
	}

	// Fields that look like they limit the key are refused, not ignored.
	for _, body := range []map[string]any{
		{"name": "x", "permissions": []string{"devices:list"}},
		{"name": "x", "role": "viewer"},
		{"name": "x", "is_admin": false},
		{"name": "x", "scopes": []string{}},
	} {
		if code, resp := create(adminToken, body); code != http.StatusBadRequest {
			t.Errorf("create %v: %d %v, want 400", body, code, resp)
		}
	}

	// A key without scopes still acts with all of its owner's permissions.
	code, resp = create(adminToken, map[string]any{"name": "full"})
	if code != http.StatusCreated || resp["scopes"] != nil {
		t.Fatalf("unscoped key: %d %v", code, resp)
	}
	if w := ts.doRequest(t, http.MethodGet, "/api/v1/users", nil, resp["key"].(string)); w.Code != http.StatusOK {
		t.Errorf("unscoped admin key GET /users: %d, want 200", w.Code)
	}
}
