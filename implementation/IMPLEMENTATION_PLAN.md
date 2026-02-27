# Rackd Implementation Plan

**Last Updated**: 2026-02-27

Remaining features for Rackd, organized by priority. Phases 1-2 and most of Phase 3 are complete.

## Status

| Phase | Remaining | Status |
|-------|-----------|--------|
| **Phase 3: Multi-User** | 0 of 6 | ✅ Complete |
| **Phase 4: Advanced** | 8 of 12 | 🟡 In Progress |
| **Phase 5: Scale** | 3 of 3 | 🔮 Future |
| **Total remaining** | **11** | |

### Completed (not listed here)

- **Phase 1**: Full-Text Search, Metrics, Health Checks, API Key Auth
- **Phase 2**: Export/Import, Bulk Operations, Rate Limiting, Audit Trail, UI/UX Enhancements
- **Phase 3.1**: User Management (users, sessions, bcrypt, login UI, CLI, bootstrap via env vars)
- **Phase 3.2**: RBAC (roles, permissions, service-layer enforcement, default roles, UI, CLI)
- **Phase 3.3**: SSO/OIDC — moved to [FUTURE_FEATURES.md](FUTURE_FEATURES.md) (implement when requested)
- **Phase 3.4**: PostgreSQL — moved to [FUTURE_FEATURES.md](FUTURE_FEATURES.md) (SQLite handles the scale)
- **Phase 3.5**: Webhook System — moved to Phase 4.7 (depends on Notifications event bus)
- **Phase 3.6**: MCP OAuth 2.1 Authorization (OAuth endpoints, PKCE, token validation, client management UI, consent screen)
- **Phase 4.1**: IP Conflict Detection
- **Phase 4.2**: IP Address Reservation & Planning
- **Phase 4.3**: Device Lifecycle & Status Tracking

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
- `wrapAuth` — Always requires authenticated session
- `wrapPerm` — Auth + RBAC permission check (skips RBAC when auth not configured)

**RBAC enforcement** happens at the service layer via `requirePermission(ctx, store, "resource", "action")`, not at the middleware level.

**Convention for new features**: model + storage interface/impl + service + API handlers + CLI command + web UI component.

---

## Phase 4: Advanced Features

**Goal**: Advanced IPAM and integration features

### 4.1 IP Conflict Detection

**Effort**: 2-3 days | **Priority**: HIGH

**What**: Detect and warn about IP conflicts (core IPAM integrity)

**Tasks**:
- [x] Duplicate IP detection
- [x] Overlapping subnet detection
- [x] Conflict resolution UI
- [x] Conflict API endpoints
- [x] Automatic conflict checking on IP assignment

**Implementation Details**:
- **Model**: `internal/model/conflict.go` — ConflictType, ConflictStatus, Conflict, ConflictResolution
- **Storage**: `internal/storage/conflict_sqlite.go` — SQLite implementation with FindDuplicateIPs, FindOverlappingSubnets, CRUD operations
- **Service**: `internal/service/conflict.go` — ConflictService with RBAC enforcement
- **API**: `internal/api/conflict_handlers.go` — REST endpoints for  - **UI**: `webui/src/components/conflicts.ts` + `webui/src/partials/pages/conflicts.html`
- **CLI**: `cmd/conflict/` — `list`, `get`, `detect`, `resolve`, `delete` commands
- **Tests**: `internal/storage/conflict_sqlite_test.go` + `cmd/conflict/conflict_test.go`
- **Integration**: Automatic conflict checking in device service,- **UI Integration**: Navigation badge with dynamic count, conflict warning banners,- device detail page warning
- Device list conflict indicator
- Network list overlap badge
- Pool heatmap "conflicted" status (orange)

### 4.2 IP Address Reservation & Planning

**Effort**: 2-3 days | **Priority**: HIGH

**What**: Reserve IPs before assignment for planning phases

**Tasks**:
- [x] Reservation model (IP, purpose, reserved_by, expires_at)
- [x] Reservation storage + service
- [x] Reserve IPs without assigning to a device
- [x] Reservation expiration (auto-release if not claimed within X days)
- [x] Reservation notes/purpose field
- [x] Reserve for a specific user
- [x] API: `POST /api/pools/{id}/reservations`, `DELETE /api/pools/{id}/reservations/{ip}`
- [x] Show reservations in pool heatmap (different color from assigned IPs)
- [x] Reservation CLI commands

**Implementation Details**:
- **Model**: `internal/model/reservation.go` — Reservation model with status (active/expired/claimed/released)
- **Storage**: `internal/storage/reservation_sqlite.go` — SQLite implementation with CRUD operations
- **Service**: `internal/service/reservation.go` — CRUD + expiration logic with RBAC enforcement
- **API**: `internal/api/reservation_handlers.go` — REST endpoints for reservations
- **CLI**: `cmd/reservation/` — `list`, `get`, `create`, `update`, `delete`, `release` commands
- **Types**: `webui/src/core/types.ts` — TypeScript types for reservations
- **API Client**: `webui/src/core/api.ts` — API methods for reservations
- **Migration**: `internal/storage/migrations.go` — Database schema with indexes
- **RBAC**: `reservation:list`, `reservation:read`, `reservation:create`, `reservation:update`, `reservation:delete` permissions

