# MCP Server

The Rackd MCP (Model Context Protocol) server provides AI tools and automation systems with direct access to device inventory, network management, and datacenter operations through a standardized protocol.

## Overview

MCP is an open protocol that enables AI assistants and automation tools to interact with external systems through a standardized interface. Rackd's MCP server exposes all core functionality as tools that can be called by MCP clients like Claude Desktop, custom automation scripts, or other AI systems.

## Authentication

The MCP server supports optional Bearer token authentication:

```bash
# Start server with authentication
rackd server --mcp-token "your-secret-token"

# Client requests must include Authorization header
Authorization: Bearer your-secret-token
```

## Available Tools

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

#### datacenter_save
Create or update a datacenter.

**Parameters:**
- `id` (string, optional): Datacenter ID (omit for new)
- `name` (string, required): Datacenter name
- `location` (string): Physical location
- `description` (string): Description

### Network Management

#### network_list
List all networks.

**Parameters:**
- `datacenter_id` (string): Filter by datacenter

#### network_save
Create or update a network.

**Parameters:**
- `id` (string, optional): Network ID (omit for new)
- `name` (string, required): Network name
- `subnet` (string, required): CIDR subnet (e.g., 192.168.1.0/24)
- `datacenter_id` (string): Datacenter ID
- `vlan_id` (number): VLAN ID
- `description` (string): Description

### IP Pool Management

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

### Claude Desktop

Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "rackd": {
      "command": "rackd",
      "args": ["mcp"],
      "env": {
        "RACKD_MCP_TOKEN": "your-secret-token"
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

- Use strong, randomly generated bearer tokens
- Run MCP server on localhost or secure networks only
- Regularly rotate authentication tokens
- Monitor MCP access logs for suspicious activity
- Consider rate limiting for production deployments