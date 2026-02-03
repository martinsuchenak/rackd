# Implementation Plan - Missing Features

Based on review of specs vs current implementation, here are the features that need to be implemented.

## Status Overview

### ✅ Fully Implemented (Core OSS)

- Device CRUD with addresses, tags, domains
- Datacenter CRUD
- Network/IPAM with CIDR, VLANs
- Network Pools with IP allocation
- Device Relationships (contains, connected_to, depends_on)
- CLI Tool (all commands)
- Web UI (all core pages)
- MCP Server (all tools)
- Basic Discovery (ping scans)
- Advanced Discovery (SSH, SNMP)
- Scheduled Scans with profiles
- Credentials storage with encryption
- SQLite storage with migrations
- REST API (all endpoints)

### 🚧 Partially Implemented

- Search (basic filtering exists, full-text search missing)
- Monitoring (logging exists, metrics endpoint missing)
- Health checks (basic, needs enhancement)

### ❌ Not Implemented (Planned Features)

## Phase 1: Core Enhancements (High Priority)

### 1.1 Full-Text Search

**Priority**: High  
**Effort**: 2-3 days  
**Dependencies**: None

**Tasks**:

- [ ] Add FTS5 virtual table to SQLite schema
- [ ] Create search index for devices (name, description, tags, domains)
- [ ] Create search index for networks (name, description)
- [ ] Implement search API endpoint with ranking
- [ ] Add search to CLI
- [ ] Update Web UI search component

**Files to modify**:

- `internal/storage/migrations.go` - Add FTS5 migration
- `internal/storage/sqlite.go` - Implement search queries
- `internal/api/device_handlers.go` - Enhance search endpoint
- `webui/src/components/search.ts` - Update UI

### 1.2 Metrics and Monitoring

**Priority**: High  
**Effort**: 2-3 days  
**Dependencies**: None

**Tasks**:

- [ ] Add Prometheus metrics library
- [ ] Implement `/metrics` endpoint
- [ ] Add HTTP request metrics (duration, count, status)
- [ ] Add application metrics (device count, network count, etc.)
- [ ] Add discovery metrics (scan duration, hosts found)
- [ ] Add database metrics (query duration, connection pool)
- [ ] Update documentation

**Files to create/modify**:

- `internal/metrics/metrics.go` - Metrics collection
- `internal/api/handlers.go` - Add metrics endpoint
- `internal/api/middleware.go` - Add metrics middleware
- `docs/monitoring.md` - Update with metrics info

### 1.3 Enhanced Health Checks

**Priority**: Medium  
**Effort**: 1 day  
**Dependencies**: None

**Tasks**:

- [ ] Implement `/healthz` (liveness probe)
- [ ] Implement `/readyz` (readiness probe)
- [ ] Check database connectivity
- [ ] Check discovery scheduler status
- [ ] Return detailed health status JSON

**Files to create/modify**:

- `internal/api/health_handlers.go` - Health check handlers
- `internal/api/handlers.go` - Register health routes

## Phase 2: Integration Features (Medium Priority)

### 2.1 Webhook System

**Priority**: Medium  
**Effort**: 5-7 days  
**Dependencies**: None

**Tasks**:

- [ ] Create webhook model and storage
- [ ] Implement webhook CRUD API
- [ ] Create event dispatcher
- [ ] Implement webhook delivery with retries
- [ ] Add HMAC signature verification
- [ ] Create webhook UI page
- [ ] Add webhook CLI commands
- [ ] Add webhook MCP tools

**Files to create**:

- `internal/model/webhook.go`
- `internal/storage/webhook_sqlite.go`
- `internal/webhook/dispatcher.go`
- `internal/webhook/delivery.go`
- `internal/api/webhook_handlers.go`
- `webui/src/components/webhooks.ts`
- `cmd/webhook/webhook.go`

**Events to support**:

- `device.created`, `device.updated`, `device.deleted`
- `network.created`, `network.updated`, `network.deleted`
- `discovery.scan.started`, `discovery.scan.completed`
- `device.promoted`

### 2.2 Bulk Operations

**Priority**: Medium  
**Effort**: 3-4 days  
**Dependencies**: None

**Tasks**:

- [ ] Implement bulk device import (CSV/JSON)
- [ ] Implement bulk device update
- [ ] Implement bulk device delete
- [ ] Implement bulk tag operations
- [ ] Add bulk operations to API
- [ ] Add bulk operations to CLI
- [ ] Add bulk import to Web UI

**Files to create/modify**:

