# Audit Trail

Rackd includes comprehensive audit logging to track all changes for compliance, security, and troubleshooting.

## Overview

Rackd records every change in the audit log, with who made it, from which address, and what changed. Auditing is always on.

**Key Features:**
- Automatic capture of changes made through the API, the web UI, MCP and the CLI
- Sign-ins (including failed and refused ones) and sign-outs
- User and IP address tracking
- Filtering, and export to JSON/CSV
- Automatic retention management

## Configuration

```bash
# Days to keep audit entries; 0 keeps everything (default: 90)
AUDIT_RETENTION_DAYS=90
```

Entries older than `AUDIT_RETENTION_DAYS` are deleted when the server starts and once a day after that. `AUDIT_ENABLED` is no longer used: setting it only logs a warning.

## What Gets Audited

| Resource | Actions |
|----------|---------|
| Devices, networks, pools, datacenters, reservations, relationships, DNS, discovery, bulk operations | `create`, `update`, `delete` and the like |
| Circuits, NAT mappings, custom fields, scan profiles, scheduled scans | `create`, `update`, `delete` |
| Conflicts | `resolve`, `delete` |
| Users | `create`, `update`, `delete`, `password_change`, `password_reset`, `assign_role`, `revoke_role` |
| Roles | `create`, `update`, `delete`, `set_permissions`, `grant_permission`, `revoke_permission` |
| API keys, credentials, webhooks | `create`, `update`, `delete` |
| OAuth clients | `register`, `authorize` (a user approving a client), `delete` |
| Sessions | `login` (`success` or `failure`, with the reason), `logout`. When an account locks after too many failures, one entry says so; refusals while it is locked aren't audited, so they can't flood the log |

Reads are not audited.

### Captured Information

Each audit log entry includes:

| Field | Description |
|-------|-------------|
| `id` | Unique audit log ID |
| `timestamp` | When the operation occurred |
| `action` | Operation type, such as `create` or `password_reset` |
| `resource` | Resource type (device, user, role, etc.) |
| `resource_id` | ID of the affected resource |
| `user_id` | ID of the user who made the change (an API key acts as its owner) |
| `username` | Their username. A failed sign-in names the account only if it exists: what was typed for an unknown user isn't stored, since it is often a password entered in the wrong field |
| `ip_address` | Client IP address |
| `changes` | The record after the change, as JSON. For credentials, API keys, webhooks and OAuth clients only names, IDs and settings are kept: never secrets, keys, passwords or full webhook URLs |
| `status` | `success` or `failure` |
| `error` | Why it failed (for a refused sign-in: unknown user, wrong password, deactivated account, or too many failed sign-ins) |

## API Usage

### List Audit Logs

```bash
GET /api/v1/audit
```

**Query Parameters:**
- `resource` - Filter by resource type (device, network, etc.)
- `resource_id` - Filter by specific resource ID
- `user_id` - Filter by user/API key ID
- `action` - Filter by action (create, update, delete)
- `start_time` - Filter by start time (RFC3339)
- `end_time` - Filter by end time (RFC3339)
- `limit` - Maximum results (default: 100)
- `offset` - Pagination offset

**Examples:**

```bash
# List recent changes
curl http://localhost:8080/api/v1/audit?limit=20

# Filter by resource
curl "http://localhost:8080/api/v1/audit?resource=device&limit=50"

# Filter by specific device
curl "http://localhost:8080/api/v1/audit?resource=device&resource_id=dev-123"

# Filter by user
curl "http://localhost:8080/api/v1/audit?user_id=key-456"

# Filter by action
curl "http://localhost:8080/api/v1/audit?action=delete"

# Time range query
curl "http://localhost:8080/api/v1/audit?start_time=2026-02-01T00:00:00Z&end_time=2026-02-03T23:59:59Z"

# Pagination
curl "http://localhost:8080/api/v1/audit?limit=50&offset=100"
```

**Response:**

```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "timestamp": "2026-02-03T16:45:00Z",
    "action": "create",
    "resource": "device",
    "resource_id": "dev-123",
    "user_id": "key-456",
    "username": "admin-key",
    "ip_address": "192.168.1.100",
    "changes": "{\"name\":\"server-01\",\"ip\":\"10.0.1.50\"}",
    "status": "success",
    "error": ""
  }
]
```

