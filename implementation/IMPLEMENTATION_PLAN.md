# Rackd Implementation Plan

**Last Updated**: 2026-02-03

This document tracks all planned features for Rackd, organized by priority and implementation status.

## Quick Status

| Phase | Features | Completed | Status |
|-------|----------|-----------|--------|
| **Phase 1: Core** | 4 | 4/4 (100%) | ✅ Complete |
| **Phase 2: Production Ready** | 4 | 1/4 (25%) | 🚧 In Progress |
| **Phase 3: Multi-User** | 5 | 0/5 (0%) | 🔜 Planned |
| **Phase 4: Advanced** | 7 | 0/7 (0%) | 🔮 Future |
| **Phase 5: Scale** | 3 | 0/3 (0%) | 🔮 Future |
| **Total** | **23** | **5/23 (22%)** | |

---

## Phase 1: Core Features ✅ COMPLETE

**Goal**: Essential features for basic IPAM functionality

### 1.1 Full-Text Search ✅ COMPLETED (2026-02-03)

**Effort**: 2-3 days | **Priority**: High

**What**: Fast FTS5-powered search across devices, networks, and datacenters

**Completed**:
- ✅ FTS5 virtual tables for devices, networks, datacenters
- ✅ Prefix matching (search "de" matches "Default")
- ✅ Unified `/api/search` endpoint
- ✅ Automatic index maintenance via triggers
- ✅ Web UI integration

**Files**: `internal/storage/migrations.go`, `internal/api/search_handlers.go`, `docs/fts.md`

### 1.2 Metrics & Monitoring ✅ COMPLETED (2026-02-03)

**Effort**: 2-3 days | **Priority**: High

**What**: Prometheus-compatible metrics for observability

**Completed**:
- ✅ `/metrics` endpoint (Prometheus text format)
- ✅ HTTP metrics (requests, duration, status codes)
- ✅ Application metrics (device/network/datacenter counts)
- ✅ Discovery metrics (scan count, duration)
- ✅ Database metrics (queries, connections)
- ✅ Runtime metrics (uptime, goroutines, memory)

**Files**: `internal/metrics/`, `internal/api/metrics_handlers.go`, `docs/monitoring.md`

### 1.3 Enhanced Health Checks ✅ COMPLETED (2026-02-03)

**Effort**: 1 day | **Priority**: High

**What**: Kubernetes-ready health endpoints

**Completed**:
- ✅ `/healthz` - Liveness probe
- ✅ `/readyz` - Readiness probe with detailed checks
- ✅ Database connectivity check
- ✅ Scheduler status check
- ✅ JSON status responses

**Files**: `internal/api/health_handlers.go`

### 1.4 API Key Authentication ✅ COMPLETED (2026-02-03)

**Effort**: 2-3 days | **Priority**: High

**What**: Secure API key management (foundation for user management)

**Completed**:
- ✅ API key CRUD (create, list, get, delete)
- ✅ Secure 256-bit random key generation
- ✅ Expiration support
- ✅ Last-used tracking
- ✅ CLI commands (`rackd apikey`)
- ✅ REST API endpoints (`/api/keys`)
- ✅ MCP server integration
- ✅ Optional by default (no auth required)

**Files**: `internal/auth/`, `internal/storage/apikey_sqlite.go`, `cmd/apikey/`, `docs/authentication.md`

---

## Phase 2: Production Ready 📋 NEXT

**Goal**: Features needed for production deployment

### 2.1 Data Export/Import ✅ COMPLETED (2026-02-03)

**Effort**: 2-3 days | **Priority**: HIGH

**What**: Export/import data for backup and migration

**Completed**:
- ✅ Export devices to CSV/JSON
- ✅ Export networks to CSV/JSON
- ✅ Export datacenters to CSV/JSON
- ✅ Export all data to JSON
- ✅ Import devices from CSV/JSON
- ✅ Import networks from CSV/JSON
- ✅ Import datacenters from CSV/JSON
- ✅ Format auto-detection from file extension
- ✅ Dry-run mode for validation
- ✅ CLI commands (`rackd export/import`)
- ✅ Comprehensive tests (13 tests, all passing)