- `internal/api/bulk_handlers.go`
- `internal/storage/bulk_operations.go`
- `cmd/device/bulk.go`
- `webui/src/components/devices.ts` - Add import UI

### 2.3 API Rate Limiting

**Priority**: Medium  
**Effort**: 2 days  
**Dependencies**: None

**Tasks**:

- [ ] Implement rate limiting middleware
- [ ] Add per-IP rate limits
- [ ] Add per-token rate limits
- [ ] Add rate limit headers (X-RateLimit-*)
- [ ] Add rate limit configuration
- [ ] Update API documentation

**Files to create/modify**:

- `internal/api/ratelimit.go`
- `internal/api/middleware.go`
- `internal/config/config.go`

## Phase 3: User Management & Security (High Priority)

### 3.1 User Management & Authentication

**Priority**: High  
**Effort**: 7-10 days  
**Dependencies**: None

**Tasks**:

- [ ] Create user model and storage
- [ ] Implement user CRUD API
- [ ] Add password hashing (bcrypt)
- [ ] Implement session management
- [ ] Add login/logout endpoints
- [ ] Create user management UI
- [ ] Add user CLI commands
- [ ] Update API to support user context

**Files to create**:

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

**Priority**: High  
**Effort**: 5-7 days  
**Dependencies**: User Management (3.1)

**Tasks**:

- [ ] Create role and permission models
- [ ] Implement RBAC storage
- [ ] Add RBAC middleware
- [ ] Define default roles (admin, operator, viewer)
- [ ] Add permission checks to all endpoints
- [ ] Create role management UI
- [ ] Add RBAC CLI commands

**Files to create**:

- `internal/model/role.go`
- `internal/storage/rbac_sqlite.go`
- `internal/auth/rbac.go`
- `internal/api/rbac_middleware.go`
- `internal/api/role_handlers.go`
- `webui/src/components/roles.ts`
- `cmd/role/role.go`

**Default Roles**:

- `admin` - Full access
- `operator` - Read/write devices, networks, discovery
- `viewer` - Read-only access

### 3.3 Audit Logging

**Priority**: High  
**Effort**: 4-5 days  
**Dependencies**: User Management (3.1)

**Tasks**:

- [ ] Create audit log model and storage
- [ ] Implement audit middleware
- [ ] Log all CRUD operations with user context
- [ ] Log authentication events
- [ ] Add audit log API endpoints
- [ ] Add audit log UI page
- [ ] Add audit log export (CSV, JSON)
- [ ] Add audit log retention policy

**Files to create**:

- `internal/model/audit.go`
- `internal/storage/audit_sqlite.go`
- `internal/api/audit_middleware.go`
- `internal/api/audit_handlers.go`
- `webui/src/components/audit.ts`

**Audit Events**:

- User login/logout
- All CRUD operations (create, update, delete)
- Permission changes
- Configuration changes

### 3.4 SSO/OIDC Integration

**Priority**: Medium  
**Effort**: 5-7 days  
**Dependencies**: User Management (3.1)

**Tasks**:

- [ ] Implement OIDC client
- [ ] Add SSO configuration
- [ ] Implement OAuth2 flow
- [ ] Add SSO login UI
- [ ] Support multiple providers (Google, Okta, Azure AD)
- [ ] Add user provisioning from SSO
- [ ] Add SSO CLI configuration

**Files to create**:

- `internal/auth/oidc.go`
- `internal/auth/oauth2.go`
- `internal/api/sso_handlers.go`
- `webui/src/components/sso-login.ts`

### 3.5 PostgreSQL Storage Backend

**Priority**: Medium  
**Effort**: 10-14 days  
**Dependencies**: None

**Tasks**:

- [ ] Implement PostgreSQL storage adapter
- [ ] Add connection pooling
- [ ] Implement all storage interfaces for PostgreSQL
- [ ] Add migration support for PostgreSQL
- [ ] Add database selection (SQLite vs PostgreSQL)
- [ ] Add PostgreSQL configuration
- [ ] Update documentation

**Files to create**:

- `internal/storage/postgres.go`
- `internal/storage/postgres_migrations.go`
- `internal/storage/postgres_device.go`
- `internal/storage/postgres_network.go`
- `internal/storage/postgres_discovery.go`

**Configuration**:

```bash
RACKD_DATABASE_TYPE=postgres  # or sqlite
RACKD_POSTGRES_URL=postgres://user:pass@host:5432/rackd
RACKD_POSTGRES_MAX_CONNS=50
RACKD_POSTGRES_MAX_IDLE=10
```

