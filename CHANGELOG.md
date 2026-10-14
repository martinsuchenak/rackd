# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Changes before v0.0.20 are recorded in the [git history](https://github.com/martinsuchenak/rackd/commits/main)
and [GitHub releases](https://github.com/martinsuchenak/rackd/releases).

## [Unreleased]

### Upgrading from v0.1.0

- Upgrade the server before the CLI. The CLI now calls `/api/v1/...`, which v0.1.0
  servers do not serve; against one it fails with "the server did not answer like a
  rackd /api/v1 server".
- Scripts and integrations that call `/api/...` keep working until the removal date,
  but should move to `/api/v1/...`.
- This release adds a database migration, and v0.1.0 refuses to start on a database
  migrated by a newer Rackd. To go back to v0.1.0, first run
  `rackd migrate rollback --to 20261013120000 --yes` with this release, or restore
  the pre-migration snapshot from `<data-dir>/backups/`.
- Webhook requests use a new format; see [Webhooks](docs/webhooks.md#webhook-delivery).
  Only the test ping was ever sent before, so this matters only for receivers built
  against it:
  - Headers are `X-Rackd-Event`, `X-Rackd-Delivery`, `X-Rackd-Timestamp` and
    `X-Rackd-Signature` instead of `X-Event-Type`, `X-Delivery-ID` and
    `X-Signature-256`.
  - The body is `{"event", "delivery_id", "timestamp", "data"}` instead of
    `{"id", "type", "timestamp", "payload"}`.
  - The signature covers `<X-Rackd-Timestamp>.<body>` instead of the body alone, so
    receivers can reject replayed requests.
- The `route` label on `http_request_duration_milliseconds` is now the route pattern,
  such as `GET /api/v1/devices/{id}`, instead of the raw method and path. Update
  dashboards and alerts that filter on old label values. The duration metrics are
  now typed as gauges, since each value is a mean.

### Added

- An extension API, `pkg/extension`, and a public entry point, `pkg/app`, for
  building Rackd binaries with extra features compiled in. A feature declares its
  permissions, event types and UI entries in a manifest, and can add its own
  migrations (in their own namespace), REST routes under `/api/v1`, MCP tools,
  CLI commands and background work. It reads core data through interfaces that
  apply the caller's permissions, and records its changes in the audit log. The
  API is unstable until 1.0. See [Extensions](docs/extensions.md).
- `rackd migrate status` shows each extension's migrations, including extensions
  recorded in the database but not in the binary; `migrate run` applies them; and
  `migrate rollback --namespace <name>` reverts them. The status output now groups
  migrations under a heading per namespace and ends with `Total: N pending` instead
  of `Total: X migrations, N pending`; update scripts that parse it.
- Event types are a registry instead of a fixed list, so extensions can add types
  that webhooks subscribe to.
- `/api/v1/config` lists the extensions in the binary, with their pages and nav
  items, and reports the edition the binary was built with.

- `tools/loadtest`, a load tool that seeds a server (by default 10,000 devices with
  100,000 addresses) and reports per-operation latency percentiles and errors
  under concurrent clients. See [Performance Testing](docs/performance.md).
- The API is versioned under `/api/v1`. Within v1, changes are backward compatible;
  breaking changes go into a new version. The web UI, the CLI and the OpenAPI
  spec use the new paths. See
  [Versioning and Deprecation](docs/api.md#versioning-and-deprecation).

### Changed

- The server's SQLite database uses a pool of 8 connections instead of one, so
  reads no longer queue behind each other or behind writes. Writes still run one at
  a time: transactions start with `BEGIN IMMEDIATE`, and a connection waits up to 5
  seconds for the write lock. With the load tool's defaults (10,000 devices, 50
  clients), throughput doubled.
- Device lists, searches and related-device lookups load addresses, tags and
  domains for all devices in three queries per 500 devices, instead of three per
  device. A page of 100 devices took about 300 queries. Under the load tool's
  defaults the median page time fell from 3.1 s to 0.29 s.
- An API key's `last_used_at` is updated at most once a minute, instead of on every
  request, so it is accurate to about a minute. Each API-key request used to add a
  database write.
- The dashboard computes network utilization with one grouped query instead of two
  queries per network. A new migration (`20261014120000`) replaces the `network_id`
  index on `addresses` with a `(network_id, ip)` index, so utilization counts read
  only the index. This also speeds up `GET /api/v1/networks/{id}/utilization`.
- The domain types moved from `internal/model` to the public `pkg/model`, so
  extensions can use the same types as core. They are unstable until 1.0.

### Deprecated

- The unversioned `/api/...` paths. They still work as aliases of `/api/v1/...` and
  are removed in the first release on or after 14 April 2027. Their responses carry
  `Deprecation`, `Sunset` and `Link: rel="successor-version"` headers. To migrate,
  insert `/v1` after `/api`.

### Fixed

- Webhooks never delivered events: nothing published them, and the delivery worker
  was never started. Device, network, discovery and conflict changes now publish the
  documented events (bulk operations publish one per item), and the worker runs with
  the server. Deliveries retry with the same delivery ID, and those interrupted by a
  shutdown are retried after the next start. `pool.utilization_high` is not published
  yet.
- Webhook delivery records stored a response code of 200 for every success; they
  now store the actual code. The test ping sent an empty event ID and timestamp.
- Bulk tag add and remove counted a device whose tags failed as both failed and
  succeeded.
- The webhooks guide described fields, per-webhook retry settings and a retry
  schedule that don't exist, said private networks were allowed by default, and
  suggested sending events straight to Slack, which rejects them.
- The server never closed its storage on shutdown, so audit entries still queued
  were lost. It now finishes in-flight requests, stops background work and closes
  the storage, flushing the queue.
- Request metrics were labelled with the raw path and method, so every device ID,
  and every random path or method from any client, added a series that was kept
  for the life of the process. Memory and `/metrics` output grew without bound. The
  `route` label is now the matched route pattern (such as
  `GET /api/v1/devices/{id}`). Requests that match no route share
  `route="unmatched"`, and requests answered before routing (for example by the
  rate limiter) share `route="unrouted"`. Label values in `/metrics` are escaped.
- Duration metrics were stored in whole milliseconds, so sub-millisecond requests
  counted as 0 and averages read low. They are now kept in nanoseconds and exported
  as fractional milliseconds.
- The monitoring guide's p95 panel used `histogram_quantile` on a metric that is an
  average, not a histogram. It now shows the average per route.
- Requests to an unknown API path fell through to the web UI: a GET returned
  `index.html` with status 200, and other methods a plain-text 405. They now return a
  JSON 404, or a JSON 405 with an `Allow` header when the path exists but the method
  is wrong.
- The Nomad job's health check probed `/api/datacenters`, which requires
  authentication, so the service never became healthy. It now uses `/healthz`.

## [0.1.0] - 2026-10-14

### Upgrading from v0.0.21

- The first start writes a verified snapshot of the database to
  `<data-dir>/backups/` before migrating, and up to five are kept. If that directory
  cannot be written (a full disk or read-only storage), the server does not start
  until you set `PRE_MIGRATION_BACKUP=false`.
- `rackd backup --output` no longer overwrites an existing file. Scripts that write
  to a fixed path need `--overwrite`.
- `rackd backup` now reads the database through SQLite instead of copying the file.
  While the server is stopped, the user running it needs write access to the data
  directory, because SQLite creates a `-shm` file there. Read access to `rackd.db`
  alone is no longer enough.
- Migration history is now recorded in `xdal_migrations`. `schema_migrations` is
  kept but no longer updated, so anything that reads the schema version from it
  sees v0.0.21's history.
- Creating a DNS provider of type `powerdns` or `bind` now returns 400. These types
  never worked; existing ones can still be edited and deleted.
- v0.0.21 can still open an upgraded database. To go back completely, restore the
  pre-migration snapshot with `rackd backup restore` before reinstalling v0.0.21.
- Releases now also publish `*.sbom.json` and `*.sigstore.json` files next to each
  archive. Download scripts that match assets by platform name alone may need a
  tighter pattern, such as `*_linux_amd64.tar.gz`.

### Added

- Signed releases.
  - `checksums.txt` and each SBOM get a keyless [cosign](https://github.com/sigstore/cosign)
    signature bundle (`*.sigstore.json`), using the release workflow's GitHub identity.
  - Each archive gets an SPDX SBOM (`*.sbom.json`) generated by syft.
  - The release carries a GitHub build provenance attestation.
  - The signing permissions are scoped to the release job, and every action in it
    is pinned to a commit SHA.
  - [Verifying a release](docs/installation.md#verifying-a-release) shows the
    `cosign verify-blob` and `gh attestation verify` commands.
- `SECURITY.md`: supported versions, private reporting, and disclosure.
- CI runs `govulncheck`. Dependabot proposes weekly updates for Go modules, the web
  UI (Bun), GitHub Actions and the Docker base images, with minor and patch updates
  grouped.

- The `rackd dns` CLI command group (providers, zones, sync, import, records) is now
  available. It existed but was never registered.
- `CONTRIBUTING.md`, and a Contributor License Agreement (`CLA.md`) that a GitHub
  workflow asks first-time contributors to sign on their pull request.
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
- `rackd backup --overwrite` replaces an existing output file, for scripts that
  write to a fixed path. The new backup is written and verified next to it first,
  so a failed backup leaves the previous file in place.
- A CI workflow for pull requests and pushes to `main`:
  - Go: formatting, `go mod tidy`, vet, golangci-lint and the full test suite.
  - A separate job runs the tests with the race detector.
  - Web UI: typecheck, the OpenAPI type check and unit tests.
- Upgrade tests against a real v0.0.21 database. A reversibility test applies each
  migration, reverts it, and checks that the schema and row counts match the state
  before it ran.

### Changed

- **Licence:** Rackd is now licensed under the Apache License 2.0, which adds an
  explicit patent grant; see `LICENSE` and `NOTICE`. Releases up to and including
  v0.0.21 remain available under the MIT License.
- Creating or updating a DNS provider now accepts only `technitium`. The `powerdns`
  and `bind` types were accepted before but failed with "not yet implemented"
  whenever the provider was used. They are planned, and the API now says so.
  Providers already stored with those types can still be edited and deleted.
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

### Removed

- The stale `webui/package-lock.json`. `bun.lock` is the web UI's only lockfile.

### Security

- Build with Go 1.27.2, which fixes nine standard library vulnerabilities that
  rackd's code reaches. These include HTTP/2 server crash, CPU and memory
  exhaustion issues in `net/http`, HTTP/1 connection desynchronisation after
  CONNECT, MIME header memory limits, and malformed ECH handling in `crypto/tls`
  (GO-2026-6603, 6605, 6607, 6608, 6610, 6611, 6612, 6613 and 6617). v0.0.21 was
  built with Go 1.27.1 and is affected.

### Fixed

- `rackd backup` copied the live database file and its WAL file separately, so a
  backup taken while the server was writing could be inconsistent. It now uses
  SQLite's `VACUUM INTO`, which produces one self-contained file. The backup is
  verified, written with mode `0600`, and does not overwrite an existing file unless
  `--overwrite` is given.
- Rolling back a migration now fully reverses it. Previously these Down steps left
  changes behind:
  - Device `hostname`, the device status columns, relationship `notes` and DNS
    record `address_id` stayed in place.
  - The `api_keys` rebuild lost its primary key, NOT NULL and UNIQUE constraints.
  - The DNS rollback left seven permissions behind.
  - The audit/logs rollback left the operator's `audit:list` grant behind.
- Discovery's ARP table was read and replaced by concurrent scans and the startup
  load without locking, a data race on Linux. It is now swapped in and read under a
  lock.
- Logging before the logger was initialised (for example from CLI commands and tests)
  panicked. Messages are now discarded until initialisation.
- The README listed Cloudflare, Route53 and PowerDNS as DNS providers, and custom
  fields on networks. Only Technitium and custom fields on devices exist. The DNS
  guide and the architecture page no longer describe PowerDNS, BIND or a feature
  interface as available.
- `docker build` failed because the Dockerfile installed Bun from Alpine's edge
  repository, which no longer packages it. The web UI is now built in an
  `oven/bun` stage, a `.dockerignore` keeps local data and build output out of the
  image, and CI builds the image on every pull request.
  - The runtime image is pinned to `alpine:3.24`.
  - Pass `--build-arg VERSION=... --build-arg COMMIT=...` (or set `RACKD_VERSION`
    and `RACKD_COMMIT` for `docker compose`) so `rackd version` reports the build.
- The Docker image and `docker-compose.yml` health checks probed
  `/api/datacenters`, which requires authentication, so the container never became
  healthy. They now use `/healthz`.
- The README and the installation guide downloaded bare binaries such as
  `rackd-linux-amd64`, which releases do not publish, so every download link
  returned 404. They now download the versioned archives.
- The installation guide also referred to a Docker Hub image that does not exist,
  `RACKD_`-prefixed environment variables, Go 1.25 and Make. It now builds the image
  locally, uses the real variable names, and requires Go 1.27 and Task.
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

[Unreleased]: https://github.com/martinsuchenak/rackd/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/martinsuchenak/rackd/compare/v0.0.21...v0.1.0
[0.0.21]: https://github.com/martinsuchenak/rackd/compare/v0.0.20...v0.0.21
[0.0.20]: https://github.com/martinsuchenak/rackd/compare/v0.0.19...v0.0.20