**Features**:
- Auto-detect format from file extension (.json/.csv)
- `--dry-run` flag to validate without importing
- `--format` flag to override auto-detection
- `--output` flag for export (stdout if omitted)
- Detailed import results (total, created, failed)
- Error reporting for failed imports

**Usage Examples**:
```bash
# Export
rackd export devices --format json --output devices.json
rackd export networks --format csv --output networks.csv
rackd export all --output backup.json

# Import
rackd import devices --file devices.json
rackd import networks --file networks.csv --dry-run
rackd import datacenters --file datacenters.json
```

**Files Created**:
- `internal/export/export.go` - Export functions
- `internal/export/export_test.go` - Export tests (6 tests)
- `internal/importdata/import.go` - Import functions
- `internal/importdata/import_test.go` - Import tests (7 tests)
- `cmd/export/export.go` - Export CLI commands
- `cmd/import/import.go` - Import CLI commands

### 2.2 Bulk Operations

**Effort**: 3-4 days | **Priority**: HIGH

**What**: Manage large numbers of devices/networks efficiently

**Tasks**:
- [ ] Bulk device create/update/delete
- [ ] Bulk tag add/remove
- [ ] Bulk network operations
- [ ] Transaction support (all-or-nothing)
- [ ] Progress reporting for large operations
- [ ] CLI commands
- [ ] API endpoints
- [ ] Web UI bulk import page

**Why High Priority**:
- Makes large-scale management practical
- Works well with export/import
- Common user request

**Files to Create**:
- `internal/api/bulk_handlers.go`
- `internal/storage/bulk_operations.go`
- `cmd/device/bulk.go`
- `webui/src/components/bulk-import.ts`

### 2.3 API Rate Limiting

**Effort**: 2 days | **Priority**: MEDIUM

**What**: Prevent API abuse

**Tasks**:
- [ ] Rate limiting middleware
- [ ] Per-IP rate limits
- [ ] Per-API-key rate limits
- [ ] Rate limit headers (X-RateLimit-*)
- [ ] Configuration (limits, windows)
- [ ] Bypass for localhost
- [ ] Documentation

**Why Medium Priority**:
- Important for public-facing deployments
- Quick to implement
- Production hardening

**Files to Create**:
- `internal/api/ratelimit.go`
- `internal/config/config.go` (add rate limit config)

### 2.4 Change History/Audit Trail

**Effort**: 4-5 days | **Priority**: HIGH

**What**: Track all changes for compliance and troubleshooting

**Tasks**:
- [ ] Audit log model and storage
- [ ] Audit middleware (capture all changes)
- [ ] Log CRUD operations with context
- [ ] Log authentication events
- [ ] API endpoints for querying audit log
- [ ] Web UI audit log page
- [ ] Export audit log (CSV/JSON)
- [ ] Retention policy configuration

**Why High Priority**:
- Essential for compliance (SOC2, ISO27001)
- Critical for troubleshooting
- Foundation for multi-user environments
- Should be implemented BEFORE full user management

**Files to Create**:
- `internal/model/audit.go`
- `internal/storage/audit_sqlite.go`
- `internal/api/audit_middleware.go`
- `internal/api/audit_handlers.go`
- `webui/src/components/audit.ts`
- `docs/audit.md`

---

## Phase 3: Multi-User Support 🔜 PLANNED

**Goal**: Full user management and access control

### 3.1 User Management

**Effort**: 7-10 days | **Priority**: HIGH

**What**: User accounts with passwords

**Tasks**:
- [ ] User model and storage
- [ ] User CRUD API
- [ ] Password hashing (bcrypt)
- [ ] Session management
- [ ] Login/logout endpoints
- [ ] User management UI
- [ ] User CLI commands
- [ ] Make API keys REQUIRED (breaking change)
- [ ] Web UI login page

**Dependencies**: Audit trail (should be in place first)

**Files to Create**:
- `internal/model/user.go`
- `internal/storage/user_sqlite.go`
- `internal/auth/session.go`
- `internal/auth/password.go`
- `internal/api/auth_handlers.go`
- `internal/api/user_handlers.go`
- `webui/src/components/users.ts`
- `webui/src/components/login.ts`
- `cmd/user/user.go`