## Phase 4: Advanced Features (Medium Priority)

### 4.1 Network Topology Visualization

**Priority**: Medium  
**Effort**: 7-10 days  
**Dependencies**: Device relationships

**Tasks**:

- [ ] Design topology data structure
- [ ] Implement topology API endpoint
- [ ] Add graph layout algorithm
- [ ] Create topology visualization component (use cytoscape.js or d3.js)
- [ ] Add interactive features (zoom, pan, filter)
- [ ] Add topology export (PNG, SVG, JSON)
- [ ] Add real-time updates via WebSocket

**Files to create**:

- `internal/api/topology_handlers.go`
- `internal/topology/graph.go`
- `webui/src/components/topology.ts`
- Add graph library dependency

### 4.2 DNS Integration

**Priority**: Medium  
**Effort**: 5-7 days  
**Dependencies**: None

**Tasks**:

- [ ] Define DNS provider interface
- [ ] Implement DNS provider for BIND (via nsupdate)
- [ ] Implement DNS provider for PowerDNS (via API)
- [ ] Add DNS record CRUD operations
- [ ] Auto-create DNS records on device creation
- [ ] Add DNS sync functionality
- [ ] Add DNS configuration UI
- [ ] Add DNS CLI commands

**Files to create**:

- `internal/dns/interface.go`
- `internal/dns/bind.go`
- `internal/dns/powerdns.go`
- `internal/api/dns_handlers.go`
- `webui/src/components/dns.ts`
- `cmd/dns/dns.go`

### 4.3 DHCP Integration

**Priority**: Medium  
**Effort**: 5-7 days  
**Dependencies**: None

**Tasks**:

- [ ] Define DHCP provider interface
- [ ] Implement DHCP provider for ISC DHCP (via omapi)
- [ ] Implement DHCP provider for Kea (via API)
- [ ] Add DHCP lease management
- [ ] Auto-reserve IPs in pools
- [ ] Add DHCP configuration UI
- [ ] Add DHCP CLI commands

**Files to create**:

- `internal/dhcp/interface.go`
- `internal/dhcp/isc.go`
- `internal/dhcp/kea.go`
- `internal/api/dhcp_handlers.go`
- `webui/src/components/dhcp.ts`
- `cmd/dhcp/dhcp.go`

### 4.4 Circuit Management

**Priority**: Medium  
**Effort**: 4-5 days  
**Dependencies**: None

**Tasks**:

- [ ] Create circuit model (provider, circuit ID, capacity, endpoints)
- [ ] Implement circuit storage
- [ ] Add circuit CRUD API
- [ ] Add circuit UI page
- [ ] Link circuits to devices
- [ ] Add circuit status tracking
- [ ] Add circuit CLI commands

**Files to create**:

- `internal/model/circuit.go`
- `internal/storage/circuit_sqlite.go`
- `internal/api/circuit_handlers.go`
- `webui/src/components/circuits.ts`
- `cmd/circuit/circuit.go`

### 4.5 NAT Tracking

**Priority**: Medium  
**Effort**: 3-4 days  
**Dependencies**: None

**Tasks**:

- [ ] Create NAT mapping model (external IP/port, internal IP/port)
- [ ] Implement NAT storage
- [ ] Add NAT CRUD API
- [ ] Add NAT UI page
- [ ] Link NAT to devices
- [ ] Add NAT validation
- [ ] Add NAT CLI commands

**Files to create**:

- `internal/model/nat.go`
- `internal/storage/nat_sqlite.go`
- `internal/api/nat_handlers.go`
- `webui/src/components/nat.ts`
- `cmd/nat/nat.go`

### 4.6 Advanced Monitoring & Dashboards

**Priority**: Medium  
**Effort**: 7-10 days  
**Dependencies**: Metrics (1.2)

**Tasks**:

- [ ] Create dashboard data aggregation
- [ ] Add time-series metrics storage
- [ ] Implement dashboard API
- [ ] Create dashboard UI with charts
- [ ] Add customizable widgets
- [ ] Add alerting rules
- [ ] Add alert notifications (email, webhook)

**Files to create**:

- `internal/monitoring/dashboard.go`
- `internal/monitoring/alerts.go`
- `internal/api/dashboard_handlers.go`
- `webui/src/components/dashboard.ts`

## Phase 5: Performance & Scale (Ongoing)

### 5.1 Query Optimization

**Priority**: Medium  
**Effort**: Ongoing  
**Dependencies**: Metrics

**Tasks**:

