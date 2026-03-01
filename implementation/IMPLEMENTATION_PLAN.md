# Rackd Implementation Plan

**Last Updated**: 2026-02-27

Remaining features for Rackd, organized by priority. Phases 1-2 and most of Phase 3 are complete.

## Status

| Phase | Remaining | Status |
|-------|-----------|--------|
| **Phase 3: Multi-User** | 0 of 6 | ✅ Complete |
| **Phase 4: Advanced** | 3 of 12 | 🟡 In Progress |
| **Phase 5: Scale** | 3 of 3 | 🔮 Future |
| **Total remaining** | **7** | |

### Completed (not listed here)

- **Phase 1**: Full-Text Search, Metrics, Health Checks, API Key Auth
- **Phase 2**: Export/Import, Bulk Operations, Rate Limiting, Audit Trail, UI/UX Enhancements
- **Phase 3.1**: User Management (users, sessions, bcrypt, login UI, CLI, bootstrap via env vars)
- **Phase 3.6**: MCP OAuth 2.1 Authorization
- **Phase 4.1**: IP Conflict Detection
- **Phase 4.2**: IP Address Reservation & Planning
- **Phase 4.3**: Device Lifecycle & Status Tracking
- **Phase 4.4**: Dashboard Reporting & Trends
- **Phase 4.5**: Network Topology Visualization
- **Phase 4.6**: Notifications & Alerting — MOVED TO FUTURE (webhooks provide external integration)
- **Phase 4.7**: Webhook System
- **Phase 3.2**: RBAC (roles, permissions, service-layer enforcement, default roles, UI, CLI)
- **Phase 3.3**: SSO/OIDC — moved to [FUTURE_FEATURES.md](FUTURE_FEATURES.md) (implement when requested)
- **Phase 3.4**: PostgreSQL — moved to [FUTURE_FEATURES.md](FUTURE_FEATURES.md) (SQLite handles the scale)
- **Phase 3.5**: Webhook System — moved to Phase 4.7 (depends on Notifications event bus)
- **Phase 3.6**: MCP OAuth 2.1 Authorization (OAuth endpoints, PKCE, token validation, client management UI, consent screen)
- **Phase 4.1**: IP Conflict Detection
- **Phase 4.2**: IP Address Reservation & Planning
- **Phase 4.3**: Device Lifecycle & Status Tracking
- **Phase 4.4**: Dashboard Reporting & Trends

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

### 4.4 Dashboard Reporting & Trends ✅

**Effort**: 3-4 days | **Priority**: HIGH

**What**: Enhanced dashboard with utilization trends, activity feeds, and summary stats

**Tasks**:
- [x] Pool utilization snapshots (periodic storage of utilization %)
- [x] Utilization trend chart (sparkline or line chart over time)
- [x] Recently discovered devices feed
- [x] Network utilization summary (% used per subnet)
- [x] Stale device detection (devices not seen in discovery for X days)
- [x] Top-level stats: total devices, networks, pools, utilization
- [x] Dashboard API endpoint for aggregated stats
- [x] Configurable dashboard refresh interval

**Implementation Details**:
- **Model**: `internal/model/snapshot.go` — UtilizationSnapshot, DashboardStats, RecentDiscovery, NetworkUtilizationSummary, UtilizationTrendPoint types
- **Storage**: `internal/storage/snapshot_sqlite.go` — CreateSnapshot, ListSnapshots, GetLatestSnapshots, DeleteOldSnapshots, GetUtilizationTrend, GetDashboardStats methods
- **Migration**: `internal/storage/migrations.go` — Migration 20260228010000 for utilization_snapshots table
- **Worker**: `internal/worker/snapshot.go` — Background worker for periodic snapshot collection
- **Config**: `internal/config/config.go` — SnapshotInterval (default 1h), SnapshotRetentionDays (default 90)
- **Service**: `internal/service/dashboard.go` — DashboardService with RBAC enforcement
- **API**: `internal/api/dashboard_handlers.go` — GET /api/dashboard, GET /api/dashboard/trend endpoints
- **Types**: `webui/src/core/types.ts` — DashboardStats, RecentDiscovery, NetworkUtilizationSummary, UtilizationTrendPoint types
- **API Client**: `webui/src/core/api.ts` — getDashboardStats(), getUtilizationTrend() methods
- **Component**: `webui/src/components/dashboard.ts` — dashboardComponent() with auto-refresh, trend chart rendering
- **Template**: `webui/src/partials/pages/dashboard.html` — Enhanced dashboard with stats cards, network utilization list, trend chart, recent discoveries, health alerts

