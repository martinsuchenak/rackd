# Dashboard Reporting & Trends

Rackd provides an enhanced dashboard with utilization trends, activity feeds, and aggregated statistics.

## Overview

The dashboard provides at-a-glance visibility into:
- Overall infrastructure health
- Pool utilization trends
- Network utilization summary
- Recent device discoveries
- Stale device detection
- Conflict alerts

## Dashboard API

### Get Dashboard Stats

```http
GET /api/dashboard
```

**Response:**
```json
{
  "total_devices": 500,
  "total_networks": 25,
  "total_pools": 40,
  "active_conflicts": 3,
  "devices_by_status": {
    "planned": 10,
    "active": 475,
    "maintenance": 5,
    "decommissioned": 10
  },
  "recent_discoveries": [
    {
      "hostname": "new-server-01",
      "ip_address": "192.168.1.100",
      "discovered_at": "2024-02-27T09:30:00Z"
    }
  ],
  "network_utilization": [
    {
      "network_id": "net-1",
      "network_name": "Production LAN",
      "cidr": "192.168.1.0/24",
      "utilization_percent": 75.5
    }
  ],
  "stale_devices": 12,
  "health_alerts": [
    {
      "type": "high_utilization",
      "resource": "Production Pool",
      "value": 92,
      "threshold": 90
    }
  ]
}
```

### Get Utilization Trend

```http
GET /api/dashboard/trend
```

Query parameters:
- `pool_id` - Filter to specific pool (optional)
- `days` - Number of days (default: 7)

**Response:**
```json
{
  "trend": [
    {"date": "2024-02-20", "utilization": 65.2},
    {"date": "2024-02-21", "utilization": 66.1},
    {"date": "2024-02-22", "utilization": 67.5},
    {"date": "2024-02-23", "utilization": 68.0},
    {"date": "2024-02-24", "utilization": 68.2},
    {"date": "2024-02-25", "utilization": 69.1},
    {"date": "2024-02-26", "utilization": 70.5}
  ]
}
```

## Dashboard Components

### Summary Stats

Top-level statistics cards:
- **Total Devices**: All devices across all statuses
- **Total Networks**: All configured networks
- **Total Pools**: All IP pools
- **Active Conflicts**: Current conflict count (clickable)

### Device Status Widget

Breakdown of devices by status:
- Planned (blue)
- Active (green)
- Maintenance (orange)
- Decommissioned (gray)

Each status is clickable to filter the device list.

### Utilization Trend Chart

Line chart showing pool utilization over time:
- Configurable time range (7/30/90 days)
- Sparkline visualization
- Hover for detailed values

### Network Utilization List

Top networks by utilization:
- Network name and CIDR
- Utilization percentage
- Visual progress bar
- Color-coded by utilization level

### Recent Discoveries

Feed of recently discovered devices:
- Hostname
- IP address
- Discovery timestamp
- Click to view device details

### Stale Devices

Count of devices not seen in discovery for the stale threshold:
- Configurable stale threshold
- Click to view stale device list

### Health Alerts

Active alerts requiring attention:
- High utilization warnings
- Conflict alerts
- Pool exhaustion warnings

## Configuration

### Snapshot Collection

Utilization snapshots are collected periodically for trend analysis:

```yaml
# config.yaml
snapshot:
  interval: 1h           # How often to collect snapshots
  retention_days: 90     # How long to keep snapshots
```

### Stale Device Threshold

Configure when devices are considered stale:

```yaml
# config.yaml
stale_device_days: 30    # Days without discovery
```

### Dashboard Refresh

Configure auto-refresh interval:

```yaml
# config.yaml
dashboard:
  refresh_interval: 5m   # Auto-refresh interval
```

## Web UI

Access the dashboard at `/` or `/dashboard`.

### Features

- **Auto-refresh**: Configurable automatic data refresh
- **Clickable Widgets**: Navigate to filtered lists
- **Trend Chart**: Interactive utilization visualization
- **Responsive Layout**: Works on desktop and mobile

### Navigation

From the dashboard, click any widget to:
- Status cards → Device list filtered by status
- Conflicts → Conflicts page
- Network utilization → Pool details
- Recent discoveries → Device details
- Stale devices → Filtered device list

## Utilization Snapshots

Snapshots are stored automatically for trend analysis:

### Snapshot Model

| Field | Type | Description |
|-------|------|-------------|
| `pool_id` | string | Pool identifier |
| `timestamp` | timestamp | When snapshot was taken |
| `total_ips` | int | Total IPs in pool |
| `used_ips` | int | Used IPs |
| `utilization_percent` | float | Utilization percentage |

### Storage

Snapshots are stored in the `utilization_snapshots` table:
- Automatic cleanup of old snapshots
- Indexed for efficient querying
- Minimal storage impact

## CLI Access

### Get Dashboard Stats

```bash
# Get JSON output
curl -H "Authorization: Bearer $TOKEN" /api/dashboard | jq
```

### Query Trends

```bash
# Get 30-day trend
curl "/api/dashboard/trend?days=30" | jq
```

## Use Cases

### Daily Operations

Start each day reviewing:
1. Active conflicts count
2. High utilization alerts
3. Recent discoveries
4. Stale device count

### Capacity Planning

Use trend data for planning:
1. Review 90-day utilization trend
2. Identify growing pools
3. Plan capacity expansion

### Compliance Reporting

Generate reports from dashboard data:
1. Device counts by status
2. Network utilization summary
3. Conflict resolution status

### Monitoring Integration

Dashboard data can feed external monitoring:
```bash
# Export metrics for Prometheus/Grafana
curl /api/dashboard | jq '{
  devices: .total_devices,
  conflicts: .active_conflicts,
  utilization: .network_utilization
}'
```

## RBAC Permissions

Dashboard access requires:
- `devices:list` - For device counts and discoveries
- `networks:list` - For network utilization
- `pools:list` - For pool statistics
- `conflicts:list` - For conflict counts

### Default Role Assignments

All roles (admin, operator, viewer) can view the dashboard.

## Best Practices

1. **Regular Review**: Check dashboard daily for alerts
2. **Trend Monitoring**: Watch utilization trends for capacity planning
3. **Conflict Resolution**: Address conflicts promptly
4. **Stale Device Cleanup**: Review and clean up stale devices
5. **Configure Thresholds**: Set appropriate alert thresholds for your environment
6. **Historical Analysis**: Use trend data for planning and reporting
