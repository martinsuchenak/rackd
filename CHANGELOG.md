# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Changes before v0.0.20 are recorded in the [git history](https://github.com/martinsuchenak/rackd/commits/main)
and [GitHub releases](https://github.com/martinsuchenak/rackd/releases).

## [Unreleased]

### Added

- Releases publish a signed multi-arch Docker image (`linux/amd64`,
  `linux/arm64`) to `ghcr.io/martinsuchenak/rackd`, tagged with the version,
  the minor version and `latest`, and built from the release binaries. The
  deployment and installation guides use it, and no longer list the removed
  `API_AUTH_TOKEN`/`MCP_AUTH_TOKEN` settings or health-check an endpoint
  that needs signing in.
- The macOS binaries are signed with a Developer ID certificate and
  notarized by Apple, so Gatekeeper no longer blocks a downloaded `rackd`.
- Scheduled backups: with `BACKUP_INTERVAL` set (for example `24h`), the
  server takes a verified backup that often into `BACKUP_DIR` (default
  `<DATA_DIR>/backups`) and keeps the newest `BACKUP_KEEP` (default 7). The
  schedule follows the newest backup on disk, so restarts don't take extra
  ones, and a failed backup is retried after at most an hour. Off by default.

### Fixed

- `rackd server` ignored `DATA_DIR`, `LISTEN_ADDR`, `LOG_LEVEL` and
  `LOG_FORMAT`: its flags defaulted to `./data`, `:8080`, `info` and `text`
  and always overrode them. `rackd backup`, `migrate`, `credentials
  rotate-key` and `user reset-password` likewise ignored `DATA_DIR`. A flag
  now overrides its variable only when given. `--discovery-interval` was
  accepted but never used; it now sets the interval.

  **Before upgrading**, check any of these variables you set without the
  matching flag: they now take effect. If `DATA_DIR` points somewhere other
  than `./data`, move the database there first, or the server refuses to
  start rather than create an empty one while `./data/rackd.db` exists.

  **Docker:** the image, `docker-compose.yml` and the deployment guide set
  `DATA_DIR=/data` and mount the volume there, but the database has been
  written to `/app/data` inside the container, which is lost when the
  container is recreated. Before upgrading, copy it into the volume from the
  running container, for example `docker cp rackd:/app/data/rackd.db ./rackd.db`
  then into the directory mounted at `/data`. Once upgraded, rackd uses
  `/data`.
- The docs still described settings, commands, flags and endpoints that don't
  exist. `API_AUTH_TOKEN` and `MCP_AUTH_TOKEN` are gone from every example:
  people sign in with user accounts, and scripts, the CLI and MCP clients use
  API keys (or OAuth for MCP). CLI examples now match
  [CLI Reference](docs/cli-reference.md). Endpoints, request and response
  fields, permission names, error bodies and the release download names now
  match the code. The docs no longer promise features that aren't there, such
  as SSH/SNMP credentials being used by scans or AAAA records.
  [Configuration](docs/configuration.md) now points to
  [Configuration Reference](docs/configuration-reference.md), which adds
  `ENCRYPTION_KEY`, `POOL_UTILIZATION_ALERT_PERCENT`, how `.env` is loaded and
  what stops the server from starting.

### Security

- Built with Go 1.27.3, which distrusts compromised intermediate certificates
  in `crypto/x509` ([GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809)).
  Rackd checks certificates when it delivers webhooks and talks to DNS
  providers.

## [0.5.1] - 2026-10-20

### Fixed

- Conflicts stayed listed after the devices or networks in them were deleted
  or changed. Listing, summarising or detecting conflicts now brings the open
  ones up to date: one that no longer applies is resolved with a note saying
  why, and one whose devices or networks changed lists the current ones.
- One device with the same IP on two addresses was detected as a duplicate
  IP, and device names containing a comma were split in two.
- **Detect Conflicts** in the web UI emptied the list until the page was
  reloaded; it now shows the recorded conflicts. The Conflicts page lists
  active conflicts only, matching its counts.
- A device page for a device that doesn't exist reloaded it ten times a
  second, flickering between "Loading..." and the error.

## [0.5.0] - 2026-10-20

### Added

- API keys can be limited to some of their owner's permissions with
  `scopes` (`resource:action`, each one the creator holds). A scoped key is
  checked against its scopes as well as its owner's permissions, on the REST
  API and over MCP. The web UI's Create API Key dialog offers "All my
  permissions" or "Only the permissions I choose", the API Keys page shows each
  key's access and notes keys with full access, and `rackd apikey create` takes
  `--scopes`. Keys without scopes keep all of their owner's permissions, as
  before.

- `SSRF_ALLOWED_TARGETS` lets webhooks and DNS providers reach just the
  private hosts or ranges you list (e.g. `192.168.10.53,10.20.0.0/24`)
  instead of every private range, as `SSRF_ALLOW_PRIVATE_TARGETS=true` does.
  Entries must be inside a private range; loopback and link-local addresses
  stay blocked.

### Fixed

- **Security:** creating an API key ignored fields it didn't know, so a
  request asking for `permissions`, `role` or `is_admin` got a key with all of
  its creator's permissions and no error (found by a Strix scan). Unknown
  fields are now refused with `400`. The authentication guide also described a
  `--user-id` flag that doesn't exist: a key always belongs to its creator.

- Device addresses bound to no pool, which the web UI's device form creates,
  were invisible to pools: a device could take a reserved address without
  claiming the reservation, an address a device held could be reserved, and
  the next free address and the heatmap could offer it. An address now
  counts toward a pool when its IP is in the pool's range and it is on the
  pool's network or on no network.
- Two enabled NAT mappings could forward the same external IP, port and
  protocol to different targets. Creating or updating onto an endpoint an
  enabled mapping uses now answers `409` naming it; port `0` (every port)
  and protocol `any` count as overlapping.

- The users page showed no roles until a user's roles were opened in the
  role manager: `GET /api/v1/users` left them out, though the API reference
  lists them. Each user now comes with its roles, loaded in one query.

### Security

- The OAuth approval is bound to what the consent screen showed. The screen
  now lists only the scopes the client and the signed-in user can actually
  grant, and approving checks a signed consent token, so a changed scope,
  redirect URI or PKCE challenge in the approval is refused with `400`
  instead of granting more than the user saw. When the user holds none of
  the requested permissions, the consent page says so instead of offering an
  approval that granted the client's other registered scopes.

- The viewer role can no longer test DNS providers: a test makes the server
  call the provider's endpoint. Testers who can't edit the provider see that
  the test failed but not why, since the cause can include the endpoint's
  own response; the cause is logged. Likewise, webhook delivery records show
  the endpoint's response code, body and error only to those who can edit
  the webhook.

## [0.4.0] - 2026-10-19

### Added

- Discovered hosts that are already in the inventory are recognised: a device
  with an address on the host's IP (on its network, or with no network) is a
  match, as is the device a host was promoted to. The discovery list shows
  "In inventory: <device>" instead of Promote, a **New only** filter hides
  them, the API returns `matched_device_id` and `matched_device_name`, and
  `rackd discovery list` has an IN INVENTORY column.

### Fixed

- Promoting a host that was already in the inventory created a second device
  with the same address; it now answers `409` naming the device.
- The dashboard counted hosts already in the inventory as discovered, and
  called a device stale even when every scan found it, unless it had been
  promoted from discovery.
- The discovery guide described automatic promotion rules and `promote`
  flags that don't exist.

## [0.3.1] - 2026-10-16

### Changed

- CI's race-detector job is several times faster: tests copy a database
  migrated once (`internal/storage/storagetest`) instead of migrating a new
  one each time, and the mDNS test no longer waits on every network
  interface. `task release` now waits for CI to pass on the release commit
  instead of rerunning the suite locally.

### Fixed

- A discovery scan could make the whole server stop responding on slow or
  network storage: every scan worker wrote the scan's progress after each
  host, and saved what it found, at the same time, which took every database
  connection, so even `/api/v1/config` timed out with a 503. A scan now saves
  results from one goroutine and writes progress every two seconds, so it
  holds at most two connections.

## [0.3.0] - 2026-10-15

### Changed

- The web UI's menu is grouped into sections: Organization, Inventory, IPAM,
  DNS, Discovery, Integrations and Administration, with Dashboard above them.
  Sections collapse, and the browser remembers which. On a phone, only the
  current page's section starts open, and a collapsed section shows its
  conflict count on the heading. Inside a section, labels are shorter: DNS
  Zones and DNS Providers are Zones and Providers, and Discovery is Scans.
- Extension menu items can name a `Section`: a core section to join, or their
  own. Without one they go in an Extensions section. `Order` now sorts items
  within their section. An extension item is hidden when its path is a core
  page's, or its label is a core item's in the same section.

## [0.2.1] - 2026-10-15

### Fixed

- Pages could take a minute to load on slow or network storage. Every API
  request wrote the session's new expiry, so a page load's parallel requests
  queued for SQLite's write lock while holding database connections, and
  reads waited behind them. The expiry is now written at most once a minute,
  and SQLite commits with `synchronous=NORMAL` (the recommended setting with
  WAL), so commits no longer wait for the disk to sync. A power cut can lose
  the last few transactions, but not corrupt the database.
- On Linux, Rackd warns at start when the data directory is on a network
  filesystem (NFS, SMB/CIFS), where SQLite is slow and its locking unreliable.

## [0.2.0] - 2026-10-15

### Upgrading from v0.1.0

- Global search (`/api/v1/search`, the search box) now requires `search:read`.
  The built-in roles have it; add it to custom roles that should search.
- `DISCOVERY_MAX_CONCURRENT` and `DISCOVERY_TIMEOUT` are no longer read (they
  never did anything); remove them from your configuration.
- **Audit retention now applies.** On the first start, audit entries older than
  `AUDIT_RETENTION_DAYS` (default 90) are deleted. To keep your full history, set
  `AUDIT_RETENTION_DAYS=0` before upgrading, or export it first
  (`rackd audit export`).
- Rackd now refuses to start if `ENCRYPTION_KEY` is set but malformed. Before,
  it ran without encryption. The docs' `RACKD_ENCRYPTION_KEY` was never read; use
  `ENCRYPTION_KEY`.
- On start, the database files are made readable only by Rackd's user. If
  another user (a backup agent, say) reads them directly, use `rackd backup`
  instead.
- `rackd credentials rotate-key --new-key` is replaced by `NEW_ENCRYPTION_KEY`
  or `--new-key-file`, plus `--yes`.
- MCP clients see error codes -32001 (unauthenticated), -32003 (forbidden),
  -32004 (not found) and -32009 (conflict), as well as -32602 and -32603.
- Login (`POST /api/v1/auth/login`) now requires `Content-Type: application/json`.
  Scripts that post JSON without that header get `415`.
- Sessions now end 7 days after sign-in even if used every day; set
  `SESSION_MAX_LIFETIME` to change this, or `0` for the old behaviour.
- `is_admin` now always matches the admin role. A migration sets it for users who
  hold the role and clears it for users who don't, so users created with
  `is_admin` but never given the role (and who never had admin rights) no longer
  show as admins. Grant them the admin role if they should be admins.
- An `id` sent when creating something, including the `id` column of a CSV
  import, must now be a UUID. Imports of Rackd's own exports are unaffected;
  clear hand-written ids such as `dev-1` and let Rackd assign them.
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
- `/metrics` now needs the new `metrics:read` permission rather than just a login.
  The built-in roles have it. Give it to the role of the user your Prometheus
  scraper's API key belongs to if that is a custom role.

### Added

- Every list endpoint accepts `limit` (at most 1000) and `offset`. Device lists,
  audit logs, logs and webhook deliveries return 100 results unless `limit` says
  otherwise; other lists return everything when no page is asked for, as the API
  reference says.
- An extension API, `pkg/extension`, and a public entry point, `pkg/app`, for
  building Rackd binaries with extra features compiled in. A feature declares its
  permissions, event types and UI entries in a manifest, and can add its own
  migrations (in their own namespace), REST routes under `/api/v1`, MCP tools,
  CLI commands and background work. It reads core data through interfaces that
  apply the caller's permissions, and records its changes in the audit log. The
  API is unstable until 1.0. See [Extensions](docs/extensions.md).
- Extensions can add to the web UI. A feature embeds a JavaScript module and
  stylesheets, which the UI loads at startup. The module draws the feature's pages
  and adds content to slots on the dashboard and on device, network and
  datacenter pages, through `window.rackd`. Pages get their titles and
  permission checks from the manifest. A binary can also replace the built-in UI
  with its own (`app.Options.UI`). `examples/hello` is a complete example.
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

- The README no longer warns against production use. It states what this release
  promises instead: `/api/v1` stays backward compatible, upgrades back up the
  database first, and only the latest release gets security fixes.
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

- Reservations (found by a Strix scan):
  - Auto-assigning a reservation (`POST /api/v1/reservations` without
    `ip_address`) failed with "high contention" once the pool had any
    reservation, because the next free address ignored reservations. It now
    skips reserved addresses, and so does `GET /api/v1/pools/{id}/next-ip`.
  - Expired reservations never expired. Their address is now free as soon as
    the expiry passes, and the status changes to `expired` within a minute.
    Expiry times given with a UTC offset other than zero are compared
    correctly.
  - A device taking a reserved address left the reservation `active`. It
    becomes `claimed` when the device is created or updated with that address
    in the pool.
  - The reservations guide described a configuration block, webhook events and
    a conflict type that don't exist; it now matches what Rackd does.
- Concurrent writes could record the same thing twice (found by a Strix scan):
  - Device writes for one address at the same moment each recorded an active
    duplicate-IP conflict, so resolving one left the others. There can now be
    only one active duplicate-IP conflict per address. On upgrade, extra
    active records are resolved by `system` as duplicates, keeping the first
    detected.
  - A duplicate IP that came back after its conflict was resolved was never
    recorded again. It now gets a new conflict; ignored conflicts still stay
    ignored.
  - Overlapping scans of a network each recorded the same host. A network now
    has one discovered-device record per address. On upgrade, duplicates are
    removed, keeping the promoted record or else the one seen last.
- Bad input and references to missing records got a 500 instead of a 4xx, on
  the REST API and over MCP. Credentials, scan profiles and scheduled scans that
  fail validation, a missing `user_id` or `role_id` when granting a role, and a
  relationship to a device that doesn't exist now answer 400; scanning or
  testing something that doesn't exist answers 404; a zone import the DNS
  provider fails answers 502, like the provider test. Deleting a user that
  doesn't exist answers 404 like every other resource, instead of 204.
- A scan profile created without `timeout_sec` or `max_workers` was rejected;
  they default to 30 seconds and 10 workers. A scan profile now needs a name.
- A NUL byte in a search query caused a 500.
- The web UI showed OAuth Clients in the menu when MCP OAuth was off, and the
  page failed with a 404. `/api/v1/config` now reports `mcp_oauth`, and the
  page and its menu entry appear only when it's on.
- The web UI hid the DNS zone Sync and Import buttons from operators, and the
  provider Test button from viewers and operators. They checked
  `dns-zone:update` and `dns-provider:update` instead of the `sync`, `import`
  and `test` permissions the server uses. The Circuits and NAT nav entries now
  show only to users who can list them.
- `pool.utilization_high` could be subscribed to but never fired. It now fires
  when a pool's utilization reaches `POOL_UTILIZATION_ALERT_PERCENT` (default
  90; `0` turns it off), checked at each snapshot, once per crossing.
