// Package extensions runs the features built on pkg/extension: it validates
// their manifests, runs their migrations, adds their permissions and event
// types, registers their routes and tools, and starts and stops them.
package extensions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/martinsuchenak/rackd/internal/api"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/mcp"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/ui"
	"github.com/martinsuchenak/rackd/internal/webhook"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/martinsuchenak/rackd/pkg/model"
	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/xdal"
)

// Store is the storage the runtime needs beyond core's.
type Store interface {
	XDAL() xdal.DBInterface
	RunExtensionMigrations(ctx context.Context, namespace string, steps []migrate.Migration) (storage.MigrationResult, error)
	EnsurePermissions(ctx context.Context, grants []storage.PermissionGrant) error
	RecordAudit(ctx context.Context, action, resource, resourceID string, changes any)
}

// Runtime holds the features of one server.
type Runtime struct {
	features []extension.Feature
	store    Store
	svc      *service.Services

	db xdal.DBInterface // shared by every feature's Host

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu           sync.Mutex
	stopping     bool
	unsubscribes []func()
}

// New validates the features' manifests and returns their runtime.
func New(features []extension.Feature, store Store, svc *service.Services) (*Runtime, error) {
	if err := Validate(features); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	rt := &Runtime{features: features, store: store, svc: svc, ctx: ctx, cancel: cancel}
	if store != nil {
		rt.db = noClose{store.XDAL()}
	}
	return rt, nil
}

// Validate checks that the features' manifests are well formed and that no two
// features share a name, a declared permission or an event type.
func Validate(features []extension.Feature) error {
	names := map[string]bool{}
	perms := map[string]string{}
	events := map[string]string{}
	for _, f := range features {
		m := f.Manifest()
		if err := m.Validate(); err != nil {
			return err
		}
		if names[m.Name] {
			return fmt.Errorf("extension %s is registered twice", m.Name)
		}
		names[m.Name] = true
		for _, p := range m.Permissions {
			if owner, ok := perms[p.String()]; ok {
				return fmt.Errorf("extension %s: permission %s is already declared by %s", m.Name, p, owner)
			}
			perms[p.String()] = m.Name
		}
		for _, e := range m.Events {
			if owner, ok := events[e.Type]; ok {
				return fmt.Errorf("extension %s: event type %s is already declared by %s", m.Name, e.Type, owner)
			}
			events[e.Type] = m.Name
		}
	}
	return nil
}

// Prepare runs the features' migrations, adds their permissions to RBAC and
// registers their event types. It runs after core's migrations and before any
// route, tool or Start.
func (rt *Runtime) Prepare(ctx context.Context) error {
	var grants []storage.PermissionGrant
	for _, f := range rt.features {
		m := f.Manifest()
		if mp, ok := f.(extension.MigrationProvider); ok {
			if _, err := rt.store.RunExtensionMigrations(ctx, m.Name, mp.Migrations()); err != nil {
				return fmt.Errorf("extension %s: %w", m.Name, err)
			}
		}
		for _, p := range m.Permissions {
			grants = append(grants, storage.PermissionGrant{Resource: p.Resource, Action: p.Action, DefaultRoles: p.DefaultRoles})
		}
		for _, e := range m.Events {
			if err := webhook.RegisterExtensionEventType(m.Name, model.EventType(e.Type), e.Label); err != nil {
				return fmt.Errorf("extension %s: %w", m.Name, err)
			}
		}
	}
	if len(grants) > 0 {
		if err := rt.store.EnsurePermissions(ctx, grants); err != nil {
			return fmt.Errorf("extension permissions: %w", err)
		}
	}
	return nil
}

// RegisterRoutes mounts the features' routes.
func (rt *Runtime) RegisterRoutes(handler *api.Handler, mux *http.ServeMux) error {
	var errs []error
	for _, f := range rt.features {
		rp, ok := f.(extension.RouteProvider)
		if !ok {
			continue
		}
		name := f.Manifest().Name
		router := handler.ExtensionRouter(mux, name)
		rp.RegisterRoutes(router, rt.host(name))
		errs = append(errs, router.Err())
	}
	return errors.Join(errs...)
}

// RegisterTools registers the features' MCP tools and instructions.
func (rt *Runtime) RegisterTools(server *mcp.Server) error {
	var errs []error
	for _, f := range rt.features {
		m := f.Manifest()
		if tp, ok := f.(extension.ToolProvider); ok {
			registry := server.ExtensionTools(m.Name)
			tp.RegisterTools(registry, rt.host(m.Name))
			errs = append(errs, registry.Err())
		}
		server.AddInstructions(m.Instructions)
	}
	return errors.Join(errs...)
}

// ConfigureUI adds the features to the UI configuration.
func (rt *Runtime) ConfigureUI(builder *api.UIConfigBuilder) {
	for _, f := range rt.features {
		builder.AddExtension(f.Manifest())
	}
}

// RegisterUIAssets serves the features' web UI files.
func (rt *Runtime) RegisterUIAssets(mux *http.ServeMux) {
	for _, f := range rt.features {
		m := f.Manifest()
		if m.UI.Assets != nil {
			ui.RegisterExtensionAssets(mux, m.Name, m.UI.Assets)
		}
	}
}

// Start calls Start on the features that have work to do while the server
// runs. On an error, features already started are stopped.
func (rt *Runtime) Start() error {
	for _, f := range rt.features {
		s, ok := f.(extension.Starter)
		if !ok {
			continue
		}
		name := f.Manifest().Name
		if err := s.Start(rt.ctx, rt.host(name)); err != nil {
			rt.Stop(10 * time.Second)
			return fmt.Errorf("extension %s: start: %w", name, err)
		}
		log.Info("Started extension", "extension", name)
	}
	return nil
}

// Stop cancels the features' context and waits up to timeout for their
// background work to return.
func (rt *Runtime) Stop(timeout time.Duration) {
	rt.mu.Lock()
	rt.stopping = true
	unsubscribes := rt.unsubscribes
	rt.unsubscribes = nil
	rt.mu.Unlock()
	for _, unsubscribe := range unsubscribes {
		unsubscribe()
	}
	rt.cancel()
	done := make(chan struct{})
	go func() {
		rt.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		log.Warn("Extension background work did not stop in time", "timeout", timeout)
	}
}

// track starts background work unless the runtime is stopping, so wg.Add never
// races Stop's wg.Wait.
func (rt *Runtime) track() bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.stopping {
		return false
	}
	rt.wg.Add(1)
	return true
}

func (rt *Runtime) subscribe(fn func(model.Event)) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.stopping {
		return
	}
	rt.unsubscribes = append(rt.unsubscribes, webhook.Subscribe(fn))
}

func (rt *Runtime) host(name string) extension.Host {
	return &host{name: name, rt: rt, logger: newLogger(name)}
}
