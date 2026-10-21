# MCP Server

The Rackd MCP (Model Context Protocol) server provides AI tools and automation systems with direct access to device inventory, network management, and datacenter operations through a standardized protocol.

## Overview

MCP is an open protocol that enables AI assistants and automation tools to interact with external systems through a standardized interface. Rackd's MCP server exposes all core functionality as tools that can be called by MCP clients like Claude Desktop, custom automation scripts, or other AI systems.

## Authentication

Rackd supports two authentication methods for the MCP endpoint:

### API Key Authentication

The simplest method - create an API key via the web UI or CLI and use it as a Bearer token:

```bash
Authorization: Bearer <api-key>
```

An API key acts with the permissions of the user who created it, limited to its scopes if it has any.

### OAuth 2.1 Authentication (Recommended for MCP Clients)

For spec-compliant MCP clients (like Claude Desktop), Rackd implements OAuth 2.1 with PKCE. This allows users to authenticate via their existing Rackd account.

**Configuration:**

```bash
MCP_OAUTH_ENABLED=true                    # Enable OAuth for MCP (default: false)
MCP_OAUTH_ISSUER_URL=http://localhost:8080  # Base URL of your Rackd server
MCP_OAUTH_ACCESS_TOKEN_TTL=1h             # Access token lifetime (default: 1h)
MCP_OAUTH_REFRESH_TOKEN_TTL=720h          # Refresh token lifetime (default: 30 days)
```

**OAuth Flow:**

1. MCP client discovers endpoints via `GET /.well-known/oauth-protected-resource`
2. Client redirects user to `/mcp-oauth/authorize` with PKCE challenge
3. User logs in and approves access on the consent screen
4. Client exchanges authorization code for tokens at `/mcp-oauth/token`
5. Client uses access token as Bearer token for MCP requests

**OAuth Endpoints:**

| Endpoint | Description |
|----------|-------------|
| `GET /.well-known/oauth-protected-resource` | RFC 9728 Protected Resource Metadata |
| `GET /.well-known/oauth-authorization-server` | RFC 8414 Authorization Server Metadata |
| `POST /mcp-oauth/register` | Dynamic client registration (limits: `client_name` 200 characters, URIs 2,048, at most 10 `redirect_uris`) |
| `GET /mcp-oauth/authorize` | Authorization endpoint (shows login/consent) |
| `POST /mcp-oauth/authorize` | Consent submission |
| `POST /mcp-oauth/token` | Token endpoint (issues access/refresh tokens) |
| `POST /mcp-oauth/revoke` | Token revocation |

**Supported Grant Types:**
- `authorization_code` with PKCE (for public clients like Claude Desktop)
- `refresh_token` (for token renewal)

`client_credentials` is listed in the server metadata but isn't usable yet: it needs a client tied to a user, and dynamically registered clients never are. For service-to-service access, use an API key.

**Backward Compatibility:**

When OAuth is enabled, both OAuth tokens and API keys are accepted. This allows a gradual migration.

## Available Tools

`tools/list` returns only the core tools: `search`, `device_list`, `device_get`, `device_save`, `device_delete`, `datacenter_list`, `datacenter_get`, `network_list`, `network_get` and `pool_get_next_ip`, plus `tool_search` and `execute_tool`. The rest are discoverable: `tool_search` (optional `query`, `max_results`) finds them with their input schemas, and `execute_tool` (`name`, `parameters`) runs one. A client can also call a discoverable tool directly by name with `tools/call`. Extensions can add tools of their own.

List tools return `{"count": ..., "has_more": ..., "items": [...]}`.

### Search

#### search
Search across devices, networks, and datacenters using full-text search.

**Parameters:**
- `query` (string, required): Search query

**Returns:** Object with `devices`, `networks`, and `datacenters` arrays.

### Device Management

#### device_save
Create or update a device in the inventory.

**Parameters:**
- `id` (string, optional): Device ID (omit for new device). An update changes only the fields passed; passing `addresses`, `tags` or `domains` replaces that whole list
- `name` (string): Device name (required for a new device)
- `hostname` (string): Hostname
- `description` (string): Device description
- `make_model` (string): Device make and model
- `os` (string): Operating system
- `status` (string): `planned`, `active`, `maintenance` or `decommissioned`
- `datacenter_id` (string): Datacenter ID
- `username` (string): Login username
- `location` (string): Physical location
- `tags` (array): Device tags
- `addresses` (array): IP addresses with `ip`, `type`, and optional `network_id`, `pool_id`, `port`
- `domains` (array): Domain names
- `custom_fields` (array): Custom field values with `field_id` and `value`

**Example:**
```json
{
  "name": "web-server-01",
  "description": "Production web server",
  "make_model": "Dell PowerEdge R740",
  "os": "Ubuntu 22.04",
  "datacenter_id": "dc-east-1",
  "tags": ["production", "web"],
  "addresses": [
    {"ip": "192.168.1.100", "type": "ipv4"}
  ],
  "domains": ["web01.example.com"]
}
```