- `search:read` was granted but never checked; global search now requires it, as
  well as each kind's list permission, and reports errors instead of returning
  empty results. `dns:import`, which nothing checked, is removed.
- The pool form's Gateway and DNS Servers fields were silently discarded; they
  are removed.
- Users, roles and API keys are timestamped in UTC like everything else.
- `DISCOVERY_MAX_CONCURRENT` and `DISCOVERY_TIMEOUT` were read but never used;
  they are removed. Discovery concurrency and timeouts are set in scan profiles.
- The CLI documentation listed flags and commands that don't exist. Every command
  and flag is now in `docs/cli-reference.md`, generated from the code
  (`go run ./tools/clidoc`), with a test that fails when it falls behind. The
  docs no longer mention settings Rackd doesn't read (`RACKD_*` server variables,
  `--api-auth-token`, `rackd mcp`, `rackd completion`). The MCP guide describes
  connecting over HTTP, and exit codes are documented as `0` or `1`.
- `rackd conflict resolve` always failed: it posted to `/api/v1/conflicts/resolve`,
  which doesn't exist. It now uses `/api/v1/conflicts/{id}/resolve`. That route
  ignored its `{id}` and resolved whatever `conflict_id` the body named; it now
  resolves the conflict in the path, and refuses a body naming another one.
