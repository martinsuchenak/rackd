# API Reference

The Rackd REST API provides programmatic access to all infrastructure management features including devices, networks, datacenters, IP pools, discovery, and relationships.

## Base URL

```
http://localhost:8080/api/v1
```

Health checks (`/healthz`, `/readyz`), metrics (`/metrics`), MCP (`/mcp`) and the OAuth endpoints (`/mcp-oauth/*`, `/.well-known/*`) are not versioned.

This page covers the core inventory endpoints. Every endpoint, including users, roles, API keys, audit and application logs, conflicts, reservations, webhooks, custom fields, circuits, NAT, DNS, credentials, scan profiles, scheduled scans, bulk operations, search and the dashboard, is described in [`api/openapi.yaml`](../api/openapi.yaml).

## Versioning and Deprecation

The API version is part of the path. Within `/api/v1`, changes are backward compatible:

- Rackd may add endpoints, optional request fields, query parameters and response fields.
- Rackd may add enum values and error codes. Clients should ignore unknown response fields and handle unknown enum values.

A breaking change goes into a new version, `/api/v2`. Breaking means removing or renaming an endpoint or field, changing a field's type, making an optional field required, or changing what a status code means.

When something is deprecated:

- It is listed under **Deprecated** in the [changelog](../CHANGELOG.md) and marked `deprecated: true` in [`api/openapi.yaml`](../api/openapi.yaml).
- Its responses carry a `Deprecation` header ([RFC 9745](https://www.rfc-editor.org/rfc/rfc9745)) and, once a removal date is set, a `Sunset` header ([RFC 8594](https://www.rfc-editor.org/rfc/rfc8594)). A `Link` header with `rel="successor-version"` points to the replacement.
- It keeps working for at least six months and two minor releases after the release that deprecates it.

When `/api/v2` ships, `/api/v1` stays supported for at least twelve months.

### Unversioned Paths

Releases up to v0.1.0 served the API under `/api/...`, without a version. Those paths still work as aliases of `/api/v1/...`, but they are deprecated and are removed in the first release on or after 14 April 2027. To migrate, insert `/v1` after `/api`.

Responses on the old paths carry these headers:

```http
Deprecation: @1791936000
Sunset: Wed, 14 Apr 2027 00:00:00 GMT
Link: </api/v1/devices>; rel="successor-version"
```

`@1791936000` is 14 October 2026, when the unversioned paths were deprecated.

## Authentication

Every endpoint requires authentication. The web UI uses a session cookie; scripts and integrations send an API key as a Bearer token:

```http
Authorization: Bearer <api-key>
```

See [Authentication](authentication.md) for creating API keys. The examples below assume the key is in `$RACKD_TOKEN`.

## Content Type

All API endpoints accept and return JSON data:

```http
Content-Type: application/json
```

## IDs

Rackd assigns IDs (UUIDs) when it creates something. A create request may include its own `id`, for example to keep IDs when importing, but it must be a UUID such as `0190f3c2-7d3e-7a6b-9c1d-2e3f4a5b6c7d`; anything else is a `400 VALIDATION_ERROR` on `id`.

## Error Handling

### Error Response Format

All errors return a consistent JSON structure:

```json
{
  "code": "ERROR_CODE",
  "error": "Human-readable error message"
}
```

Validation errors add `details`, a list of the fields that failed (see below).

### HTTP Status Codes

- `200` - Success
- `201` - Created
- `204` - No Content (successful deletion)
- `400` - Bad Request (validation errors, invalid JSON)
- `401` - Unauthorized (missing or invalid credentials)
- `403` - Forbidden (authenticated, but the role lacks the permission)
- `404` - Not Found (unknown resource, or unknown endpoint)
- `405` - Method Not Allowed (the endpoint exists; the `Allow` header lists its methods)
- `409` - Conflict (a record with that name already exists, no free IPs, or the last admin)
- `415` - Unsupported Media Type (login without `Content-Type: application/json`)
- `429` - Too Many Requests (rate limited, too many failed sign-ins, or too many scans running)
- `500` - Internal Server Error
- `502` - Bad Gateway (a DNS provider or webhook target couldn't be reached or refused the request)

### Error Codes

| Code | Status | Meaning |
|------|--------|---------|
| `VALIDATION_ERROR` | 400 | Fields failed validation; `details` lists them |
| `INVALID_JSON` | 400 | The body isn't valid JSON |
| `INVALID_INPUT` | 400 | A parameter or the request is malformed |
| `SYSTEM_ROLE` | 400 | System roles can't be changed this way |
| `CANNOT_DELETE_SELF` | 400 | Users can't delete their own account |
| `UNAUTHORIZED` | 401 | Missing or invalid credentials, or the session ended |
| `INVALID_CREDENTIALS` | 401 | Wrong username or password |
| `EXPIRED_KEY` | 401 | The API key has expired |
| `LEGACY_API_KEY_UNSUPPORTED` | 401 | The API key belongs to no user; create a new one |
| `FORBIDDEN` | 403 | The caller lacks the permission; for delegated user and role management the message says what is missing |
| `CSRF_FAILED` | 403 | A session request that changes something lacks `X-Requested-With: XMLHttpRequest` |
| `NOT_FOUND` | 404 | No such endpoint, or the resource doesn't exist |
| `METHOD_NOT_ALLOWED` | 405 | The endpoint doesn't accept this method |
| `ALREADY_EXISTS` | 409 | A record with that name (or other unique field) already exists |
| `IP_NOT_AVAILABLE` | 409 | The pool has no free addresses |
| `LAST_ADMIN` | 409 | The change would leave no active admin |
| `UNSUPPORTED_MEDIA_TYPE` | 415 | Login requires `Content-Type: application/json` |
| `RATE_LIMIT_EXCEEDED`, `LOGIN_RATE_LIMIT_EXCEEDED` | 429 | Too many requests from this address |
| `TOO_MANY_ATTEMPTS` | 429 | Too many failed sign-ins for this account |
| `TOO_MANY_SCANS` | 429 | The maximum number of discovery scans is running |
| `INTERNAL_ERROR` | 500 | Server error; details are in the server log, not the response |
| `PROVIDER_UNREACHABLE` | 502 | A DNS provider test or sync failed |
| `WEBHOOK_UNREACHABLE` | 502 | A webhook ping couldn't be delivered |

### Validation Errors

Validation errors list each field that failed:

```json
{
  "code": "VALIDATION_ERROR",
  "error": "name: name is required; subnet: invalid CIDR notation",
  "details": [
    {"field": "name", "message": "name is required"},
    {"field": "subnet", "message": "invalid CIDR notation"}
  ]
}
```

Fields in bulk requests are prefixed with the item, such as `devices[3].name`.

## Pagination

List endpoints return every result unless you ask for a page:

| Parameter | Meaning |
| --- | --- |
| `limit` | Results per page, at most 1000 |
| `offset` | Results to skip |

Device lists (`GET /api/v1/devices`), audit logs, application logs and webhook deliveries can grow large, so they always return a page: 100 results unless `limit` says otherwise.

## Filtering

Many list endpoints support query parameters for filtering results. Common filters include:

- `name` - Filter by name (partial match)
- `tags` - Filter devices by tags (exact match; repeat the parameter or separate values with commas)
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
GET /api/v1/datacenters
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
POST /api/v1/datacenters
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
GET /api/v1/datacenters/{id}
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
PUT /api/v1/datacenters/{id}
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
DELETE /api/v1/datacenters/{id}
```

**Response:** `204 No Content`

### List Datacenter Devices

```http
GET /api/v1/datacenters/{id}/devices
```

**Response:** `200 OK` (returns array of devices)

## Networks

### List Networks

```http
GET /api/v1/networks
```

**Query Parameters:**
- `name` (optional) - Filter by name
- `datacenter_id` (optional) - Filter by datacenter
- `vlan_id` (optional) - Filter by VLAN ID

**Response:** `200 OK` (returns array of networks)

### Create Network

```http
POST /api/v1/networks
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
GET /api/v1/networks/{id}
```

**Response:** `200 OK` (returns network details)

### Update Network

```http
PUT /api/v1/networks/{id}
```

**Request Body:** (same as create, all fields optional)

**Response:** `200 OK` (returns updated network)

### Delete Network

```http
DELETE /api/v1/networks/{id}
```

**Response:** `204 No Content`

### List Network Devices

```http
GET /api/v1/networks/{id}/devices
```

**Response:** `200 OK` (returns array of devices in network)

### Get Network Utilization

```http
GET /api/v1/networks/{id}/utilization
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
GET /api/v1/networks/{id}/pools
```

**Response:** `200 OK` (returns array of pools)

### Create Network Pool

```http
POST /api/v1/networks/{id}/pools
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
GET /api/v1/pools/{id}
```

**Response:** `200 OK` (returns pool details)

### Update Pool

```http
PUT /api/v1/pools/{id}
```

**Request Body:** (same as create, all fields optional)

**Response:** `200 OK` (returns updated pool)

### Delete Pool

```http
DELETE /api/v1/pools/{id}
```

**Response:** `204 No Content`

### Get Next Available IP

```http
GET /api/v1/pools/{id}/next-ip
```

Returns the first address in the pool's range that no device holds and no
unexpired reservation sets aside.

**Response:** `200 OK`
```json
{
  "ip": "192.168.1.15"
}
```

**Error:** `409 Conflict` if no IPs available

### Get Pool Heatmap

```http
GET /api/v1/pools/{id}/heatmap
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
GET /api/v1/devices
```

**Query Parameters:**
- `tags` (optional) - Filter by tags (multiple values supported)
- `datacenter_id` (optional) - Filter by datacenter
- `network_id` (optional) - Filter by network
- `pool_id` (optional) - Filter by IP pool
- `status` (optional) - `planned`, `active`, `maintenance` or `decommissioned`
- `stale_days` (optional) - Only active devices discovery hasn't seen for that many days; `stale=true` alone means 7

**Response:** `200 OK` (returns array of devices)

### Create Device

```http
POST /api/v1/devices
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
GET /api/v1/devices/{id}
```

**Response:** `200 OK` (returns device details)

### Update Device

```http
PUT /api/v1/devices/{id}
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
DELETE /api/v1/devices/{id}
```

**Response:** `204 No Content`

### Search Devices

```http
GET /api/v1/devices/search?q=web
```

**Query Parameters:**
- `q` (required) - Search query (max 256 characters)

Searches across device names, descriptions, IP addresses, tags, and domains.

**Response:** `200 OK` (returns array of matching devices)

## Device Relationships

### Get Device Relationships

```http
GET /api/v1/devices/{id}/relationships
```

**Response:** `200 OK`
```json
[
  {
    "parent_id": "parent-uuid",
    "child_id": "child-uuid", 
    "type": "contains",
    "notes": "",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

### Add Device Relationship

```http
POST /api/v1/devices/{id}/relationships
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
GET /api/v1/devices/{id}/related
```

**Query Parameters:**
- `type` (optional) - Filter by relationship type

**Response:** `200 OK` (returns array of related devices)

### Remove Device Relationship

```http
DELETE /api/v1/devices/{id}/relationships/{child_id}/{type}
```

**Response:** `204 No Content`

## Discovery

### Start Network Scan

```http
POST /api/v1/discovery/networks/{id}/scan
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
GET /api/v1/discovery/scans
```

**Query Parameters:**
- `network_id` (optional) - Filter by network

**Response:** `200 OK` (returns array of scans)

### Get Discovery Scan

```http
GET /api/v1/discovery/scans/{id}
```

**Response:** `200 OK` (returns scan details with progress)

### Cancel Discovery Scan

```http
POST /api/v1/discovery/scans/{id}/cancel
```

**Response:** `200 OK` with the scan, now `failed` with `error_message` `scan cancelled`

### Delete Discovery Scan

```http
DELETE /api/v1/discovery/scans/{id}
```

**Response:** `204 No Content`

### List Discovered Devices

```http
GET /api/v1/discovery/devices
```

**Query Parameters:**
- `network_id` (optional) - Filter by network

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
DELETE /api/v1/discovery/devices/{id}
```

**Response:** `204 No Content`

### Delete All Discovered Devices by Network

```http
DELETE /api/v1/discovery/devices?network_id={network_id}
```

**Response:** `204 No Content`

### Promote Discovered Device

```http
POST /api/v1/discovery/devices/{id}/promote
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
GET /api/v1/discovery/rules
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
POST /api/v1/discovery/rules
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
GET /api/v1/discovery/rules/{id}
```

**Response:** `200 OK` (returns rule details)

### Update Discovery Rule

```http
PUT /api/v1/discovery/rules/{id}
```

**Request Body:** (same as create, all fields optional)

**Response:** `200 OK` (returns updated rule)

### Delete Discovery Rule

```http
DELETE /api/v1/discovery/rules/{id}
```

**Response:** `204 No Content`

## Examples

### Complete Device Creation Workflow

1. **Create a datacenter:**
```bash
curl -X POST http://localhost:8080/api/v1/datacenters \
  -H "Authorization: Bearer $RACKD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Primary DC",
    "location": "New York",
    "description": "Primary datacenter"
  }'
```

2. **Create a network:**
```bash
curl -X POST http://localhost:8080/api/v1/networks \
  -H "Authorization: Bearer $RACKD_TOKEN" \
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
curl -X POST http://localhost:8080/api/v1/networks/net1-uuid/pools \
  -H "Authorization: Bearer $RACKD_TOKEN" \
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
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer $RACKD_TOKEN" \
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
curl -X POST http://localhost:8080/api/v1/discovery/networks/net1-uuid/scan \
  -H "Authorization: Bearer $RACKD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"scan_type": "full"}'
```

2. **Check scan progress:**
```bash
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/discovery/scans/scan-uuid
```

3. **List discovered devices:**
```bash
curl -H "Authorization: Bearer $RACKD_TOKEN" "http://localhost:8080/api/v1/discovery/devices?network_id=net1-uuid"
```

4. **Promote a discovered device:**
```bash
curl -X POST http://localhost:8080/api/v1/discovery/devices/discovered-uuid/promote \
  -H "Authorization: Bearer $RACKD_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "discovered-server",
    "datacenter_id": "dc1-uuid"
  }'
```

## Rate Limiting

Rate limiting is **enabled by default** to protect against API abuse:

- **General API**: 100 requests per minute per client (configurable)
- **Login endpoints**: 5 requests per minute per IP (configurable)

Rate limit headers are included in all responses:
- `X-RateLimit-Limit`: Maximum requests per window
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Time when the window resets

When rate limited, the API returns HTTP 429 with a `Retry-After` header.

See [Rate Limiting](ratelimit.md) for detailed configuration options.