#### device_get
Retrieve a device by ID.

**Parameters:**
- `id` (string, required): Device ID

#### device_list
List devices with optional filtering.

**Parameters:**
- `query` (string): Search query
- `tags` (array): Filter by tags
- `datacenter_id` (string): Filter by datacenter
- `network_id` (string): Filter by network
- `pool_id` (string): Filter by IP pool
- `status` (string): Filter by status
- `limit` (number): Max results (default 100, max 1000)
- `offset` (number): Results to skip

#### device_delete
Delete a device from inventory.

**Parameters:**
- `id` (string, required): Device ID

### Device Relationships

#### device_add_relationship
Create a relationship between two devices.

**Parameters:**
- `parent_id` (string, required): Parent device ID
- `child_id` (string, required): Child device ID
- `type` (string, required): Relationship type: `contains`, `connected_to`, `depends_on`
- `notes` (string): Optional notes

**Example:**
```json
{
  "parent_id": "rack-01",
  "child_id": "server-01",
  "type": "contains"
}
```

#### device_get_relationships
Get all relationships for a device.

**Parameters:**
- `id` (string, required): Device ID

### Datacenter Management

#### datacenter_list
List all datacenters.

**Parameters:**
- `limit` (number): Max results (default 100, max 1000)
- `offset` (number): Results to skip

#### datacenter_get
Get a datacenter by ID.

**Parameters:**
- `id` (string, required): Datacenter ID

#### datacenter_save
Create or update a datacenter.

**Parameters:**
- `id` (string, optional): Datacenter ID (omit for new). An update changes only the fields passed
- `name` (string): Datacenter name (required for a new datacenter)
- `location` (string): Physical location
- `description` (string): Description

#### datacenter_delete
Delete a datacenter.

**Parameters:**
- `id` (string, required): Datacenter ID

### Network Management

#### network_list
List all networks.

**Parameters:**
- `datacenter_id` (string): Filter by datacenter
- `limit` (number): Max results (default 100, max 1000)
- `offset` (number): Results to skip

#### network_get
Get a network by ID.

**Parameters:**
- `id` (string, required): Network ID

#### network_save
Create or update a network.

**Parameters:**
- `id` (string, optional): Network ID (omit for new). An update changes only the fields passed
- `name` (string): Network name (required for a new network)
- `subnet` (string): CIDR subnet, e.g. 192.168.1.0/24 (required for a new network)
- `datacenter_id` (string): Datacenter ID
- `vlan_id` (number): VLAN ID
- `description` (string): Description

#### network_delete
Delete a network.

**Parameters:**
- `id` (string, required): Network ID

### IP Pool Management

#### pool_list
List IP pools for a network.

**Parameters:**
- `network_id` (string, required): Network ID

#### pool_get_next_ip
Get the next available IP address from a pool.

**Parameters:**
- `pool_id` (string, required): Pool ID

### Network Discovery

#### discovery_scan
Start a network discovery scan.

**Parameters:**
- `network_id` (string, required): Network ID to scan
- `scan_type` (string): Scan type: `quick`, `full`, `deep` (default: quick)

#### discovery_list
List discovered devices.

**Parameters:**
- `network_id` (string): Filter by network ID

#### discovery_promote
Promote a discovered device to inventory.

**Parameters:**
- `discovered_id` (string, required): Discovered device ID
- `name` (string, required): Device name for inventory

### Discoverable Tools

Found with `tool_search`; see each tool's schema there for its parameters.

| Tool | Purpose |
|------|---------|
| `device_get_custom_fields` | Get a device's custom field values with their definitions |
| `circuit_list`, `circuit_get`, `circuit_save`, `circuit_delete` | List, get, create or update, and delete circuits |
| `nat_list`, `nat_get`, `nat_save`, `nat_delete` | List, get, create or update, and delete NAT mappings |
| `reservation_list`, `reservation_get` | List and get IP reservations |
| `reservation_create`, `reservation_update` | Reserve an IP from a pool, or update a reservation |
| `reservation_release`, `reservation_delete` | Release a reservation back to the pool, or delete its record |
| `webhook_list`, `webhook_get`, `webhook_save`, `webhook_delete` | List, get, create or update, and delete webhooks |
| `webhook_ping` | Send a test ping to a webhook |
| `custom_field_list`, `custom_field_get`, `custom_field_save`, `custom_field_delete` | Manage custom field definitions |
| `conflict_list`, `conflict_detect`, `conflict_resolve` | List conflicts, run detection, resolve one |
| `audit_list` | List audit log entries |
| `dns_provider_list`, `dns_provider_get`, `dns_provider_save`, `dns_provider_delete`, `dns_provider_test` | Manage DNS providers and test a connection |
| `dns_zone_list`, `dns_zone_get`, `dns_zone_save`, `dns_zone_delete` | Manage DNS zones |
| `dns_zone_sync`, `dns_zone_import` | Sync a zone to its provider, or import records from it |
| `dns_record_list`, `dns_record_get`, `dns_record_save`, `dns_record_delete` | Manage DNS records in a zone |
| `dns_record_link` | Link a DNS record to a device |

