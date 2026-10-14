package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/martinsuchenak/rackd/internal/config"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/pkg/extension"
)

type probe struct {
	pattern string
	stopped atomic.Bool
}

func (p *probe) Manifest() extension.Manifest { return extension.Manifest{Name: "probe", Version: "1"} }

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
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{Config: cfg, Store: newStore(t, cfg), Features: []extension.Feature{p}, Edition: "test"})
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
		Edition  string   `json:"edition"`
		Features []string `json:"features"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&uiConfig)
	_ = resp.Body.Close()
	if uiConfig.Edition != "test" || len(uiConfig.Features) != 1 || uiConfig.Features[0] != "probe" {
		t.Errorf("config = %+v", uiConfig)
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
