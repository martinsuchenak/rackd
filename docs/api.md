# API Reference

The Rackd REST API provides programmatic access to all infrastructure management features including devices, networks, datacenters, IP pools, discovery, and relationships.

## Base URL

```
http://localhost:8080/api
```

## Authentication

Rackd supports optional Bearer token authentication. When enabled, include the token in the Authorization header:

```http
Authorization: Bearer <your-token>
```

## Content Type

All API endpoints accept and return JSON data:

```http
Content-Type: application/json
```

## Error Handling

### Error Response Format

All errors return a consistent JSON structure:

```json
{
  "code": "ERROR_CODE",
  "error": "Human-readable error message",
  "details": {}
}
```

### HTTP Status Codes

- `200` - Success
- `201` - Created
- `204` - No Content (successful deletion)
- `400` - Bad Request (validation errors)
- `404` - Not Found
- `409` - Conflict (resource already exists or no available IPs)
- `500` - Internal Server Error

### Common Error Codes

- `VALIDATION_ERROR` - Request validation failed
- `INVALID_INPUT` - Invalid JSON or request format
- `DEVICE_NOT_FOUND` - Device does not exist
- `NETWORK_NOT_FOUND` - Network does not exist
- `DATACENTER_NOT_FOUND` - Datacenter does not exist
- `POOL_NOT_FOUND` - IP pool does not exist
- `INTERNAL_ERROR` - Server error

### Validation Errors

Validation errors include detailed field-level information:

```json
{
  "code": "VALIDATION_ERROR",
  "error": "name: name is required; ip: invalid IP address",
  "details": {
    "name": {
      "field": "name",
      "message": "name is required"
    },
    "ip": {
      "field": "ip", 
      "message": "invalid IP address"
    }
  }
}
```

## Pagination

Currently, the API returns all results without pagination. Future versions may implement cursor-based pagination for large datasets.

## Filtering

Many list endpoints support query parameters for filtering results. Common filters include:

- `name` - Filter by name (partial match)
- `tags` - Filter by tags (exact match, multiple values supported)
- `datacenter_id` - Filter by datacenter ID
- `network_id` - Filter by network ID

## Data Models

### Datacenter