### 4.3 Device Lifecycle & Status Tracking ✅

**Effort**: 2-3 days | **Priority**: HIGH

**What**: Track device lifecycle states with history and scheduled transitions

**Tasks**:
- [x] Add `status` field to Device model (`planned`, `active`, `maintenance`, `decommissioned`)
- [x] Status change history (stored in audit trail or dedicated table)
- [x] Scheduled decommission date field
- [x] Filter/search devices by lifecycle status
- [x] Status badge in device list and detail UI
- [x] Status change dropdown in device detail UI
- [x] Dashboard widget: device count by status
- [x] CLI: `rackd device list --status active`

**Implementation Details**:
- **Model**: `internal/model/device.go` — DeviceStatus type with validation, Status/DecommissionDate/StatusChangedAt/StatusChangedBy fields
- **Storage**: `internal/storage/device_sqlite.go` — Status filtering in ListDevices, GetDeviceStatusCounts method
- **Migration**: `internal/storage/migrations.go` — Migration 20260228000000 for status columns
- **Service**: `internal/service/device.go` — Status validation, setStatusChangedBy helper, GetStatusCounts method
- **API**: `internal/api/device_handlers.go` — Status filter param, /api/devices/status-counts endpoint
- **CLI**: `cmd/device/list.go` — --status and --pool flags
- **Types**: `webui/src/core/types.ts` — DeviceStatus and DeviceStatusCounts types
- **API Client**: `webui/src/core/api.ts` — getDeviceStatusCounts method
- **UI Components**: `webui/src/components/devices.ts` — statusFilter state, URL param handling
- **UI Templates**: `webui/src/partials/pages/devices.html` — Status filter dropdown, status badge styling
- **UI Templates**: `webui/src/partials/pages/dashboard.html` — Device Status section with clickable cards
- **Modal**: `webui/src/partials/modals/device-form.html` — Status dropdown for editing
- **Tests**: `internal/storage/device_sqlite_test.go` — Device status tests

### 4.4 Dashboard Reporting & Trends

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

### 4.5 Network Topology Visualization

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

### 4.6 Notifications & Alerting

**Effort**: 4-5 days | **Priority**: HIGH

**What**: Configurable notifications for infrastructure events via email, Slack, and Teams

**Tasks**:
- [ ] Notification channel model (email, Slack webhook, Teams webhook)
- [ ] Notification channel storage + service
- [ ] Channel CRUD API
- [ ] Internal event bus (reusable by webhooks)
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

### 4.7 Webhook System

**Effort**: 5-7 days | **Priority**: LOW (Optional)

**What**: Event notifications for external automation

**Tasks**:
- [ ] Webhook model and storage
- [ ] Webhook CRUD API + service
- [ ] Webhook delivery with retries and backoff (subscribes to event bus from Notifications)
- [ ] HMAC signature verification for payloads
- [ ] Webhook management UI
- [ ] Webhook CLI commands
- [ ] Webhook MCP tools

**Events**:
- `device.created`, `device.updated`, `device.deleted`
- `network.created`, `network.updated`, `network.deleted`
- `discovery.scan.started`, `discovery.scan.completed`
- `device.promoted`

**Note**: Can be skipped if not doing heavy automation. Depends on Notifications (4.6) for event bus.

**Files to Create**:
- `internal/model/webhook.go`
- `internal/storage/webhook_sqlite.go`
- `internal/service/webhook.go` — CRUD + permission checks
- `internal/webhook/delivery.go` — HTTP delivery with retries
- `internal/api/webhook_handlers.go`
- `webui/src/components/webhooks.ts`
- `cmd/webhook/webhook.go`

### 4.8 DNS Integration

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

### 4.9 DHCP Integration

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

### 4.10 Circuit Management

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

### 4.11 NAT Tracking

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

### 4.12 Custom Fields/Metadata

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
- [x] MCP clients authenticate via OAuth 2.1 with PKCE
- [x] MCP operations enforced under the authenticating user's RBAC permissions

### Phase 4
- [ ] IP conflicts detected and flagged before assignment
- [ ] IP reservations expire automatically
- [ ] Device lifecycle transitions tracked with full history
- [ ] Dashboard loads aggregated stats in <200ms
- [ ] Topology renders 500+ devices smoothly
- [ ] Notifications delivered within 30s of trigger event
- [ ] Webhooks deliver with retry and HMAC verification
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
- See [FUTURE_FEATURES.md](FUTURE_FEATURES.md) for ideas not yet planned (SSO, PostgreSQL, etc.)
- 