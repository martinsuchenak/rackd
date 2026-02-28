# NAT Tracking

Rackd provides comprehensive NAT (Network Address Translation) mapping management for tracking external-to-internal IP/port translations.

## Overview

NAT tracking allows you to document and manage NAT mappings in your network infrastructure. This is essential for:

- Documenting firewall rules and port forwards
- Tracking which external IPs map to internal services
- Managing IP address utilization across NAT boundaries
- Audit trail for NAT configuration changes

## NAT Mapping Model

Each NAT mapping contains:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (auto-generated UUID) |
| `name` | string | Descriptive name for the mapping |
| `external_ip` | string | External/public IP address |
| `external_port` | int | External port number (1-65535) |
| `internal_ip` | string | Internal/private IP address |
| `internal_port` | int | Internal port number (1-65535) |
| `protocol` | string | Protocol: `tcp`, `udp`, or `any` (default: `tcp`) |
| `device_id` | string | Optional linked device ID |
| `datacenter_id` | string | Optional datacenter ID |
| `network_id` | string | Optional network ID |
| `description` | string | Optional description |
| `enabled` | bool | Whether the mapping is active (default: `true`) |
| `tags` | []string | Optional tags for categorization |
| `created_at` | timestamp | Creation timestamp |
| `updated_at` | timestamp | Last update timestamp |

## API Endpoints

### List NAT Mappings

```http
GET /api/nat
```

Query parameters:
- `protocol` - Filter by protocol (tcp/udp/any)
- `external_ip` - Filter by external IP address
- `internal_ip` - Filter by internal IP address
- `device_id` - Filter by device ID
- `datacenter_id` - Filter by datacenter ID
- `enabled` - Filter by enabled status (true/false)

**Response:**
```json
[
  {
    "id": "nat-abc123",
    "name": "Web Server HTTPS",
    "external_ip": "203.0.113.10",
    "external_port": 443,
    "internal_ip": "192.168.1.10",
    "internal_port": 443,
    "protocol": "tcp",
    "enabled": true,
    "description": "HTTPS to internal web server",
    "tags": ["production", "web"]
  }
]
```

### Get NAT Mapping

```http
GET /api/nat/{id}
```

### Create NAT Mapping

```http
POST /api/nat
```

**Request body:**
```json
{
  "name": "Web Server HTTPS",
  "external_ip": "203.0.113.10",
  "external_port": 443,
  "internal_ip": "192.168.1.10",
  "internal_port": 443,
  "protocol": "tcp",
  "description": "HTTPS to internal web server",
  "enabled": true,
  "tags": ["production", "web"]
}
```

Required fields: `name`, `external_ip`, `external_port`, `internal_ip`, `internal_port`

### Update NAT Mapping

```http
PUT /api/nat/{id}
```

All fields are optional for partial updates.

### Delete NAT Mapping

```http
DELETE /api/nat/{id}
```

## CLI Commands

### List NAT Mappings

```bash
# List all NAT mappings
rackd nat list

# Filter by protocol
rackd nat list --protocol tcp

# Filter by external IP
rackd nat list --external-ip 203.0.113.10

# Output as JSON
rackd nat list --output json
```

### Get NAT Mapping

```bash
rackd nat get --id nat-abc123
```

### Create NAT Mapping

```bash
rackd nat create \
  --name "Web Server HTTPS" \
  --external-ip 203.0.113.10 \
  --external-port 443 \
  --internal-ip 192.168.1.10 \
  --internal-port 443 \
  --protocol tcp \
  --description "HTTPS to internal web server" \
  --tags "production,web"
```

### Update NAT Mapping

```bash
rackd nat update \
  --id nat-abc123 \
  --name "Updated Name" \
  --external-port 8443 \
  --disabled
```

### Delete NAT Mapping

```bash
rackd nat delete --id nat-abc123
```

## Web UI

Access NAT management at `/nat` in the web interface.

### Features

- **List View**: Table view with filtering by protocol, external IP, and enabled status
- **Create/Edit Modal**: Form for creating and editing NAT mappings
- **Delete Confirmation**: Confirmation dialog before deletion
- **Status Indicators**: Visual indicators for enabled/disabled mappings
- **Tag Support**: Add and manage tags for categorization

## RBAC Permissions

NAT tracking uses the following permissions:

| Permission | Description |
|------------|-------------|
| `nat:list` | View list of NAT mappings |
| `nat:read` | View individual NAT mapping details |
| `nat:create` | Create new NAT mappings |
| `nat:update` | Modify existing NAT mappings |
| `nat:delete` | Delete NAT mappings |

### Default Role Assignments

- **admin**: All NAT permissions
- **operator**: All NAT permissions
- **viewer**: `nat:list`, `nat:read`

## Validation Rules

1. **Name**: Required, cannot be empty
2. **External IP**: Required, must be a valid IP address
3. **External Port**: Required, must be 1-65535
4. **Internal IP**: Required, must be a valid IP address
5. **Internal Port**: Required, must be 1-65535
6. **Protocol**: Must be one of `tcp`, `udp`, or `any` (defaults to `tcp`)

## Use Cases

### Port Forwarding Documentation

Document firewall port forwards for compliance and troubleshooting:

```json
{
  "name": "Mail Server SMTP",
  "external_ip": "203.0.113.25",
  "external_port": 25,
  "internal_ip": "192.168.1.25",
  "internal_port": 25,
  "protocol": "tcp",
  "tags": ["email", "production"]
}
```

### Load Balancer Backends

Track load balancer VIP to pool member mappings:

```json
{
  "name": "Web Pool Member 1",
  "external_ip": "203.0.113.100",
  "external_port": 80,
  "internal_ip": "10.0.1.101",
  "internal_port": 8080,
  "protocol": "tcp"
}
```

### Service Discovery Integration

Link NAT mappings to devices for automatic documentation:

```json
{
  "name": "API Gateway",
  "external_ip": "203.0.113.50",
  "external_port": 443,
  "internal_ip": "10.0.2.50",
  "internal_port": 8443,
  "protocol": "tcp",
  "device_id": "dev-123"
}
```

## Best Practices

1. **Naming Convention**: Use descriptive names that include the service and environment
2. **Tagging**: Use tags to group related mappings (e.g., by service, environment, customer)
3. **Device Linking**: Link NAT mappings to devices when applicable for better visibility
4. **Documentation**: Use the description field to document the purpose and any relevant ticket numbers
5. **Disabled State**: Disable mappings instead of deleting them when decommissioning services (for audit trail)
