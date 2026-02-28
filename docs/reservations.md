# IP Address Reservations

Rackd provides IP reservation functionality for planning and managing IP address allocations before devices are provisioned.

## Overview

IP reservations allow you to:
- Reserve IPs for future use
- Plan IP allocations during infrastructure planning
- Prevent accidental IP conflicts
- Track reservation purposes and owners
- Auto-expire unused reservations

## Reservation Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `pool_id` | string | Parent IP pool |
| `ip_address` | string | Reserved IP address |
| `purpose` | string | Reason for reservation |
| `reserved_by` | string | User who made the reservation |
| `device_id` | string | Optional linked device |
| `status` | string | Reservation status |
| `expires_at` | timestamp | Expiration date (optional) |
| `created_at` | timestamp | Creation timestamp |
| `updated_at` | timestamp | Last update timestamp |

## Reservation Status

| Status | Description |
|--------|-------------|
| `active` | Reservation is currently active |
| `expired` | Reservation has passed expiration date |
| `claimed` | IP has been assigned to a device |
| `released` | Reservation was manually released |

## API Endpoints

### List Reservations

```http
GET /api/pools/{pool_id}/reservations
```

Query parameters:
- `status` - Filter by status (active/expired/claimed/released)
- `reserved_by` - Filter by user

**Response:**
```json
[
  {
    "id": "res-abc123",
    "pool_id": "pool-1",
    "ip_address": "192.168.1.50",
    "purpose": "New database server",
    "reserved_by": "admin",
    "status": "active",
    "expires_at": "2024-03-30T00:00:00Z",
    "created_at": "2024-02-27T10:00:00Z"
  }
]
```

### Get Reservation

```http
GET /api/pools/{pool_id}/reservations/{ip}
```

### Create Reservation

```http
POST /api/pools/{pool_id}/reservations
```

**Request body:**
```json
{
  "ip_address": "192.168.1.50",
  "purpose": "New database server - ticket #12345",
  "reserved_by": "john.doe",
  "expires_at": "2024-03-30T00:00:00Z"
}
```

If `ip_address` is omitted, the next available IP is automatically assigned.

### Update Reservation

```http
PUT /api/pools/{pool_id}/reservations/{ip}
```

### Release Reservation

```http
DELETE /api/pools/{pool_id}/reservations/{ip}
```

Marks the reservation as `released` and frees the IP.

### Claim Reservation

When creating a device with a reserved IP, the reservation is automatically claimed:

```http
POST /api/devices
```

```json
{
  "hostname": "db-server-01",
  "ip_address": "192.168.1.50"
}
```

## CLI Commands

### List Reservations

```bash
# List reservations for a pool
rackd reservation list --pool pool-1

# Filter by status
rackd reservation list --pool pool-1 --status active

# Output as JSON
rackd reservation list --pool pool-1 --output json
```

### Get Reservation

```bash
rackd reservation get --pool pool-1 --ip 192.168.1.50
```

### Create Reservation

```bash
# Reserve specific IP
rackd reservation create \
  --pool pool-1 \
  --ip 192.168.1.50 \
  --purpose "Database server - ticket #12345" \
  --reserved-by "john.doe"

# Reserve next available IP
rackd reservation create \
  --pool pool-1 \
  --purpose "Load balancer" \
  --expires "2024-03-30"
```

### Update Reservation

```bash
rackd reservation update \
  --pool pool-1 \
  --ip 192.168.1.50 \
  --purpose "Updated purpose"
```

### Release Reservation

```bash
rackd reservation release --pool pool-1 --ip 192.168.1.50
```

### Delete Reservation

```bash
rackd reservation delete --pool pool-1 --ip 192.168.1.50
```

## Web UI

Reservations are managed within pool details at `/pools/{id}`.

### Pool Heatmap

The pool heatmap displays reservations with distinct styling:
- **Blue**: Reserved IPs
- **Green**: Assigned IPs
- **Orange**: Conflicted IPs
- **Gray**: Available IPs

### Reservation List

View all reservations for a pool with:
- IP address
- Purpose
- Reserved by
- Status badge
- Expiration date
- Actions (edit/release)

## Automatic Expiration

Reservations can have an optional expiration date. Configure expiration handling:

```yaml
# config.yaml
reservation:
  default_expiration_days: 30
  cleanup_interval: 1h
```

Expired reservations:
- Status changes to `expired`
- IP becomes available for new reservations
- Original reservation record preserved for audit

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `reservation:list` | View reservations |
| `reservation:read` | View reservation details |
| `reservation:create` | Create new reservations |
| `reservation:update` | Modify reservations |
| `reservation:delete` | Release/delete reservations |

### Default Role Assignments

- **admin**: All reservation permissions
- **operator**: All reservation permissions
- **viewer**: `reservation:list`, `reservation:read`

## Use Cases

### Infrastructure Planning

Reserve IPs during capacity planning:

```bash
rackd reservation create \
  --pool production \
  --ip 10.0.1.100 \
  --purpose "Q3 expansion - web tier" \
  --reserved-by "infrastructure-team"
```

### Change Management

Reserve IPs for upcoming changes:

```bash
rackd reservation create \
  --pool dmz \
  --ip 192.168.100.50 \
  --purpose "New firewall - CHG-12345" \
  --expires "2024-03-15"
```

### Project Allocation

Reserve IP ranges for projects:

```bash
# Reserve multiple IPs for a project
for i in $(seq 50 60); do
  rackd reservation create \
    --pool development \
    --ip "10.10.0.$i" \
    --purpose "Project Phoenix - dev environment"
done
```

### Temporary Assignments

Reserve IPs for temporary workloads:

```bash
rackd reservation create \
  --pool testing \
  --purpose "Load testing - week of 3/15" \
  --expires "2024-03-22" \
  --reserved-by "qa-team"
```

## Integration with Other Features

### Conflict Detection

Reserved IPs that are assigned to devices trigger `reserved_ip_assigned` conflicts.

### Device Creation

When creating a device:
- If IP is reserved, reservation is automatically claimed
- If IP is already assigned to another device, conflict is created

### Webhooks

Subscribe to reservation events:
- `reservation.created`
- `reservation.claimed`
- `reservation.released`
- `reservation.expired`

## Best Practices

1. **Always Add Purpose**: Document why an IP is reserved
2. **Set Expiration**: Use expiration dates for temporary reservations
3. **Include Ticket Numbers**: Reference change tickets in purpose field
4. **Regular Cleanup**: Review and release unused reservations
5. **Naming Convention**: Use consistent purpose formatting
6. **Owner Tracking**: Set `reserved_by` for accountability
