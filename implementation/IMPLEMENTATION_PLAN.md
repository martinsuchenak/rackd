# Rackd Implementation Plan

**Last Updated**: 2026-02-10

Remaining features for Rackd, organized by priority. Phases 1-2 and most of Phase 3 are complete.

## Status

| Phase | Remaining | Status |
|-------|-----------|--------|
| **Phase 3: Multi-User** | 2 of 6 | 🚧 In Progress |
| **Phase 4: Advanced** | 10 of 10 | 🔮 Future |
| **Phase 5: Scale** | 3 of 3 | 🔮 Future |
| **Total remaining** | **15** | |

### Completed (not listed here)

- **Phase 1**: Full-Text Search, Metrics, Health Checks, API Key Auth
- **Phase 2**: Export/Import, Bulk Operations, Rate Limiting, Audit Trail, UI/UX Enhancements
- **Phase 3.1**: User Management (users, sessions, bcrypt, login UI, CLI, bootstrap via env vars)
- **Phase 3.2**: RBAC (roles, permissions, service-layer enforcement, default roles, UI, CLI)
- **Phase 3.3**: SSO/OIDC — moved to [FUTURE_FEATURES.md](FUTURE_FEATURES.md) (implement when requested)
- **Phase 3.4**: PostgreSQL — moved to [FUTURE_FEATURES.md](FUTURE_FEATURES.md) (SQLite handles the scale)

## Architecture Reference

```
cmd/              CLI commands (github.com/paularlott/cli)
internal/
  model/          Data models
  storage/        SQLite storage (implements interfaces from storage.go)
  service/        Business logic + RBAC enforcement (requirePermission)
  api/            HTTP handlers + route registration (handlers.go)
  auth/           Session management, password hashing, RBAC checker
  config/         Configuration
  server/         HTTP server setup + middleware
webui/src/
  components/     TypeScript UI components
  partials/pages/ HTML templates
```

**Route auth wrappers** (in `handlers.go`):
- `wrap` — optional auth (respects `cfg.requireAuth`)
- `wrapAuth` — always requires authenticated session
- `wrapPerm` — auth + RBAC permission check (skips RBAC when auth not configured)

**RBAC enforcement** happens at the service layer via `requirePermission(ctx, store, "resource", "action")`, not at the middleware level.

**Convention for new features**: model + storage interface/impl + service + API handlers + CLI command + web UI component.

---

## Phase 3: Multi-User (remaining)

### 3.6 MCP OAuth 2.1 Authorization

**Effort**: 5-7 days | **Priority**: HIGH (next up)

**What**: OAuth 2.1 authorization for the MCP server endpoint, ensuring MCP operations are performed as the correct user with proper RBAC enforcement. Also upgrades paularlott/mcp from v0.9.2 to v0.11.1.

**Why top priority**: Currently MCP auth uses a manual API key check in `HandleRequest()`. This doesn't follow the MCP authorization spec and means MCP clients like Claude Desktop have no standard way to authenticate. With OAuth, any spec-compliant MCP client can authenticate via the user's existing Rackd account and inherit their RBAC permissions.

**MCP Auth Spec Requirements** (2025-11-25):
- MCP server acts as an OAuth 2.1 **Resource Server** (validates access tokens)
- Separate **Authorization Server** issues tokens (can be built into Rackd since we own the user database)
- **Protected Resource Metadata** (RFC 9728) for client discovery
- Returns `401` with `WWW-Authenticate` header when unauthorized

**Tasks**:
- [ ] Upgrade `github.com/paularlott/mcp` from v0.9.2 to v0.11.1
- [ ] OAuth 2.1 Authorization Server:
  - [ ] `GET /oauth/authorize` — Authorization endpoint (renders login/consent page)
  - [ ] `POST /oauth/token` — Token endpoint (issues access + refresh tokens)
  - [ ] `POST /oauth/revoke` — Token revocation
  - [ ] Authorization Code grant with PKCE (for public clients like Claude Desktop)
  - [ ] Client Credentials grant (for service-to-service)