- Reservations: create took only `expires_in_days` and update only `expires_at`,
  so `rackd reservation update --expires` did nothing. Both now take either form
  (and refuse both at once).
- The CLI printed only "HTTP 400" for validation errors, because it couldn't decode
  the `details` list. It now shows the code, the message and each field.
- `rackd device list` returned at most 100 devices, and `--query` was ignored. It
  now fetches every page unless `--limit` is given, adds `--offset`, and searches
  with `--query`.
- Duplicate names and other unique fields now return `409 ALREADY_EXISTS`
  everywhere. Before, roles, circuits and user emails gave `500`, and scan
  profiles and scheduled scans gave `400` with the database's message.
  Deleting an unknown OAuth client or conflict now returns `404` instead of `500`
  or `204`.
- Scheduled scans with an invalid cron expression, no name or no network are
  refused when saved, instead of being stored and failing silently in the
  scheduler. The documented 5-minute minimum interval is now enforced.
- A reservation's expiration can be removed (`clear_expiry`, or
  `rackd reservation update --expires 0`).
- The API reference's error section, and the OpenAPI conflict and reservation
  schemas, described fields and codes Rackd doesn't use. They now match, as do
  `docs/conflicts.md` and `docs/reservations.md`.
- The web UI showed only the first 100 devices: the device list, the relationship
  graph and the device pickers in NAT, DNS and pools. It now fetches every page.