### 3.2 Role-Based Access Control (RBAC)

**Effort**: 5-7 days | **Priority**: HIGH

**What**: Permissions and roles

**Tasks**:
- [ ] Role and permission models
- [ ] RBAC storage
- [ ] RBAC middleware
- [ ] Default roles (admin, operator, viewer)
- [ ] Permission checks on all endpoints
- [ ] Role management UI
- [ ] RBAC CLI commands

**Dependencies**: User Management

**Default Roles**:
- `admin` - Full access
- `operator` - Read/write devices, networks, discovery
- `viewer` - Read-only access

**Files to Create**:
- `internal/model/role.go`
- `internal/storage/rbac_sqlite.go`
- `internal/auth/rbac.go`
- `internal/api/rbac_middleware.go`
- `internal/api/role_handlers.go`
- `webui/src/components/roles.ts`
- `cmd/role/role.go`

### 3.3 SSO/OIDC Integration

**Effort**: 5-7 days | **Priority**: MEDIUM

**What**: Enterprise authentication

**Tasks**:
- [ ] OIDC client implementation
- [ ] OAuth2 flow
- [ ] SSO configuration
- [ ] SSO login UI
- [ ] Support multiple providers (Google, Okta, Azure AD)
- [ ] User provisioning from SSO
- [ ] SSO CLI configuration

**Dependencies**: User Management, RBAC

**Files to Create**:
- `internal/auth/oidc.go`
- `internal/auth/oauth2.go`
- `internal/api/sso_handlers.go`
- `webui/src/components/sso-login.ts`

### 3.4 PostgreSQL Storage Backend

**Effort**: 10-14 days | **Priority**: MEDIUM

**What**: Scale beyond SQLite

**Tasks**:
- [ ] PostgreSQL storage adapter
- [ ] Connection pooling
- [ ] Implement all storage interfaces
- [ ] PostgreSQL migrations
- [ ] Database selection config
- [ ] Migration tool (SQLite → PostgreSQL)
- [ ] Documentation

**Configuration**:
```bash
RACKD_DATABASE_TYPE=postgres  # or sqlite
RACKD_POSTGRES_URL=postgres://user:pass@host:5432/rackd
```

**Files to Create**:
- `internal/storage/postgres.go`
- `internal/storage/postgres_migrations.go`
- `internal/storage/postgres_*.go` (per entity)

### 3.5 Webhook System

**Effort**: 5-7 days | **Priority**: LOW (Optional)

**What**: Event notifications for automation

**Tasks**:
- [ ] Webhook model and storage
- [ ] Webhook CRUD API
- [ ] Event dispatcher
- [ ] Webhook delivery with retries
- [ ] HMAC signature verification
- [ ] Webhook UI page
- [ ] Webhook CLI commands
- [ ] Webhook MCP tools

**Events**:
- `device.created`, `device.updated`, `device.deleted`
- `network.created`, `network.updated`, `network.deleted`
- `discovery.scan.started`, `discovery.scan.completed`
- `device.promoted`

**Note**: Can be skipped if not doing heavy automation

**Files to Create**:
- `internal/model/webhook.go`
- `internal/storage/webhook_sqlite.go`
- `internal/webhook/dispatcher.go`
- `internal/webhook/delivery.go`
- `internal/api/webhook_handlers.go`
- `webui/src/components/webhooks.ts`
- `cmd/webhook/webhook.go`

---

## Phase 4: Advanced Features 🔮 FUTURE

**Goal**: Advanced IPAM and integration features

### 4.1 Network Topology Visualization

**Effort**: 7-10 days

**What**: Visual network topology based on relationships

**Tasks**:
- [ ] Topology data structure
- [ ] Topology API endpoint
- [ ] Graph layout algorithm
- [ ] Interactive visualization (Cytoscape.js or D3.js)
- [ ] Zoom, pan, filter
- [ ] Export (PNG, SVG, JSON)
- [ ] Real-time updates via WebSocket

**Note**: Basic relationship graph already exists at `/devices/graph`

### 4.2 DNS Integration

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

### 4.3 DHCP Integration

**Effort**: 5-7 days

**What**: DHCP server integration

