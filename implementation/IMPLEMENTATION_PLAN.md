# Rackd Implementation Plan

**Last Updated**: 2026-02-06

This document tracks all planned features for Rackd, organized by priority and implementation status.

## Quick Status

| Phase | Features | Completed | Status |
|-------|----------|-----------|--------|
| **Phase 1: Core** | 4 | 4/4 (100%) | ✅ Complete |
 | **Phase 2: Production Ready** | 5 | 5/5 (100%) | ✅ Complete |
| **Phase 3: Multi-User** | 5 | 1/5 (20%) | 🚧 In Progress |
| **Phase 4: Advanced** | 7 | 0/7 (0%) | 🔮 Future |
| **Phase 5: Scale** | 3 | 0/3 (0%) | 🔮 Future |
| **Total** | **24** | **9/24 (38%)** | |

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

### 2.2 Bulk Operations ✅ COMPLETED (2026-02-03)

**Effort**: 3-4 days | **Priority**: HIGH

**What**: Manage large numbers of devices/networks efficiently

**Completed**:
- ✅ Bulk device create/update/delete
- ✅ Bulk tag add/remove
- ✅ Bulk network create/delete
- ✅ Transaction support (all-or-nothing)
- ✅ Detailed result reporting (total, success, failed, errors)
- ✅ API endpoints (`/api/bulk/*`)
- ✅ Comprehensive tests (6 tests, all passing)

**Features**:
- All operations run in transactions for atomicity
- Detailed error reporting per item
- No deadlocks (direct SQL within transactions)
- Result includes: total, success count, failed count, error messages

**API Endpoints**:
```
POST   /api/devices/bulk          - Bulk create devices
PUT    /api/devices/bulk          - Bulk update devices
DELETE /api/devices/bulk          - Bulk delete devices (body: {"ids": [...]})
POST   /api/devices/bulk/tags     - Bulk add tags (body: {"device_ids": [...], "tags": [...]})
DELETE /api/devices/bulk/tags     - Bulk remove tags
POST   /api/networks/bulk         - Bulk create networks
DELETE /api/networks/bulk         - Bulk delete networks
```

**Files Created**:
- `internal/storage/bulk.go` - Bulk operations implementation
- `internal/storage/bulk_test.go` - Bulk operations tests (6 tests)

**Files Modified**:
- `internal/storage/storage.go` - Added BulkOperations interface
- `internal/storage/sqlite.go` - Refactored to support transaction-based operations
- `internal/api/handlers.go` - Registered bulk routes
- `internal/api/device_handlers.go` - Added bulk device handlers
- `internal/api/network_handlers.go` - Added bulk network handlers
- `cmd/import/import.go` - Updated to use bulk endpoints for better performance

### 2.3 API Rate Limiting ✅ COMPLETED (2026-02-03)

**Effort**: 2 days | **Priority**: MEDIUM

**What**: Prevent API abuse

**Completed**:
- ✅ Rate limiting middleware with token bucket algorithm
- ✅ Per-IP rate limits
- ✅ Per-API-key rate limits
- ✅ Rate limit headers (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset)
- ✅ Configuration via environment variables (RATE_LIMIT_ENABLED, RATE_LIMIT_REQUESTS, RATE_LIMIT_WINDOW)
- ✅ Localhost bypass (127.0.0.1, ::1 always allowed)
- ✅ Comprehensive tests (8 tests, all passing)
- ✅ Documentation (docs/ratelimit.md)

**Features**:
- Disabled by default (opt-in via RATE_LIMIT_ENABLED=true)
- Token bucket algorithm with configurable window
- Client identification by API key (preferred) or IP address
- X-Forwarded-For and X-Real-IP header support
- Automatic cleanup of inactive clients
- HTTP 429 responses with Retry-After header

