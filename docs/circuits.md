# Circuit Management

Rackd provides circuit tracking for managing network circuits including fiber, copper, microwave, and dark fiber connections.

## Overview

Circuit management allows you to:

- Track circuit provider information and circuit IDs
- Monitor circuit status and capacity
- Link circuits to devices and datacenters
- Maintain circuit documentation and history
- Plan circuit decommissioning

## Circuit Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (auto-generated UUID) |
| `name` | string | Descriptive name for the circuit |
| `provider` | string | Circuit provider name (e.g., "AT&T", "Lumen") |
| `circuit_id` | string | Provider's circuit identifier |
| `type` | string | Circuit type: `fiber`, `copper`, `microwave`, `dark_fiber` |
| `status` | string | Status: `active`, `inactive`, `planned`, `decommissioned` |
| `capacity_mbps` | int | Circuit capacity in Mbps |
| `a_endpoint` | string | A-side endpoint location/device |
| `z_endpoint` | string | Z-side endpoint location/device |
| `datacenter_id` | string | Optional associated datacenter |
| `device_id` | string | Optional linked device |
| `description` | string | Optional description |
| `tags` | []string | Optional tags for categorization |
| `install_date` | date | Installation date |
| `termination_date` | date | Contract termination date |
| `created_at` | timestamp | Creation timestamp |
| `updated_at` | timestamp | Last update timestamp |

## Circuit Types

| Type | Description |
|------|-------------|
| `fiber` | Active fiber optic circuit |
| `copper` | Copper-based circuit (T1, DS3, etc.) |
| `microwave` | Wireless microwave link |
| `dark_fiber` | Unlit fiber (point-to-point) |

## Circuit Status

| Status | Description |
|--------|-------------|
| `active` | Circuit is operational |
| `inactive` | Circuit is temporarily down |
| `planned` | Circuit is planned but not yet installed |
| `decommissioned` | Circuit has been retired |

## API Endpoints

### List Circuits

```http
GET /api/circuits
```

Query parameters:
- `provider` - Filter by provider name
- `status` - Filter by status
- `type` - Filter by circuit type
- `datacenter_id` - Filter by datacenter

**Response:**
```json
[
  {
    "id": "circuit-abc123",
    "name": "Primary Internet",
    "provider": "Lumen",
    "circuit_id": "LUM-12345-ABC",
    "type": "fiber",
    "status": "active",
    "capacity_mbps": 10000,
    "a_endpoint": "DC1 - Meet Me Room",
    "z_endpoint": "Lumen POP - Downtown",
    "description": "Primary 10G internet circuit"
  }
]
```

### Get Circuit

```http
GET /api/circuits/{id}
```

### Create Circuit

```http
POST /api/circuits
```

**Request body:**
```json
{
  "name": "Primary Internet",
  "provider": "Lumen",
  "circuit_id": "LUM-12345-ABC",
  "type": "fiber",
  "status": "active",
  "capacity_mbps": 10000,
  "a_endpoint": "DC1 - Meet Me Room",
  "z_endpoint": "Lumen POP - Downtown",
  "description": "Primary 10G internet circuit",
  "install_date": "2024-01-15",
  "tags": ["production", "internet"]
}
```

Required fields: `name`, `provider`, `type`, `status`

### Update Circuit

```http
PUT /api/circuits/{id}
```

All fields are optional for partial updates.

### Delete Circuit

```http
DELETE /api/circuits/{id}
```

## CLI Commands

### List Circuits

```bash
# List all circuits
rackd circuit list

# Filter by provider
rackd circuit list --provider Lumen

# Filter by status
rackd circuit list --status active

# Filter by type
rackd circuit list --type fiber

# Output as JSON
rackd circuit list --output json
```

### Get Circuit

```bash
rackd circuit get --id circuit-abc123
```

### Create Circuit

```bash
rackd circuit create \
  --name "Primary Internet" \
  --provider "Lumen" \
  --circuit-id "LUM-12345-ABC" \
  --type fiber \
  --status active \
  --capacity 10000 \
  --a-endpoint "DC1 - Meet Me Room" \
  --z-endpoint "Lumen POP - Downtown"
```

### Update Circuit

```bash
rackd circuit update \
  --id circuit-abc123 \
  --status inactive \
  --description "Temporarily down for maintenance"
```

### Delete Circuit

```bash
rackd circuit delete --id circuit-abc123
```

## Web UI

Access circuit management at `/circuits` in the web interface.

### Features

- **List View**: Table view with filtering by provider, status, and type
- **Create/Edit Modal**: Form for creating and editing circuits
- **Status Badges**: Color-coded status indicators
- **Capacity Display**: Visual representation of circuit capacity
- **Endpoint Information**: Clear display of A/Z endpoints

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `circuit:list` | View list of circuits |
| `circuit:read` | View individual circuit details |
| `circuit:create` | Create new circuits |
| `circuit:update` | Modify existing circuits |
| `circuit:delete` | Delete circuits |

### Default Role Assignments

- **admin**: All circuit permissions
- **operator**: All circuit permissions
- **viewer**: `circuit:list`, `circuit:read`

## Use Cases

### ISP Circuit Tracking

```json
{
  "name": "Secondary ISP",
  "provider": "AT&T",
  "circuit_id": "ATT-98765-XYZ",
  "type": "fiber",
  "status": "active",
  "capacity_mbps": 1000,
  "a_endpoint": "Building A - Demarc",
  "z_endpoint": "AT&T Central Office"
}
```

### MPLS WAN Circuit

```json
{
  "name": "Branch Office MPLS",
  "provider": "Verizon",
  "circuit_id": "VZ-MPLS-54321",
  "type": "fiber",
  "status": "active",
  "capacity_mbps": 100,
  "a_endpoint": "HQ Data Center",
  "z_endpoint": "Branch Office 5",
  "tags": ["wan", "mpls"]
}
```

### Dark Fiber

```json
{
  "name": "Inter-DC Dark Fiber",
  "provider": "Zayo",
  "circuit_id": "ZAYO-DF-11111",
  "type": "dark_fiber",
  "status": "active",
  "a_endpoint": "DC1",
  "z_endpoint": "DC2",
  "description": "24-strand dark fiber between data centers"
}
```

## Best Practices

1. **Naming**: Use consistent naming conventions that include location and purpose
2. **Provider Info**: Always include the provider's circuit ID for support tickets
3. **Endpoints**: Document both A and Z endpoints clearly
4. **Contract Dates**: Track install and termination dates for renewal planning
5. **Status Updates**: Keep status current to maintain accurate inventory
6. **Tagging**: Use tags to group circuits by function, region, or customer
