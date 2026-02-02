# CLI Reference

Complete command-line interface reference for Rackd.

## Overview

The Rackd CLI provides full access to all functionality via command-line commands. It communicates with the Rackd server via the REST API.

## Global Options

```bash
rackd [global options] command [command options] [arguments...]
```

### Global Flags

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--api-url` | `RACKD_API_URL` | `http://localhost:8080` | API server URL |
| `--api-token` | `RACKD_API_TOKEN` | - | API authentication token |
| `--timeout` | `RACKD_TIMEOUT` | `30s` | Request timeout |
| `--output` | `RACKD_OUTPUT` | `table` | Output format (table, json, yaml) |
| `--help, -h` | - | - | Show help |
| `--version, -v` | - | - | Show version |

### Configuration File

The CLI reads configuration from `~/.rackd/config.yaml`:

```yaml
api_url: http://localhost:8080
api_token: your-secret-token
timeout: 30s
output: table
```

## Commands

### server

Start the Rackd HTTP/MCP server.

```bash
rackd server [options]
```

#### Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--listen-addr` | `RACKD_LISTEN_ADDR` | `:8080` | Listen address |
| `--data-dir` | `RACKD_DATA_DIR` | `./data` | Data directory |
| `--api-auth-token` | `RACKD_API_AUTH_TOKEN` | - | API auth token |
| `--mcp-auth-token` | `RACKD_MCP_AUTH_TOKEN` | - | MCP auth token |
| `--log-level` | `RACKD_LOG_LEVEL` | `info` | Log level |
| `--log-format` | `RACKD_LOG_FORMAT` | `text` | Log format (text/json) |
| `--discovery-enabled` | `RACKD_DISCOVERY_ENABLED` | `true` | Enable discovery |
| `--discovery-interval` | `RACKD_DISCOVERY_INTERVAL` | `24h` | Discovery interval |
| `--encryption-key` | `RACKD_ENCRYPTION_KEY` | - | Credential encryption key |

#### Examples

```bash
# Start with defaults
rackd server

# Start with custom port
rackd server --listen-addr :9000

# Start with authentication
rackd server --api-auth-token mysecret

# Start with custom data directory
rackd server --data-dir /var/lib/rackd

# Start with debug logging
rackd server --log-level debug --log-format json
```

### device

Manage devices in the inventory.

#### device list

List all devices.

```bash
rackd device list [options]
```

**Options:**
- `--datacenter <id>` - Filter by datacenter ID
- `--tags <tag1,tag2>` - Filter by tags
- `--network <id>` - Filter by network ID
- `--output <format>` - Output format (table, json, yaml)

**Examples:**

```bash
# List all devices
rackd device list

# List devices in a datacenter
rackd device list --datacenter dc1

# List devices with specific tags
rackd device list --tags production,web

# Output as JSON
rackd device list --output json
```

#### device get

Get device details.

```bash
rackd device get <id> [options]
```

**Examples:**

```bash
# Get device by ID
rackd device get dev-123

# Output as JSON
rackd device get dev-123 --output json
```

#### device add

Add a new device.

```bash
rackd device add [options]
```

**Options:**
- `--name <name>` - Device name (required)
- `--description <desc>` - Description
- `--make-model <model>` - Make and model
- `--os <os>` - Operating system
- `--datacenter <id>` - Datacenter ID
- `--username <user>` - Login username
- `--location <loc>` - Physical location
- `--tags <tag1,tag2>` - Tags
- `--domains <domain1,domain2>` - Domain names
- `--addresses <type:ip,type:ip>` - IP addresses

**Examples:**

```bash
# Add a basic device
rackd device add --name web-01 --description "Web server"

# Add device with full details
rackd device add \
  --name web-01 \
  --description "Production web server" \
  --make-model "Dell PowerEdge R640" \
  --os "Ubuntu 22.04" \
  --datacenter dc1 \
  --username admin \
  --location "Rack 5, U10" \
  --tags production,web \
  --domains web-01.example.com,www.example.com \
  --addresses management:10.0.1.10,primary:192.168.1.10

# Add device with JSON input
cat device.json | rackd device add --from-stdin
```

#### device update

Update an existing device.

```bash
rackd device update <id> [options]
```

**Options:** Same as `device add`

**Examples:**

```bash
# Update device name
rackd device update dev-123 --name web-02

# Update multiple fields
rackd device update dev-123 \
  --description "Updated description" \
  --tags production,web,updated
```

#### device delete

Delete a device.

```bash
rackd device delete <id>
```

**Examples:**

```bash
# Delete device
rackd device delete dev-123

# Delete with confirmation
rackd device delete dev-123 --confirm
```

### network

Manage networks and IP address pools.

#### network list

List all networks.

```bash
rackd network list [options]
```

**Options:**
- `--datacenter <id>` - Filter by datacenter
- `--vlan <vlan>` - Filter by VLAN
- `--name <name>` - Filter by name

**Examples:**

```bash
# List all networks
rackd network list

# List networks in datacenter
rackd network list --datacenter dc1

# List networks by VLAN
rackd network list --vlan 100
```

#### network get

Get network details.

```bash
rackd network get <id>
```

#### network add

Add a new network.

```bash
rackd network add [options]
```

**Options:**
- `--name <name>` - Network name (required)
- `--cidr <cidr>` - CIDR notation (required)
- `--vlan <vlan>` - VLAN ID
- `--datacenter <id>` - Datacenter ID
- `--gateway <ip>` - Gateway IP
- `--description <desc>` - Description

**Examples:**

```bash
# Add basic network
rackd network add --name prod-net --cidr 10.0.1.0/24

# Add network with full details
rackd network add \
  --name prod-net \
  --cidr 10.0.1.0/24 \
  --vlan 100 \
  --datacenter dc1 \
  --gateway 10.0.1.1 \
  --description "Production network"
```

