# Device Lifecycle & Status Tracking

Rackd tracks device lifecycle states to help manage infrastructure from planning through decommissioning.

## Overview

Device lifecycle tracking provides:
- Clear visibility into device status
- Historical tracking of status changes
- Scheduled decommission planning
- Status-based filtering and reporting
- Dashboard summaries by status

## Device Status

| Status | Description |
|--------|-------------|
| `planned` | Device is planned but not yet deployed |
| `active` | Device is operational |
| `maintenance` | Device is under maintenance |
| `decommissioned` | Device has been retired |

## Status Model

The device model includes status-related fields:

| Field | Type | Description |
|-------|------|-------------|
| `status` | string | Current device status |
| `status_changed_at` | timestamp | When status was last changed |
| `status_changed_by` | string | ID of the user who changed the status |
| `decommission_date` | timestamp | Scheduled decommission date (RFC 3339) |

## API Endpoints

### List Devices by Status

```http
GET /api/v1/devices?status=active
```

Filter devices by status:
- `status=planned`
- `status=active`
- `status=maintenance`
- `status=decommissioned`

### Get Status Counts

```http
GET /api/v1/devices/status-counts
```

**Response:**
```json
{
  "planned": 5,
  "active": 150,
  "maintenance": 3,
  "decommissioned": 12
}
```

### Update Device Status

```http
PUT /api/v1/devices/{id}
```

```json
{
  "status": "maintenance"
}
```

Status changes are automatically tracked:
- `status_changed_at` updated to current time
- `status_changed_by` set to current user
- Change logged in audit trail

### Schedule Decommission

```http
PUT /api/v1/devices/{id}
```

```json
{
  "decommission_date": "2024-06-30T00:00:00Z"
}
```

The date must be an RFC 3339 timestamp; any other format is ignored.

## CLI Commands

### List by Status

```bash
# List active devices
rackd device list --status active

# List devices in maintenance
rackd device list --status maintenance

# List decommissioned devices
rackd device list --status decommissioned

# Combine with other filters
rackd device list --status active --pool <pool-id>
```

### Set Status

`rackd device update` has no status or decommission-date flag; change those through the API or the web UI. To create a device with a status, put it in a JSON file:

```bash
# device.json: {"name": "server-future-01", "status": "planned"}
rackd device add --input device.json
```

## Web UI

### Device List

- **Status Filter**: Dropdown to filter by status
- **Status Badge**: Color-coded status indicator per device
- **Link to a status**: `/devices?status=<status>` opens the list filtered by that status

Status badge colors:
- **Planned**: Blue
- **Active**: Green
- **Maintenance**: Yellow
- **Decommissioned**: Gray

Status changes are recorded in the audit log (see [Audit Trail](#audit-trail)).

### Device Form

- **Status Dropdown**: Select status on create/edit

The web UI doesn't show or set `decommission_date`; use the API.

### Dashboard

- **Status Summary**: Card showing counts by status
- **Clickable Cards**: Click to filter device list by status

## Audit Trail

All status changes are logged in the audit trail as a device update:

| Field | Value |
|-------|-------|
| `action` | `update` |
| `resource` | `device` |
| `resource_id` | Device ID |
| `changes` | JSON of the updated device, including its new `status` |
| `username` | Current user |
| `timestamp` | Change time |

## Use Cases

### Planned Deployment

1. Create device with `status: planned`
2. Add to rack and network planning
3. Update to `status: active` when deployed

```bash
# device.json: {"name": "server-future-01", "status": "planned"}
rackd device add --input device.json
```

### Maintenance Window

1. Set device to `status: maintenance` before work
2. Perform maintenance tasks
3. Return to `status: active` when complete

```bash
curl -H "Authorization: Bearer $RACKD_TOKEN" -X PUT http://localhost:8080/api/v1/devices/dev-123 \
  -H "Content-Type: application/json" -d '{"status": "maintenance"}'
# ... perform maintenance ...
curl -H "Authorization: Bearer $RACKD_TOKEN" -X PUT http://localhost:8080/api/v1/devices/dev-123 \
  -H "Content-Type: application/json" -d '{"status": "active"}'
```

### Planned Decommission

1. Set `decommission_date` for planning
2. Update to `status: decommissioned` when removed
3. Historical record preserved

```bash
# Schedule future decommission
curl -H "Authorization: Bearer $RACKD_TOKEN" -X PUT http://localhost:8080/api/v1/devices/dev-123 \
  -H "Content-Type: application/json" -d '{"decommission_date": "2024-06-30T00:00:00Z"}'

# Mark as decommissioned
curl -H "Authorization: Bearer $RACKD_TOKEN" -X PUT http://localhost:8080/api/v1/devices/dev-123 \
  -H "Content-Type: application/json" -d '{"status": "decommissioned"}'
```

### Hardware Refresh

Track devices being replaced:

1. Set the old device's status to `decommissioned` (API or web UI)
2. Create the replacement with `"status": "planned"` (`rackd device add --input`, API or web UI)
3. Set the replacement to `active` when it's deployed

## Reporting

### Status Distribution

Get overview of infrastructure state:

```bash
# Via API
curl -H "Authorization: Bearer $RACKD_TOKEN" /api/v1/devices/status-counts

# Via dashboard
# View "Device Status" widget
```

### Decommission Queue

Find devices scheduled for decommission:

```bash
# List all devices with decommission dates
rackd device list --output json | jq '.[] | select(.decommission_date != null)'
```

### Maintenance Tracking

Track devices currently in maintenance:

```bash
rackd device list --status maintenance
```

## Integration with Other Features

### Webhooks

Subscribe to device status changes:
- `device.updated` (includes status changes)

### Custom Fields

Combine with custom fields for additional tracking:
- `warranty_expiry` - Track warranty end dates
- `last_maintenance` - Track maintenance history

### Dashboard

The dashboard widget shows status distribution with clickable cards.

## Best Practices

1. **Use All States**: Don't skip planned/decommissioned states
2. **Document Changes**: Note the reason in the device description; status changes have no notes field
3. **Schedule Decommissions**: Set dates for planned removals
4. **Regular Reviews**: Audit decommissioned devices periodically
5. **Consistent Process**: Follow same workflow for all devices
