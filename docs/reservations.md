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
| GET | `/api/v1/reservations` | List reservations; filter with `pool_id`, `status`, `reserved_by`, `ip` |
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

Fields left out are unchanged. The expiration takes the same two forms as on create; send `"clear_expiry": true` to remove it.

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
rackd reservation update --id res-abc123 --expires 0   # never expires
rackd reservation release --id res-abc123
rackd reservation delete --id res-abc123
```

## Web UI

Reservations are managed within pool details at `/pools/detail?id=<pool-id>`.

### Pool Heatmap

The pool heatmap colours each address:
- **Green**: Available (click to reserve it)
- **Red**: Used by a device (click to open the device)
- **Yellow**: Reserved

### Reservation List

View a pool's reservations with:
- IP address
- Hostname
- Purpose
- Reserved by
- Expiration date
- Actions (delete)

## Automatic Expiration

A reservation with an expiration stops holding its address once that time
passes:

- Reserving the address again, or asking for the next free address in the
  pool, treats it as free straight away.
- Within a minute, its status changes to `expired`, and the pool views and
  dashboards count the address as available.
- The reservation record stays, and the change is in the audit log.

Reservations without an expiration never expire. There is nothing to
configure.

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
- **operator**: `reservation:list`, `reservation:read`, `reservation:create`, `reservation:update`
- **viewer**: `reservation:list`, `reservation:read`

## Use Cases

### Infrastructure Planning

Reserve IPs during capacity planning:

```bash
rackd reservation create \
  --pool production \
  --ip 10.0.1.100 \
  --purpose "Q3 expansion - web tier"
```

### Change Management

Reserve IPs for upcoming changes:

```bash
rackd reservation create \
  --pool dmz \
  --ip 192.168.100.50 \
  --purpose "New firewall - CHG-12345" \
  --expires 30
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
  --expires 7
```

## Integration with Other Features

### Devices

An address is in a pool when its IP is in the pool's range and it is bound to
the pool, or bound to no pool and on the pool's network or on no network. The
web UI's device form creates addresses with neither, and they count too.

When a device is created or updated (including in bulk, and by promoting a
discovered device) with an address in a pool, the active reservation for that
address becomes `claimed`. The address is no longer just set aside; it is in
use. The claim is in the audit log. An address on no network claims only when
a single pool has the IP reserved: with several, they're on different
networks and the address could belong to any of them.

If the address is already assigned to another device, a duplicate-IP conflict
is created.

Reserving an address a device already holds is refused. The next free address
in a pool, and the pool heatmap, count both assigned and reserved addresses.

### Webhooks

There are no reservation webhook events yet.

## Best Practices

1. **Always Add Purpose**: Document why an IP is reserved
2. **Set Expiration**: Use expiration dates for temporary reservations
3. **Include Ticket Numbers**: Reference change tickets in purpose field
4. **Regular Cleanup**: Review and release unused reservations
5. **Naming Convention**: Use consistent purpose formatting
6. **Owner Tracking**: `reserved_by` records who made each reservation
