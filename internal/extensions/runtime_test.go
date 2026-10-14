package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/mcp"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/martinsuchenak/rackd/pkg/model"
	"github.com/martinsuchenak/xdal/pkg/migrate"
	mcplib "github.com/paularlott/mcp"
)

// sample is a feature built only on pkg/extension, as an extension in another
// repository would be.
type sample struct {
	host    extension.Host
	running atomic.Bool
	stopped atomic.Bool
}

func (f *sample) Manifest() extension.Manifest {
	return extension.Manifest{
		Name:    "sample",
		Version: "0.0.1",
		Permissions: []extension.Permission{
			{Resource: "things", Action: "list", DefaultRoles: []string{"admin", "viewer"}},
			{Resource: "things", Action: "create", DefaultRoles: []string{"admin"}},
		},
		Events: []extension.EventType{{Type: "thing.created", Label: "Thing Created"}},
		UI: extension.UI{
			Pages: []extension.Page{{Path: "/things", Title: "Things", Permission: &extension.Permission{Resource: "things", Action: "list"}}},
			Nav:   []extension.NavItem{{Label: "Things", Path: "/things", Icon: "box", Order: 95}},
		},
		Instructions: "Use sample_count to count things.",
	}
}

func (f *sample) Migrations() []extension.Migration {
	return []extension.Migration{{
		Version: "20260101000000",
		Name:    "create_things",
		Up: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `CREATE TABLE sample_things (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`)
		},
		Down: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.ExecSQL(ctx, `DROP TABLE sample_things`)
		},
	}}
}

func (f *sample) count(ctx context.Context) (int, error) {
	var n int
	err := f.host.DB().QueryRow(ctx, `SELECT COUNT(*) FROM sample_things`).Scan(&n)
	return n, err
}

func (f *sample) RegisterRoutes(r extension.Router, host extension.Host) {
	f.host = host
	r.Handle("GET /sample/things", func(w http.ResponseWriter, req *http.Request) {
		n, err := f.count(req.Context())
		if err != nil {
			extension.WriteError(w, err)
			return
		}
		extension.WriteJSON(w, http.StatusOK, map[string]int{"count": n})
	}, extension.Require("things", "list"))
	r.Handle("POST /sample/things", func(w http.ResponseWriter, req *http.Request) {
		var body struct{ Name string }
		if err := extension.DecodeJSON(req, &body); err != nil {
			extension.WriteError(w, err)
			return
		}
		if _, err := host.DB().Exec(req.Context(), `INSERT INTO sample_things (name) VALUES (?)`, body.Name); err != nil {
			extension.WriteError(w, err)
			return
		}
		host.Audit(req.Context(), "create", "thing", body.Name, body)
		host.Events().Publish("thing.created", map[string]string{"name": body.Name})
		extension.WriteJSON(w, http.StatusCreated, body)
	}, extension.Require("things", "create"))
	r.Handle("GET /sample/devices", func(w http.ResponseWriter, req *http.Request) {
		devices, err := host.Core().Devices().List(req.Context(), nil)
		if err != nil {
			extension.WriteError(w, err)
			return
		}
		extension.WriteJSON(w, http.StatusOK, map[string]int{"devices": len(devices)})
	})
	r.Handle("GET /sample/ping", func(w http.ResponseWriter, req *http.Request) {
		extension.WriteJSON(w, http.StatusOK, map[string]bool{"authenticated": extension.CallerFrom(req.Context()) != nil})
	}, extension.Public())
}

func (f *sample) RegisterTools(t extension.ToolRegistry, host extension.Host) {
	t.Register(mcplib.NewTool("sample_count", "Count things"), func(ctx context.Context, req *mcplib.ToolRequest) (*mcplib.ToolResponse, error) {
		n, err := f.count(ctx)
		if err != nil {
			return nil, err
		}
		return mcplib.NewToolResponseText(strings.Repeat("x", n)), nil
	}, extension.Require("things", "list"))
}