- [ ] Add query performance logging
- [ ] Identify slow queries
- [ ] Add missing indexes
- [ ] Optimize N+1 queries
- [ ] Add query result caching

### 5.2 Pagination Enhancement

**Priority**: Medium  
**Effort**: 2-3 days  
**Dependencies**: None

**Tasks**:

- [ ] Implement cursor-based pagination
- [ ] Add pagination to all list endpoints
- [ ] Update CLI to support pagination
- [ ] Update UI to support pagination

### 5.3 Caching Layer

**Priority**: Low  
**Effort**: 3-4 days  
**Dependencies**: None

**Tasks**:
- [ ] Implement in-memory cache
- [ ] Cache frequently accessed data
- [ ] Add cache invalidation
- [ ] Add cache metrics

## Implementation Priority

### Immediate (Next Sprint)

1. Full-Text Search (1.1)
2. Metrics and Monitoring (1.2)
3. Enhanced Health Checks (1.3)

### Short-term (1-2 months)

4. Webhook System (2.1)
5. Bulk Operations (2.2)
6. API Rate Limiting (2.3)
7. User Management & Authentication (3.1)
8. RBAC (3.2)
9. Audit Logging (3.3)

### Medium-term (3-6 months)

10. SSO/OIDC Integration (3.4)
11. PostgreSQL Storage (3.5)
12. Network Topology (4.1)
13. DNS Integration (4.2)
14. DHCP Integration (4.3)
15. Circuit Management (4.4)
16. NAT Tracking (4.5)
17. Advanced Monitoring (4.6)

### Long-term (6+ months)

18. Query Optimization (5.1)
19. Enhanced Pagination (5.2)
20. Caching Layer (5.3)

## Effort Summary

| Phase | Features | Total Effort |
|-------|----------|--------------|
| Phase 1 (Core) | 3 features | 5-7 days |
| Phase 2 (Integration) | 3 features | 10-13 days |
| Phase 3 (Security & Users) | 5 features | 31-43 days |
| Phase 4 (Advanced) | 6 features | 35-48 days |
| Phase 5 (Performance) | 3 features | Ongoing |
| **Total** | **20 features** | **81-111 days** |

## Dependencies

```
Phase 1 (Core Enhancements)
  ├─ No dependencies
  └─ Enables: Better monitoring, search

Phase 2 (Integration)
  ├─ Depends on: Phase 1 (metrics for rate limiting)
  └─ Enables: External integrations, automation

Phase 3 (Security & Users)
  ├─ Depends on: Phase 1, Phase 2
  ├─ User Management → RBAC → Audit Logging
  ├─ User Management → SSO/OIDC
  └─ Enables: Multi-user support, compliance, security

Phase 4 (Advanced Features)
  ├─ Depends on: Phase 1, Phase 2, Phase 3
  ├─ Topology depends on: Device relationships (implemented)
  ├─ Monitoring depends on: Metrics (Phase 1)
  └─ Enables: Advanced IPAM, visualization, integrations

Phase 5 (Performance)
  ├─ Depends on: Phase 1 (metrics)
  ├─ PostgreSQL enables: Better scalability
  └─ Enables: Production-scale deployments
```

## Success Criteria

### Phase 1

- [ ] Search returns relevant results in <100ms
- [ ] Metrics endpoint returns data in <50ms
- [ ] Health checks respond in <10ms
- [ ] All tests passing

### Phase 2

- [ ] Webhooks deliver within 5 seconds
- [ ] Bulk import handles 1000+ devices
- [ ] Rate limiting prevents abuse
- [ ] All tests passing

### Phase 3 (Security & Users)

- [ ] User authentication works reliably
- [ ] RBAC enforces permissions correctly
- [ ] Audit log captures all changes
- [ ] SSO integration works with major providers
- [ ] PostgreSQL supports 10,000+ devices
- [ ] All tests passing

### Phase 4 (Advanced)

- [ ] Topology renders 500+ devices smoothly
- [ ] DNS/DHCP sync works reliably
- [ ] Circuit/NAT tracking accurate
- [ ] Monitoring dashboards responsive
- [ ] All tests passing

### Phase 5 (Performance)

- [ ] API p95 latency <200ms
- [ ] Database queries <50ms
- [ ] Support 50,000+ devices (with PostgreSQL)
- [ ] Cache hit rate >80%
- [ ] All tests passing

## Notes

- All features should include tests
- All features should include documentation
- All features should include CLI, API, and UI (where applicable)
- All features should include MCP tools (where applicable)
- PostgreSQL is optional (SQLite remains default)
