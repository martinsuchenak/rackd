# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Changes before v0.0.20 are recorded in the [git history](https://github.com/martinsuchenak/rackd/commits/main)
and [GitHub releases](https://github.com/martinsuchenak/rackd/releases).

## [Unreleased]

### Added

- Automatic backup before migrations. When the server starts or `rackd migrate run`
  runs with pending migrations on an existing database, a verified snapshot is written
  to `<data-dir>/backups/` first. The five newest are kept. If the snapshot fails, no
  migration runs. Set `PRE_MIGRATION_BACKUP=false` to skip the snapshot, for example
  on a full disk.
- `rackd migrate rollback --steps N` / `--to VERSION` reverts migrations after taking
  the same snapshot. It requires `--yes`, because some Down steps drop data.
  The target is validated first, and nothing is written when there is nothing to
  revert.
- `rackd backup verify --file F` checks a backup's integrity and migration history, and
  warns when it was made by a newer Rackd.
- `rackd backup restore --file F --yes` replaces the database with a verified backup.
  The backup is first consolidated into a single file, so a backup copied together
  with its `-wal` file keeps all its data. The restored database is left in WAL mode.
  The current database is kept as `rackd.db.pre-restore-<timestamp>`. The command
  refuses to run while the server has the database open, and refuses backups from a
  newer Rackd.
- `rackd migrate run --no-backup` skips the pre-migration snapshot.
- A CI workflow for pull requests and pushes to `main`:
  - Go: formatting, `go mod tidy`, vet, golangci-lint and the full test suite.
  - A separate job runs the tests with the race detector.
  - Web UI: typecheck, the OpenAPI type check and unit tests.
- Upgrade tests against a real v0.0.21 database. A reversibility test applies each
  migration, reverts it, and checks that the schema and row counts match the state
  before it ran.

### Changed

- Migrations run on [xdal](https://github.com/martinsuchenak/xdal) `pkg/migrate`
  (v0.1.0). This is groundwork for PostgreSQL and MySQL support.
  - Applied migrations are recorded in `xdal_migrations` under the `core` namespace.
  - On first start, the existing history is copied from `schema_migrations`, once:
    a marker row stops a later full rollback from triggering the copy again. The old
    table is left in place so an older binary can still open the database.
- `rackd migrate status` also reports dirty migrations, and migrations applied by a
  newer Rackd. It never writes to the database; before the first upgrade it reads
  the old `schema_migrations` table directly.
- A database migrated by a newer Rackd is refused at startup with a clear error,
  instead of being run against an older schema definition.
- Per-migration log lines are now at debug level. One summary line is logged at info.
- The `scan_profiles`, `scheduled_scans` and `credentials` tables are now created by a
  versioned migration (`20261013120000`) instead of when their stores start.
  - Existing tables and data are kept.
  - The default scan profiles are seeded once. A deleted default profile no longer
    comes back on the next restart.
  - Opening one of these stores before migrations have run now returns a clear error.

### Fixed

- `rackd backup` copied the live database file and its WAL file separately, so a
  backup taken while the server was writing could be inconsistent. It now uses
  SQLite's `VACUUM INTO`, which produces one self-contained file. The backup is
  verified, written with mode `0600`, and never overwrites an existing file.
- Rolling back a migration now fully reverses it. Previously these Down steps left
  changes behind:
  - Device `hostname`, the device status columns, relationship `notes` and DNS
    record `address_id` stayed in place.
  - The `api_keys` rebuild lost its primary key, NOT NULL and UNIQUE constraints.
  - The DNS rollback left seven permissions behind.
  - The audit/logs rollback left the operator's `audit:list` grant behind.
- Logging before the logger was initialised (for example from CLI commands and tests)
  panicked. Messages are now discarded until initialisation.
- The backup guide referred to `rackd server --dry-run`, `--init-db` and `/api/health`,
  which do not exist. It now uses the real commands and `/healthz` / `/readyz`.

## [0.0.21] - 2026-10-13

### Added

- This changelog. Each release's section is published as its GitHub release
  notes and on the [documentation site](https://getrackd.dev/changelog/).

### Changed

- RBAC permission checks now use the [go-rbac](https://github.com/martinsuchenak/go-rbac)
  engine (v0.1.0). Role and permission definitions are kept in memory and reloaded
  whenever a role or permission changes; permissions still match exactly
  (wildcard grants are disabled).
- The permissions reported for a user (UI configuration, OAuth scope checks) are now
  derived from the same source as enforcement, so they always agree.
- DNS provider (Technitium) calls and webhook test pings now time out after 20s
  instead of 30s, so failures are reported before the request timeout.
- The network topology graph uses Cytoscape's default scroll-zoom sensitivity
  (removes the "custom wheel sensitivity" console warning).
- Updated Go dependencies (including `paularlott/mcp` v0.32.4, `paularlott/cli`
  v0.10.1, `modernc.org/sqlite` v1.60.1, `golang.org/x/crypto` v0.58.0) and web UI
  dependencies (Alpine.js 3.17.4, Cytoscape 3.34.3, Playwright 1.64.0). Run
  `bun run test:e2e:install` to fetch the matching Playwright browser.

### Fixed

- Requests running longer than 15s received an empty reply because the HTTP server
  write timeout was shorter than `REQUEST_TIMEOUT`. The write timeout now outlasts
  the request timeout (`REQUEST_TIMEOUT` + 5s).
- DNS provider "Test Connection" failures against unreachable servers are shown
  again (502 `PROVIDER_UNREACHABLE` with the cause) instead of hanging.
- OAuth client registration without a `scope` returns 400
  `invalid_client_metadata` instead of a 500 server error.
- Approving an OAuth authorization whose requested scopes are not permitted for the
  user returns `invalid_scope` instead of `server_error`.
- Deleting a permission now revokes it immediately rather than at the next role
  change or restart.

### Removed

- Internal `HasPermission`, `HasAnyPermission` and `HasAllPermissions` helpers,
  replaced by go-rbac.

## [0.0.20] - 2026-09-19

### Changed

- Build tooling moved from a Makefile to a Taskfile.
- Go bumped to 1.27.1 and dependencies updated.
- Tests can lower the bcrypt cost via `RACKD_BCRYPT_COST` for faster runs.

### Fixed

- Resolved all golangci-lint findings.

[Unreleased]: https://github.com/martinsuchenak/rackd/compare/v0.0.21...HEAD
[0.0.21]: https://github.com/martinsuchenak/rackd/compare/v0.0.20...v0.0.21
[0.0.20]: https://github.com/martinsuchenak/rackd/compare/v0.0.19...v0.0.20