The DNS tools are present only when `ENCRYPTION_KEY` is set.

## Integration Examples

### Connecting a Client

Rackd serves MCP over Streamable HTTP at `<server URL>/mcp`, for example `https://rackd.example.com/mcp`. There is no separate MCP command to run.

**With OAuth** (`MCP_OAUTH_ENABLED=true`), give that URL to a client that supports MCP OAuth, such as a custom connector in Claude. The client discovers the endpoints through `/.well-known/oauth-protected-resource`, opens a browser for you to sign in and approve it, and refreshes its tokens.

**With an API key**, send it as a bearer token. In Claude Code:

```bash
claude mcp add --transport http rackd https://rackd.example.com/mcp \
  --header "Authorization: Bearer <api key>"
```

A client that can only launch local commands can use a bridge such as [`mcp-remote`](https://www.npmjs.com/package/mcp-remote):

```json
{
  "mcpServers": {
    "rackd": {
      "command": "npx",
      "args": ["mcp-remote", "https://rackd.example.com/mcp", "--header", "Authorization: Bearer <api key>"]
    }
  }
}
```

The key acts with its owner's permissions, limited to its scopes if it has any.

### Python Client

```python
import requests

def call_mcp_tool(tool_name, params):
    response = requests.post(
        "http://localhost:8080/mcp",
        headers={"Authorization": "Bearer <api-key>"},
        json={
            "jsonrpc": "2.0",
            "id": 1,
            "method": "tools/call",
            "params": {
                "name": tool_name,
                "arguments": params
            }
        }
    )
    return response.json()

# Create a new device
device = call_mcp_tool("device_save", {
    "name": "db-server-01",
    "description": "Database server",
    "tags": ["database", "production"]
})

# List all devices
devices = call_mcp_tool("device_list", {})
```

### Automation Scripts

```bash
#!/bin/bash
# Bulk device creation via MCP

ENDPOINT="http://localhost:8080/mcp"

for i in {1..10}; do
  curl -X POST "$ENDPOINT" \
    -H "Authorization: Bearer $RACKD_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
      \"jsonrpc\": \"2.0\",
      \"id\": $i,
      \"method\": \"tools/call\",
      \"params\": {
        \"name\": \"device_save\",
        \"arguments\": {
          \"name\": \"server-$(printf %02d $i)\",
          \"description\": \"Auto-generated server $i\",
          \"tags\": [\"auto-generated\"]
        }
      }
    }"
done
```

## AI Assistant Integration

The MCP server enables natural language interaction with your infrastructure:

**Example prompts:**
- "Show me all production web servers"
- "Create a new database server in the east datacenter"
- "What devices are connected to switch-01?"
- "Scan the 192.168.1.0/24 network for new devices"
- "Get the next available IP from the production pool"

The AI assistant will automatically translate these requests into appropriate MCP tool calls and present the results in a human-readable format.

## Error Handling

A failed tool call returns a JSON-RPC error:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32004,
    "message": "not found"
  }
}
```

Error codes:
- `-32602`: Invalid or missing parameters
- `-32603`: Internal error (details are in the server log, not the response)
- `-32001`: Not authenticated
- `-32003`: Forbidden (the user or token lacks the permission)
- `-32004`: Not found
- `-32009`: Conflict (already exists, no free IPs, last admin, scan limits)
- `-32010`: An outside service (DNS provider, webhook target) failed

A request without a valid token gets HTTP `401` before any JSON-RPC handling, with a `WWW-Authenticate` header that points OAuth clients to `/.well-known/oauth-protected-resource`.

## Security Considerations

- **OAuth (recommended)**: Users authenticate with their own credentials; tokens are scoped to their RBAC permissions
- **API Keys**: Keys act with their creator's permissions; give them scopes to limit them further
- Run MCP server on localhost or secure networks only
- Use HTTPS in production (set `MCP_OAUTH_ISSUER_URL` to your HTTPS URL)
- Regularly rotate API keys and review OAuth client registrations
- Monitor MCP access logs for suspicious activity
- Consider rate limiting for production deployments

## Managing OAuth Clients

Registered OAuth clients can be viewed and revoked in the web UI at `/oauth-clients`. Each client shows:
- Client name and ID
- Redirect URIs
- Client type (public/confidential)
- Registration date

Deleting a client revokes all its active tokens.