func (f *sample) Start(ctx context.Context, host extension.Host) error {
	host.Go("ticker", func(ctx context.Context) {
		f.running.Store(true)
		<-ctx.Done()
		f.stopped.Store(true)
	})
	return nil
}

type fixture struct {
	store   *storage.SQLiteStorage
	svc     *service.Services
	rt      *Runtime
	mux     *http.ServeMux
	handler *api.Handler
	tokens  map[string]string // role -> API key
}

func newFixture(t *testing.T, path string, features ...extension.Feature) *fixture {
	t.Helper()
	store, err := storage.NewSQLiteStorageWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	svc := service.NewServices(store, nil, nil)
	rt, err := New(features, store, svc)
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler := api.NewHandler(store, nil, api.WithServices(svc))
	handler.RegisterRoutes(mux)
	return &fixture{store: store, svc: svc, rt: rt, mux: mux, handler: handler, tokens: map[string]string{}}
}

// user creates a user with role and returns an API key for them.
func (f *fixture) user(t *testing.T, role string) string {
	t.Helper()
	ctx := context.Background()
	u := &model.User{ID: "user-" + role, Username: role, Email: role + "@example.test", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := f.store.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	r, err := f.store.GetRoleByName(ctx, role)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.store.AssignRoleToUser(ctx, u.ID, r.ID); err != nil {
		t.Fatal(err)
	}
	token := "rk_test_" + role
	if err := f.store.CreateAPIKey(ctx, &model.APIKey{Name: role, Key: auth.HashToken(token), UserID: u.ID}); err != nil {
		t.Fatal(err)
	}
	return token
}

func (f *fixture) do(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func TestRuntimeRunsAFeature(t *testing.T) {
	ctx := context.Background()
	feature := &sample{}
	f := newFixture(t, filepath.Join(t.TempDir(), "rackd.db"), feature)
	if err := f.rt.RegisterRoutes(f.handler, f.mux); err != nil {
		t.Fatal(err)
	}
	admin, viewer := f.user(t, "admin"), f.user(t, "viewer")

	// Permissions were added and granted to their default roles.
	for role, want := range map[string][]string{"admin": {"things:list", "things:create"}, "viewer": {"things:list"}, "operator": nil} {
		r, err := f.store.GetRoleByName(ctx, role)
		if err != nil {
			t.Fatal(err)
		}
		perms, err := f.store.GetRolePermissions(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, p := range perms {
			if p.Resource == "things" {
				got[p.Resource+":"+p.Action] = true
			}
		}
		if len(got) != len(want) {
			t.Errorf("%s has %v, want %v", role, got, want)
		}
	}

	// The feature's events reach subscribers, and webhooks accept the type.
	if !webhook.IsKnownEventType("thing.created") {
		t.Fatal("thing.created is not a known event type")
	}
	received := make(chan model.Event, 1)
	webhook.Subscribe(func(e model.Event) {
		if e.Type == "thing.created" {
			received <- e
		}
	})

	for _, tc := range []struct {
		method, path, token, body string
		status                    int
	}{
		{"GET", "/api/v1/sample/things", "", "", http.StatusUnauthorized},
		{"GET", "/api/v1/sample/things", viewer, "", http.StatusOK},
		{"POST", "/api/v1/sample/things", viewer, `{"name":"a"}`, http.StatusForbidden},
		{"POST", "/api/v1/sample/things", admin, `{"name":"a"}`, http.StatusCreated},
		{"POST", "/api/v1/sample/things", admin, `not json`, http.StatusBadRequest},
		{"GET", "/api/v1/sample/devices", viewer, "", http.StatusOK},
		{"GET", "/api/v1/sample/ping", "", "", http.StatusOK},
		{"GET", "/api/sample/things", viewer, "", http.StatusNotFound}, // no deprecated alias
	} {
		rec := f.do(t, tc.method, tc.path, tc.token, tc.body)
		if rec.Code != tc.status {
			t.Errorf("%s %s = %d, want %d: %s", tc.method, tc.path, rec.Code, tc.status, rec.Body.String())
		}
	}

	select {
	case e := <-received:
		if e.Payload.(map[string]string)["name"] != "a" {
			t.Errorf("event payload = %v", e.Payload)
		}
	case <-time.After(2 * time.Second):
		t.Error("thing.created was not published")
	}

	// The write was audited as the admin, through rackd's audit log.
	waitFor(t, func() bool {
		logs, err := f.store.ListAuditLogs(ctx, &model.AuditFilter{Resource: "thing"})
		return err == nil && len(logs) == 1 && logs[0].Username == "admin" && logs[0].Action == "create"
	})

	rec := f.do(t, "GET", "/api/v1/sample/things", viewer, "")
	var body map[string]int
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["count"] != 1 {
		t.Errorf("count = %v", body)
	}

	// Background work runs until Stop.
	if err := f.rt.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, feature.running.Load)
	f.rt.Stop(time.Second)
	if !feature.stopped.Load() {
		t.Error("background work did not stop")
	}
}

func TestRuntimeToolsAndUI(t *testing.T) {
	ctx := context.Background()
	feature := &sample{}
	f := newFixture(t, filepath.Join(t.TempDir(), "rackd.db"), feature)
	f.user(t, "viewer")
	f.user(t, "operator")
	if err := f.rt.RegisterRoutes(f.handler, f.mux); err != nil {
		t.Fatal(err)
	}

	server := mcp.NewServer(f.svc, f.store, true)
	if err := f.rt.RegisterTools(server); err != nil {
		t.Fatal(err)
	}
	asUser := func(id string) context.Context {
		return extension.WithCaller(ctx, &extension.Caller{Type: extension.CallerTypeUser, UserID: id})
	}
	if _, err := server.Inner().CallTool(asUser("user-viewer"), "sample_count", nil); err != nil {
		t.Errorf("viewer calling sample_count: %v", err)
	}
	if _, err := server.Inner().CallTool(asUser("user-operator"), "sample_count", nil); err == nil {
		t.Error("operator lacks things:list but could call sample_count")
	}

	builder := api.NewUIConfigBuilder()
	f.rt.ConfigureUI(builder)
	cfg := builder.Build()
	if len(cfg.Features) != 1 || cfg.Features[0] != "sample" || len(cfg.Pages) != 1 || cfg.Pages[0].RequiredPermission == nil {
		t.Errorf("UI config = %+v", cfg)
	}
	found := false
	for _, n := range cfg.NavItems {
		found = found || n.Path == "/things"
	}
	if !found {
		t.Error("nav item missing")
	}
}

func TestRuntimeIsIdempotentAcrossRestarts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rackd.db")
	f := newFixture(t, path, &sample{})
	ctx := context.Background()

	// An administrator removes a default grant; a restart must not restore it.
	viewer, _ := f.store.GetRoleByName(ctx, "viewer")
	perms, _ := f.store.GetRolePermissions(ctx, viewer.ID)
	var keep []string
	for _, p := range perms {
		if p.Resource != "things" {
			keep = append(keep, p.ID)
		}
	}
	if err := f.store.SetRolePermissions(ctx, viewer.ID, keep); err != nil {
		t.Fatal(err)
	}
	_ = f.store.Close()

	f2 := newFixture(t, path, &sample{})
	perms, _ = f2.store.GetRolePermissions(ctx, viewer.ID)
	for _, p := range perms {
		if p.Resource == "things" {
			t.Errorf("restart re-granted %s:%s to viewer", p.Resource, p.Action)
		}
	}
}

// conflicting registers routes and tools that clash with core's.
type conflicting struct{}

func (conflicting) Manifest() extension.Manifest { return extension.Manifest{Name: "conflicting"} }

func (conflicting) RegisterRoutes(r extension.Router, host extension.Host) {
	r.Handle("GET /devices", func(http.ResponseWriter, *http.Request) {})
	r.Handle("no-method", func(http.ResponseWriter, *http.Request) {})
}

func (conflicting) RegisterTools(t extension.ToolRegistry, host extension.Host) {
	t.Register(mcplib.NewTool("device_list", "clashes with core"), nil)
}

func TestRuntimeReportsConflicts(t *testing.T) {
	f := newFixture(t, filepath.Join(t.TempDir(), "rackd.db"), conflicting{})
	err := f.rt.RegisterRoutes(f.handler, f.mux)
	if err == nil || !strings.Contains(err.Error(), `"GET /devices"`) || !strings.Contains(err.Error(), `"no-method"`) {
		t.Errorf("route conflicts: %v", err)
	}
	if err := f.rt.RegisterTools(mcp.NewServer(f.svc, f.store, true)); err == nil || !strings.Contains(err.Error(), "device_list") {
		t.Errorf("tool conflict: %v", err)
	}
}

type named string

func (n named) Manifest() extension.Manifest {
	return extension.Manifest{Name: string(n), Permissions: []extension.Permission{{Resource: "x", Action: "y"}}}
}

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		features []extension.Feature
		want     string
	}{
		{[]extension.Feature{named("Bad Name")}, "must match"},
		{[]extension.Feature{named("a"), named("a")}, "registered twice"},
		{[]extension.Feature{named("a"), named("b")}, "already declared by a"},
	} {
		if err := Validate(tc.features); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("Validate(%v) = %v, want %q", tc.features, err, tc.want)
		}
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatal("condition not met within 2s")
}

