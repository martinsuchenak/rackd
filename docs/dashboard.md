# Dashboard Reporting & Trends

Rackd provides a dashboard with aggregated statistics, network utilization, recent discoveries and stale-device alerts, plus an API for utilization trends.

## Overview

The dashboard provides at-a-glance visibility into:
- Inventory totals
- Devices by status
- Network utilization summary
- Recent device discoveries
- Stale device detection

## Dashboard API

### Get Dashboard Stats

```http
GET /api/v1/dashboard
```

Query parameters:
- `stale_days` - Days without being seen in discovery before an active device counts as stale (default: 7, 1-365)
- `recent_limit` - Number of recent discoveries to return (default: 10, 1-100)

**Response:**
```json
{
  "total_devices": 500,
  "total_networks": 25,
  "total_pools": 40,
  "total_datacenters": 3,
  "device_status_counts": {
    "planned": 10,
    "active": 475,
    "maintenance": 5,
    "decommissioned": 10
  },
  "discovered_devices": 14,
  "recent_discoveries": [
    {
      "id": "disc-123",
      "ip": "192.168.1.100",
      "hostname": "new-server-01",
      "vendor": "Dell",
      "network_id": "net-1",
      "first_seen": "2024-02-27T09:30:00Z",
      "last_seen": "2024-02-27T09:30:00Z"
    }
  ],
  "overall_utilization": 42.5,
  "network_utilization": [
    {
      "network_id": "net-1",
      "network_name": "Production LAN",
      "subnet": "192.168.1.0/24",
      "total_ips": 254,
      "used_ips": 192,
      "utilization": 75.5
    }
  ],
  "stale_devices": 12,
  "stale_threshold_days": 7,
  "stale_device_list": [
    {"id": "dev-1", "name": "old-server", "hostname": "old-server.example.com"}
  ]
}
```

### Get Utilization Trend

```http
GET /api/v1/dashboard/trend?type=pool&resource_id=<pool-id>&days=30
```

Query parameters:
- `type` - `network` or `pool` (default: `network`)
- `resource_id` - ID of the network or pool (required)
- `days` - Number of days (default: 30, 1-365)

**Response:**
```json
[
  {"timestamp": "2024-02-20T00:00:00Z", "utilization": 65.2, "used_ips": 166},
  {"timestamp": "2024-02-20T01:00:00Z", "utilization": 65.6, "used_ips": 167}
]
```

## Dashboard Components

### Summary Stats

Top-level statistics cards:
- **Devices**: All devices across all statuses (links to the device list)
- **Networks**: All configured networks (links to the network list)
- **Pools**: All IP pools
- **Datacenters**: All datacenters (links to the datacenter list)
- **Overall Utilization**: Across all networks

### Device Status Widget

Breakdown of devices by status:
- Planned (blue)
- Active (green)
- Maintenance (yellow)
- Decommissioned (gray)

Each status is clickable to filter the device list.

### Network Utilization List

Networks by utilization:
- Network name and subnet
- Utilization percentage
- Used and total IPs
- Click to open the network

### Recently Discovered

Feed of recently discovered hosts that haven't been promoted:
- IP address
- Hostname or vendor
- **Promote** link to the host on the Discovery page
- Total count of discovered hosts not yet in the inventory

### Health Alerts

Shows the count of stale devices: active devices not seen in discovery for `stale_threshold_days`. Expand it to list up to 20 of them, with a link to the full filtered device list when there are more.

## Configuration

### Snapshot Collection

Utilization snapshots of every network and pool are collected periodically for trend analysis. They are set with environment variables:

```bash
SNAPSHOT_INTERVAL=1h          # How often to collect snapshots
SNAPSHOT_RETENTION_DAYS=90    # How long to keep snapshots
```

When a pool's utilization reaches `POOL_UTILIZATION_ALERT_PERCENT` (default 90), the snapshot sends a `pool.utilization_high` webhook event, once each time it crosses the threshold.

### Stale Device Threshold

The stale threshold is the `stale_days` query parameter on `GET /api/v1/dashboard` (default 7). The web UI uses the default.

## Web UI

Access the dashboard at `/`.

### Features

- **Auto-refresh**: Refreshes every 60 seconds; can be switched off
- **Clickable Widgets**: Navigate to filtered lists
- **Responsive Layout**: Works on desktop and mobile

### Navigation

From the dashboard, click a widget to go to:
- Summary cards → Device, network or datacenter list
- Status cards → Device list filtered by status
- Network utilization → Network details
- Recent discoveries → Discovery page
- Stale devices → Device details, or the filtered device list

## Utilization Snapshots

Snapshots are stored automatically for trend analysis:

### Snapshot Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Snapshot identifier |
| `type` | string | `network` or `pool` |
| `resource_id` | string | Network or pool ID |
| `resource_name` | string | Network or pool name |
| `total_ips` | int | Total IPs |
| `used_ips` | int | Used IPs |
| `utilization` | float | Utilization percentage (0-100) |
| `timestamp` | timestamp | When the snapshot was taken |

### Storage

Snapshots are stored in the `utilization_snapshots` table:
- Snapshots older than `SNAPSHOT_RETENTION_DAYS` are deleted automatically
- Indexed for efficient querying

## CLI Access

### Get Dashboard Stats

```bash
# Get JSON output
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/dashboard | jq
```

### Query Trends

```bash
# Get a pool's 30-day trend
curl -H "Authorization: Bearer $RACKD_TOKEN" \
  "http://localhost:8080/api/v1/dashboard/trend?type=pool&resource_id=<pool-id>&days=30" | jq
```

## Use Cases

### Daily Operations

Start each day reviewing:
1. Network utilization
2. Recent discoveries
3. Stale device count

### Capacity Planning

Use trend data for planning:
1. Review a pool's or network's 90-day utilization trend
2. Identify growing pools
3. Plan capacity expansion

### Compliance Reporting

Generate reports from dashboard data:
1. Device counts by status
2. Network utilization summary

### Monitoring Integration

Dashboard data can feed external monitoring:
```bash
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/dashboard | jq '{
  devices: .total_devices,
  stale: .stale_devices,
  utilization: .network_utilization
}'
```

## RBAC Permissions

Both dashboard endpoints require `dashboard:read`.

### Default Role Assignments

All roles (admin, operator, viewer) have `dashboard:read`.

## Best Practices

1. **Regular Review**: Check the dashboard daily
2. **Trend Monitoring**: Watch utilization trends for capacity planning
3. **Stale Device Cleanup**: Review and clean up stale devices
4. **Alert Threshold**: Set `POOL_UTILIZATION_ALERT_PERCENT` for your environment
5. **Historical Analysis**: Use trend data for planning and reporting
