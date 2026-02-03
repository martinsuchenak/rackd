# Audit Trail

Rackd includes comprehensive audit logging to track all changes for compliance, security, and troubleshooting.

## Overview

The audit trail captures all mutating API operations with full context including user, IP address, resource details, and changes. This provides accountability, compliance support, and troubleshooting capabilities.

**Key Features:**
- Automatic capture of all API changes
- User and IP address tracking
- Request body capture for change details
- Flexible filtering and querying
- Export to JSON/CSV
- Automatic retention management
- Disabled by default (opt-in)

## Configuration

Audit logging is configured via environment variables:

```bash
# Enable audit logging (default: false)
AUDIT_ENABLED=true

# Retention period in days (default: 90)
AUDIT_RETENTION_DAYS=90
```

### Example Configurations

**Production with compliance:**
```bash
export AUDIT_ENABLED=true
export AUDIT_RETENTION_DAYS=365
rackd server
```

**Development (disabled):**
```bash
# Default - no audit logging
rackd server
```

## What Gets Audited

### Captured Operations

- ✅ **POST** - Create operations
- ✅ **PUT/PATCH** - Update operations
- ✅ **DELETE** - Delete operations
- ❌ **GET/HEAD** - Read operations (not audited)

### Captured Information

Each audit log entry includes:

| Field | Description |
|-------|-------------|
| `id` | Unique audit log ID |
| `timestamp` | When the operation occurred |
| `action` | Operation type (create, update, delete) |
| `resource` | Resource type (device, network, datacenter, etc.) |
| `resource_id` | ID of the affected resource |
| `user_id` | API key ID (if authenticated) |
| `username` | API key name (if authenticated) |
| `ip_address` | Client IP address |
| `changes` | Request body (up to 10KB) |
| `status` | success or failure |
| `error` | Error message (if failed) |

### Excluded Endpoints

These endpoints are not audited:
- Health checks (`/healthz`, `/readyz`)
- Metrics (`/metrics`)
- MCP server (`/mcp`)
- Static files (`/static/`)
- Search (`/api/search`)

## API Usage

### List Audit Logs

```bash
GET /api/audit
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
curl http://localhost:8080/api/audit?limit=20

# Filter by resource
curl "http://localhost:8080/api/audit?resource=device&limit=50"

# Filter by specific device
curl "http://localhost:8080/api/audit?resource=device&resource_id=dev-123"

# Filter by user
curl "http://localhost:8080/api/audit?user_id=key-456"

# Filter by action
curl "http://localhost:8080/api/audit?action=delete"

# Time range query
curl "http://localhost:8080/api/audit?start_time=2026-02-01T00:00:00Z&end_time=2026-02-03T23:59:59Z"

# Pagination
curl "http://localhost:8080/api/audit?limit=50&offset=100"
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
GET /api/audit/{id}
```

**Example:**

```bash
curl http://localhost:8080/api/audit/550e8400-e29b-41d4-a716-446655440000
```

### Export Audit Logs

```bash
GET /api/audit/export
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
curl "http://localhost:8080/api/audit/export?format=json" > audit.json

# Export to CSV
curl "http://localhost:8080/api/audit/export?format=csv" > audit.csv

# Export specific resource
curl "http://localhost:8080/api/audit/export?resource=device&format=json" > device-audit.json

# Export time range
curl "http://localhost:8080/api/audit/export?start_time=2026-02-01T00:00:00Z&format=csv" > february-audit.csv
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
curl "http://localhost:8080/api/audit/export?start_time=2025-11-04T00:00:00Z&format=csv" > compliance-audit.csv
```

### Security Investigation

Investigate suspicious activity:

```bash
# Check all delete operations
curl "http://localhost:8080/api/audit?action=delete&limit=100"

# Check activity from specific IP
curl "http://localhost:8080/api/audit" | jq '.[] | select(.ip_address=="192.168.1.100")'

# Check failed operations
curl "http://localhost:8080/api/audit" | jq '.[] | select(.status=="failure")'
```

### Change Tracking

Track changes to specific resources:

```bash
# View device history
curl "http://localhost:8080/api/audit?resource=device&resource_id=dev-123"

# View network changes
curl "http://localhost:8080/api/audit?resource=network&limit=50"
```

### User Activity

Monitor user actions:

```bash
# View user activity
curl "http://localhost:8080/api/audit?user_id=key-456"

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

Cleanup runs automatically when the server starts and periodically during operation.

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

Check configuration:
```bash
# Verify audit logging is enabled
curl http://localhost:8080/api/config | jq .audit

# Check server logs
rackd server
# Should see: "Audit logging enabled retention_days=90"
```

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

Verify the operation is being audited:
- Only POST, PUT, DELETE are audited
- GET operations are not audited
- Health checks and metrics are excluded

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

Audit logs may contain sensitive information:
- Request bodies (including passwords, tokens)
- IP addresses
- User identifiers

**Recommendations:**
- Restrict access to audit logs
- Use authentication for audit endpoints
- Consider encrypting audit log storage
- Sanitize sensitive data before logging

### Access Control

Protect audit log endpoints:
```bash
# Enable API authentication
export API_AUTH_TOKEN="your-secure-token"

# Access audit logs with authentication
curl -H "Authorization: Bearer your-secure-token" http://localhost:8080/api/audit
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