type claimsCoreEvent struct{}

func (claimsCoreEvent) Manifest() extension.Manifest {
	return extension.Manifest{Name: "claims", Events: []extension.EventType{{Type: "device.created", Label: "Device Created"}}}
}

func TestRuntimeRefusesCoreEventTypes(t *testing.T) {
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(t.TempDir(), "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	rt, err := New([]extension.Feature{claimsCoreEvent{}}, store, service.NewServices(store, nil, nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Prepare(context.Background()); err == nil || !strings.Contains(err.Error(), "already registered by core") {
		t.Errorf("Prepare = %v, want a core event type clash", err)
	}
}

func TestRuntimeStopEndsWorkAndSubscriptions(t *testing.T) {
	f := newFixture(t, filepath.Join(t.TempDir(), "rackd.db"), &sample{})
	h := f.rt.host("sample")

	var calls atomic.Int32
	h.Events().Subscribe(func(e model.Event) {
		if e.Type == "thing.created" {
			calls.Add(1)
		}
	})
	h.Events().Publish("thing.created", nil)
	waitFor(t, func() bool { return calls.Load() == 1 })

	f.rt.Stop(time.Second)
	h.Events().Publish("thing.created", nil)
	var started atomic.Bool
	h.Go("late", func(context.Context) { started.Store(true) })
	time.Sleep(50 * time.Millisecond)
	if calls.Load() != 1 {
		t.Errorf("subscription still active after Stop (%d calls)", calls.Load())
	}
	if started.Load() {
		t.Error("background work started after Stop")
	}
}

type publicWithPermission struct{}

func (publicWithPermission) Manifest() extension.Manifest {
	return extension.Manifest{Name: "public-perm"}
}

func (publicWithPermission) RegisterRoutes(r extension.Router, host extension.Host) {
	r.Handle("GET /public-perm", func(http.ResponseWriter, *http.Request) {}, extension.Public(), extension.Require("x", "y"))
}

func TestRouterRejectsPublicWithPermission(t *testing.T) {
	f := newFixture(t, filepath.Join(t.TempDir(), "rackd.db"), publicWithPermission{})
	if err := f.rt.RegisterRoutes(f.handler, f.mux); err == nil || !strings.Contains(err.Error(), "is Public but requires") {
		t.Errorf("RegisterRoutes = %v", err)
	}
}