- [ ] OAuth token model and storage (access tokens, refresh tokens, authorization codes)
- [ ] Token scoping tied to user's RBAC permissions
- [ ] Protected Resource Metadata endpoint:
  - [ ] `GET /.well-known/oauth-protected-resource` (RFC 9728)
  - [ ] Points MCP clients to the authorization server endpoints
- [ ] Token validation middleware for MCP endpoint (replace current manual API key check)
- [ ] OAuth client registration (store client_id/redirect_uri pairs)
- [ ] OAuth client management UI (register/revoke MCP clients)
- [ ] Consent screen UI (user approves MCP client access)
- [ ] Tests for full OAuth flow (authorize → token → MCP call → RBAC check)

**Current auth flow** (to be replaced):
```
MCP client → Bearer API-key → HandleRequest() validates manually → system/API key caller
```

**New auth flow**:
```
MCP client → discovers /.well-known/oauth-protected-resource
           → redirects user to /oauth/authorize (login + consent)
           → receives authorization code
           → exchanges code for access token at /oauth/token
           → Bearer access-token → MCP endpoint validates token → user caller with RBAC
```

**Configuration**:
```bash
MCP_OAUTH_ENABLED=true                    # Enable OAuth for MCP (default: false)
MCP_OAUTH_ACCESS_TOKEN_TTL=1h             # Access token lifetime
MCP_OAUTH_REFRESH_TOKEN_TTL=30d           # Refresh token lifetime
MCP_OAUTH_AUTHORIZATION_CODE_TTL=10m      # Auth code lifetime
```

**Files to Create**:
- `internal/model/oauth.go` — OAuthClient, OAuthToken, OAuthAuthorizationCode models
- `internal/storage/oauth_sqlite.go` — Token/client/code storage
- `internal/auth/oauth.go` — Token generation, validation, PKCE verification
- `internal/service/oauth.go` — OAuth flow orchestration, consent logic
- `internal/api/oauth_handlers.go` — `/oauth/authorize`, `/oauth/token`, `/oauth/revoke`
- `internal/api/resource_metadata_handler.go` — `/.well-known/oauth-protected-resource`
- `webui/src/components/oauth-consent.ts` — Consent screen
- `webui/src/components/oauth-clients.ts` — Client management UI

**Files to Modify**:
- `internal/mcp/server.go` — Replace manual API key auth with OAuth token validation
- `internal/api/handlers.go` — Register OAuth + metadata routes
- `go.mod` — Upgrade paularlott/mcp to v0.11.1

**Backward Compatibility**:
- When `MCP_OAUTH_ENABLED=false` (default), existing API key auth continues to work
- When enabled, both OAuth tokens and API keys are accepted during transition period

### 3.5 Webhook System

**Effort**: 5-7 days | **Priority**: LOW (Optional)

**What**: Event notifications for external automation

**Tasks**:
- [ ] Webhook model and storage
- [ ] Webhook CRUD API + service
- [ ] Event dispatcher (publish/subscribe within app)
- [ ] Webhook delivery with retries and backoff
- [ ] HMAC signature verification for payloads
- [ ] Webhook management UI
- [ ] Webhook CLI commands
- [ ] Webhook MCP tools

**Events**:
- `device.created`, `device.updated`, `device.deleted`
- `network.created`, `network.updated`, `network.deleted`
- `discovery.scan.started`, `discovery.scan.completed`
- `device.promoted`

**Note**: Can be skipped if not doing heavy automation.

**Files to Create**:
- `internal/model/webhook.go`
- `internal/storage/webhook_sqlite.go`
- `internal/service/webhook.go` — CRUD + permission checks
- `internal/webhook/dispatcher.go` — Event bus
- `internal/webhook/delivery.go` — HTTP delivery with retries
- `internal/api/webhook_handlers.go`
- `webui/src/components/webhooks.ts`
- `cmd/webhook/webhook.go`

---

## Phase 4: Advanced Features

**Goal**: Advanced IPAM and integration features

### 4.1 Notifications & Alerting

**Effort**: 4-5 days | **Priority**: HIGH

**What**: Configurable notifications for infrastructure events via email, Slack, and Teams