### 4.5 Network Topology Visualization

**Effort**: 7-10 days | **Status**: ✅ COMPLETE

**What**: Interactive visual network topology based on device relationships

**Tasks**:
- [x] Topology data structure (nodes, edges from relationships)
- [x] Topology API endpoint
- [x] Graph layout algorithm
- [x] Interactive visualization (Cytoscape.js or D3.js)
- [x] Zoom, pan, filter controls
- [x] Export (PNG, SVG, JSON)
- [ ] Real-time updates via WebSocket (deferred)

**Files Modified**:
- `webui/src/components/graph.ts` — Enhanced Cytoscape.js component with filters, export, layout options
- `webui/src/partials/pages/device-graph.html` — Toolbar, filter controls, export buttons, stats

**Note**: Basic relationship graph already existed at `/devices/graph`. Enhanced with zoom/pan controls, filtering by status/datacenter/relationship type, multiple layout algorithms, PNG/JSON export, and node tooltips.

### 4.6 Notifications & Alerting — MOVED TO FUTURE

**Effort**: 4-5 days | **Priority**: DEFERRED

**What**: Configurable notifications for infrastructure events via email, Slack, and Teams

**Note**: Moved to [FUTURE_FEATURES.md](FUTURE_FEATURES.md). The webhook system (4.7) provides external event integration, which satisfies immediate automation needs. In-app notifications can be added later.

### 4.7 Webhook System ✅

**Effort**: 5-7 days | **Status**: COMPLETE

**What**: Event notifications for external automation

**Tasks**:
- [x] Webhook model and storage
- [x] Webhook CRUD API + service
- [x] Event bus for internal events (independent of notifications)
- [x] Webhook delivery with retries and backoff
- [x] HMAC signature verification for payloads
- [x] Webhook management UI
- [x] Webhook CLI commands
- [ ] Webhook MCP tools (deferred)

**Events**:
- `device.created`, `device.updated`, `device.deleted`, `device.promoted`
- `network.created`, `network.updated`, `network.deleted`
- `discovery.started`, `discovery.completed`, `discovery.device_found`
- `conflict.detected`, `conflict.resolved`
- `pool.utilization_high`

**Implementation Details**:
- **Model**: `internal/model/webhook.go` — Webhook, WebhookDelivery, Event types, DeliveryStatus
- **Storage**: `internal/storage/webhook_sqlite.go` — SQLite implementation with CRUD operations
- **Migration**: `internal/storage/migrations.go` — Migration 20260228030000 for webhooks and deliveries tables
- **Event Bus**: `internal/webhook/eventbus.go` — Simple pub/sub event dispatcher
- **Delivery**: `internal/webhook/delivery.go` — HTTP delivery with retries, exponential backoff, HMAC signatures
- **Worker**: `internal/webhook/worker.go` — Background worker for processing pending deliveries
- **Service**: `internal/service/webhook.go` — CRUD + RBAC enforcement
- **API**: `internal/api/webhook_handlers.go` — REST endpoints for webhooks and deliveries
- **Types**: `webui/src/core/types.ts` — TypeScript types for webhooks
- **API Client**: `webui/src/core/api.ts` — API methods for webhooks
- **Component**: `webui/src/components/webhooks.ts` — Webhook management UI component
- **Template**: `webui/src/partials/pages/webhooks.html` — Webhook management page
- **CLI**: `cmd/webhook/` — `list`, `get`, `create`, `update`, `delete`, `ping`, `events` commands
- **RBAC**: `webhook:list`, `webhook:read`, `webhook:create`, `webhook:update`, `webhook:delete` permissions
- `internal/service/webhook.go` — CRUD + permission checks
- `internal/webhook/delivery.go` — HTTP delivery with retries
- `internal/api/webhook_handlers.go`
- `webui/src/components/webhooks.ts`
- `cmd/webhook/webhook.go`

