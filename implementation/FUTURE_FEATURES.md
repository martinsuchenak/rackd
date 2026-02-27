# Rackd Future Feature Ideas

**Last Updated**: 2026-02-27

Ideas and improvements that are not yet planned for implementation. These may be promoted to the main [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) when prioritized.

---

## Notifications & Alerting

**Effort**: 4-5 days | **Priority**: HIGH

**What**: Configurable notifications for infrastructure events via email, Slack, and Teams

Moved from IMPLEMENTATION_PLAN.md section 4.6. The webhook system (4.7) provides external event integration which satisfies immediate automation needs. In-app notifications can be added later.

**Tasks**:
- [ ] Notification channel model (email, Slack webhook, Teams webhook)
- [ ] Notification channel storage + service
- [ ] Channel CRUD API
- [ ] Internal event bus (already implemented in webhook system - reuse)
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

**Dependencies**: Webhook System (complete - provides event bus)

---

## SSO/OIDC Integration

**Effort**: 5-7 days

Enterprise authentication via OpenID Connect. Only needed when integrating into environments with mandatory SSO policies.

- OIDC client implementation (authorization code flow)
- SSO configuration (issuer URL, client ID/secret, scopes)
- SSO login UI (provider buttons on login page)
- Support multiple providers (Google, Okta, Azure AD)
- User auto-provisioning from SSO claims
- Role mapping from SSO groups/claims
- Configuration: `OIDC_ENABLED`, `OIDC_ISSUER_URL`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URL`

**Dependencies**: User Management, RBAC (both complete)

---

## PostgreSQL Storage Backend

**Effort**: 10-14 days

Alternative storage backend for horizontal scaling. SQLite with WAL mode comfortably handles 100K+ rows and is sufficient for most deployments. Only consider PostgreSQL if you need multiple server instances writing simultaneously.

- PostgreSQL storage adapter implementing all storage interfaces
- Connection pooling (pgxpool)
- PostgreSQL-specific migrations
- Database selection via config (`RACKD_DATABASE_TYPE=postgres`)
- Migration tool (SQLite to PostgreSQL)
- The existing storage interface pattern makes this possible without changing business logic

---

## Tagging System Enhancement

**Effort**: 2-3 days

Tags currently exist on devices as flat strings. A structured tagging system would improve organization.

- Key:value tag format (e.g., `env:prod`, `team:networking`, `location:rack-12`)
- Tag taxonomy management (define allowed keys and values)
- Tag-based views and filtered dashboards
- Tag policies (require certain tag keys on devices, e.g., every device must have `env`)
- Bulk tag editing UI improvements
- Tag autocomplete in UI

---

## Database Backup API

**Effort**: 1-2 days

Docs describe SQLite backup strategies but there's no programmatic backup support.

- `POST /api/backup` — trigger SQLite online backup (using `.backup` command)
- `GET /api/backup/download` — download latest backup file
- Scheduled automatic backups (configurable interval)
- Backup retention policy (keep last N backups)
- Backup status and history endpoint
- Configuration: `BACKUP_ENABLED`, `BACKUP_INTERVAL`, `BACKUP_RETENTION`, `BACKUP_DIR`

---

## OpenAPI Spec Completion + Swagger UI

**Effort**: 2-3 days

An OpenAPI spec exists at `api/openapi.yaml` but is incomplete — many endpoints added since initial creation are not documented.

- Audit and complete the OpenAPI spec for all current endpoints
- Serve Swagger UI at `/docs` (embed swagger-ui assets)
- Auto-validate request/response against spec in tests
- Generate API client libraries from spec (Go, Python, TypeScript)

---

## Change Diffing in Audit Trail

**Effort**: 2-3 days

The audit trail logs changes but doesn't show what specifically changed.

- Store before/after snapshots for update operations
- Field-level diff computation ("hostname changed from `sw-01` to `sw-core-01`")
- Diff display in audit log UI (highlighted additions/removals)
- API: include `changes` field in audit log responses with structured diffs

---

## Multi-Tenancy / Resource Scoping

**Effort**: 7-10 days

Group resources by tenant or project for teams managing infrastructure for multiple customers.

- Tenant model with name, description, contact info
- Scope devices, networks, pools, datacenters to a tenant
- Tenant-level RBAC (viewer in tenant A, operator in tenant B)
- Tenant switcher in UI
- Cross-tenant search for admins
- Tenant-scoped API keys
- Default tenant for backward compatibility

---

## Import from External Sources

**Effort**: 3-5 days per source

Beyond CSV/JSON, support importing from common infrastructure tools and cloud providers.

**Nmap XML Import**:
- Parse Nmap XML scan output
- Map hosts, ports, services to Rackd devices
- Merge with existing devices by IP

**NetBox Import**:
- Import devices, prefixes, IP addresses, sites from NetBox API
- Migration path for teams switching from NetBox
- Field mapping configuration

**Cloud Provider Sync**:
- AWS: Import VPCs, subnets, EC2 instances
- Azure: Import VNets, subnets, VMs
- GCP: Import VPC networks, subnetworks, instances
- Periodic sync with configurable interval
- Tag cloud resources with source metadata

---

## IPAM Reporting & Analytics

**Effort**: 3-4 days

Generate reports for capacity planning and compliance.

- Subnet utilization report (% used across all networks)
- IP allocation history (who allocated what, when)
- Growth projections (based on utilization trends)
- Export reports as PDF
- Scheduled report generation and email delivery
- Compliance reports (unused IPs, stale devices, orphaned records)

---

## Device Configuration Backup

**Effort**: 5-7 days

Fetch and store device configurations via SSH/SNMP for change tracking.

- SSH-based config fetch (show running-config, etc.)
- Configurable per-device commands
- Config version history with diffs
- Scheduled config collection
- Alert on config changes
- Config search across all devices

---

## REST API Versioning

**Effort**: 2-3 days

Prepare for API evolution without breaking existing clients.

- Version prefix (`/api/v1/`, `/api/v2/`)
- Deprecation headers for old versions
- Migration guide between versions
- Version negotiation via Accept header
