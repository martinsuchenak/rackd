# Audit And Logs Web UI Plan

Status:

- implemented in the application codebase
- audit and logs now have Web UI pages, backend APIs, and explicit RBAC permissions
- migrations are required and included for the new permission set

This document outlines a practical plan for exposing both audit history and runtime logs in the Rackd Web UI.

The two features should not be treated as the same thing:

- **Audit** is already a product feature with structured storage, filtering, RBAC, and export support.
- **Logs** are currently an operational surface driven by deployment/runtime (`journalctl`, Docker logs, file logs, stderr/stdout), not a first-class application API.

Because of that, the implementation plan should ship **audit first** and treat **logs** as a separate admin/operator feature with stricter scope and security controls.

## RBAC And Migration Requirement

Both features must follow the existing Rackd pattern:

- backend-enforced RBAC in service/API layers
- frontend route/nav/action gating through the shared feature registry and permissions store
- permission definitions added through migrations, not only in code or docs

This means the work is not just Web UI work. It requires:

- new permissions
- migration entries to create those permissions
- role seeding updates for built-in roles
- RBAC documentation updates
- frontend feature gating from the beginning

### Required New Permissions

These should be introduced explicitly from the start, even if the first UI uses only a subset.

Recommended resources/actions:

- `audit:list`
- `audit:read`
- `audit:export`
- `logs:list`
- `logs:read`
- `logs:export`

Why define all three actions per resource up front:

- it matches the rest of the RBAC model
- it avoids redesigning permission semantics later
- it allows route access, detail access, and export access to be separated cleanly if needed

### Migration Requirement

Adding these permissions means adding a new RBAC migration in [`internal/storage/migrations.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/storage/migrations.go), following the same pattern used for other resources.

That migration should:

- create the new permission rows
- grant all new permissions to the `admin` role
- decide the initial built-in role behavior for `operator` and `viewer`

Recommended initial seeding:

- `admin`: all `audit:*` and `logs:*`
- `operator`: `audit:list`, `audit:read`, `logs:list`, `logs:read`
- `viewer`: no audit or logs access by default

Notes:

- `audit:export` and `logs:export` should stay admin-only initially
- if operator access to logs feels too broad, keep logs admin-only in v1 and still define the permissions now

### Documentation Requirement

The following documentation should be updated as part of the same feature track:

- [`docs/rbac.md`](/Users/martinsuchenak/Devel/projects/rackd/docs/rbac.md)
- [`docs/audit.md`](/Users/martinsuchenak/Devel/projects/rackd/docs/audit.md)
- any future logs documentation once the logs backend model is implemented

## Current State

### Audit

Already present:

- API endpoints:
  - `GET /api/audit`
  - `GET /api/audit/{id}`
  - `GET /api/audit/export`
- service and RBAC support:
  - [`internal/service/audit_svc.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/service/audit_svc.go)
  - resource/action: `audit:list`
- storage support:
  - [`internal/storage/audit_sqlite.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/storage/audit_sqlite.go)
- model:
  - [`internal/model/audit.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/model/audit.go)
- docs:
  - [`docs/audit.md`](/Users/martinsuchenak/Devel/projects/rackd/docs/audit.md)

Gaps:

- no web UI page
- no frontend API/type wiring
- no per-object audit history surface
- current API handler does not parse the `source` filter even though the model supports it
- current permission model is too narrow for the planned UI if detail/export separation is desired

### Logs

Already present:

- structured application logging package:
  - [`internal/log/log.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/log/log.go)
- deployment guidance for viewing logs externally:
  - `journalctl`
  - Docker logs
  - file tails

Gaps:

- no `GET /api/logs` style endpoint
- no storage/indexing/query layer for logs inside Rackd
- no Web UI page
- no log RBAC resource documented or enforced
- no redaction/filtering rules for UI-safe log exposure
- no migration-backed permission rows for logs

## Product Direction

### Audit UI

This should be implemented as a **first-class product feature**.

Primary users:

- administrators
- security/compliance reviewers
- operators investigating configuration changes

Primary goals:

- answer **who changed what, when, and from where**
- support investigation and export
- link actions back to resources where possible

### Logs UI

This should be implemented as an **operator feature**, not a general user feature.

Primary users:

- admins
- operators

Primary goals:

- support troubleshooting
- expose recent application/runtime issues
- avoid requiring shell access for common diagnosis

Non-goals for the first version:

- full log management platform
- persistent centralized log search
- replacing `journalctl`, Docker logging, or external log aggregation

## Recommended Delivery Order

1. Audit page and frontend integration
2. Per-resource audit history links
3. Audit export UX polish
4. Logs backend design decision
5. Minimal logs UI
6. Optional advanced logs features

## Audit Implementation Plan

## Phase A1: Backend Cleanup For Audit UI

Goal:

- close the small API contract gaps before building the frontend

Tasks:

- add/confirm RBAC support for the intended audit UI surface:
  - `audit:list`
  - `audit:read`
  - `audit:export`
- update [`internal/api/audit_handlers.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/audit_handlers.go) to parse and pass through:
  - `source`