### 4.8 DNS Integration

**Effort**: 5-7 days

**What**: Automatic DNS record management

**Detailed Plan**: See [DNS_INTEGRATION.md](DNS_INTEGRATION.md)

**Tasks**:
- [ ] DNS provider interface
- [ ] Technitium DNS support (via API), Docs: https://github.com/TechnitiumSoftware/DnsServer/blob/master/APIDOCS.md
- [ ] PowerDNS support (via API) - future
- [ ] BIND support (via nsupdate) - future
- [ ] PI-Hole DNS support (via API) - future
- [ ] DNS record CRUD
- [ ] Auto-create DNS records on device creation
- [ ] DNS sync functionality
- [ ] Import records from DNS
- [ ] PTR zone auto-generation
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

### 4.10 Circuit Management ✅

**Effort**: 4-5 days | **Status**: COMPLETE

**What**: Track network circuits (provider, circuit ID, capacity, endpoints)

**Tasks**:
- [x] Circuit model
- [x] Circuit storage + service
- [x] Circuit CRUD API
- [x] Circuit UI page
- [x] Link circuits to devices
- [x] Circuit status tracking
- [x] Circuit CLI commands

**Implementation Details**:
- **Model**: `internal/model/circuit.go` — CircuitType (fiber/copper/microwave/dark_fiber), CircuitStatus (active/inactive/planned/decommissioned), Circuit model with endpoints, capacity, provider info
- **Storage**: `internal/storage/circuit_sqlite.go` — SQLite implementation with CRUD operations, filtering by provider/status/datacenter/type
- **Migration**: `internal/storage/migrations.go` — Migration 20260228070000 for circuits table with RBAC permissions
- **Service**: `internal/service/circuit.go` — CRUD + RBAC enforcement
- **API**: `internal/api/circuit_handlers.go` — REST endpoints for circuits
- **Types**: `webui/src/core/types.ts` — CircuitType, CircuitStatus, Circuit, CircuitFilter, CreateCircuitRequest, UpdateCircuitRequest types
- **API Client**: `webui/src/core/api.ts` — listCircuits, getCircuit, createCircuit, updateCircuit, deleteCircuit methods
- **Component**: `webui/src/components/circuits.ts` — Circuit management UI component with filters, forms, CRUD
- **Template**: `webui/src/partials/pages/circuits.html` — Circuit management page with table, filters, create/edit modals
- **CLI**: `cmd/circuit/` — `list`, `get`, `create`, `update`, `delete` commands
- **RBAC**: `circuit:list`, `circuit:read`, `circuit:create`, `circuit:update`, `circuit:delete` permissions

### 4.11 NAT Tracking ✅

**Effort**: 3-4 days | **Status**: COMPLETE

**What**: Track NAT mappings (external IP/port to internal IP/port)

**Tasks**:

- [x] NAT mapping model
- [x] NAT storage + service
- [x] NAT CRUD API
- [x] NAT UI page
- [x] Link NAT to devices
- [x] NAT validation
- [x] NAT CLI commands

