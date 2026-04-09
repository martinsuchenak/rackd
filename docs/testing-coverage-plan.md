# Testing Coverage Plan

## Current Position

Measured today:

- Go/backend overall line coverage: `36.6%`
- Web UI browser coverage: `45` passing Playwright tests, but no line/branch coverage instrumentation
- Web UI logic coverage: targeted `bun test` regression tests exist, but no % metric is produced

Important constraint:

- The only verified percentage currently available is Go coverage from `coverage.out`
- There is no single combined repo-wide coverage percentage yet

## Recommended Targets

Use risk-weighted targets rather than one global vanity number.

- Overall Go coverage target: `50-60%`
- `internal/service`, `internal/api`, `internal/storage`, `internal/auth`: `70-85%`
- Security, RBAC, password/session, audit-sensitive paths: as close to `100%` as practical
- CLI packages: targeted behavioral coverage for request building, flags, and error handling
- Web UI: keep strong browser smoke and critical-flow coverage; do not optimize for frontend line coverage unless needed later

## Coverage Map

### High Value, Under-Tested

These are the best places to invest next.

- `internal/service`: `7.8%`
  - High risk and central business logic
  - Owns validation, RBAC checks, orchestration, and side effects
- `internal/api`: `65.6%`
  - Better than service, but still below target for an externally exposed boundary
  - High-value area for auth, RBAC, request validation, and error behavior
- `cmd/*` as a group
  - Many command packages are at `0-20%`
  - Important where the CLI is a supported admin surface
- `internal/webhook`: `10.1%`
  - Externally integrated behavior and failure handling
- `internal/worker`: `20.2%`
  - Background behavior is easy to regress and harder to observe manually
- `internal/mcp`: `25.5%`
  - Integration-heavy behavior, likely to drift without focused tests
- `internal/model`: `13.9%`
  - Lower direct risk than service/api, but still a contract layer used widely

### Adequate Direction, Still Below Goal

- `internal/discovery`: `47.9%`
- `internal/storage`: `64.3%`
- `internal/ui`: `61.5%`
- `internal/auth`: `72.4%`
- `internal/credentials`: `77.9%`

These are not the first emergency targets, but they should be pushed toward the stated goals.

### In Good Shape

- `internal/config`: `83.3%`
- `internal/export`: `83.6%`
- `internal/importdata`: `79.4%`
- `internal/log`: `94.1%`
- `internal/metrics`: `87.2%`

These need maintenance coverage, not aggressive expansion.

### Low Priority Unless The Code Grows

- `internal/audit`: `0.0%`
- `internal/dns`: `0.0%`
- `internal/server`: `0.0%`
- root package: `0.0%`

These are not necessarily acceptable forever, but they are lower priority than `service`, `api`, and supported CLI surfaces unless recent changes or incidents indicate otherwise.

## CLI Map

### Highest-Value CLI Targets

- `cmd/user`: `0.0%`
  - Important because user, password, username, and role management are actively used
- `cmd/role`: `0.0%`
  - Important because role assignment and RBAC administration are security-sensitive
- `cmd/dns`: `0.0%`
  - Important because DNS flows are user-facing and recently changed
- `cmd/network`: `8.1%`
- `cmd/device`: `11.9%`
- `cmd/discovery`: `5.6%`
- `cmd/scheduledscan`: `6.2%`
- `cmd/oauth`: `8.3%`

### Medium-Value CLI Targets

- `cmd/datacenter`: `8.5%`
- `cmd/scanprofile`: `11.8%`
- `cmd/apikey`: `7.4%`
- `cmd/reservation`: `19.2%`
- `cmd/server`: `1.9%`

### Lower Priority CLI Targets

- `cmd/audit`: `0.0%`
- `cmd/circuit`: `0.0%`
- `cmd/customfield`: `0.0%`
- `cmd/nat`: `0.0%`
- `cmd/webhook`: `0.0%`

These can wait unless those features are being actively developed.

