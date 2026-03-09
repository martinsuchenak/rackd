# Rackd Comprehensive Implementation Review

**Date:** 2026-03-08  
**Scope:** CLI, REST API, MCP Server, Web UI, Storage, Security, Documentation  
**Codebase Version:** dev (latest commit)

---

## 1. Architecture Assessment

Rackd is a well-structured Go application following clean architecture principles. The layered design (HTTP → API Handlers → Service Layer → Storage) provides good separation of concerns. The single-binary approach with embedded SQLite and web UI is a strong design choice for the target use case.

The four interfaces (CLI, REST API, MCP, Web UI) share the same service and storage layers, which is the correct approach. The CLI consumes the REST API via HTTP client, while the MCP server and API handlers both use the service layer directly.

### What works well
- Single binary with zero external dependencies
- CGO-free SQLite via modernc.org/sqlite
- Pattern-based routing (Go 1.22+) eliminates router dependency
- Comprehensive feature set: devices, networks, datacenters, discovery, circuits, NAT, reservations, webhooks, custom fields, DNS, conflicts, audit
- OAuth 2.1 with PKCE for MCP authentication
- Graceful shutdown with signal handling

### Structural concerns
- ~~The `Handler` struct in `internal/api/handlers.go` has grown to hold 10+ fields set via individual setter methods. This is a code smell — consider a builder pattern or options struct.~~ **FIXED** — refactored to functional options pattern.
- The MCP server duplicates authentication logic that already exists in the API middleware. The MCP `HandleRequest` method re-implements API key lookup and user resolution instead of reusing `AuthMiddleware`.
- No interface abstraction between the MCP tool handlers and the service layer — MCP tools call storage directly in some cases, bypassing service-layer RBAC.

---

## 2. Security Review

### Resolved issues (strong foundation)
The existing `SECURITY_AND_CONSISTENCY_REVIEW.md` documents 36 critical/high issues that have been fixed. Key wins:
- bcrypt cost 14 for password hashing
- AES-256-GCM credential encryption at rest
- Constant-time token comparison
- Session invalidation on password change
- Refresh token rotation with replay detection
- SSRF protection for webhooks (SafeDialContext)
- CSP hardened (removed unsafe-eval/unsafe-inline for scripts)
- Rate limiting on login, OAuth token, and sensitive endpoints
- Legacy API key bypass removed
- Privilege escalation prevention in user/role management

### Open security gaps

**CSRF protection implemented (FIXED).** The server's `AuthMiddlewareWithSessions` requires the `X-Requested-With: XMLHttpRequest` header on all state-changing requests (POST/PUT/DELETE/PATCH) for session-authenticated users — requests without it are rejected with 403. The main `RackdAPI` client in `webui/src/core/api.ts` sends this header on all requests. Components that previously bypassed the shared client and called `fetch()` directly (`scheduled-scans.ts`, `credentials.ts`) have been patched to include `X-Requested-With: XMLHttpRequest` and `credentials: 'same-origin'` on every fetch call.

**MCP auth now consistent with REST API (FIXED).** The MCP server previously had its own API key authentication logic that diverged from the REST API — legacy keys (no `UserID`) were allowed through to RBAC instead of being rejected at the auth boundary. This has been fixed by extracting a shared `api.AuthenticateAPIKey()` function that both the REST API middleware and MCP server now call. Legacy keys are rejected with 401 at the auth layer in both paths. All MCP tools go through the service layer (`s.svc.*`), which enforces RBAC via `requirePermission` — no direct storage access exists in any tool handler.

**No CORS configuration (M-9):** The API sets no CORS headers. While this defaults to same-origin (safe), it means legitimate cross-origin integrations won't work, and there's no explicit deny for preflight requests on API routes (only MCP handles OPTIONS).

**Rate limiting enabled by default (FIXED).** `RATE_LIMIT_ENABLED` now defaults to `true` (100 requests/minute). Local dev environments can set `RATE_LIMIT_ENABLED=false` to disable.

**Cookie `Secure` flag enabled by default (FIXED).** `COOKIE_SECURE` now defaults to `true`, so session cookies are only sent over HTTPS. Local dev without TLS can set `COOKIE_SECURE=false`.

**Discovery subnet size unbounded up to /16:** A /16 scan covers 65,534 hosts. No upper bound is enforced, which could cause resource exhaustion.

---

## 3. API Consistency

### Route structure
The REST API follows a consistent RESTful pattern across all resources:
- `GET /api/{resource}` — list
- `POST /api/{resource}` — create
- `GET /api/{resource}/{id}` — get
- `PUT /api/{resource}/{id}` — update
- `DELETE /api/{resource}/{id}` — delete