**Tasks**:
- [ ] DHCP provider interface
- [ ] ISC DHCP support (via omapi)
- [ ] Kea support (via API)
- [ ] DHCP lease management
- [ ] Auto-reserve IPs in pools
- [ ] DHCP configuration UI
- [ ] DHCP CLI commands

### 4.4 Circuit Management

**Effort**: 4-5 days

**What**: Track network circuits

**Tasks**:
- [ ] Circuit model (provider, circuit ID, capacity, endpoints)
- [ ] Circuit storage
- [ ] Circuit CRUD API
- [ ] Circuit UI page
- [ ] Link circuits to devices
- [ ] Circuit status tracking
- [ ] Circuit CLI commands

### 4.5 NAT Tracking

**Effort**: 3-4 days

**What**: Track NAT mappings

**Tasks**:
- [ ] NAT mapping model (external IP/port, internal IP/port)
- [ ] NAT storage
- [ ] NAT CRUD API
- [ ] NAT UI page
- [ ] Link NAT to devices
- [ ] NAT validation
- [ ] NAT CLI commands

### 4.6 IP Conflict Detection

**Effort**: 2-3 days

**What**: Detect and warn about IP conflicts

**Tasks**:
- [ ] Duplicate IP detection
- [ ] Overlapping subnet detection
- [ ] Conflict resolution UI
- [ ] Conflict API endpoints
- [ ] Automatic conflict checking on IP assignment

### 4.7 Custom Fields/Metadata

**Effort**: 4-5 days

**What**: User-defined fields for devices/networks

**Tasks**:
- [ ] Custom field model
- [ ] Custom field storage (JSON or key-value)
- [ ] Custom field CRUD API
- [ ] Custom field UI
- [ ] Custom field validation
- [ ] Search/filter by custom fields

---

## Phase 5: Scale & Performance 🔮 FUTURE

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

## Implementation Priorities

### Immediate (Next 2 weeks)

1. **Data Export/Import** (2-3 days) - Start here!
2. **Bulk Operations** (3-4 days)
3. **API Rate Limiting** (2 days)

### Short-term (Next 1-2 months)

4. **Change History/Audit Trail** (4-5 days)
5. **User Management** (7-10 days)
6. **RBAC** (5-7 days)

### Medium-term (3-6 months)

7. **SSO/OIDC** (5-7 days)
8. **PostgreSQL Backend** (10-14 days)
9. **Network Topology** (7-10 days)
10. **DNS Integration** (5-7 days)
11. **DHCP Integration** (5-7 days)

### Long-term (6+ months)

12. **Circuit Management** (4-5 days)
13. **NAT Tracking** (3-4 days)
14. **Performance Optimizations** (Ongoing)

---

## Success Criteria

### Phase 1 ✅
- ✅ Search returns results in <100ms
- ✅ Metrics endpoint responds in <50ms
- ✅ Health checks respond in <10ms
- ✅ API keys can be created and used

### Phase 2
- [ ] Can import 1000+ devices from CSV
- [ ] Can export all data to JSON
- [ ] Rate limiting prevents abuse
- [ ] All changes are audited

### Phase 3
- [ ] Users can log in with password
- [ ] RBAC enforces permissions
- [ ] SSO works with major providers
- [ ] PostgreSQL supports 10,000+ devices

### Phase 4
- [ ] Topology renders 500+ devices smoothly
- [ ] DNS/DHCP sync works reliably
- [ ] Circuit/NAT tracking accurate

### Phase 5
- [ ] API p95 latency <200ms
- [ ] Database queries <50ms
- [ ] Support 50,000+ devices (with PostgreSQL)
- [ ] Cache hit rate >80%

---

## Notes

- All features include tests, documentation, CLI, API, and UI (where applicable)
- All features include MCP tools (where applicable)
- PostgreSQL is optional (SQLite remains default)
- Webhook system is optional (can be skipped)
- Focus on production-ready features before advanced features

---

## Change Log

- **2026-02-03**: Merged FEATURE_STATUS.md, reorganized by priority, completed Phase 1
- **2026-02-03**: Added API Key Authentication (1.4)
- **2026-02-03**: Completed Full-Text Search (1.1), Metrics (1.2), Health Checks (1.3)