- MCP `device_save`, `network_save` and `datacenter_save` replaced the whole record
  on update, so `{id, description}` wiped a device's addresses, tags and hostname.
  An update now changes only the fields passed. Addresses can carry `network_id`,
  `pool_id` and `port`, and custom field errors are reported instead of ignored.
- Field validation (CIDRs, IP formats, VLAN range, hostnames, pool ranges, lengths)
  ran only in the REST handlers. MCP and bulk operations skipped it, and bulk
  device creation also skipped IP conflict detection and DNS sync. The services
  now validate every write, and bulk writes get the same side effects as single
  ones.
- **Security:** writes to devices through other features were checked only
  against that feature's permission. Promoting a discovered device now needs
  `devices:create`. Resolving a duplicate-IP conflict, which takes the IP off
  other devices, needs `devices:update`, and so does linking a DNS record that
  adds a domain to a device. Promoting now also runs device validation, conflict
  detection and DNS sync.
- Device list filters accept `tags` both repeated (`tags=a&tags=b`) and
  comma-separated (`tags=a,b`). The web UI and CLI sent the second form, which
  matched nothing. New tags can't contain a comma.
- **Security:** much of what an attacker would do left no audit trail. Sign-ins
  (successful, failed and refused), sign-outs, user changes, password changes and
  resets, role and permission changes, role assignments, API keys, credentials,
  webhooks, OAuth client registration, approval and deletion, circuits, NAT, custom
  fields, conflict resolution, scan profiles and scheduled scans are now audited.
  Entries for credentials, keys, webhooks and OAuth clients hold names and
  settings, never secrets or full webhook URLs.