### Get Specific Audit Log

```bash
GET /api/v1/audit/{id}
```

**Example:**

```bash
curl http://localhost:8080/api/v1/audit/550e8400-e29b-41d4-a716-446655440000
```

### Export Audit Logs

```bash
GET /api/v1/audit/export
```

**Query Parameters:**
- `format` - Export format: `json` or `csv` (default: json)
- `resource` - Filter by resource type
- `resource_id` - Filter by resource ID
- `user_id` - Filter by user ID
- `start_time` - Filter by start time
- `end_time` - Filter by end time

**Examples:**

```bash
# Export to JSON
curl "http://localhost:8080/api/v1/audit/export?format=json" > audit.json

# Export to CSV
curl "http://localhost:8080/api/v1/audit/export?format=csv" > audit.csv

# Export specific resource
curl "http://localhost:8080/api/v1/audit/export?resource=device&format=json" > device-audit.json

# Export time range
curl "http://localhost:8080/api/v1/audit/export?start_time=2026-02-01T00:00:00Z&format=csv" > february-audit.csv
```

## CLI Usage

### List Audit Logs

```bash
rackd audit list [flags]
```

**Flags:**
- `--resource` - Filter by resource type
- `--resource-id` - Filter by resource ID
- `--action` - Filter by action
- `--limit` - Limit results (default: 50)
- `--data-dir` - Data directory (default: ./data)

**Examples:**

```bash
# List recent changes
rackd audit list --limit 20

# Filter by resource
rackd audit list --resource device --limit 50

# Filter by action
rackd audit list --action delete

# Filter by specific resource
rackd audit list --resource device --resource-id dev-123
```

**Output:**

```
TIMESTAMP                 ACTION  RESOURCE   RESOURCE_ID  USER        IP              STATUS
2026-02-03T16:45:00Z      create  device     dev-123      admin-key   192.168.1.100   success
2026-02-03T16:44:30Z      update  device     dev-123      admin-key   192.168.1.100   success
2026-02-03T16:43:15Z      delete  network    net-456      ops-key     192.168.1.101   success
```

### Export Audit Logs

```bash
rackd audit export [flags]
```

**Flags:**
- `--format` - Export format: json or csv (default: json)
- `--output` - Output file (default: stdout)
- `--resource` - Filter by resource type
- `--resource-id` - Filter by resource ID
- `--data-dir` - Data directory (default: ./data)

**Examples:**

```bash
# Export to JSON file
rackd audit export --format json --output audit-backup.json

# Export to CSV
rackd audit export --format csv --output audit.csv

# Export specific resource
rackd audit export --resource device --output device-audit.json

# Export to stdout
rackd audit export --format json
```

## Use Cases

### Compliance Auditing

Track all changes for compliance requirements:

```bash
# Export last 90 days for audit
curl "http://localhost:8080/api/v1/audit/export?start_time=2025-11-04T00:00:00Z&format=csv" > compliance-audit.csv
```

### Security Investigation

Investigate suspicious activity:

```bash
# Check all delete operations
curl "http://localhost:8080/api/v1/audit?action=delete&limit=100"

# Check activity from specific IP
curl "http://localhost:8080/api/v1/audit" | jq '.[] | select(.ip_address=="192.168.1.100")'

# Check failed operations
curl "http://localhost:8080/api/v1/audit" | jq '.[] | select(.status=="failure")'
```

### Change Tracking

Track changes to specific resources:

```bash
# View device history
curl "http://localhost:8080/api/v1/audit?resource=device&resource_id=dev-123"

# View network changes
curl "http://localhost:8080/api/v1/audit?resource=network&limit=50"
```

### User Activity

Monitor user actions:

```bash
# View user activity
curl "http://localhost:8080/api/v1/audit?user_id=key-456"

# Export user activity report
rackd audit export --format csv --output user-activity.csv
```

## Retention Management

### Automatic Cleanup

Old audit logs are automatically deleted based on the retention policy:

```bash
# Set retention to 365 days
export AUDIT_RETENTION_DAYS=365
```

Cleanup runs when the server starts and once a day after that. Set `AUDIT_RETENTION_DAYS=0` to keep every entry.

