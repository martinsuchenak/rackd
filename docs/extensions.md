# Extensions

Rackd features can be built as extensions on `pkg/extension` and compiled into a Rackd binary with `pkg/app`. An extension adds its own tables, REST endpoints, permissions, events, MCP tools, CLI commands and background work, and reaches core data only through the interfaces Rackd gives it.

The API is unstable until Rackd 1.0: it may change in any release, and changes are listed in the [changelog](../CHANGELOG.md). Extensions are compiled in. Rackd loads nothing from disk or the network at runtime.

## Building a Binary

A binary with extensions is a `main` package that passes them to `app.Main`, which builds the CLI with all of core's commands and the server:

```go
package main

import (
	"github.com/martinsuchenak/rackd/pkg/app"
	"github.com/martinsuchenak/rackd/pkg/extension"

	"example.com/my-rackd/inventorytags"
)

var version = "dev"

func main() {
	app.Main(app.Options{
		Version:  version,
		Edition:  "custom", // shown in /api/v1/config
		Features: []extension.Feature{inventorytags.New()},
	})
}
```

Pin the Rackd version in `go.mod`, and build each release against a Rackd tag.

## Writing a Feature

A feature implements `extension.Feature`, which returns its manifest, plus any of the optional interfaces it needs:

| Interface | Method | What Rackd does with it |
| --- | --- | --- |
| `Feature` | `Manifest() Manifest` | Validates the name, permissions, events and UI entries |
| `MigrationProvider` | `Migrations() []Migration` | Runs them in the feature's own namespace, after core's |
| `RouteProvider` | `RegisterRoutes(Router, Host)` | Mounts the routes under `/api/v1` |
| `ToolProvider` | `RegisterTools(ToolRegistry, Host)` | Adds the MCP tools |
| `CommandProvider` | `Commands() []*cli.Command` | Adds top-level CLI commands |
| `Starter` | `Start(ctx, Host) error` | Calls it once the server is ready; `ctx` ends at shutdown |

### Manifest

```go
func (f *Feature) Manifest() extension.Manifest {
	return extension.Manifest{
		Name:    "inventory-tags", // [a-z][a-z0-9-]*
		Version: "0.1.0",
		Permissions: []extension.Permission{
			{Resource: "inventory-tags", Action: "list", DefaultRoles: []string{"admin", "operator", "viewer"}},
			{Resource: "inventory-tags", Action: "create", DefaultRoles: []string{"admin", "operator"}},
		},
		Events: []extension.EventType{{Type: "inventory_tag.created", Label: "Inventory Tag Created"}},
		UI: extension.UI{
			Pages: []extension.Page{{Path: "/inventory-tags", Title: "Inventory Tags"}},
			Nav:   []extension.NavItem{{Label: "Inventory Tags", Path: "/inventory-tags", Icon: "tag", Order: 95}},
		},
		Instructions: "Use inventory_tag_list to find a device's inventory tags.",
	}
}
```

- **Permissions** are added to RBAC when the server starts. A new permission is granted to its `DefaultRoles` once, when it is first added; later changes administrators make are kept. A route or tool can also require one of core's permissions; don't declare those, since an existing permission's `DefaultRoles` are not applied.
- **Events** become event types that webhooks can subscribe to. A feature can't declare one of core's types or another feature's. Publishing an undeclared type is logged and dropped.
- **UI** entries are reported in `/api/v1/config` as `pages` and `nav_items`.
- **Instructions** is one line added to the MCP server instructions.

Rackd refuses to start when two features share a name, a declared permission, an event type, a route, a tool name or a command name, or when one clashes with core.

Removing a feature from the binary leaves its tables, its migration records and its permissions in the database, so adding it back restores everything.

### Migrations

A feature's migrations are [xdal](https://github.com/martinsuchenak/xdal) migrations, recorded in the `xdal_migrations` table under the feature's name. Use the schema builder (`tx.Schema()`) so they stay portable:

```go
func (f *Feature) Migrations() []extension.Migration {
	return []extension.Migration{{
		Version: "20261014000000",
		Name:    "create_inventory_tags",
		Up: func(ctx context.Context, tx *migrate.Tx) error {
			s := tx.Schema()
			return tx.Apply(ctx, s.CreateTable("inventory_tags").
				Column(xdal.NewColumn("id", xdal.TypeAutoIncrement)).
				Column(xdal.NewColumn("device_id", xdal.TypeString).Size(64).NotNull()).
				Column(xdal.NewColumn("tag", xdal.TypeString).Size(255).NotNull()))
		},
		Down: func(ctx context.Context, tx *migrate.Tx) error {
			return tx.Apply(ctx, tx.Schema().DropTable("inventory_tags"))
		},
	}}
}
```

- Prefix table names with the feature's name.
- Don't add foreign keys to core tables. Core may rebuild its tables in a migration, and a feature must not depend on core's schema. Refer to core records by ID and read them through `Host.Core`.
- When a feature already has migrations applied, Rackd backs up the database before running new ones, as it does for core.

### Routes

```go
func (f *Feature) RegisterRoutes(r extension.Router, host extension.Host) {
	r.Handle("GET /devices/{id}/inventory-tags", f.list(host), extension.Require("inventory-tags", "list"))
	r.Handle("GET /inventory-tags/health", f.health, extension.Public())
}
```

- Patterns are `net/http` patterns relative to `/api/v1`. Extension routes have no deprecated `/api/...` alias.
- Routes require an authenticated user (session cookie or API key) unless `Public()` is given, and the body is limited to 1 MiB. `extension.CallerFrom(r.Context())` returns the caller. A public route has no caller, so `Public()` can't be combined with `Require`.
- `Require(resource, action)` checks the caller's permission before the handler runs.
- `WriteJSON`, `WriteError` and `DecodeJSON` produce Rackd's standard responses. `WriteError` maps `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrForbidden` and `ErrUnauthenticated` to their statuses; wrap them for detail: `fmt.Errorf("%w: body is required", extension.ErrValidation)`.

### Host

| Method | Use |
| --- | --- |
| `Core()` | Core data (`Devices()`, `Networks()`), with the permissions of the caller in the context. It grows as features need more. |
| `DB()` | The database, for the feature's own tables. |
| `Audit(ctx, action, resource, id, details)` | Records a change to the feature's data in Rackd's audit log, attributed to the caller. Core data changed through `Core()` is audited already. |
| `Authorize(ctx, resource, action)` | A permission check inside a handler or tool. |
| `Events()` | `Publish` a declared event type; `Subscribe` to all events until shutdown. |
| `Logger()` | A `*slog.Logger` that writes through Rackd's logs, tagged with the feature's name. |
| `Go(name, fn)` | Runs background work until shutdown; shutdown waits up to 10 seconds for it. Work started once shutdown has begun is not run. |

Background work runs without a user. Use `extension.SystemContext(ctx, "my-feature-job")` for calls into `Core()` from it.

### MCP Tools and CLI Commands

```go
func (f *Feature) RegisterTools(t extension.ToolRegistry, host extension.Host) {
	t.Register(mcp.NewTool("inventory_tag_list", "List a device's inventory tags",
		mcp.String("device_id", "Device ID", mcp.Required())),
		f.listTool(host), extension.Require("inventory-tags", "list"))
}
```

Tools run as the MCP caller. CLI commands run without a server; to call the API they can use `github.com/martinsuchenak/rackd/cmd/client`, which reads the same configuration as Rackd's own commands.

## Current Limits

- `rackd migrate status` and `rackd migrate rollback` cover core's migrations only.
- The web UI shows extension nav items, but extension pages and their scripts can't be added to the UI build yet.