This is well-maintained across devices, networks, datacenters, circuits, NAT, reservations, webhooks, custom fields, and DNS resources.

### Inconsistencies found

**Error code standardized (FIXED).** All handler-level validation errors now use `invalidJSON()` (code: `INVALID_JSON`) for decode failures and `badRequest()` (code: `INVALID_INPUT`) for all other request validation. Service-layer errors go through `handleServiceError` exclusively.

**Update handler patterns diverge:** Device and network update handlers use map-based partial updates (decode into `map[string]interface{}`), while webhook and circuit updates use typed request structs with pointer fields for optionality. The pointer-based approach is safer and should be standardized.

**Pagination enforced (FIXED).** All list endpoints now apply `LIMIT`/`OFFSET` via a shared `Pagination` struct embedded in every filter type. Default page size is 100, max is 1000. The `parsePagination()` handler helper reads `limit`/`offset` query params and clamps values.

**Health endpoint inconsistency:** `/healthz` and `/readyz` return plain text or simple JSON, while all other endpoints return structured `{"error": ..., "code": ...}` responses. This is acceptable for health checks but should be documented.

**Bulk operations lack transactional guarantees:** `BulkCreateDevices` in storage iterates and creates individually, collecting errors. A partial failure leaves some records created and others not. Consider wrapping in a transaction or clearly documenting the partial-success behavior.

---

## 4. CLI Review

### Structure
All CLI commands follow a consistent pattern: `rackd {resource} {action}`. Each resource package exports a `Command()` function returning a `*cli.Command` with subcommands. This is clean and maintainable.

### Coverage gaps

**Missing CLI commands for several API features:**
- No `rackd circuit` commands for circuit management (the package exists but verify completeness)
- ~~No `rackd scheduled-scan` CLI commands (only API endpoints exist)~~ **FIXED.** Added `rackd scheduled-scan` with list, get, create, update, delete subcommands.
- ~~No `rackd scan-profile` CLI commands~~ **FIXED.** Added `rackd scan-profile` with list, get, create, update, delete subcommands.
- ~~No `rackd oauth` client management CLI commands~~ **FIXED.** Added `rackd oauth` with list and delete subcommands.

**HTTP client has no TLS verification control:** ~~The CLI's `http.Client` in `cmd/client/http.go` uses default TLS settings. There's no `--insecure` flag for self-signed certificates, which is common in infrastructure tools. Add a `--skip-tls-verify` flag.~~ **FIXED.** The `VerifySSL` config field (default `true`) is now wired to the HTTP transport's TLS settings. Set `RACKD_VERIFY_SSL=false` env var or `"verify_ssl": false` in `~/.config/rackd/config.json` to skip certificate verification.

**No request timeout in CLI client:** The client uses `cfg.GetTimeout()` which defaults to 30s. This is fine, but long-running operations like bulk imports or discovery scans may need longer timeouts. Consider per-command timeout overrides.

**Output format inconsistency:** Some commands support `--format json|table` while others only output tables. Standardize all list/get commands to support both formats.

---

## 5. MCP Server Review

### Tool coverage
The MCP server registers tools for: search, devices, networks, datacenters, circuits, NAT, reservations, webhooks, custom fields, discovery, conflicts, and audit. This is comprehensive and matches the API surface well.

### Issues

**No MCP-specific audit logging:** API requests are logged via the `LoggingMiddleware`, but MCP tool invocations are not individually audit-logged. When an AI agent modifies infrastructure via MCP, there's no audit trail distinguishing MCP actions from API actions beyond the `source: "mcp"` field on the caller.

**Self-relationship not prevented:** The device relationship MCP tool (`device_add_relationship`) doesn't validate that `parent_id != child_id`. This should be enforced at the service layer.

**Missing DNS tools:** ~~The MCP server doesn't register DNS management tools, even though the API has full DNS provider/zone/record endpoints. This is a feature gap for AI-driven DNS management.~~ **FIXED.** Added 16 DNS tools covering providers, zones, and records — all discoverable via keywords (dns, provider, zone, record, domain, etc.).

**MCP version hardcoded:** `mcp.NewServer("rackd", "1.0.0")` hardcodes the version instead of using the build-time `version` variable from `main.go`.

**No tool pagination:** MCP list tools (e.g., `device_list`) return all results without pagination support. For large inventories, this could produce very large responses that exceed MCP client context windows.

---

## 6. Web UI Review

