# Web UI Consistency Plan

This document tracks the web UI audit follow-up work as a sequence of PR-sized changes.

## Goals

- Remove duplicated route, title, nav, and permission metadata
- Align all frontend permission checks with backend RBAC resource names
- Reduce Alpine-specific type escapes and isolate unsafe framework access
- Standardize modal behavior and CRUD/list component structure
- Reduce drift between frontend shared types and the OpenAPI contract
- Add regression coverage for the fragile UI paths

## Validation For Every PR

- Run `cd webui && bun run typecheck`
- Run `cd webui && bun run build:js`
- If a PR changes permissions or route behavior, manually verify at least one affected page
- If a PR changes modal behavior, verify overlay click, `Esc`, and close button behavior

## PR 1: Central Feature Registry

### Status

- Completed on 2026-04-09

### Goal

Create one source of truth for route prefixes, titles, nav entries, and required permissions.

### Files

- [`webui/src/core/features.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/features.ts)
- [`webui/src/app.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/app.ts)
- [`webui/src/components/nav.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/nav.ts)

### Tasks

- Add a feature registry that defines:
  - route path or prefix
  - page title
  - nav label
  - nav icon
  - nav order
  - required permission resource/action
- Replace the title map in `app.ts`
- Replace `routePermissions` in `app.ts`
- Replace hardcoded nav base items in `app.ts`
- Replace hardcoded nav base items in `nav.ts`
- Merge dynamic nav items through the same filtering path

### Acceptance Criteria

- Titles, route guards, and nav visibility derive from one shared definition
- `app.ts` and `nav.ts` no longer duplicate feature metadata

### Progress Notes

- Added [`webui/src/core/features.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/features.ts) as the shared registry for:
  - page titles
  - route access checks
  - base nav items
  - nav filtering and dynamic-item merging
- Updated [`webui/src/app.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/app.ts) to use shared helpers for:
  - page titles
  - route permission checks
  - base nav construction
- Updated [`webui/src/components/nav.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/nav.ts) to use the same shared nav definitions and filtering rules

### Validation

- `cd webui && bun run typecheck`
- `cd webui && bun run build:js`

## PR 2: RBAC Name Audit And Fixes

### Goal

Make all frontend permission checks use backend RBAC resource names consistently.

### Backend References

- [`internal/service`](/Users/martinsuchenak/Devel/projects/rackd/internal/service)
- [`internal/storage/migrations.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/storage/migrations.go)

### Primary Frontend Targets

- [`webui/src/partials/pages/scan-profiles.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/scan-profiles.html)
- [`webui/src/partials/pages/webhooks.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/webhooks.html)
- [`webui/src/partials/pages/pool-detail.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/pool-detail.html)
- [`webui/src/partials/pages/dns-providers.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/dns-providers.html)
- [`webui/src/partials/pages/dns-zones.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/dns-zones.html)
- [`webui/src/partials/pages/dns-records.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/dns-records.html)
- [`webui/src/app.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/app.ts)
- [`webui/src/components/nav.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/nav.ts)

### Tasks

- Audit every `canList`, `canCreate`, `canUpdate`, `canDelete`, and `canRead` use in the web UI
- Compare each resource string with backend RBAC names
- Fix known mismatches first:
  - `scan-profiles`
  - `webhooks`
  - `reservations`
  - DNS resource naming
- Ensure route guard, nav entry, and action buttons use the same resource name per feature

### Acceptance Criteria

- Every frontend permission string matches a backend permission resource
- No feature is gated by one resource in routes and another in templates

## PR 3: Typed Alpine Helper Layer

### Goal

Move Alpine-specific store and event access behind typed helpers.

### Files

- [`webui/src/core/alpine.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/alpine.ts)
- [`webui/src/app.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/app.ts)
- [`webui/src/components/search.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/search.ts)
- [`webui/src/components/users.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/users.ts)
- [`webui/src/components/devices.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/devices.ts)
- [`webui/src/components/networks.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/networks.ts)
- [`webui/src/components/nat.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/nat.ts)

### Tasks

- Add typed helpers for:
  - permissions store access
  - toast store access
  - navigation dispatch
  - optional Alpine watch wrapper where practical
- Replace `(this as any).$dispatch`
- Replace `@ts-ignore` around `Alpine.store(...)`
- Remove easy `as any` cases tied to stores and event dispatch
- Localize CSP directive Alpine internals behind helper functions where possible

### Acceptance Criteria

- No `@ts-ignore` remains for permissions store access
- Cross-component navigation and store access are typed and reusable