**Tasks**:
- [ ] Notification channel model (email, Slack webhook, Teams webhook)
- [ ] Notification channel storage + service
- [ ] Channel CRUD API
- [ ] Internal event bus (reusable by webhooks later)
- [ ] Notification triggers with configurable thresholds:
  - Pool utilization exceeds threshold (e.g., 80%)
  - New device discovered
  - Discovery scan failure
  - IP conflict detected
  - Device status change
- [ ] Per-user notification preferences
- [ ] Notification history/log
- [ ] Notification management UI
- [ ] Notification CLI commands
- [ ] Email sender (SMTP)
- [ ] Slack sender (incoming webhook)
- [ ] Teams sender (incoming webhook)

**Configuration**:
```bash
NOTIFICATIONS_ENABLED=true
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_USERNAME=alerts@example.com
SMTP_PASSWORD=xxx
SMTP_FROM=rackd@example.com
```

**Files to Create**:
- `internal/model/notification.go` — Channel, trigger, and history models
- `internal/storage/notification_sqlite.go`
- `internal/service/notification.go` — CRUD + permission checks
- `internal/notification/dispatcher.go` — Event bus + trigger evaluation
- `internal/notification/email.go` — SMTP sender
- `internal/notification/slack.go` — Slack webhook sender
- `internal/notification/teams.go` — Teams webhook sender
- `internal/api/notification_handlers.go`
- `webui/src/components/notifications.ts`
- `cmd/notification/notification.go`

### 4.2 Device Lifecycle & Status Tracking

**Effort**: 2-3 days | **Priority**: HIGH

**What**: Track device lifecycle states with history and scheduled transitions

**Tasks**:
- [ ] Add `status` field to Device model (`planned`, `active`, `maintenance`, `decommissioned`)
- [ ] Status change history (stored in audit trail or dedicated table)
- [ ] Scheduled decommission date field
- [ ] Filter/search devices by lifecycle status
- [ ] Status badge in device list and detail UI
- [ ] Status change dropdown in device detail UI
- [ ] Dashboard widget: device count by status
- [ ] CLI: `rackd device list --status active`

**Files to Modify**:
- `internal/model/device.go` — Add Status and DecommissionDate fields
- `internal/storage/device_sqlite.go` — Migration + query filters
- `internal/service/device.go` — Status validation, transition logic
- `internal/api/device_handlers.go` — Accept status in create/update
- `webui/src/components/devices.ts` — Status filter, badge, dropdown
- `webui/src/components/dashboard.ts` — Status summary widget

### 4.3 Dashboard Reporting & Trends

**Effort**: 3-4 days | **Priority**: HIGH

**What**: Enhanced dashboard with utilization trends, activity feeds, and summary stats

**Tasks**:
- [ ] Pool utilization snapshots (periodic storage of utilization %)
- [ ] Utilization trend chart (sparkline or line chart over time)
- [ ] Recently discovered devices feed
- [ ] Network utilization summary (% used per subnet)
- [ ] Stale device detection (devices not seen in discovery for X days)
- [ ] Top-level stats: total devices, networks, pools, utilization
- [ ] Dashboard API endpoint for aggregated stats
- [ ] Configurable dashboard refresh interval

**Files to Create/Modify**:
- `internal/model/snapshot.go` — Utilization snapshot model
- `internal/storage/snapshot_sqlite.go` — Snapshot storage + periodic writer
- `internal/service/dashboard.go` — Aggregation queries, stale detection
- `internal/api/dashboard_handlers.go` — `/api/dashboard` endpoint
- `webui/src/components/dashboard.ts` — Trend charts, activity feed, stats cards

### 4.4 Network Topology Visualization

**Effort**: 7-10 days

**What**: Interactive visual network topology based on device relationships

**Tasks**:
- [ ] Topology data structure (nodes, edges from relationships)
- [ ] Topology API endpoint
- [ ] Graph layout algorithm
- [ ] Interactive visualization (Cytoscape.js or D3.js)
- [ ] Zoom, pan, filter controls
- [ ] Export (PNG, SVG, JSON)
- [ ] Real-time updates via WebSocket

**Note**: Basic relationship graph already exists at `/devices/graph`.

### 4.5 DNS Integration

**Effort**: 5-7 days

**What**: Automatic DNS record management