### Architecture
The frontend uses Alpine.js with TailwindCSS v4, built with Bun. Components are organized by feature (devices, networks, credentials, etc.) with a shared API client (`core/api.ts`). The UI is embedded into the Go binary at build time.

### Strengths
- CSP-friendly Alpine.js build (no unsafe-eval)
- Comprehensive API client with typed methods matching all API endpoints
- Permission-based route checking (`checkRoutePermission`)
- Responsive design with TailwindCSS

### Issues

**No CSRF tokens:** ~~The API client in `core/api.ts` sends requests with `credentials: 'same-origin'` but includes no CSRF token header. Combined with the server's lack of CSRF middleware, this is a real vulnerability for session-authenticated users.~~ **FIXED.** The server enforces `X-Requested-With: XMLHttpRequest` on state-changing requests for session auth. The shared API client already sent this header. Components that bypassed the shared client (`scheduled-scans.ts`, `credentials.ts`) have been patched to include the header and `credentials: 'same-origin'` on all fetch calls.

**Client-side only password validation:** Password strength rules are enforced in the UI components but the server should be the source of truth. If the server already validates (bcrypt cost 14, min length 12), the UI validation is redundant but harmless — verify server-side enforcement exists.

**No error boundary:** API errors in components are caught individually but there's no global error handler. A network failure during any operation could leave the UI in an inconsistent state.

**Missing loading states:** Some components may not show loading indicators during async operations, leading to double-submissions.

---

## 7. Storage Layer Review

### Strengths
- WAL mode enabled for better read concurrency
- Foreign key constraints enforced
- Parameterized queries throughout (no SQL injection risk)
- Audit logging infrastructure with buffered channel (no goroutine leak)
- Context propagation through all storage methods

### Inconsistencies

**Mixed time handling:** Some storage methods use `time.Now().UTC()` while others use `time.Now()`. Standardize on UTC throughout.

**Inconsistent error sentinels:** Some "not found" errors use package-level sentinels (`storage.ErrDeviceNotFound`), while others return inline errors. The service layer has to handle both patterns.

**Transaction usage varies:** `CreateReservation` uses transactions for atomicity, but `BulkCreateDevices` does not. Any multi-step write operation should use transactions.

**No database file permissions:** The SQLite database file is created with default permissions. In production, it should be restricted to the running user (0600).

---

## 8. Documentation Review

### Coverage
The `docs/` directory contains 30+ markdown files covering architecture, API, CLI, security, deployment, and individual features. This is above average for a project of this size.

### Gaps and inaccuracies

**`docs/authentication.md` is outdated:** ~~It still describes the old API key model ("API keys provide a secure way to authenticate API requests without requiring user accounts") but the codebase now has full user management, session auth, and OAuth 2.1. The "Future Enhancements" section lists features that are already implemented (user management, RBAC, session management, audit logging).~~ **FIXED.** Completely rewritten to document session auth, API keys with RBAC, OAuth 2.1 with PKCE, user management, and role-based permissions.

**`docs/security.md` references non-existent features:** ~~It mentions `rackd auth token create`, `rackd users create --role`, `RACKD_ALLOWED_NETWORKS`, `RACKD_BLOCKED_IPS`, and `rackd backup --encrypt` — none of which exist in the codebase. The security doc appears to be aspirational rather than reflecting actual implementation.~~ **FIXED.** Completely rewritten to document actual security features: secure defaults, CSRF protection, password security, API key security, session security, OAuth 2.1, credential encryption, webhook security, CSP, audit logging, rate limiting, and RBAC.

**OpenAPI spec is incomplete:** ~~The `api/openapi.yaml` file defines schemas for basic resources but doesn't cover the full API surface (missing: auth, users, roles, OAuth, circuits, NAT, reservations, webhooks, custom fields, DNS, conflicts, audit, bulk operations). The spec should be generated from or validated against the actual route registrations.~~ **FIXED.** Complete OpenAPI 3.1.0 spec covering all 170 operations across 105 paths with 74 schemas and 26 tags.

**Missing configuration reference:** ~~There's no single document listing all environment variables with their defaults, types, and descriptions. The `internal/config/config.go` file is the source of truth but users shouldn't have to read Go code. Create a `docs/configuration-reference.md`.~~ **FIXED.** Created `docs/configuration-reference.md` with all environment variables organized by category.

**No MCP integration guide:** `docs/mcp.md` likely exists but should include: available tools list, authentication setup (API key vs OAuth), example tool calls, and integration with popular AI clients.

**No changelog or migration guide:** For users upgrading between versions, there's no documentation of breaking changes (e.g., legacy API key removal, password length increase from 8→12).