**Configuration**:
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m
```

**Files Created**:
- `internal/api/ratelimit.go` - Rate limiter implementation
- `internal/api/ratelimit_test.go` - Rate limiter tests (8 tests)
- `docs/ratelimit.md` - Comprehensive documentation

**Files Modified**:
- `internal/config/config.go` - Added rate limit configuration
- `internal/server/server.go` - Integrated rate limit middleware

### 2.4 Change History/Audit Trail ✅ COMPLETED (2026-02-03)

**Effort**: 4-5 days | **Priority**: HIGH

**What**: Track all changes for compliance and troubleshooting

**Completed**:
- ✅ Audit log model and storage
- ✅ Audit middleware (capture all changes)
- ✅ Log CRUD operations with context
- ✅ Log authentication events (via API key context)
- ✅ API endpoints for querying audit log
- ✅ Export audit log (JSON/CSV)
- ✅ Retention policy configuration
- ✅ CLI commands (list, export)
- ✅ Comprehensive tests (6 tests, all passing)

**Features**:
- Disabled by default (opt-in via AUDIT_ENABLED=true)
- Captures all mutating API operations (POST, PUT, DELETE)
- Tracks user (API key), IP address, resource, action, status
- Stores request body as changes (up to 10KB)
- Automatic cleanup of old logs (configurable retention)
- Pagination support for large result sets
- Time-based filtering (start_time, end_time)
- Resource and action filtering

**Configuration**:
```bash
AUDIT_ENABLED=true
AUDIT_RETENTION_DAYS=90
```

**API Endpoints**:
```
GET /api/audit                - List audit logs (with filters)
GET /api/audit/{id}           - Get specific audit log
GET /api/audit/export         - Export audit logs (JSON/CSV)
```

**CLI Commands**:
```bash
rackd audit list --resource device --limit 50
rackd audit export --format json --output audit.json
```

**Files Created**:
- `internal/model/audit.go` - Audit log model
- `internal/storage/audit_sqlite.go` - SQLite implementation
- `internal/storage/audit_test.go` - Audit tests (6 tests)
- `internal/api/audit_middleware.go` - Audit middleware
- `internal/api/audit_handlers.go` - API handlers
- `cmd/audit/audit.go` - CLI commands

**Files Modified**:
- `internal/storage/storage.go` - Added AuditStorage interface
- `internal/storage/migrations.go` - Added audit_logs table migration
- `internal/api/handlers.go` - Registered audit routes
- `internal/server/server.go` - Integrated audit middleware
- `internal/config/config.go` - Added audit configuration
- `main.go` - Registered audit CLI command

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

### 2.5 UI/UX Enhancements ✅ COMPLETED (2026-02-03)

**Effort**: 3-5 days | **Priority**: MEDIUM

**What**: Improve data visibility and navigation in web UI

**Completed**:
- ✅ Device list: Added network and pool columns
- ✅ Device list: Added filters for network and pool
- ✅ Network/pool columns show clickable links to detail pages
- ✅ Client-side filtering for network and pool
- ✅ Network table: Added "Devices" button to view devices in network
- ✅ Network detail: Added linked devices section (shows first 5, link to view all)
- ✅ Pool detail: Added linked devices section (shows first 5, link to view all)
- ✅ Pool heatmap: Made used IPs clickable to navigate to device

**All Features Complete!**

**Files Modified**:
- `webui/src/components/devices.ts` - Added network/pool filters and helper methods
- `webui/src/partials/pages/devices.html` - Added network/pool columns and filter dropdowns
- `webui/src/components/networks.ts` - Added loadNetworkDevices method
- `webui/src/partials/pages/networks.html` - Added "Devices" button in actions
- `webui/src/partials/pages/network-detail.html` - Added devices section
- `webui/src/components/pools.ts` - Added loadPoolDevices method
- `webui/src/partials/pages/pool-detail.html` - Added devices section, made heatmap clickable

---

## Phase 3: Multi-User Support 🔜 PLANNED

**Goal**: Full user management and access control

### 3.1 User Management ✅ COMPLETED (2026-02-06)

**Effort**: 7-10 days | **Priority**: HIGH

**What**: User accounts with passwords

**Completed**:
- ✅ User model and storage
- ✅ User CRUD API
- ✅ Password hashing (bcrypt)
- ✅ Session management
- ✅ Login/logout endpoints
- ✅ User CLI commands
- ✅ Environment variable bootstrapping for initial admin
- ✅ Comprehensive tests
- ⏳ User management UI (pending)
- ⏳ Web UI login page (pending)
- ⏳ Make API keys REQUIRED (breaking change - requires careful deployment)

**Dependencies**: Audit trail (should be in place first)

**Files Created**:
- ✅ `internal/model/user.go`
- ✅ `internal/storage/user_sqlite.go`
- ✅ `internal/storage/user_test.go` (12 tests, all passing)
- ✅ `internal/storage/bootstrap.go` - Initial admin bootstrapping
- ✅ `internal/storage/bootstrap_test.go` (7 tests, all passing)
- ✅ `internal/auth/session.go`
- ✅ `internal/auth/session_test.go` (8 tests, all passing)
- ✅ `internal/auth/password.go`
- ✅ `internal/auth/password_test.go` (3 tests, all passing)
- ✅ `internal/api/auth_handlers.go`
- ✅ `internal/api/user_handlers.go`
- ⏳ `webui/src/components/users.ts` (pending)
- ⏳ `webui/src/components/login.ts` (pending)
- ✅ `cmd/user/user.go`
- ✅ `docs/user-authentication.md` - Full documentation

**API Endpoints**:
```
POST   /api/auth/login           - User login
POST   /api/auth/logout          - User logout
GET    /api/auth/me             - Get current user
GET    /api/users               - List users
POST   /api/users               - Create user
GET    /api/users/{id}          - Get user
PUT    /api/users/{id}          - Update user
DELETE /api/users/{id}          - Delete user
POST   /api/users/{id}/password - Change password
```

**CLI Commands**:
```bash
rackd user list              - List users
rackd user create            - Create user
rackd user update            - Update user
rackd user delete            - Delete user
rackd user password          - Change password
```

**Environment Variables**:
```bash
INITIAL_ADMIN_USERNAME=admin       # Required for initial admin
INITIAL_ADMIN_PASSWORD=pass123    # Required for initial admin (min 8 chars)
INITIAL_ADMIN_EMAIL=admin@local   # Optional (default: admin@localhost)
INITIAL_ADMIN_FULL_NAME="Admin"   # Optional (default: System Administrator)
SESSION_TTL=24h                    # Optional (default: 24h)
```

**Bootstrapping Flow**:
1. Server checks if any users exist in database
2. If no users exist:
   - Checks for `INITIAL_ADMIN_USERNAME` and `INITIAL_ADMIN_PASSWORD` env vars
   - If both are set, creates initial admin user
   - If not set, logs warning with instructions
   - Server continues running (admin can be created via CLI)
3. If users already exist, skips bootstrapping

**Features**:
- bcrypt password hashing with cost factor 12
- Session tokens with configurable TTL
- Automatic session expiration and cleanup
- Password change with old password verification
- User filtering (username, email, active status, admin status)
- User activation/deactivation
- Admin status management
- Cannot delete own account
- Password changes invalidate all sessions
- Industry-standard environment variable bootstrapping for initial admin
- Graceful handling when no initial admin is configured

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
- ✅ Can import 1000+ devices from CSV
- ✅ Can export all data to JSON
- ✅ Rate limiting prevents abuse
- ✅ All changes are audited

### Phase 3
- ✅ Users can log in with password
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

- **2026-02-06**: Added initial admin bootstrapping via environment variables (industry standard for deployments)
- **2026-02-06**: Completed Phase 2 (Production Ready)
- **2026-02-06**: Completed User Management (3.1) - backend implementation with full API, CLI, and tests
- **2026-02-03**: Merged FEATURE_STATUS.md, reorganized by priority, completed Phase 1
- **2026-02-03**: Added API Key Authentication (1.4)
- **2026-02-03**: Completed Full-Text Search (1.1), Metrics (1.2), Health Checks (1.3)