#### network update

Update a network.

```bash
rackd network update <id> [options]
```

#### network delete

Delete a network.

```bash
rackd network delete <id>
```

#### network pool

Manage IP address pools within networks.

##### pool list

List pools for a network.

```bash
rackd network pool list <network-id>
```

##### pool add

Add a pool to a network.

```bash
rackd network pool add <network-id> [options]
```

**Options:**
- `--name <name>` - Pool name (required)
- `--start-ip <ip>` - Start IP (required)
- `--end-ip <ip>` - End IP (required)
- `--description <desc>` - Description
- `--tags <tag1,tag2>` - Tags

**Examples:**

```bash
# Add IP pool
rackd network pool add net-123 \
  --name dhcp-pool \
  --start-ip 10.0.1.100 \
  --end-ip 10.0.1.200 \
  --description "DHCP pool" \
  --tags dhcp,dynamic
```

### datacenter

Manage datacenters.

#### datacenter list

List all datacenters.

```bash
rackd datacenter list [options]
```

**Options:**
- `--name <name>` - Filter by name

#### datacenter get

Get datacenter details.

```bash
rackd datacenter get <id>
```

#### datacenter add

Add a new datacenter.

```bash
rackd datacenter add [options]
```

**Options:**
- `--name <name>` - Datacenter name (required)
- `--location <location>` - Physical location
- `--description <desc>` - Description

**Examples:**

```bash
# Add datacenter
rackd datacenter add \
  --name dc1 \
  --location "New York, NY" \
  --description "Primary datacenter"
```

#### datacenter update

Update a datacenter.

```bash
rackd datacenter update <id> [options]
```

#### datacenter delete

Delete a datacenter.

```bash
rackd datacenter delete <id>
```

### discovery

Network discovery and scanning.

#### discovery scan

Start a network discovery scan.

```bash
rackd discovery scan <network-cidr> [options]
```

**Options:**
- `--type <type>` - Scan type (basic, advanced) [default: basic]
- `--profile <id>` - Scan profile ID
- `--wait` - Wait for scan to complete

**Examples:**

```bash
# Start basic scan
rackd discovery scan 10.0.1.0/24

# Start advanced scan with profile
rackd discovery scan 10.0.1.0/24 --type advanced --profile prof-123

# Start scan and wait for completion
rackd discovery scan 10.0.1.0/24 --wait
```

#### discovery list

List discovered devices.

```bash
rackd discovery list [options]
```

**Options:**
- `--network <id>` - Filter by network ID
- `--scan <id>` - Filter by scan ID

**Examples:**

```bash
# List all discovered devices
rackd discovery list

# List devices from specific scan
rackd discovery list --scan scan-123
```

#### discovery promote

Promote a discovered device to inventory.

```bash
rackd discovery promote <discovered-device-id> [options]
```

**Options:**
- `--name <name>` - Device name
- `--datacenter <id>` - Datacenter ID
- `--tags <tag1,tag2>` - Tags

**Examples:**

```bash
# Promote device
rackd discovery promote disc-123 \
  --name web-server-01 \
  --datacenter dc1 \
  --tags production,web
```

### version

Show version information.

```bash
rackd version
```

**Output:**

```
Version: 1.0.0
Commit: abc123
Built: 2024-01-20T10:30:00Z
```

## Output Formats

### Table (Default)

Human-readable table format:

```bash
rackd device list
```

```
ID       NAME      DATACENTER  IP ADDRESSES      TAGS
dev-001  web-01    dc1         10.0.1.10         production,web
dev-002  db-01     dc1         10.0.1.20         production,database
```

### JSON

Machine-readable JSON:

```bash
rackd device list --output json
```

```json
[
  {
    "id": "dev-001",
    "name": "web-01",
    "datacenter_id": "dc1",
    "addresses": [
      {"type": "primary", "address": "10.0.1.10"}
    ],
    "tags": ["production", "web"]
  }
]
```

### YAML

YAML format:

```bash
rackd device list --output yaml
```

```yaml
- id: dev-001
  name: web-01
  datacenter_id: dc1
  addresses:
    - type: primary
      address: 10.0.1.10
  tags:
    - production
    - web
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments |
| 3 | API error |
| 4 | Authentication error |
| 5 | Not found |

## Environment Variables

All CLI options can be set via environment variables:

```bash
export RACKD_API_URL=http://localhost:8080
export RACKD_API_TOKEN=mysecret
export RACKD_OUTPUT=json

rackd device list
```

## Shell Completion

Generate shell completion scripts:

```bash
# Bash
rackd completion bash > /etc/bash_completion.d/rackd

# Zsh
rackd completion zsh > /usr/local/share/zsh/site-functions/_rackd

# Fish
rackd completion fish > ~/.config/fish/completions/rackd.fish
```

## Examples

### Complete Workflow

```bash
# 1. Start server
rackd server --api-auth-token mysecret &

# 2. Configure CLI
export RACKD_API_TOKEN=mysecret

# 3. Add datacenter
rackd datacenter add --name dc1 --location "New York"

# 4. Add network
rackd network add --name prod-net --cidr 10.0.1.0/24 --datacenter dc1

# 5. Add device
rackd device add \
  --name web-01 \
  --datacenter dc1 \
  --addresses primary:10.0.1.10

# 6. Run discovery
rackd discovery scan 10.0.1.0/24

# 7. List discovered devices
rackd discovery list

# 8. Promote discovered device
rackd discovery promote disc-123 --name db-01
```

## Related Documentation

- [API Reference](api.md) - REST API documentation
- [Configuration](configuration.md) - Configuration options
- [Quick Start](quickstart.md) - Getting started guide