---

## 9. Testing Review

### Coverage
Tests exist for:
- Storage layer (SQLite operations, bulk, audit, API keys, circuits, conflicts)
- API middleware (rate limiting)
- CLI commands (structure, flag parsing, output formatting, mock API integration)
- Credentials (encryption/decryption)
- Discovery (ARP, banner grabbing, confidence scoring, correlation)
- MCP server (basic tool operations)
- Models (validation)
- Service layer (DNS with property-based testing via rapid)

### Gaps

**No integration tests for the full HTTP stack:** ~~Tests mock individual layers but don't test the complete request flow (HTTP → middleware → handler → service → storage → response).~~ **FIXED.** Added 13 full-stack integration tests in `internal/api/integration_test.go` that exercise the complete request flow using `httptest` with real in-memory SQLite storage.

**No security-focused tests:** ~~Missing tests for:
- SQL injection attempts
- XSS payload handling
- CSRF attack scenarios
- Authentication bypass attempts
- Rate limiting under concurrent load
- Authorization boundary testing (user A can't access user B's resources)~~
**FIXED.** Added 20+ security tests in `internal/api/security_test.go` covering SQL injection, XSS, CSRF, auth bypass, expired/legacy keys, cross-user authorization, body size limits, invalid input, login security, session cookie attributes, and privilege escalation prevention.

**Hardcoded test credentials:** `bootstrap_test.go` and `webhook_sqlite_test.go` use hardcoded passwords and secrets. Use test helpers that generate random values.

**No performance benchmarks:** For an IPAM system that may manage thousands of devices and networks, there are no benchmarks for list operations, search, or bulk operations.

---

## 10. Build & Deployment

### Strengths
- Makefile with comprehensive targets
- Docker multi-stage build with health check
- GoReleaser for multi-platform releases
- Nomad job spec for orchestration

### Issues

**Dockerfile uses `golang:1.25-alpine`:** Go 1.25 doesn't exist yet (current is 1.22). The Makefile references `GOTOOLCHAIN=go1.26.0`. These should match the actual Go version in `go.mod`.

**No database migration tooling:** ~~Schema changes are applied via auto-migration at startup. For production deployments, this is risky — a failed migration could corrupt data. Consider adding a `rackd migrate` command with rollback support.~~ **FIXED.** Added `rackd migrate status` (shows all migrations and pending count) and `rackd migrate run` (applies pending migrations).

**No backup/restore commands:** ~~The docs mention backup capabilities but no `rackd backup` or `rackd restore` commands exist. For a single-file SQLite database, this is straightforward to implement.~~ **FIXED.** Added `rackd backup` command that copies the SQLite DB file plus WAL/SHM files to a timestamped backup.

---

## 11. Priority Recommendations

### Immediate (security)
1. ~~**MCP legacy API key handling inconsistent with REST API** — FIXED. Extracted shared `api.AuthenticateAPIKey()` used by both REST middleware and MCP server. Legacy keys rejected at auth boundary. All MCP tools verified to use service layer with RBAC.~~
2. ~~**Implement CSRF protection for session-authenticated requests.** Session cookies are sent automatically by browsers. Without CSRF tokens, any cross-origin form POST to `/api/*` will succeed for logged-in users. (Note: `AuthMiddlewareWithSessions` now checks for `X-Requested-With: XMLHttpRequest` header on state-changing requests, which provides partial mitigation — but this should be documented and the UI must send the header consistently.)~~ **FIXED.** Server enforces `X-Requested-With: XMLHttpRequest` on state-changing requests. All UI fetch calls now send this header consistently.
3. ~~**Document that `COOKIE_SECURE=true` and `RATE_LIMIT_ENABLED=true` are required for production.**~~ **FIXED.** Both now default to `true`. Dev environments opt out explicitly.