## PR 4: Config Bootstrap Consolidation

### Goal

Fetch UI config once and reuse it everywhere.

### Files

- [`webui/src/components/nav.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/nav.ts)
- [`webui/src/app.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/app.ts)

### Tasks

- Remove direct `fetch('/api/config')` from `nav.ts`
- Reuse bootstrapped config already loaded by `app.ts`
- Keep nav filtering and feature visibility tied to the shared config path

### Acceptance Criteria

- Config is fetched through one code path
- Nav does not maintain a separate fetch/bootstrap implementation

## PR 5: Modal Shell Standardization

### Goal

Reduce duplicated modal markup and normalize dialog behavior.

### Files To Add

- [`webui/src/partials/modals/modal-shell.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/modals/modal-shell.html)
- optional helper file such as [`webui/src/core/ui.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/ui.ts)

### First Migration Batch

- [`webui/src/partials/pages/api-keys.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/api-keys.html)
- [`webui/src/partials/pages/custom-fields.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/custom-fields.html)
- [`webui/src/partials/pages/circuits.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/circuits.html)
- [`webui/src/partials/pages/webhooks.html`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/partials/pages/webhooks.html)

### Second Migration Batch

- DNS pages
- users and roles pages
- detail page modals
- discovery modals

### Tasks

- Standardize:
  - overlay wrapper
  - dialog container
  - close button
  - `Esc` behavior
  - focus trap
  - max height behavior
  - modal widths
- Define modal size conventions:
  - confirm
  - standard form
  - large form
  - complex detail

### Acceptance Criteria

- Migrated dialogs use one shell pattern
- Close behavior and focus behavior are uniform

## PR 6: Shared List-Page State Pattern

### Goal

Normalize CRUD/list component state shape and method names.

### First Targets

- [`webui/src/components/scan-profiles.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/scan-profiles.ts)
- [`webui/src/components/webhooks.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/webhooks.ts)
- [`webui/src/components/users.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/users.ts)
- [`webui/src/components/roles.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/roles.ts)

### Second Targets

- [`webui/src/components/dns.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/dns.ts)
- [`webui/src/components/circuits.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/circuits.ts)
- [`webui/src/components/nat.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/components/nat.ts)

### Tasks

- Converge on a common state shape:
  - `items`
  - `selectedItem`
  - `modalType`
  - `loading`
  - `saving`
  - `deleting`
  - `error`
  - `validationErrors`
- Converge on common methods:
  - `openCreateModal`
  - `openEditModal`
  - `openDeleteModal`
  - `closeModal`
  - `save`
  - `deleteConfirmed`

### Acceptance Criteria

- Migrated components share the same mental model and naming pattern

## PR 7: OpenAPI Contract Hardening

### Goal

Reduce drift between frontend shared types and backend API schemas.

### Files

- [`api/openapi.yaml`](/Users/martinsuchenak/Devel/projects/rackd/api/openapi.yaml)
- [`webui/src/core/types.ts`](/Users/martinsuchenak/Devel/projects/rackd/webui/src/core/types.ts)

### Tasks

- Decide whether to:
  - generate frontend API-facing types from OpenAPI
  - or add a validation step that checks for drift
- Document the chosen source of truth
- Remove manual duplication where feasible

### Acceptance Criteria

- CI or a documented validation step can detect schema drift

## PR 8: Frontend Regression Coverage

### Goal

Lock in the consistency work with automated coverage.

### Files To Add

- test files under [`webui/tests`](/Users/martinsuchenak/Devel/projects/rackd/webui/tests) or project-standard location

### Coverage Targets

- Route access and access-denied rendering
- Nav visibility by permission set
- Modal close behavior:
  - close button
  - overlay click
  - `Esc`
- CSP-safe nested form updates
- OAuth disabled path returning HTML instead of JSON
- At least one CRUD smoke flow for a list page

### Acceptance Criteria

- Permission and modal regressions are covered by automated tests

## Recommended Order

1. PR 1: Central Feature Registry
2. PR 2: RBAC Name Audit And Fixes
3. PR 3: Typed Alpine Helper Layer
4. PR 4: Config Bootstrap Consolidation
5. PR 5: Modal Shell Standardization
6. PR 6: Shared List-Page State Pattern
7. PR 7: OpenAPI Contract Hardening
8. PR 8: Frontend Regression Coverage

## Notes

- PR 2 has the highest immediate user-facing value because permission drift can hide or expose the wrong actions
- PR 5 and PR 6 should be incremental to avoid broad regressions
- PR 7 should not block the earlier refactors unless code generation is chosen immediately
