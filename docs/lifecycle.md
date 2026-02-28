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
| `status_changed_by` | string | User who changed the status |
| `decommission_date` | date | Scheduled decommission date |

## API Endpoints

### List Devices by Status

```http
GET /api/devices?status=active
```

Filter devices by status:
- `status=planned`
- `status=active`
- `status=maintenance`
- `status=decommissioned`

### Get Status Counts

```http
GET /api/devices/status-counts
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
PUT /api/devices/{id}
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
PUT /api/devices/{id}
```

```json
{
  "decommission_date": "2024-06-30"
}
```

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
rackd device list --status active --pool production
```

### Update Status

```bash
# Set device to maintenance
rackd device update --id dev-123 --status maintenance

# Decommission a device
rackd device update --id dev-123 --status decommissioned

# Schedule decommission
rackd device update --id dev-123 --decommission-date 2024-06-30
```

## Web UI

### Device List

- **Status Filter**: Dropdown to filter by status
- **Status Badge**: Color-coded status indicator per device
- **URL State**: Filter persisted in URL for sharing

Status badge colors:
- **Planned**: Blue
- **Active**: Green
- **Maintenance**: Orange
- **Decommissioned**: Gray

### Device Detail

- **Status Display**: Prominent status badge
- **Status History**: In audit trail
- **Decommission Date**: Displayed when set

### Device Form

- **Status Dropdown**: Select status on create/edit
- **Decommission Date**: Date picker for scheduling

### Dashboard

- **Status Summary**: Card showing counts by status
- **Clickable Cards**: Click to filter device list by status

## Audit Trail

All status changes are logged in the audit trail:

| Field | Value |
|-------|-------|
| Action | `device.updated` |
| Field | `status` |
| Old Value | `active` |
| New Value | `maintenance` |
| User | Current user |
| Timestamp | Change time |

## Use Cases

### Planned Deployment

1. Create device with `status: planned`
2. Add to rack and network planning
3. Update to `status: active` when deployed

```bash
rackd device create \
  --hostname "server-future-01" \
  --status planned \
  --pool production
```

### Maintenance Window

1. Set device to `status: maintenance` before work
2. Perform maintenance tasks
3. Return to `status: active` when complete

```bash
rackd device update --id dev-123 --status maintenance
# ... perform maintenance ...
rackd device update --id dev-123 --status active
```

### Planned Decommission

1. Set `decommission_date` for planning
2. Update to `status: decommissioned` when removed
3. Historical record preserved

```bash
# Schedule future decommission
rackd device update --id dev-123 --decommission-date 2024-06-30

# Mark as decommissioned
rackd device update --id dev-123 --status decommissioned
```

### Hardware Refresh

Track devices being replaced:

```bash
# Mark old device for decommission
rackd device update --id old-server --status decommissioned

# Create replacement as planned
rackd device create --hostname new-server --status planned

# Activate replacement
rackd device update --id new-server --status active
```

## Reporting

### Status Distribution

Get overview of infrastructure state:

```bash
# Via API
curl /api/devices/status-counts

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
2. **Document Changes**: Add notes when changing status
3. **Schedule Decommissions**: Set dates for planned removals
4. **Regular Reviews**: Audit decommissioned devices periodically
5. **Consistent Process**: Follow same workflow for all devices