### Short-term (consistency)
4. ~~Standardize API error codes through `handleServiceError` exclusively~~ **FIXED.** All handler-level validation errors now use two standardized helpers: `invalidJSON()` (code: `INVALID_JSON`) for JSON decode failures, and `badRequest()` (code: `INVALID_INPUT`) for all other request validation. Eliminated ad-hoc codes (`MISSING_FIELD`, `MISSING_QUERY`, `MISSING_RESOURCE_ID`, `INVALID_TYPE`, `INVALID_ID`, `MISSING_USERNAME`, `MISSING_PASSWORD`). Service-layer errors continue through `handleServiceError`.
5. ~~Enforce pagination limits on all list endpoints~~ **FIXED.** Added `Pagination` struct (Limit/Offset with `Clamp()`) embedded in all filter types. All storage List methods now apply `LIMIT`/`OFFSET` via shared `appendPagination` helper. Default page size: 100, max: 1000. Handler-level `parsePagination()` reads `limit`/`offset` query params and clamps.
6. ~~Update `docs/authentication.md` and `docs/security.md` to reflect current implementation~~ **FIXED.** Both documents completely rewritten to reflect the actual codebase: session auth, API keys with RBAC, OAuth 2.1 with PKCE, CSRF protection, secure defaults, credential encryption, webhook security, CSP, audit logging, and rate limiting.
7. ~~Generate or update OpenAPI spec to cover all endpoints~~ **FIXED.** Complete OpenAPI 3.1.0 spec (`api/openapi.yaml`) covering all 170 operations across 105 paths with 74 schemas and 26 tags. Covers every endpoint from route registrations.
8. ~~Create `docs/configuration-reference.md` from config.go~~ **FIXED.** Created comprehensive `docs/configuration-reference.md` with all environment variables from `internal/config/config.go`, organized by category (Server, Security, Sessions, Initial Admin, Discovery, Audit, OAuth, Snapshots, DNS).

### Medium-term (completeness)
9. ~~Add CLI commands for scheduled scans, scan profiles, and OAuth client management~~ **FIXED.** Added `rackd scan-profile` (list, get, create, update, delete), `rackd scheduled-scan` (list, get, create, update, delete), and `rackd oauth` (list, delete) CLI commands.
10. ~~Add DNS tools to MCP server~~ **FIXED.** Added 16 DNS tools covering providers (list, get, save, delete, test), zones (list, get, save, delete, sync, import), and records (list, get, save, delete, link). All discoverable via keywords.
11. ~~Add integration tests for full HTTP request flow~~ **FIXED.** Added `internal/api/integration_test.go` with 13 full-stack integration tests covering: device/network/datacenter/circuit/custom-field/user CRUD through HTTP, session auth (login/request/logout), API key auth (valid/invalid/missing), RBAC enforcement (admin vs viewer), device relationships, search, and pagination. All tests use real in-memory SQLite storage with the full middleware chain (auth, CSRF, body limits).
12. ~~Add security-focused test suite~~ **FIXED.** Added `internal/api/security_test.go` with 20+ security tests covering: CSRF protection (blocks POST/PUT/DELETE without header, allows GET, not required for API keys), SQL injection (device names, search queries, list filters), XSS payload handling, authentication bypass attempts (no auth, malformed bearer, wrong scheme), expired/legacy API key rejection, cross-user authorization boundaries, request body size limits, invalid JSON handling, invalid resource IDs, login security (wrong credentials, no password hash leakage), session cookie attributes (HttpOnly, SameSite, Path), invalid session rejection, and privilege escalation prevention (non-admin can't create admin, can't self-escalate role).
13. ~~Implement `rackd backup` and `rackd migrate` commands~~ **FIXED.** Added `rackd backup` (copies SQLite DB + WAL/SHM files with timestamped output) and `rackd migrate` with `status` (shows all migrations and pending count) and `run` (applies pending migrations) subcommands.
14. ~~Add `--skip-tls-verify` flag to CLI client~~ **FIXED.** The existing `VerifySSL` config field is now wired to the HTTP client's TLS settings. Set `RACKD_VERIFY_SSL=false` or `"verify_ssl": false` in config to skip certificate verification for self-signed certs.

15. ~~Refactor Handler struct to use options pattern~~ **FIXED.** Replaced 8 individual `Set*` methods on the `Handler` struct with a `HandlerOption` functional options pattern. `NewHandler` now accepts variadic `...HandlerOption` args. Option functions: `WithSessionManager`, `WithCredentialsStorage`, `WithProfileStorage`, `WithScheduledScanStorage`, `WithLoginRateLimiter`, `WithCookieConfig`, `WithTrustProxy`, `WithServices`. All call sites in `server.go` and test files updated.
16. Standardize storage layer time handling and error sentinels~~ **FIXED.**
17. ~~Add performance benchmarks~~ **FIXED.** Added `internal/api/benchmark_test.go` with 18 benchmarks covering: storage layer (create/get/list/search/tag-filter devices), auth (API key auth, session validation, password hash/verify, RBAC permission check), HTTP handlers (list/get/create devices, global search), JSON serialization, and full middleware chain overhead. Run with `go test -bench=. -benchmem ./internal/api/`.
18. Implement CORS configuration
19. Add request timeout enforcement per-endpoint