### Manual Cleanup

You can manually clean up old logs via the database:

```sql
-- Delete logs older than 90 days
DELETE FROM audit_logs WHERE timestamp < datetime('now', '-90 days');
```

## Performance Considerations

### Write Performance

- Audit logs are written **asynchronously** (via goroutine)
- No blocking of API requests
- Failed audit writes are logged but don't fail requests
- Overhead: <1ms per request

### Query Performance

- Indexed on: timestamp, resource, user_id, action
- Typical query time: <10ms
- Pagination recommended for large result sets

### Storage

- Average size: ~500 bytes per log entry
- 1 million logs ≈ 500 MB
- Request body limited to 10KB per log

## Compliance Support

### SOC 2

- ✅ **Access Logging** - All API access tracked
- ✅ **Change Tracking** - All modifications logged
- ✅ **User Attribution** - User/API key recorded
- ✅ **Retention Policy** - Configurable retention

### ISO 27001

- ✅ **Audit Trail** - Complete change history
- ✅ **Security Event Logging** - Failed operations tracked
- ✅ **Accountability** - User and IP tracking

### GDPR

- ✅ **Data Access Logging** - Who accessed what
- ✅ **Modification Tracking** - Changes to personal data
- ✅ **Retention Controls** - Automatic cleanup

## Troubleshooting

### No audit logs appearing

Auditing is always on. Check that the change was made by an authenticated caller (entries record who acted), and that the entry isn't older than `AUDIT_RETENTION_DAYS`. The server logs "Audit log channel full, dropping log entry" if writes outpace the database.

### Audit logs growing too large

Reduce retention period:
```bash
export AUDIT_RETENTION_DAYS=30
```

Or manually clean up:
```bash
# Delete logs older than 30 days
sqlite3 data/rackd.db "DELETE FROM audit_logs WHERE timestamp < datetime('now', '-30 days')"
```

### Missing audit entries

Check the table under [What Gets Audited](#what-gets-audited); reads are not audited.

### Performance impact

If audit logging impacts performance:
- Ensure database is on fast storage (SSD)
- Consider reducing retention period
- Check for slow disk I/O

## Database Schema

```sql
CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    resource_id TEXT,
    user_id TEXT,
    username TEXT,
    ip_address TEXT,
    changes TEXT,
    status TEXT NOT NULL,
    error TEXT
);

CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource, resource_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
```

## Security Considerations

### Sensitive Data

Audit entries never hold passwords, secrets, API keys or full webhook URLs, but they do hold IP addresses, usernames and the records' other fields, so:
- Grant `audit:list` and `audit:read` only to roles that need them (by default, admin and operator)
- Set `AUDIT_RETENTION_DAYS` to what your policies require

### Access Control

Protect audit log endpoints:
```bash
# Enable API authentication
export API_AUTH_TOKEN="your-secure-token"

# Access audit logs with authentication
curl -H "Authorization: Bearer your-secure-token" http://localhost:8080/api/v1/audit
```

### Audit Log Integrity

To ensure audit logs cannot be tampered with:
- Use read-only database replicas for audit queries
- Export logs to immutable storage (S3, etc.)
- Consider write-once storage for compliance

## Best Practices

### For Administrators

1. **Enable in production** - Essential for compliance and troubleshooting
2. **Set appropriate retention** - Balance compliance needs with storage
3. **Regular exports** - Backup audit logs to external storage
4. **Monitor growth** - Track audit log size and adjust retention
5. **Protect access** - Require authentication for audit endpoints

### For Developers

1. **Review audit logs** - Check what's being captured
2. **Sanitize sensitive data** - Don't log passwords or tokens
3. **Use bulk operations** - Reduce audit log volume
4. **Test with auditing enabled** - Ensure no performance issues

### For Compliance

1. **Document retention policy** - Match regulatory requirements
2. **Regular reviews** - Audit the audit logs
3. **Export for archival** - Long-term storage outside Rackd
4. **Access controls** - Limit who can view/export logs

## See Also

- [Authentication](authentication.md) - API key management
- [Configuration](configuration.md) - Environment variables
- [Monitoring](monitoring.md) - Metrics and observability
- [Security](security.md) - Security best practices
