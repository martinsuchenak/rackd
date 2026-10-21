# IP Conflict Detection

Rackd automatically detects and tracks IP address conflicts to help maintain network integrity.

## Overview

IP conflicts occur when:
- Multiple devices are assigned the same IP address
- Network subnets overlap

Rackd detects these conflicts automatically and provides tools for resolution.

## Conflict Types

| Type | Description |
|------|-------------|
| `duplicate_ip` | Same IP assigned to multiple devices |
| `overlapping_subnet` | Network subnets overlap |

## Conflict Status

| Status | Description |
|--------|-------------|
| `active` | Conflict currently exists |
| `resolved` | Conflict has been resolved |
| `ignored` | Conflict marked as acceptable. No API or CLI command sets this status yet. |

## Conflict Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `type` | string | Conflict type |
| `status` | string | Current status |
| `description` | string | Human-readable description |
| `ip_address` | string | Affected IP address (for duplicate_ip) |
| `device_ids` | []string | IDs of conflicting devices (for duplicate_ip) |
| `device_names` | []string | Names of conflicting devices (for duplicate_ip) |
| `network_ids` | []string | IDs of overlapping networks (for overlapping_subnet) |
| `network_names` | []string | Names of overlapping networks (for overlapping_subnet) |
| `subnets` | []string | Overlapping subnets (for overlapping_subnet) |
| `detected_at` | timestamp | When the conflict was detected |
| `resolved_at` | timestamp | When the conflict was resolved |
| `resolved_by` | string | User who resolved the conflict |
| `notes` | string | Resolution notes |

## Automatic Detection

Creating or updating a device checks each of its addresses. When another device already has the same IP, Rackd records a duplicate-IP conflict listing every device that holds it.

- One device with the same IP on two of its addresses is not a conflict.
- An IP has at most one active duplicate-IP conflict. While it has an active or ignored one, device writes and detection don't add another, so an ignored conflict stays ignored. Once its conflict is resolved, a new duplicate on that IP is recorded as a new conflict.
- Overlapping subnets, and duplicates created another way (such as a bulk import), are found by running detection explicitly; see Detect Conflicts below.

Recorded conflicts follow the inventory. Whenever conflicts are listed, summarised or detected, the open ones (active and ignored) are checked against the devices and networks as they are now:

- A duplicate IP that no device shares any more, or an overlap whose network was deleted or whose subnets no longer overlap, is resolved by `rackd`, with a note saying why. Notes already on it are kept.
- A duplicate IP that other devices still share, or an overlap whose networks were renamed, is updated to list the current devices or networks.

The web UI's Conflicts page lists active conflicts.

## API Endpoints

### List Conflicts

```http
GET /api/v1/conflicts
```

Query parameters:
- `status` - Filter by status (active/resolved/ignored)
- `type` - Filter by conflict type

**Response:**
```json
[
  {
    "id": "conflict-abc123",
    "type": "duplicate_ip",
    "status": "active",
    "ip_address": "192.168.1.10",
    "device_ids": ["dev-1", "dev-2"],
    "device_names": ["web-01", "web-02"],
    "description": "IP 192.168.1.10 is assigned to 2 devices",
    "detected_at": "2024-02-27T10:00:00Z"
  }
]
```

### Get Conflict

```http
GET /api/v1/conflicts/{id}
```

### Detect Conflicts

Trigger manual conflict detection scan:

```http
POST /api/v1/conflicts/detect
```

Add `?type=duplicate_ip` or `?type=overlapping_subnet` to detect only one type.

One run records at most 1,000 new overlapping-subnet pairs; pairs already recorded don't count. Networks may overlap freely, so many copies of one subnet would otherwise make an unbounded number of pairs. If a run hits the limit, run detection again to record the next 1,000. The response lists the active duplicate-IP conflicts, and the overlap conflicts this run recorded.

**Response:**
```json
{
  "conflicts": [
    {"id": "conflict-abc123", "type": "duplicate_ip", "status": "active", "ip_address": "192.168.1.10"}
  ],
  "count": 1
}
```

Each entry in `conflicts` is a full conflict object.

### Resolve Conflict

```http
POST /api/v1/conflicts/{id}/resolve
```

**Request body:**
```json
{
  "keep_device_id": "dev-1",
  "notes": "dev-1 owns this address; dev-2 was re-addressed"
}
```

For a `duplicate_ip` conflict, `keep_device_id` names the device that keeps the address; it is removed from the others, which needs `device:update` as well as `conflict:resolve`. For an `overlapping_subnet` conflict, `keep_network_id` names the network to keep. The body may repeat the conflict as `conflict_id`, but not name a different one.

### Delete Conflict

Remove a conflict record:

```http
DELETE /api/v1/conflicts/{id}
```

## CLI Commands

### List Conflicts

```bash
# List all active conflicts
rackd conflict list

# List resolved conflicts
rackd conflict list --status resolved

# List specific type
rackd conflict list --type duplicate_ip

# Output as JSON
rackd conflict list --output json
```

### Get Conflict Details

```bash
rackd conflict get --id conflict-abc123
```

### Detect Conflicts

```bash
rackd conflict detect
```

### Resolve Conflict

```bash
# Keep the address on dev-1 and remove it from the other devices
rackd conflict resolve --id conflict-abc123 --keep-device-id dev-1 --notes "dev-2 re-addressed"

# Keep one of two overlapping networks
rackd conflict resolve --id conflict-def456 --keep-network-id net-1
```

### Delete Conflict

```bash
rackd conflict delete --id conflict-abc123
```

## Web UI

Access conflict management at `/conflicts` in the web interface.

### Features

- **Summary Cards**: Counts of duplicate IPs, overlapping subnets and all active conflicts; click a card to filter the list by type
- **Detect Conflicts**: Button that runs detection
- **Conflict List**: Each conflict with links to the affected devices or networks
- **Navigation Badge**: Red badge with the active conflict count

Resolve and delete conflicts with the CLI or the API.

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `conflict:list` | View list of conflicts |
| `conflict:read` | View conflict details |
| `conflict:detect` | Trigger conflict detection |
| `conflict:resolve` | Resolve conflicts |
| `conflict:delete` | Delete conflict records |

### Default Role Assignments

- **admin**: All conflict permissions
- **operator**: `conflict:list`, `conflict:read`, `conflict:detect`
- **viewer**: `conflict:list`, `conflict:read`

## Conflict Resolution Strategies

### Duplicate IP

1. **Identify**: Determine which device should keep the IP
2. **Update**: Change the IP on the other device
3. **Resolve**: Mark conflict as resolved with explanation

### Overlapping Subnet

1. **Review**: Check if overlap is intentional (VXLAN, etc.)
2. **Adjust**: Modify network CIDR if needed
3. **Document**: Add resolution notes

## Best Practices

1. **Regular Scans**: Run conflict detection after bulk imports
2. **Prompt Resolution**: Address conflicts quickly to prevent issues
3. **Documentation**: Always add resolution notes
4. **Prevention**: Use IP reservations for planned assignments
5. **Monitoring**: Watch the navigation badge for new conflicts

## Integration with Other Features

### Device Service

Conflicts are automatically checked when:
- Creating a device
- Updating a device's IP address

If a conflict is detected, it's created immediately.

### Webhooks

Subscribe to conflict events for automated notifications:
- `conflict.detected` - New conflict found
- `conflict.resolved` - Conflict resolved

