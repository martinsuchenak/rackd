package extension

import (
	"context"
	"fmt"
	"regexp"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/paularlott/cli"
)

// Feature is a unit of functionality compiled into a Rackd binary.
type Feature interface {
	Manifest() Manifest
}

// Manifest describes a feature to Rackd.
type Manifest struct {
	// Name identifies the feature. It must match [a-z][a-z0-9-]* and be unique
	// in the binary; it names the feature's migration namespace and appears in
	// logs and /api/v1/config.
	Name string
	// Version is the feature's own version, shown in /api/v1/config.
	Version string
	// Permissions the feature adds to RBAC. Routes and tools may also require
	// core permissions without declaring them.
	Permissions []Permission
	// Events the feature publishes, added to the event types webhooks can
	// subscribe to.
	Events []EventType
	// UI pages and navigation entries, reported in /api/v1/config.
	UI UI
	// Instructions is one line added to the MCP server instructions, telling
	// assistants what the feature's tools are for. Leave empty for none.
	Instructions string
}

// Permission is an RBAC resource and action.
type Permission struct {
	Resource string
	Action   string
	// DefaultRoles are the built-in roles ("admin", "operator", "viewer")
	// granted the permission when it is first added. Later changes made by
	// administrators are kept.
	DefaultRoles []string
}

// String returns the permission as "resource:action".
func (p Permission) String() string { return p.Resource + ":" + p.Action }

// EventType is an event a feature publishes.
type EventType struct {
	// Type is the event name, conventionally "<object>.<verb>" in the past
	// tense, such as "backup.completed".
	Type string
	// Label is a short human-readable name for the UI.
	Label string
}

// UI lists a feature's pages and navigation entries.
type UI struct {
	Pages []Page
	Nav   []NavItem
}

// Page is a route in the web UI.
type Page struct {
	Path  string
	Title string
	// Permission, if set, is required to open the page.
	Permission *Permission
}

// NavItem is an entry in the web UI's navigation.
type NavItem struct {
	Label string
	Path  string
	Icon  string
	// Order positions the item among core's; lower comes first.
	Order int
	// Permission, if set, is required to see the item.
	Permission *Permission
}

// Migration is a schema change for a feature's own tables. Use tx.Schema() to
// stay portable across the databases Rackd supports.
type Migration = migrate.Migration

// MigrationProvider is implemented by features that own database tables.
// Migrations run in the feature's own namespace, after core's, and before the
// feature's routes, tools or Start.
type MigrationProvider interface {
	Migrations() []Migration
}

// RouteProvider is implemented by features with REST endpoints.
type RouteProvider interface {
	RegisterRoutes(r Router, host Host)
}

// ToolProvider is implemented by features with MCP tools.
type ToolProvider interface {
	RegisterTools(t ToolRegistry, host Host)
}

// CommandProvider is implemented by features with CLI commands. Commands are
// added to the binary's top level and run without a server.
type CommandProvider interface {
	Commands() []*cli.Command
}

// Starter is implemented by features with work to do while the server runs.
// Start is called once all features are registered; ctx is cancelled when the
// server shuts down. Run background work with host.Go so shutdown waits for it.
type Starter interface {
	Start(ctx context.Context, host Host) error
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Validate reports whether the manifest is well formed.
func (m Manifest) Validate() error {
	if !namePattern.MatchString(m.Name) {
		return fmt.Errorf("extension: feature name %q must match %s", m.Name, namePattern)
	}
	for _, p := range m.Permissions {
		if p.Resource == "" || p.Action == "" {
			return fmt.Errorf("extension: feature %s: permission %q needs a resource and an action", m.Name, p)
		}
	}
	for _, e := range m.Events {
		if e.Type == "" {
			return fmt.Errorf("extension: feature %s: event type is empty", m.Name)
		}
	}
	return nil
}
