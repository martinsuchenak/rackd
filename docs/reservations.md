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
| `pool_id` | string | Pool the address is reserved in |
| `ip_address` | string | Reserved IP address |
| `hostname` | string | Hostname the address is for (optional) |
| `purpose` | string | Reason for the reservation (optional) |
| `notes` | string | Additional notes (optional) |
| `reserved_by` | string | User who made the reservation |
| `reserved_at` | timestamp | When it was made |
| `status` | string | Reservation status |
| `expires_at` | timestamp | Expiration (optional) |
| `created_at` | timestamp | Creation timestamp |
| `updated_at` | timestamp | Last update timestamp |

## Reservation Status

| Status | Description |
|--------|-------------|
| `active` | Reservation is currently active |
| `expired` | Reservation has passed its expiration |
| `claimed` | The address has been assigned to a device |
| `released` | Reservation was released |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/reservations` | List reservations; filter with `pool_id`, `status`, `reserved_by`, `ip_address` |
| GET | `/api/v1/pools/{id}/reservations` | List a pool's reservations |
| POST | `/api/v1/reservations` | Create a reservation |
| GET | `/api/v1/reservations/{id}` | Get a reservation |
| PUT | `/api/v1/reservations/{id}` | Update hostname, purpose, notes or expiration |
| POST | `/api/v1/reservations/{id}/release` | Release a reservation, freeing the address |
| DELETE | `/api/v1/reservations/{id}` | Delete a reservation |

### Create Reservation

```http
POST /api/v1/reservations
```

```json
{
  "pool_id": "pool-1",
  "ip_address": "192.168.1.50",
  "hostname": "db-01",
  "purpose": "New database server - ticket #12345",
  "expires_in_days": 30
}
```

If `ip_address` is omitted, the next free address in the pool is reserved. Give the expiration as `expires_in_days` (1 to 365) or as an `expires_at` timestamp, not both; without either, the reservation doesn't expire.

**Response:** `201 Created` with the reservation.

### Update Reservation

```http
PUT /api/v1/reservations/{id}
```

```json
{
  "purpose": "Updated purpose",
  "expires_at": "2026-12-31T00:00:00Z"
}
```

Fields left out are unchanged. The expiration takes the same two forms as on create.

## CLI Commands

```bash
# List reservations, optionally by pool, status or who reserved them
rackd reservation list --pool pool-1 --status active

# Reserve a specific address
rackd reservation create --pool pool-1 --ip 192.168.1.50 --hostname db-01 --purpose "Database server"

# Reserve the next free address for 30 days
rackd reservation create --pool pool-1 --purpose "Load balancer" --expires 30

# Show, update, release or delete a reservation by ID
rackd reservation get --id res-abc123
rackd reservation update --id res-abc123 --purpose "Updated purpose" --expires 14
rackd reservation release --id res-abc123
rackd reservation delete --id res-abc123
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