**Implementation Details**:
- **Model**: `internal/model/nat.go` — NATProtocol (tcp/udp/any), NATMapping, NATFilter, CreateNATRequest, UpdateNATRequest
- **Storage**: `internal/storage/nat_sqlite.go` — SQLite implementation with full CRUD and filtering
- **Migration**: `internal/storage/migrations.go` — Migration for nat_mappings table with indexes and RBAC permissions
- **Service**: `internal/service/nat.go` — CRUD + RBAC enforcement, validation (ports, protocols)
- **API**: `internal/api/nat_handlers.go` — REST endpoints for NAT mappings
- **Types**: `webui/src/core/types.ts` — NATProtocol, NATMapping, NATFilter, CreateNATRequest, UpdateNATRequest types
- **API Client**: `webui/src/core/api.ts` — listNATMappings, getNATMapping, createNATMapping, updateNATMapping, deleteNATMapping methods
- **Component**: `webui/src/components/nat.ts` — NAT management UI component with filters, forms, CRUD
- **Template**: `webui/src/partials/pages/nat.html` — NAT management page with table, filters, create/edit modals
- **CLI**: `cmd/nat/` — `list`, `get`, `create`, `update`, `delete` commands
- **RBAC**: `nat:list`, `nat:read`, `nat:create`, `nat:update`, `nat:delete` permissions

### 4.12 Custom Fields/Metadata ✅

**Effort**: 4-5 days | **Status**: COMPLETE

**What**: User-defined fields for devices

**Tasks**:
- [x] Custom field model (definitions + values)
- [x] Custom field storage (hybrid: definitions table + values table)
- [x] Custom field CRUD API + service with RBAC
- [x] Custom field admin UI
- [x] Custom field validation (type, required, options for select)
- [x] Custom fields tab in device form
- [x] Custom fields display on device detail page
- [x] Custom fields CLI commands
- [ ] Search/filter by custom fields (deferred)
- [ ] Custom fields on networks/pools (deferred)

**Implementation Details**:
- **Model**: `internal/model/custom_field.go` — CustomFieldType (text/number/boolean/select), CustomFieldDefinition, CustomFieldValue, CustomFieldValueInput
- **Storage**: `internal/storage/custom_field_sqlite.go` — SQLite implementation with definitions and values tables
- **Migration**: `internal/storage/migrations.go` — Migration for custom_field_definitions and custom_field_values tables
- **Service**: `internal/service/custom_field.go` — CRUD + RBAC enforcement, validation
- **API**: `internal/api/custom_field_handlers.go` — REST endpoints for definitions and types
- **Device Integration**: `internal/storage/device_sqlite.go` — Custom fields loaded with device, saved on create/update
- **Device Handler**: `internal/api/device_handlers.go` — toCustomFieldSlice helper, custom_fields handling in updateDevice
- **Types**: `webui/src/core/types.ts` — CustomFieldDefinition, CustomFieldType, CustomFieldValueInput types
- **API Client**: `webui/src/core/api.ts` — listCustomFieldDefinitions, getCustomFieldTypes, createCustomFieldDefinition, updateCustomFieldDefinition, deleteCustomFieldDefinition methods
- **Component**: `webui/src/components/custom-fields.ts` — Custom field management UI component
- **Template**: `webui/src/partials/pages/custom-fields.html` — Custom field management page
- **Device Form**: `webui/src/partials/modals/device-form.html` — Custom Fields tab with dynamic inputs
- **Device Detail Edit**: `webui/src/partials/modals/device-detail-edit.html` — Custom Fields tab
- **Device Detail**: `webui/src/partials/pages/device-detail.html` — Custom fields display section
- **Device Component**: `webui/src/components/devices.ts` — getCustomFieldValue, setCustomFieldValue helpers in both devices and deviceDetail components
- **CLI**: `cmd/customfield/` — `list`, `get`, `create`, `update`, `delete`, `types` commands
- **RBAC**: `custom-fields:list`, `custom-fields:read`, `custom-fields:create`, `custom-fields:update`, `custom-fields:delete` permissions
- **Tests**: `internal/storage/custom_field_sqlite_test.go`, `internal/api/custom_field_handlers_test.go`

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