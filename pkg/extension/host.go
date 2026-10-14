package extension

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/martinsuchenak/rackd/pkg/model"
	"github.com/martinsuchenak/xdal/pkg/xdal"
	"github.com/paularlott/mcp"
)

// Host is what Rackd gives a feature to work with.
type Host interface {
	// Core reads and changes core data with the permissions of the caller in
	// the context passed to each method.
	Core() Core
	// Events publishes and subscribes to events.
	Events() EventBus
	// Authorize returns ErrForbidden (or ErrUnauthenticated) unless the caller
	// in ctx holds resource:action. System callers always pass.
	Authorize(ctx context.Context, resource, action string) error
	// Audit records a change in Rackd's audit log, attributed to the caller in
	// ctx: action is a verb such as "create", resource names what changed
	// (such as "device-note"), and details are stored as JSON. Record every
	// change to the feature's own data; core data changed through Core is
	// audited already.
	Audit(ctx context.Context, action, resource, resourceID string, details any)
	// DB is the database, for the feature's own tables only. Close returns an
	// error.
	DB() xdal.DBInterface
	// Logger logs with the feature's name attached.
	Logger() *slog.Logger
	// Go runs fn in the background until ctx is cancelled at shutdown, and
	// shutdown waits for it to return.
	Go(name string, fn func(ctx context.Context))
}

// Core is the core data a feature can use. It grows as features need more;
// every method checks the caller's permissions.
type Core interface {
	Devices() Devices
	Networks() Networks
}

// Devices reads devices.
type Devices interface {
	Get(ctx context.Context, id string) (*model.Device, error)
	List(ctx context.Context, filter *model.DeviceFilter) ([]model.Device, error)
}

// Networks reads networks.
type Networks interface {
	Get(ctx context.Context, id string) (*model.Network, error)
	List(ctx context.Context, filter *model.NetworkFilter) ([]model.Network, error)
}

// EventBus delivers events to subscribers, including webhooks.
type EventBus interface {
	// Publish sends an event to all subscribers asynchronously. The type must
	// be one of core's or declared in a feature's Manifest.
	Publish(eventType string, payload any)
	// Subscribe calls fn for every event published.
	Subscribe(fn func(model.Event))
}

// Router mounts a feature's REST endpoints under /api/v1.
type Router interface {
	// Handle registers handler for pattern, a net/http pattern such as
	// "GET /things/{id}" relative to /api/v1. By default the route requires an
	// authenticated user (session or API key) and its body is limited to 1 MiB;
	// the caller is in the request context.
	Handle(pattern string, handler http.HandlerFunc, opts ...Option)
}

// ToolRegistry registers a feature's MCP tools.
type ToolRegistry interface {
	// Register adds a tool. The handler runs as the MCP caller; options such as
	// Require check permissions before it runs.
	Register(tool *mcp.ToolBuilder, handler mcp.ToolHandler, opts ...Option)
}

// Option configures a route or tool.
type Option func(*Options)

// Options holds what Option functions set. Rackd reads it; features use the
// Option functions.
type Options struct {
	Permission *Permission
	Public     bool
}

// Require makes a route or tool require resource:action.
func Require(resource, action string) Option {
	return func(o *Options) { o.Permission = &Permission{Resource: resource, Action: action} }
}

// Public makes a route reachable without authentication. It has no effect on
// tools, which always run as the MCP caller.
func Public() Option {
	return func(o *Options) { o.Public = true }
}

// ApplyOptions returns the options opts set.
func ApplyOptions(opts []Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
