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

API keys inherit the permissions of the user who created them.

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
| `POST /mcp-oauth/register` | Dynamic client registration |
| `GET /mcp-oauth/authorize` | Authorization endpoint (shows login/consent) |
| `POST /mcp-oauth/token` | Token endpoint (issues access/refresh tokens) |
| `POST /mcp-oauth/revoke` | Token revocation |

**Supported Grant Types:**
- `authorization_code` with PKCE (for public clients like Claude Desktop)
- `client_credentials` (for service-to-service)
- `refresh_token` (for token renewal)

**Backward Compatibility:**

When OAuth is enabled, both OAuth tokens and API keys are accepted. This allows a gradual migration.

## Available Tools

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
- `id` (string, optional): Device ID (omit for new device)
- `name` (string, required): Device name
- `description` (string): Device description
- `make_model` (string): Device make and model
- `os` (string): Operating system
- `datacenter_id` (string): Datacenter ID
- `username` (string): Login username
- `location` (string): Physical location
- `tags` (array): Device tags
- `addresses` (array): IP addresses with `ip` and `type` fields
- `domains` (array): Domain names

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

**Parameters:** None

#### datacenter_get
Get a datacenter by ID.

**Parameters:**
- `id` (string, required): Datacenter ID

#### datacenter_save
Create or update a datacenter.

**Parameters:**
- `id` (string, optional): Datacenter ID (omit for new)
- `name` (string, required): Datacenter name
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

#### network_get
Get a network by ID.

**Parameters:**
- `id` (string, required): Network ID

#### network_save
Create or update a network.

**Parameters:**
- `id` (string, optional): Network ID (omit for new)
- `name` (string, required): Network name
- `subnet` (string, required): CIDR subnet (e.g., 192.168.1.0/24)
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

## Integration Examples

### Claude Desktop (with OAuth)

When OAuth is enabled, Claude Desktop can authenticate via the standard MCP OAuth flow. Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "rackd": {
      "command": "rackd",
      "args": ["mcp"],
      "env": {
        "RACKD_URL": "http://localhost:8080"
      }
    }
  }
}
```

Claude Desktop will automatically:
1. Discover OAuth endpoints via `/.well-known/oauth-protected-resource`
2. Open a browser for you to log in to Rackd
3. Store and refresh tokens automatically

### Claude Desktop (with API Key)

For simpler setups without OAuth, use an API key:

```json
{
  "mcpServers": {
    "rackd": {
      "command": "rackd",
      "args": ["mcp"],
      "env": {
        "RACKD_API_KEY": "your-api-key"
      }
    }
  }
}
```

### Python Client

```python
import requests

def call_mcp_tool(tool_name, params):
    response = requests.post(
        "http://localhost:8080/mcp",
        headers={"Authorization": "Bearer your-secret-token"},
        json={
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

TOKEN="your-secret-token"
ENDPOINT="http://localhost:8080/mcp"

for i in {1..10}; do
  curl -X POST "$ENDPOINT" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
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

All tools return standardized error responses:

```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Device not found"
  }
}
```

Common error codes:
- `INVALID_PARAMS`: Invalid or missing parameters
- `INTERNAL_ERROR`: Server-side error
- `NOT_FOUND`: Resource not found
- `UNAUTHORIZED`: Authentication failed

## Security Considerations

- **OAuth (recommended)**: Users authenticate with their own credentials; tokens are scoped to their RBAC permissions
- **API Keys**: Use strong, randomly generated keys; keys inherit creator's permissions
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