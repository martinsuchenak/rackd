# IP Conflict Detection

Rackd automatically detects and tracks IP address conflicts to help maintain network integrity.

## Overview

IP conflicts occur when:
- Multiple devices are assigned the same IP address
- Network subnets overlap
- Reserved IPs are assigned to devices

Rackd detects these conflicts automatically and provides tools for resolution.

## Conflict Types

| Type | Description |
|------|-------------|
| `duplicate_ip` | Same IP assigned to multiple devices |
| `overlapping_subnet` | Network subnets overlap |
| `reserved_ip_assigned` | Reserved IP assigned to a device |

## Conflict Status

| Status | Description |
|--------|-------------|
| `active` | Conflict currently exists |
| `resolved` | Conflict has been resolved |
| `ignored` | Conflict marked as acceptable |

## Conflict Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `type` | string | Conflict type |
| `status` | string | Current status |
| `ip_address` | string | Affected IP address (for duplicate_ip) |
| `network_id` | string | Affected network (for overlapping_subnet) |
| `device_ids` | []string | IDs of conflicting devices |
| `description` | string | Human-readable description |
| `resolution` | string | How the conflict was resolved |
| `resolved_at` | timestamp | When conflict was resolved |
| `resolved_by` | string | User who resolved the conflict |
| `created_at` | timestamp | When conflict was detected |
| `updated_at` | timestamp | Last update timestamp |

## Automatic Detection

Conflicts are automatically detected when:
- A new device is created
- A device's IP address is updated
- A new network is added
- A network's subnet is modified
- An IP reservation is created

## API Endpoints

### List Conflicts

```http
GET /api/conflicts
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
    "description": "IP 192.168.1.10 is assigned to 2 devices",
    "created_at": "2024-02-27T10:00:00Z"
  }
]
```

### Get Conflict

```http
GET /api/conflicts/{id}
```

### Detect Conflicts

Trigger manual conflict detection scan:

```http
POST /api/conflicts/detect
```

**Response:**
```json
{
  "conflicts_found": 2,
  "conflicts": ["conflict-abc123", "conflict-def456"]
}
```

### Resolve Conflict

```http
POST /api/conflicts/{id}/resolve
```

**Request body:**
```json
{
  "resolution": "Updated device dev-2 to use 192.168.1.11"
}
```

### Ignore Conflict

Mark a conflict as acceptable:

```http
POST /api/conflicts/{id}/ignore
```

**Request body:**
```json
{
  "reason": "Both devices are in different VLANs"
}
```

### Delete Conflict

Remove a resolved/ignored conflict record:

```http
DELETE /api/conflicts/{id}
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
rackd conflict resolve \
  --id conflict-abc123 \
  --resolution "Updated device IP to 192.168.1.11"
```

### Ignore Conflict

```bash
rackd conflict ignore \
  --id conflict-abc123 \
  --reason "Devices in different broadcast domains"
```

### Delete Conflict

```bash
rackd conflict delete --id conflict-abc123
```

## Web UI

Access conflict management at `/conflicts` in the web interface.

### Features

- **Conflict List**: Table view with filtering by status and type
- **Conflict Details**: View affected devices and networks
- **Resolution Actions**: Resolve or ignore buttons
- **Navigation Badge**: Active conflict count in navigation
- **Device Warnings**: Conflict indicators on device list and detail pages
- **Network Warnings**: Overlap badges on network list

### UI Indicators

- **Red Badge**: Active conflicts in navigation
- **Warning Icon**: Devices with IP conflicts
- **Orange Status**: Networks with overlapping subnets
- **Pool Heatmap**: Conflicted IPs shown in orange

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
- **operator**: All conflict permissions
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

### Reserved IP Assigned

1. **Review**: Check reservation purpose
2. **Either**: Remove reservation or change device IP
3. **Resolve**: Document action taken

## Best Practices

1. **Regular Scans**: Run conflict detection after bulk imports
2. **Prompt Resolution**: Address conflicts quickly to prevent issues
3. **Documentation**: Always add resolution notes
4. **Ignored Conflicts**: Use sparingly and document reason
5. **Prevention**: Use IP reservations for planned assignments
6. **Monitoring**: Watch the navigation badge for new conflicts

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

### Network Topology

The device graph shows conflict indicators on affected devices.