- A burst of changes bigger than the audit queue (1,000 entries) silently
  dropped audit entries while the changes themselves succeeded (found by a
  Strix scan). Entries are now written in batches, and when the queue is full
  the request waits for room instead of dropping its entry.
- `AUDIT_RETENTION_DAYS` was never applied, so audit entries were kept forever. They
  are now deleted once they are older than it, at start and daily. `AUDIT_ENABLED`
  did nothing, since changes were always audited; it is now documented as unused
  and logs a warning if set.
- **Security:** secrets and internal details were easier to reach than they
  should have been:
  - The install guide and CLI reference told operators to set
    `RACKD_ENCRYPTION_KEY` or `--encryption-key`, which Rackd never read, so
    following them ran without encryption. A malformed `ENCRYPTION_KEY` also only
    printed a warning. The docs now name `ENCRYPTION_KEY`, and Rackd refuses to
    start if it is set but isn't 64 hex characters.
  - The database and its WAL files were created world-readable (`0644`) in a
    `0755` directory, and audit exports were written `0644`. They are now `0600`,
    in a `0700` directory for new installs, and existing database files are
    tightened on start.
  - Webhook URLs, which can themselves be credentials (Slack and Teams incoming
    webhooks, `?token=`), were shown to every role that can read webhooks. Only
    those who can update webhooks now see them in full; others see the scheme and
    host. Delivery errors no longer repeat the URL, and a migration removes it
    from errors already stored.
  - The in-app log kept debug and trace entries whatever `LOG_LEVEL` was set to.
    It now keeps what the level shows, and hides sensitive values in logged query
    strings.
  - MCP tools, the OAuth token endpoint and `/readyz` returned raw internal
    errors, including database messages. MCP errors now say forbidden, not
    found, invalid or conflict, with the service's message, and anything else is
    logged and reported as an internal error.
  - HSTS was only sent when Rackd itself served TLS, which it never does. It is
    now sent whenever `COOKIE_SECURE` is on (the default), without
    `includeSubDomains`, so it doesn't affect other sites on the same domain.
  - `rackd credentials rotate-key` took the new key as a command-line argument,
    visible in the process list, and updated rows one at a time. It now reads the
    key from `NEW_ENCRYPTION_KEY` or `--new-key-file`, needs `--yes`, and rotates
    in one transaction that changes nothing if the current key is wrong.
