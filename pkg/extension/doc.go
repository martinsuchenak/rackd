// Package extension is the API for adding features to Rackd.
//
// A Feature is compiled into a Rackd binary and handed to app.Main. It
// declares itself with a Manifest, and implements any of the optional
// interfaces (MigrationProvider, RouteProvider, ToolProvider, CommandProvider,
// Starter) for the parts of Rackd it extends. Rackd runs a feature's
// migrations in their own namespace, adds its permissions to RBAC, mounts its
// routes under /api/v1 with Rackd's authentication, registers its MCP tools
// and CLI commands, and starts and stops it with the server.
//
// Features reach core data only through Host, which applies the calling
// user's permissions, and keep their own data in their own tables.
//
// Rackd's own built-in features use this API too. Until Rackd 1.0 it is
// unstable: it may change in any release, with changes listed in the
// changelog.
package extension