```json
{
  "id": "uuid",
  "name": "string",
  "location": "string", 
  "description": "string",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Network

```json
{
  "id": "uuid",
  "name": "string",
  "subnet": "192.168.1.0/24",
  "vlan_id": 100,
  "datacenter_id": "uuid",
  "description": "string",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Device

```json
{
  "id": "uuid",
  "name": "string",
  "hostname": "string",
  "description": "string",
  "make_model": "string",
  "os": "string",
  "datacenter_id": "uuid",
  "username": "string",
  "location": "string",
  "tags": ["tag1", "tag2"],
  "addresses": [
    {
      "ip": "192.168.1.10",
      "port": 22,
      "type": "ipv4",
      "label": "management",
      "network_id": "uuid",
      "switch_port": "eth0/1",
      "pool_id": "uuid"
    }
  ],
  "domains": ["example.com"],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Network Pool

```json
{
  "id": "uuid",
  "network_id": "uuid",
  "name": "string",
  "start_ip": "192.168.1.10",
  "end_ip": "192.168.1.100",
  "description": "string",
  "tags": ["production"],
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

## Datacenters

### List Datacenters

```http
GET /api/datacenters
```

**Query Parameters:**
- `name` (optional) - Filter by name (partial match)

**Response:**
```json
[
  {
    "id": "dc1-uuid",
    "name": "Primary DC",
    "location": "New York",
    "description": "Primary datacenter",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

### Create Datacenter

```http
POST /api/datacenters
```

**Request Body:**
```json
{
  "name": "Primary DC",
  "location": "New York", 
  "description": "Primary datacenter"
}
```

**Response:** `201 Created`
```json
{
  "id": "dc1-uuid",
  "name": "Primary DC",
  "location": "New York",
  "description": "Primary datacenter",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Get Datacenter

```http
GET /api/datacenters/{id}
```

**Response:** `200 OK`
```json
{
  "id": "dc1-uuid",
  "name": "Primary DC",
  "location": "New York",
  "description": "Primary datacenter",
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### Update Datacenter

```http
PUT /api/datacenters/{id}
```

**Request Body:**
```json
{
  "name": "Updated DC Name",
  "location": "Boston",
  "description": "Updated description"
}
```

**Response:** `200 OK` (returns updated datacenter)

### Delete Datacenter

```http
DELETE /api/datacenters/{id}
```

**Response:** `204 No Content`

### List Datacenter Devices

```http
GET /api/datacenters/{id}/devices
```

**Response:** `200 OK` (returns array of devices)

## Networks

### List Networks

```http
GET /api/networks
```

**Query Parameters:**
- `name` (optional) - Filter by name
- `datacenter_id` (optional) - Filter by datacenter
- `vlan_id` (optional) - Filter by VLAN ID

**Response:** `200 OK` (returns array of networks)

### Create Network

```http
POST /api/networks
```

**Request Body:**
```json
{
  "name": "Production Network",
  "subnet": "192.168.1.0/24",
  "vlan_id": 100,
  "datacenter_id": "dc1-uuid",
  "description": "Production network"
}
```

**Response:** `201 Created` (returns created network)

### Get Network

```http
GET /api/networks/{id}
```

**Response:** `200 OK` (returns network details)

### Update Network

```http
PUT /api/networks/{id}
```

**Request Body:** (same as create, all fields optional)

**Response:** `200 OK` (returns updated network)

### Delete Network

```http
DELETE /api/networks/{id}
```

**Response:** `204 No Content`

### List Network Devices

```http
GET /api/networks/{id}/devices
```

**Response:** `200 OK` (returns array of devices in network)

### Get Network Utilization

```http
GET /api/networks/{id}/utilization
```

**Response:** `200 OK`
```json
{
  "network_id": "net1-uuid",
  "total_ips": 254,
  "used_ips": 45,
  "available_ips": 209,
  "utilization": 17.7
}
```

## IP Pools

### List Network Pools

```http
GET /api/networks/{id}/pools
```

**Query Parameters:**
- `tags` (optional) - Filter by tags

**Response:** `200 OK` (returns array of pools)

### Create Network Pool

```http
POST /api/networks/{id}/pools
```

**Request Body:**
```json
{
  "name": "DHCP Pool",
  "start_ip": "192.168.1.10",
  "end_ip": "192.168.1.100",
  "description": "DHCP address pool",
  "tags": ["dhcp", "production"]
}
```

**Response:** `201 Created` (returns created pool)

### Get Pool

```http
GET /api/pools/{id}
```

**Response:** `200 OK` (returns pool details)

### Update Pool

```http
PUT /api/pools/{id}
```

**Request Body:** (same as create, all fields optional)

**Response:** `200 OK` (returns updated pool)

### Delete Pool

```http
DELETE /api/pools/{id}
```

**Response:** `204 No Content`

### Get Next Available IP

```http
GET /api/pools/{id}/next-ip
```

**Response:** `200 OK`
```json
{
  "ip": "192.168.1.15"
}
```

**Error:** `409 Conflict` if no IPs available

### Get Pool Heatmap

```http
GET /api/pools/{id}/heatmap
```

**Response:** `200 OK`
```json
[
  {
    "ip": "192.168.1.10",
    "status": "available"
  },
  {
    "ip": "192.168.1.11", 
    "status": "used",
    "device_id": "device-uuid"
  },
  {
    "ip": "192.168.1.12",
    "status": "reserved"
  }
]
```

## Devices

### List Devices

```http
GET /api/devices
```

**Query Parameters:**
- `tags` (optional) - Filter by tags (multiple values supported)
- `datacenter_id` (optional) - Filter by datacenter
- `network_id` (optional) - Filter by network

**Response:** `200 OK` (returns array of devices)

### Create Device

```http
POST /api/devices
```

**Request Body:**
```json
{
  "name": "web-server-01",
  "hostname": "web01.example.com",
  "description": "Production web server",
  "make_model": "Dell PowerEdge R740",
  "os": "Ubuntu 22.04",
  "datacenter_id": "dc1-uuid",
  "username": "admin",
  "location": "Rack A1",
  "tags": ["web", "production"],
  "addresses": [
    {
      "ip": "192.168.1.10",
      "port": 22,
      "type": "ipv4",
      "label": "management",
      "network_id": "net1-uuid"
    }
  ],
  "domains": ["example.com"]
}
```

**Response:** `201 Created` (returns created device)

### Get Device

```http
GET /api/devices/{id}
```

**Response:** `200 OK` (returns device details)

### Update Device

```http
PUT /api/devices/{id}
```

**Request Body:** (partial updates supported)
```json
{
  "description": "Updated description",
  "tags": ["web", "production", "updated"]
}
```

**Response:** `200 OK` (returns updated device)

### Delete Device

```http
DELETE /api/devices/{id}
```

**Response:** `204 No Content`

### Search Devices

```http
GET /api/devices/search?q=web
```

**Query Parameters:**
- `q` (required) - Search query (max 256 characters)

Searches across device names, descriptions, IP addresses, tags, and domains.

**Response:** `200 OK` (returns array of matching devices)

## Device Relationships

### Get Device Relationships

```http
GET /api/devices/{id}/relationships
```

**Response:** `200 OK`
```json
[
  {
    "parent_id": "parent-uuid",
    "child_id": "child-uuid", 
    "type": "contains",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

### Add Device Relationship

```http
POST /api/devices/{id}/relationships
```

**Request Body:**
```json
{
  "child_id": "child-device-uuid",
  "type": "contains"
}
```

**Relationship Types:**
- `contains` - Parent contains child (e.g., chassis contains blade)
- `connected_to` - Devices are connected (e.g., switch to server)
- `depends_on` - Parent depends on child (e.g., VM depends on host)

**Response:** `201 Created`
```json
{
  "status": "created"
}
```

### Get Related Devices

```http
GET /api/devices/{id}/related
```

**Query Parameters:**
- `type` (optional) - Filter by relationship type

**Response:** `200 OK` (returns array of related devices)

### Remove Device Relationship

```http
DELETE /api/devices/{id}/relationships/{child_id}/{type}
```

**Response:** `204 No Content`

## Discovery

### Start Network Scan

```http
POST /api/discovery/networks/{id}/scan
```

**Request Body:**
```json
{
  "scan_type": "quick"
}
```

**Scan Types:**
- `quick` - Basic ping sweep
- `full` - Port scanning and OS detection
- `deep` - Comprehensive service discovery

**Response:** `202 Accepted`
```json
{
  "id": "scan-uuid",
  "network_id": "net1-uuid",
  "status": "pending",
  "scan_type": "quick",
  "total_hosts": 254,
  "scanned_hosts": 0,
  "found_hosts": 0,
  "progress_percent": 0.0,
  "created_at": "2024-01-01T00:00:00Z",
  "updated_at": "2024-01-01T00:00:00Z"
}
```

### List Discovery Scans

```http
GET /api/discovery/scans
```

**Query Parameters:**
- `network_id` (optional) - Filter by network

**Response:** `200 OK` (returns array of scans)

### Get Discovery Scan

```http
GET /api/discovery/scans/{id}
```

**Response:** `200 OK` (returns scan details with progress)

### Cancel Discovery Scan

```http
POST /api/discovery/scans/{id}/cancel
```

**Response:** `204 No Content`

### Delete Discovery Scan

```http
DELETE /api/discovery/scans/{id}
```

**Response:** `204 No Content`

### List Discovered Devices

```http
GET /api/discovery/devices?network_id={network_id}
```

**Query Parameters:**
- `network_id` (required) - Network to list devices for

**Response:** `200 OK`
```json
[
  {
    "id": "discovered-uuid",
    "ip": "192.168.1.50",
    "mac_address": "00:11:22:33:44:55",
    "hostname": "unknown-device",
    "network_id": "net1-uuid",
    "status": "up",
    "confidence": 85,
    "os_guess": "Linux 3.x",
    "vendor": "Dell Inc.",
    "open_ports": [22, 80, 443],
    "services": [
      {
        "port": 22,
        "protocol": "tcp",
        "service": "ssh",
        "version": "OpenSSH 8.0"
      }
    ],
    "first_seen": "2024-01-01T00:00:00Z",
    "last_seen": "2024-01-01T01:00:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T01:00:00Z"
  }
]
```

### Delete Discovered Device

```http
DELETE /api/discovery/devices/{id}
```

**Response:** `204 No Content`

### Delete All Discovered Devices by Network

```http
DELETE /api/discovery/devices?network_id={network_id}
```

**Response:** `204 No Content`

### Promote Discovered Device

```http
POST /api/discovery/devices/{id}/promote
```

**Request Body:**
```json
{
  "name": "web-server-02",
  "make_model": "Dell PowerEdge",
  "datacenter_id": "dc1-uuid"
}
```

**Response:** `201 Created` (returns created device)

### List Discovery Rules

```http
GET /api/discovery/rules
```

**Response:** `200 OK`
```json
[
  {
    "id": "rule-uuid",
    "network_id": "net1-uuid",
    "enabled": true,
    "scan_type": "quick",
    "interval_hours": 24,
    "exclude_ips": "192.168.1.1-192.168.1.10",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
]
```

### Create Discovery Rule

```http
POST /api/discovery/rules
```

**Request Body:**
```json
{
  "network_id": "net1-uuid",
  "enabled": true,
  "scan_type": "quick",
  "interval_hours": 24,
  "exclude_ips": "192.168.1.1-192.168.1.10"
}
```

**Response:** `201 Created` (returns created rule)

### Get Discovery Rule

```http
GET /api/discovery/rules/{id}
```

**Response:** `200 OK` (returns rule details)

### Update Discovery Rule

```http
PUT /api/discovery/rules/{id}
```

**Request Body:** (same as create, all fields optional)

**Response:** `200 OK` (returns updated rule)

### Delete Discovery Rule

```http
DELETE /api/discovery/rules/{id}
```

**Response:** `204 No Content`

## Examples

### Complete Device Creation Workflow

1. **Create a datacenter:**
```bash
curl -X POST http://localhost:8080/api/datacenters \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Primary DC",
    "location": "New York",
    "description": "Primary datacenter"
  }'
```

2. **Create a network:**
```bash
curl -X POST http://localhost:8080/api/networks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production Network",
    "subnet": "192.168.1.0/24",
    "vlan_id": 100,
    "datacenter_id": "dc1-uuid",
    "description": "Production network"
  }'
```

3. **Create an IP pool:**
```bash
curl -X POST http://localhost:8080/api/networks/net1-uuid/pools \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Server Pool",
    "start_ip": "192.168.1.10",
    "end_ip": "192.168.1.100",
    "description": "Server IP pool"
  }'
```

4. **Create a device:**
```bash
curl -X POST http://localhost:8080/api/devices \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server-01",
    "hostname": "web01.example.com",
    "description": "Production web server",
    "make_model": "Dell PowerEdge R740",
    "os": "Ubuntu 22.04",
    "datacenter_id": "dc1-uuid",
    "tags": ["web", "production"],
    "addresses": [{
      "ip": "192.168.1.10",
      "type": "ipv4",
      "label": "management",
      "network_id": "net1-uuid"
    }]
  }'
```

### Network Discovery Workflow

1. **Start a network scan:**
```bash
curl -X POST http://localhost:8080/api/discovery/networks/net1-uuid/scan \
  -H "Content-Type: application/json" \
  -d '{"scan_type": "full"}'
```

2. **Check scan progress:**
```bash
curl http://localhost:8080/api/discovery/scans/scan-uuid
```

3. **List discovered devices:**
```bash
curl "http://localhost:8080/api/discovery/devices?network_id=net1-uuid"
```

4. **Promote a discovered device:**
```bash
curl -X POST http://localhost:8080/api/discovery/devices/discovered-uuid/promote \
  -H "Content-Type: application/json" \
  -d '{
    "name": "discovered-server",
    "datacenter_id": "dc1-uuid"
  }'
```

## Rate Limiting

Currently, no rate limiting is implemented. Consider implementing appropriate rate limiting for production deployments.

## Versioning

The API is currently unversioned. Future versions will use URL-based versioning (e.g., `/api/v2/`).