- **Security:** sessions and OAuth tokens could outlive revocation, and sign-in
  could be abused:
  - Sessions never re-checked their user, and the refresh on each request was an
    upsert, so a request in flight could bring back a session that a password
    reset or deactivation had just deleted. Refresh now only extends existing
    sessions, and every request rejects a session whose user is missing or
    inactive.
  - Sessions had no absolute lifetime. They now end 7 days after sign-in
    (`SESSION_MAX_LIFETIME`).
  - When a reused OAuth refresh token was detected, only the tokens issued
    directly from it were revoked, so a thief who refreshed first kept a working
    refresh token. Revocation now covers every descendant. Two refreshes of the
    same token at once no longer both succeed.
  - A password change or reset now revokes the user's OAuth tokens.
  - A cross-site form could sign a visitor into another account (login CSRF).
    Login now requires `Content-Type: application/json`.
  - The login page's `?redirect=` could send users to another site with a path
    like `/.//evil.example`.
  - Guessing from many addresses got around the per-address login limit. Failed
    sign-ins are now also limited per account (`LOGIN_ACCOUNT_MAX_FAILURES`,
    `LOGIN_ACCOUNT_WINDOW`).
  - Login, OAuth, API key, user and bulk routes shared one rate-limit bucket per
    address, so failed logins could block token refreshes. Each now has its own.
  - The OAuth consent page now shows where an approval sends access, and warns
    that the application's name can't be verified.
- **Security:** several ways to overload the server or reach places it shouldn't:
  - Overlapping-subnet detection compared every pair of networks and listed every
    recorded conflict once per new one. An operator could create thousands of
    identical networks and exhaust the server's memory. It now sweeps sorted
    networks, stops when the request is cancelled, records at most 1,000 pairs per
    run, and reads the recorded conflicts once.
  - `/mcp` read request bodies of any size; it now has the same 1 MB limit as the
    API.
  - Discovery scans could target loopback, link-local (including the cloud
    metadata address) and other non-host ranges, which probed the server itself.
    Those subnets are now refused, and at most 4 scans run at once
    (`429 TOO_MANY_SCANS`). A subnet that is too large is now a `400` instead of a
    `500`.
  - Changing a DNS provider's endpoint kept its stored token and sent it to the
    new host on the next sync or test. The token must now be entered again.
  - Outbound requests (webhooks, DNS providers) now apply the IPv4 rules to NAT64,
    6to4 and Teredo addresses. They block reserved IPv4 space, and check the target
    before handing a request to an environment proxy. DNS provider responses are
    capped at 32 MB.
  - Anonymous OAuth client registration limits field lengths and the number of
    redirect URIs.
