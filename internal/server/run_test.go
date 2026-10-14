package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/martinsuchenak/rackd/pkg/model"
)

type probe struct {
	pattern string
	stopped atomic.Bool
}

func (p *probe) Manifest() extension.Manifest {
	return extension.Manifest{Name: "probe", Version: "1", UI: extension.UI{
		Assets:  fstest.MapFS{"ui/probe.js": {Data: []byte("export {};")}},
		Scripts: []string{"ui/probe.js"},
	}}
}

func (p *probe) RegisterRoutes(r extension.Router, host extension.Host) {
	r.Handle(p.pattern, func(w http.ResponseWriter, _ *http.Request) {
		extension.WriteJSON(w, http.StatusOK, map[string]string{"ok": "yes"})
	}, extension.Public())
}

func (p *probe) Start(ctx context.Context, host extension.Host) error {
	host.Go("wait", func(ctx context.Context) {
		<-ctx.Done()
		p.stopped.Store(true)
	})
	return nil
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	cfg := config.Load()
	cfg.ListenAddr = addr
	cfg.DataDir = t.TempDir()
	cfg.RateLimitEnabled = false
	cfg.SessionStoreType = "memory"
	cfg.MCPOAuthEnabled = false
	cfg.DNSSyncInterval = 0
	return cfg
}

func newStore(t *testing.T, cfg *config.Config) *storage.SQLiteStorage {
	t.Helper()
	store, err := storage.NewSQLiteStorageWithPath(filepath.Join(cfg.DataDir, "rackd.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestRunServesExtensionsAndShutsDown(t *testing.T) {
	cfg := testConfig(t)
	p := &probe{pattern: "GET /probe"}
	store := newStore(t, cfg)
	if err := store.CreateNetwork(context.Background(), &model.Network{Name: "trend", Subnet: "10.7.0.0/24"}); err != nil {
		t.Fatal(err)
	}
	hook := &model.Webhook{Name: "local", URL: "http://127.0.0.1:9/hook", Active: true, Events: []model.EventType{model.EventTypeNetworkCreated}}
	if err := store.CreateWebhook(context.Background(), hook); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{Config: cfg, Store: store, Features: []extension.Feature{p}, Edition: "test",
			UI: fstest.MapFS{"index.html": {Data: []byte("<!DOCTYPE html>replacement")}}})
	}()

	base := "http://" + cfg.ListenAddr
	var resp *http.Response
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		var err error
		if resp, err = http.Get(base + "/api/v1/probe"); err == nil {
			break
		}
	}
	if resp == nil {
		t.Fatal("server did not start")
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("extension route = %d", resp.StatusCode)
	}

	resp, err := http.Get(base + "/api/v1/config")
	if err != nil {
		t.Fatal(err)
	}
	var uiConfig struct {
		Edition    string   `json:"edition"`
		Features   []string `json:"features"`
		Extensions []struct {
			Scripts []string `json:"scripts"`
		} `json:"extensions"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&uiConfig)
	_ = resp.Body.Close()
	if uiConfig.Edition != "test" || len(uiConfig.Features) != 1 || uiConfig.Features[0] != "probe" {
		t.Errorf("config = %+v", uiConfig)
	}
	if len(uiConfig.Extensions) != 1 || strings.Join(uiConfig.Extensions[0].Scripts, ",") != "/ext/probe/ui/probe.js" {
		t.Errorf("extension scripts = %+v", uiConfig.Extensions)
	}

	// The extension's UI files, and the replacement UI, are served.
	for _, tc := range []struct{ path, want string }{
		{"/ext/probe/ui/probe.js", "export {};"},
		{"/devices", "<!DOCTYPE html>replacement"},
	} {
		resp, err := http.Get(base + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != tc.want {
			t.Errorf("GET %s = %d %q, want %q", tc.path, resp.StatusCode, body, tc.want)
		}
	}

	// The snapshot worker runs: it takes a snapshot of each network on start.
	waitFor := func(what string, cond func() bool) {
		t.Helper()
		for deadline := time.Now().Add(5 * time.Second); !cond(); time.Sleep(20 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("timed out waiting for %s", what)
			}
		}
	}
	waitFor("a network snapshot", func() bool {
		snaps, _ := store.GetLatestSnapshots(context.Background(), model.SnapshotTypeNetwork)
		return len(snaps) == 1
	})

	// The webhook worker runs: a published event produces a delivery attempt
	// (refused here, since loopback targets are blocked).
	webhook.Publish(model.EventTypeNetworkCreated, map[string]string{"id": "net-1"})
	deadline := time.Now().Add(5 * time.Second)
	for {
		ds, _ := store.ListDeliveries(context.Background(), &model.DeliveryFilter{WebhookID: hook.ID})
		if len(ds) == 1 && strings.Contains(ds[0].Error, "SSRF prevention") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no delivery attempt recorded: %+v", ds)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// A connection the client dialled but never used would hold Shutdown for 5s
	// (net/http treats new connections as idle only after that).
	http.DefaultClient.CloseIdleConnections()
	cancelled := time.Now()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run returned %v after cancel", err)
		}
		if took := time.Since(cancelled); took > 2*time.Second {
			t.Errorf("shutdown took %s", took)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if !p.stopped.Load() {
		t.Error("extension background work was not stopped")
	}
}

func TestRunReportsRouteClashWithCore(t *testing.T) {
	cfg := testConfig(t)
	err := Run(context.Background(), Options{Config: cfg, Store: newStore(t, cfg), Features: []extension.Feature{&probe{pattern: "GET /config"}}})
	if err == nil || !strings.Contains(err.Error(), `"GET /config"`) {
		t.Errorf("Run = %v, want a route clash error", err)
	}
	if errors.Is(err, http.ErrServerClosed) {
		t.Error("server started despite the clash")
	}
}