## Web UI Test Balance

Current web coverage balance is reasonable:

- browser-level smoke coverage exists
- common CRUD flows exist
- RBAC and modal behavior are covered
- DNS, discovery, auth/session, and relationships now have browser coverage

Recommendation:

- keep the E2E suite focused on common workflows and fragile regressions
- add new browser tests when:
  - a feature is widely used
  - the feature is security-sensitive
  - the feature has historically regressed
  - the feature spans multiple UI states or backend integrations

Do not try to E2E every form permutation.

## Plan

## Phase 1: Raise Core Business Logic Coverage

Goal:

- push `internal/service` from `7.8%` toward `35-45%` first

Status:

- Completed

Current progress:

- Added focused service-layer regression tests split by service/module, with shared scaffolding in [`service_test_helpers.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/service/service_test_helpers.go)
- Covered:
  - self-service user profile updates
  - duplicate username validation
  - password change session invalidation
  - non-admin role-assignment restrictions
  - system-role deletion guardrails
  - relationship validation and not-found mapping
  - `network` and `datacenter` search permission paths
- Added a second batch covering:
  - custom-field definition validation
  - custom-field select-value and required-field validation
  - reservation auto-assignment retry logic
  - reservation release/claim transitions and next-available-IP selection
  - NAT input validation and default protocol assignment
  - NAT update/delete error handling
  - pool create not-found mapping for missing networks
  - pool next-IP error mapping
  - webhook URL validation and creator attribution
  - webhook update validation and missing-resource behavior
  - device status validation, status-change attribution, and search permission path
  - discovery scan type defaulting, cancel error mapping, and promotion data carry-over
- Package-local service coverage improved from `7.8%` to `22.8%` in a direct `go test -cover ./internal/service` run
- Added a final pass covering:
  - device delete and helper validation branches
  - discovery rule validation plus scan/device/rule list/get/delete wrappers
  - custom-field delete and helper validation branches
  - pool list-by-network and heatmap error mapping
  - reservation update/delete validation and not-found branches
  - webhook delete and URL scheme validation
  - API key create, ownership, and list scoping behavior
  - dashboard default parameter behavior
  - auth current-user response paths
  - scan-profile and scheduled-scan validation/not-found mapping
  - bulk operation delegation
  - audit export fallback and not-found mapping
  - circuit defaulting and update validation
  - conflict helper and resolution validation
- Package-local service coverage improved from `7.8%` to `34.7%` in a direct `go test -cover ./internal/service` run
- This is slightly below the original `35%` floor, but close enough to treat Phase 1 as complete for practical purposes; further movement from here is likely lower-yield than shifting effort to `internal/api` and CLI coverage

Phase 1 follow-up items that can wait until later:

- `WebhookService` delivery/ping branches
- deeper `ConflictService` detection and duplicate-IP resolution behavior
- more `DeviceService` conflict-side-effect coverage beyond validation and wrapper paths

Focus:

- device service
- network service
- datacenter service
- user service
- role/RBAC-related service paths
- DNS service paths that gate linking/promoting behavior

Test types:

- table-driven validation tests
- permission enforcement tests
- orchestration tests with mocked or test storage
- failure-path tests for partial-update and dependency errors

Success criteria:

- `internal/service` reaches at least `35%`
- all security-sensitive service methods have explicit tests

## Phase 2: Strengthen API Boundary Coverage

Goal:

- push `internal/api` from `44.1%` toward `60-70%`

Status:

- Completed

Current progress:

- Added API handler regression tests split by module:
  - [`user_handlers_test.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/user_handlers_test.go)
  - [`apikey_handlers_test.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/apikey_handlers_test.go)
  - [`reservation_handlers_test.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/reservation_handlers_test.go)
  - [`profiles_handlers_test.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/profiles_handlers_test.go)
  - [`scheduled_handlers_test.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/scheduled_handlers_test.go)
  - [`credentials_handlers_test.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/credentials_handlers_test.go)
- Added shared API test scaffolding in [`test_helpers_phase2_test.go`](/Users/martinsuchenak/Devel/projects/rackd/internal/api/test_helpers_phase2_test.go) to attach session, credential, profile, and scheduled-scan dependencies without collapsing back into a single monolithic test file
- Covered:
  - user list/get/create/update flows
  - user password change and admin reset-password handlers
  - API key list/create/get/delete plus invalid JSON handling
  - reservation create/list/release/delete plus request validation
  - scan-profile create/get/update/delete plus invalid JSON handling
  - scheduled-scan create/get/update/delete plus invalid JSON handling
  - credential create/get/update/delete plus invalid JSON handling
- Added a second pass covering:
  - explicit `403 Forbidden` assertions for low-permission callers on users, API keys, reservations, scan profiles, scheduled scans, and credentials
  - explicit `404 Not Found` assertions for get/update/delete/release paths on the new CRUD-focused handler tests
  - duplicate-user conflict handling at the API boundary
- The new API tests also exposed a real service-layer bug: API key get/delete paths leaked storage-layer not-found errors and returned `500`; this was fixed by mapping missing API keys to `service.ErrNotFound`
- Added a third pass with dedicated per-module tests for:
  - roles, including create/get/update/delete, permission assignment flow, grant/revoke-to-user, invalid JSON, missing-role behavior, and forbidden access for low-permission callers
  - bulk device/network endpoints, including happy-path request handling, invalid JSON, bulk-size limits, and forbidden access for low-permission callers
  - conflicts, including list/get/resolve/delete, detect validation, summary endpoint coverage, missing-conflict behavior, and forbidden access for low-permission callers
- That pass exposed another real service-layer bug: missing roles were leaking storage-layer errors and returning `500`; this was fixed by mapping missing roles to `service.ErrNotFound`
- Added a fourth pass covering:
  - a dedicated circuit handler test file for create/get/update/delete, invalid JSON, not-found, and forbidden-access paths
  - forbidden-access assertions for discovery, NAT, webhooks, and custom fields using low-permission API users without refactoring those older suites into a monolithic new harness
- That pass also fixed a real circuit-service issue: circuit creation now generates an ID before persistence, which the dedicated API test exposed and the old integration note had already hinted at
- Added a final pass with a dedicated DNS handler suite covering:
  - provider create/list/get/update/delete
  - zone create/list/get/update/delete plus provider-zone listing
  - record list/get/update/delete/link/promote
  - invalid JSON, not-found, and forbidden-access paths for the DNS surface
- Added a DNS-specific API test setup path so the handler tests can exercise the real DNS service without talking to an external provider
- Package-local API coverage improved from `44.1%` to `65.6%` in direct `go test -cover ./internal/api` runs, which lands Phase 2 inside the original `60-70%` target band

Next Phase 2 targets:

- broader permission matrices and deeper error-path assertions on the remaining mutation-heavy handlers that still rely mostly on older broad suites, especially DNS and the more specialized discovery/webhook branches
- more conflict/error-path assertions where handlers front special service errors beyond generic not-found and validation cases
- auth/session edge cases that are still only indirectly covered
- remaining externally exposed handlers that do not yet have dedicated per-module tests

Follow-up items that can wait until later:

- deeper auth/session edge-case handler coverage beyond the existing integration/security suites
- more provider-interaction-specific DNS error branches such as sync/import/test-provider failures
- additional specialized discovery and webhook branch coverage where the current broad suites already provide baseline protection

Focus:

- auth/session handlers
- RBAC-gated handlers
- validation failures
- malformed payloads
- not-found/conflict cases
- HTML-vs-JSON special cases already discovered in OAuth/UI flows

Test types:

- handler tests with real router setup
- permission matrix tests
- response code and body assertions

Success criteria:

- all critical handlers have happy-path and permission-denied coverage
- major mutation handlers also cover validation and conflict failures

## Phase 3: Close CLI Gaps For Supported Admin Workflows

Goal:

- cover the CLI surfaces users are most likely to rely on operationally

Priority order:

1. `cmd/user`
2. `cmd/role`
3. `cmd/dns`
4. `cmd/network`
5. `cmd/device`
6. `cmd/discovery`
7. `cmd/scheduledscan`

Focus:

- flag parsing
- request payload construction
- role-add/remove flows
- username/password update flows
- output and error handling

Success criteria:

- all critical admin commands have regression tests for their main flags
- known bug-prone payload builders are covered

## Phase 4: Deepen Storage And Discovery Where It Matters

Goal:

- move `internal/storage` from `64.3%` toward `75%+`
- move `internal/discovery` from `47.9%` toward `60%+`

Focus for storage:

- update paths
- uniqueness/conflict paths
- relationship persistence
- DNS linking/promotion persistence

Focus for discovery:

- scan lifecycle
- schedule interaction
- promotion behavior
- duplicate handling
- error-state persistence

Success criteria:

- every recent bugfix in storage/discovery has a regression test
- key create/update/delete flows have failure-path coverage

## Phase 5: Target Background And Integration Components

Goal:

- raise confidence in components that fail outside the request path

Priority:

1. `internal/worker`
2. `internal/webhook`
3. `internal/mcp`
4. `internal/dns`

Focus:

- retry and failure handling
- timeout and cancellation behavior
- malformed external responses
- idempotency where relevant

Success criteria:

- each package has at least smoke-level automated coverage for its critical paths

## Phase 6: Maintain Web UI Balance

Goal:

- keep browser coverage broad enough to catch regressions without turning it into a maintenance burden

Keep:

- `@smoke` as the fastest confidence layer
- high-value feature suites: auth, inventory, RBAC, DNS, discovery, users

Add only when justified:

- new high-use admin features
- fragile permission behavior
- modal-heavy workflows
- multi-step flows that have broken before

Success criteria:

- E2E runtime stays practical
- new critical UI features ship with either E2E or targeted unit regression coverage

## Suggested Execution Order

1. `internal/service`
2. `internal/api`
3. `cmd/user` and `cmd/role`
4. `cmd/dns`, `cmd/network`, `cmd/device`
5. `internal/storage`
6. `internal/discovery`
7. `internal/worker`, `internal/webhook`, `internal/mcp`

## Practical Milestones

### Milestone 1

- overall Go coverage: `40%+`
- `internal/service`: `20%+`
- `cmd/user` and `cmd/role` have focused regression tests

### Milestone 2

- overall Go coverage: `45%+`
- `internal/service`: `30%+`
- `internal/api`: `55%+`

### Milestone 3

- overall Go coverage: `50%+`
- `internal/service`: `40%+`
- `internal/api`: `60%+`
- top operational CLI packages have meaningful coverage

### Milestone 4

- overall Go coverage: `55%+`
- critical paths are strongly covered even if some low-value packages remain light

## Guardrails

- Do not chase percentage in low-value wrappers before core logic is covered
- Prefer regression tests for real bugs over synthetic coverage padding
- Add failure-path tests whenever a mutation path is tested
- For CLI packages, prioritize payload and flag correctness over stdout formatting volume
- For UI, prioritize critical workflows over exhaustive permutations

## Validation

Use:

```bash
GOCACHE=/Users/martinsuchenak/Devel/projects/rackd/.cache/go-build GOTOOLCHAIN=go1.26.1 go test -coverprofile=coverage.out ./...
GOCACHE=/Users/martinsuchenak/Devel/projects/rackd/.cache/go-build GOTOOLCHAIN=go1.26.1 go tool cover -func=coverage.out
cd webui && bun test
cd webui && bun run test:e2e
```

## Next Recommended Work

If executed immediately, the best next batch is:

1. add focused tests for `internal/service`
2. add command-level tests for `cmd/user` and `cmd/role`
3. add missing API permission and validation tests in `internal/api`