- **Security:** delegated user and role management could be escalated to full
  admin:
  - The "only admins can manage admins" checks read the `is_admin` column, which
    stayed false for admins promoted by granting the admin role. So a user with
    `users:update` could reset such an admin's password and sign in as them.
  - `roles:update` could grant any permission, including to the caller's own role.
  - `users:create` could give a new user any role except the one named `admin`.

  Callers who aren't admins can now only grant, revoke or assign permissions they
  hold. They can only act on users whose permissions they all hold. The last
  active admin can't be removed. See [RBAC](docs/rbac.md#delegating-user-and-role-management).
- **Security:** an operator could make an admin's browser delete or rename users,
  API keys, roles and other records. Creating a device, network, datacenter, pool,
  scan profile or scheduled scan kept any `id` the client sent, and the web UI put
  IDs into request paths unescaped. So a device with the id `../users/<admin-id>`
  sent `DELETE /api/v1/users/<admin-id>` when an admin deleted it from the list.
  - Client-supplied IDs on create must now be UUIDs.
  - The web UI escapes every value it puts in a request path, and refuses
    requests whose path contains `.` or `..` segments, so IDs already stored are
    harmless too.
  - CSV exports neutralise formulas in ID columns.
- Bulk network creation (`POST /networks/bulk`) now validates each network like
  single creation does, instead of storing empty names and invalid subnets.
- The web UI's notifications (such as "Permissions refreshed successfully" and
  permission-denied messages) never appeared: the notification area showed a
  separate, always-empty list.
- Lists were cut off at 100 records with no way to page, although the API reference
  said they returned everything. Among the effects:
  - The API and web UI showed only the first 100 networks, datacenters, users,
    webhooks, circuits, NAT mappings, reservations, DNS records and others.
  - Rackd has 118 permissions, so the last 18 could not be assigned to a role
    ("Unknown permission ID"), were missing from the role editor, and could not be
    requested as OAuth scopes.
  - Audit log exports held at most 100 entries.
  - The Prometheus `devices_total`, `networks_total` and `datacenters_total` metrics,
    dashboard utilization and utilization snapshots covered at most 100 records.
- Without an encryption key (so without the DNS service), the MCP server still
  offered the DNS tools, and calling one crashed the request with a nil pointer.
  The DNS tools, and their mention in the server instructions, now appear only when
  DNS is available.
- The utilization snapshot worker was never started, so the dashboard's trend chart
  had no data. It now runs with the server: a snapshot of every network and pool on
  start, then every `SNAPSHOT_INTERVAL` (default 1h; `0` disables it), with
  snapshots older than `SNAPSHOT_RETENTION_DAYS` (default 90) removed.
- Creating or updating a device never checked for duplicate IPs, although the
  conflicts guide said it did: the check was never connected. It now runs, and looks
  up the devices holding each IP with one indexed query. The old check, had it run,
  compared only against the first 100 devices by name. The conflicts guide now says
  which detection is automatic and which needs an explicit run.
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
- Any signed-in user, even one with no roles, could read `/metrics`, including
  inventory counts. It now needs `metrics:read`. The monitoring guide said the
  endpoint needed no authentication; it shows the scraper an API key now.
- A webhook ping that didn't reach the receiver answered 200 "Webhook ping sent
  successfully". It now answers 502 `WEBHOOK_UNREACHABLE` with the cause, and
  the web UI and CLI report it as a failure.
- An unparsable `start_time` or `end_time` on the audit and log list and export
  endpoints was ignored, so the filter quietly returned everything. It is now a
  400, as is an `end_time` before `start_time`.

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

[Unreleased]: https://github.com/martinsuchenak/rackd/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/martinsuchenak/rackd/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/martinsuchenak/rackd/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/martinsuchenak/rackd/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/martinsuchenak/rackd/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/martinsuchenak/rackd/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/martinsuchenak/rackd/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/martinsuchenak/rackd/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/martinsuchenak/rackd/compare/v0.0.21...v0.1.0
[0.0.21]: https://github.com/martinsuchenak/rackd/compare/v0.0.20...v0.0.21
[0.0.20]: https://github.com/martinsuchenak/rackd/compare/v0.0.19...v0.0.20
