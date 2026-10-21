# Circuit Management

Rackd provides circuit tracking for managing network circuits including fiber, copper, microwave, and dark fiber connections.

## Overview

Circuit management allows you to:

- Track circuit provider information and circuit IDs
- Monitor circuit status and capacity
- Link circuits to devices and datacenters at each endpoint
- Track contract, cost and provider contact details

## Circuit Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (auto-generated UUID) |
| `name` | string | Descriptive name for the circuit |
| `provider` | string | Circuit provider name (e.g., "AT&T", "Lumen") |
| `circuit_id` | string | Provider's circuit identifier |
| `type` | string | Circuit type: `fiber`, `copper`, `microwave`, `dark_fiber` |
| `status` | string | Status: `active`, `maintenance`, `down`, `decommissioned` (default `active`) |
| `capacity_mbps` | int | Circuit capacity in Mbps |
| `datacenter_a_id` | string | Endpoint A datacenter |
| `datacenter_b_id` | string | Endpoint B datacenter |
| `device_a_id` | string | Device at endpoint A |
| `device_b_id` | string | Device at endpoint B |
| `port_a` | string | Port/interface at endpoint A |
| `port_b` | string | Port/interface at endpoint B |
| `ip_address_a` | string | IP address at endpoint A |
| `ip_address_b` | string | IP address at endpoint B |
| `vlan_id` | int | VLAN ID |
| `description` | string | Optional description |
| `install_date` | timestamp | Installation date (RFC 3339) |
| `terminate_date` | timestamp | Contract termination date (RFC 3339) |
| `monthly_cost` | float | Monthly cost |
| `contract_number` | string | Contract reference |
| `contact_name` | string | Provider contact |
| `contact_phone` | string | Provider phone |
| `contact_email` | string | Provider email |
| `tags` | []string | Optional tags for categorization |
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
| `maintenance` | Circuit is under maintenance |
| `down` | Circuit is down |
| `decommissioned` | Circuit has been retired |

## API Endpoints

### List Circuits

```http
GET /api/v1/circuits
```

Query parameters:
- `provider` - Filter by provider name
- `status` - Filter by status
- `type` - Filter by circuit type
- `datacenter_id` - Filter by datacenter (matches endpoint A or B)

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
    "datacenter_a_id": "dc-1",
    "port_a": "Meet Me Room - Panel 3",
    "description": "Primary 10G internet circuit"
  }
]
```

### Get Circuit

```http
GET /api/v1/circuits/{id}
```

### Create Circuit

```http
POST /api/v1/circuits
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
  "datacenter_a_id": "dc-1",
  "port_a": "Meet Me Room - Panel 3",
  "description": "Primary 10G internet circuit",
  "install_date": "2024-01-15T00:00:00Z",
  "tags": ["production", "internet"]
}
```

Required fields: `name`, `circuit_id`, `provider`. `status` defaults to `active`.

### Update Circuit

```http
PUT /api/v1/circuits/{id}
```

All fields are optional for partial updates.

### Delete Circuit

```http
DELETE /api/v1/circuits/{id}
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
  --datacenter-a dc-1 \
  --port-a "Meet Me Room - Panel 3"
```

### Update Circuit

```bash
rackd circuit update \
  --id circuit-abc123 \
  --status maintenance \
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
- **Create/Edit Modal**: Form for creating and editing circuits, including endpoint datacenters
- **Status Badges**: Color-coded status indicators

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `circuits:list` | View list of circuits |
| `circuits:read` | View individual circuit details |
| `circuits:create` | Create new circuits |
| `circuits:update` | Modify existing circuits |
| `circuits:delete` | Delete circuits |

### Default Role Assignments

- **admin**: All circuit permissions
- **operator**: `circuits:list`, `circuits:read`, `circuits:create`, `circuits:update`
- **viewer**: `circuits:list`, `circuits:read`

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
  "datacenter_a_id": "dc-building-a",
  "port_a": "Demarc",
  "contact_phone": "<provider-support-phone>"
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
  "datacenter_a_id": "dc-hq",
  "datacenter_b_id": "dc-branch-5",
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
  "datacenter_a_id": "dc-1",
  "datacenter_b_id": "dc-2",
  "description": "24-strand dark fiber between data centers"
}
```

## Best Practices

1. **Naming**: Use consistent naming conventions that include location and purpose
2. **Provider Info**: Always include the provider's circuit ID for support tickets
3. **Endpoints**: Record both A and B endpoints
4. **Contract Dates**: Track install and termination dates for renewal planning
5. **Status Updates**: Keep status current to maintain accurate inventory
6. **Tagging**: Use tags to group circuits by function, region, or customer