- verify `openapi.yaml` matches the full supported query surface
- ensure audit responses are stable for:
  - empty filter set
  - date range filter
  - resource/resource_id filter
  - user/action/source filter
- confirm export endpoint respects the same filters as list

Acceptance:

- API supports all filter fields already present in [`internal/model/audit.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/model/audit.go)
- OpenAPI and handler behavior match
- audit permissions are fully defined and migration-backed before the Web UI ships

## Phase A2: Frontend Contract

Goal:

- expose audit through the central frontend API/types layer

Files to add/update:

- [`webui/src/core/types.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/types.ts)
- [`webui/src/core/api.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/api.ts)
- [`webui/src/core/features.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/features.ts)

Tasks:

- add `AuditLog` and `AuditFilter` frontend types
- add API methods:
  - `listAuditLogs(filter)`
  - `getAuditLog(id)`
  - `getAuditExportUrl(filter, format)` or equivalent helper
- add feature registry entry for `/audit`
- gate the route with `audit:list`
- gate detail access and export controls consistently with:
  - `audit:read`
  - `audit:export`

Acceptance:

- audit is a first-class frontend feature using the shared API/type layer
- audit route, detail view, and export controls are permission-aware from v1

## Phase A3: Audit List Page

Goal:

- build the main audit investigation screen

Files to add:

- [`webui/src/components/audit.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/audit.ts)
- [`webui/src/partials/pages/audit.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/audit.html)

Core UI:

- filter bar:
  - resource
  - action
  - actor/user
  - source
  - start time
  - end time
- paginated table with:
  - timestamp
  - action
  - resource
  - resource ID
  - username
  - IP address
  - status
  - source
- empty state
- loading/error state
- export buttons for JSON/CSV

Recommended default sort:

- timestamp descending

Acceptance:

- admin can investigate recent changes without leaving the Web UI

## Phase A4: Audit Detail View

Goal:

- make a single entry inspectable without cluttering the table

Recommended UX:

- detail drawer or modal, not a separate route for v1

Show:

- full metadata
- pretty-printed `changes`
- error text if present
- linked resource target when possible

Implementation note:

- `changes` is stored as a string and may contain JSON; parse when valid, fall back to raw text

Acceptance:

- complex audit entries are readable and useful

## Phase A5: Per-Resource Audit History

Goal:

- connect audit to everyday workflows

Recommended placements:

- device detail
- network detail
- datacenter detail
- user detail/edit surfaces
- DNS provider/zone detail

Two viable options:

1. lightweight “Recent activity” card on detail pages
2. “View audit history” link that opens `/audit?resource=device&resource_id=<id>`

Recommendation:

- start with links, not embedded cards

Acceptance:

- object-level change history is one click away

## Phase A6: Audit Tests

Add:

- frontend unit tests for filter serialization and export URL generation
- Playwright E2E:
  - audit page access for admin
  - audit page hidden/denied for non-admin
  - filter interaction
  - detail drawer rendering
  - export link generation

## Logs Implementation Plan

## Phase L1: Decide The Logs Backend Model

This is the critical design decision.

There are three realistic options:

1. **Read from existing runtime output**
   - examples: systemd journal, Docker logs, local log file
   - pros: minimal duplication
   - cons: deployment-specific, hard to make portable, security-sensitive

2. **Add an in-app rolling log buffer**
   - keep recent structured logs in memory and optionally on disk
   - pros: portable, app-controlled, easy to expose in UI
   - cons: only recent history unless persisted, more application responsibility

3. **Persist selected log events into storage**
   - structured operational events table
   - pros: queryable and portable
   - cons: overlaps with audit and increases DB write volume

Recommendation:

- for Rackd, use **option 2** for the first version:
  - a bounded in-app ring buffer
  - optionally exposed via API
  - focused on recent application logs only

Why:

- avoids platform-specific `journalctl`/Docker coupling
- avoids turning SQLite into a full log store
- is enough for Web UI troubleshooting

## Phase L2: Define Log Access Rules

Before any UI work, define:

- allowed viewers:
  - admins only by default
- new RBAC resource:
  - `logs:list`
  - `logs:read`
  - `logs:export`
- data redaction rules:
  - never expose passwords, tokens, secrets, authorization headers, provider secrets
- retention:
  - fixed recent window, for example last `1000-5000` entries
- filtering:
  - level
  - component/source
  - free-text search

Acceptance:

- security boundaries are explicit before implementation
- logs permissions are defined up front and scheduled as migration-backed RBAC additions

## Phase L3: Minimal Logs Backend

Likely files:

- [`internal/log/log.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/log/log.go)
- new API/service/model files for logs

Tasks:

- add migration-backed RBAC support for:
  - `logs:list`
  - `logs:read`
  - `logs:export`
- add a recent log entry model
- attach a ring buffer sink to the application logger
- add service methods:
  - `ListRecentLogs(filter)`
- add API route:
  - `GET /api/logs`
- add optional tail-like polling semantics later, not in v1

Recommended fields:

- timestamp
- level
- component/source
- message
- structured fields

Acceptance:

- recent logs are available in a portable, app-managed way
- logs backend and API are guarded by explicit permissions, not implicit admin assumptions

## Phase L4: Logs Web UI

Files to add:

- [`webui/src/components/logs.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/logs.ts)
- [`webui/src/partials/pages/logs.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/logs.html)

Core UI:

- filters:
  - level
  - component/source
  - text search
- table or list:
  - timestamp
  - level badge
  - message
  - source
- optional detail drawer for structured fields
- auto-refresh toggle

Keep v1 small:

- no live websocket tail
- no arbitrary file browsing
- no download of host logs

Permission behavior:

- page visible only with `logs:list`
- row/detail expansion only if `logs:read` is granted
- export action hidden unless `logs:export` is granted

Acceptance:

- admins can inspect recent application errors without shell access

## Phase L5: Logs Tests

Add:

- unit tests for ring buffer behavior and redaction
- API tests for `logs:list` and filters
- Playwright smoke test:
  - logs page visible to admin
  - hidden/denied to non-admin
  - recent seeded log entry renders

## Cross-Cutting UX Recommendations

### Navigation

Recommended nav layout:

- `Audit` as a normal admin feature
- `Logs` under admin/operations section, or below `Audit`

### Filtering

Keep filters URL-driven where possible:

- `/audit?resource=device&resource_id=...`
- `/logs?level=error&source=discovery`

That makes deep links and troubleshooting handoff easier.

### Empty States

Important messaging:

- audit empty state should clarify whether auditing is disabled
- logs empty state should clarify that only recent in-app logs are shown

### Export

Audit:

- yes, export is already a product fit

Logs:

- not in v1 unless the backend model is clearly scoped

## Risks And Decisions

### Audit Risks

- `changes` payload may be inconsistent across resources
- some entries may not link cleanly to a current UI route
- large filters/date ranges may need pagination tuning

### Logs Risks

- accidental exposure of secrets
- expectation mismatch if users think Web UI logs are full system logs
- deployment portability if tied to systemd/Docker too early

## Recommended Scope For The First Delivery

Ship first:

1. audit backend cleanup
2. audit Web UI page
3. audit detail drawer
4. audit deep links from detail pages

Then:

5. define logs RBAC + redaction
6. implement recent in-app log buffer
7. ship minimal logs page for admins

## Suggested PR Breakdown

1. Audit API cleanup and OpenAPI alignment
2. Audit RBAC migration and docs alignment
3. Frontend audit types/API/feature registration
4. Audit list page
5. Audit detail drawer and export UI
6. Audit deep links from object pages
7. Log access design and RBAC
8. Logs backend migration/API
9. Logs Web UI
10. Audit/logs E2E coverage

## Recommendation

The highest-value path is:

- make **audit** a complete Web UI feature now
- treat **logs** as a separate operator-focused feature with a bounded in-app backend

That gives users real accountability and investigation tools quickly, without overcommitting to a brittle “browser view of server logs” design.