**Tasks**:
- [ ] DNS provider interface
- [ ] BIND support (via nsupdate)
- [ ] PowerDNS support (via API)
- [ ] DNS record CRUD
- [ ] Auto-create DNS records on device creation
- [ ] DNS sync functionality
- [ ] DNS configuration UI
- [ ] DNS CLI commands

### 4.6 DHCP Integration

**Effort**: 5-7 days

**What**: DHCP server integration for IP reservation

**Tasks**:
- [ ] DHCP provider interface
- [ ] ISC DHCP support (via omapi)
- [ ] Kea support (via API)
- [ ] DHCP lease management
- [ ] Auto-reserve IPs in pools
- [ ] DHCP configuration UI
- [ ] DHCP CLI commands

### 4.7 Circuit Management

**Effort**: 4-5 days

**What**: Track network circuits (provider, circuit ID, capacity, endpoints)

**Tasks**:
- [ ] Circuit model
- [ ] Circuit storage + service
- [ ] Circuit CRUD API
- [ ] Circuit UI page
- [ ] Link circuits to devices
- [ ] Circuit status tracking
- [ ] Circuit CLI commands

### 4.8 NAT Tracking

**Effort**: 3-4 days

**What**: Track NAT mappings (external IP/port to internal IP/port)

**Tasks**:
- [ ] NAT mapping model
- [ ] NAT storage + service
- [ ] NAT CRUD API
- [ ] NAT UI page
- [ ] Link NAT to devices
- [ ] NAT validation
- [ ] NAT CLI commands

### 4.9 IP Conflict Detection

**Effort**: 2-3 days

**What**: Detect and warn about IP conflicts

**Tasks**:
- [ ] Duplicate IP detection
- [ ] Overlapping subnet detection
- [ ] Conflict resolution UI
- [ ] Conflict API endpoints
- [ ] Automatic conflict checking on IP assignment

### 4.10 Custom Fields/Metadata

**Effort**: 4-5 days

**What**: User-defined fields for devices/networks

**Tasks**:
- [ ] Custom field model
- [ ] Custom field storage (JSON or key-value)
- [ ] Custom field CRUD API + service
- [ ] Custom field UI
- [ ] Custom field validation
- [ ] Search/filter by custom fields

---

## Phase 5: Scale & Performance

**Goal**: Optimize for large deployments

### 5.1 Query Optimization

**Effort**: Ongoing

**Tasks**:
- [ ] Query performance logging
- [ ] Identify slow queries
- [ ] Add missing indexes
- [ ] Optimize N+1 queries
- [ ] Query result caching

### 5.2 Enhanced Pagination

**Effort**: 2-3 days

**Tasks**:
- [ ] Cursor-based pagination
- [ ] Pagination on all list endpoints
- [ ] CLI pagination support
- [ ] UI pagination support

### 5.3 Caching Layer

**Effort**: 3-4 days

**Tasks**:
- [ ] In-memory cache implementation
- [ ] Cache frequently accessed data
- [ ] Cache invalidation strategy
- [ ] Cache metrics
- [ ] Cache configuration

---

## Success Criteria (remaining)

### Phase 3
- [ ] MCP clients authenticate via OAuth 2.1 with PKCE
- [ ] MCP operations enforced under the authenticating user's RBAC permissions

### Phase 4
- [ ] Notifications delivered within 30s of trigger event
- [ ] Device lifecycle transitions tracked with full history
- [ ] Dashboard loads aggregated stats in <200ms
- [ ] Topology renders 500+ devices smoothly
- [ ] DNS/DHCP sync works reliably
- [ ] Circuit/NAT tracking accurate

### Phase 5
- [ ] API p95 latency <200ms
- [ ] Database queries <50ms
- [ ] Cache hit rate >80%

---

## Notes

- All features include tests, documentation, CLI, API, and UI where applicable
- All features include MCP tools where applicable
- New features follow the service-layer pattern: model -> storage -> service (with RBAC) -> API handler
- Webhook system is optional (can be skipped)
- See [FUTURE_FEATURES.md](FUTURE_FEATURES.md) for ideas not yet planned (SSO, PostgreSQL, etc.)
