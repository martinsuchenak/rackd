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

### user

Manage users.

#### user list

List all users.

```bash
rackd user list [options]
```

**Options:**
- `--username <username>` - Filter by username
- `--email <email>` - Filter by email

**Examples:**

```bash
# List all users
rackd user list

# Filter by username
rackd user list --username admin
```

#### user create

Create a new user.

```bash
rackd user create [options]
```

**Options:**
- `--username <username>` - Username (required)
- `--email <email>` - Email address (required)
- `--full-name <name>` - Full name
- `--admin` - Make user an admin
- `--role-id <role-id>` - Assign a role ID to the new user (repeatable)

**Examples:**

```bash
# Create a user
rackd user create --username jsmith --email john@example.com --full-name "John Smith"

# Create an admin user
rackd user create --username admin --email admin@example.com --admin

# Create a user and assign RBAC roles
rackd user create --username operator --email op@example.com --role-id <role-id-1> --role-id <role-id-2>
```

#### user update

Update a user.

```bash
rackd user update --id <user-id> [options]
```

**Options:**
- `--id <id>` - User ID (required)
- `--username <username>` - Username
- `--email <email>` - Email address
- `--full-name <name>` - Full name
- `--active` - Set user active
- `--inactive` - Set user inactive
- `--admin` - Grant admin status
- `--not-admin` - Remove admin status
- `--add-role-id <role-id>` - Assign a role ID to the user (repeatable)
- `--remove-role-id <role-id>` - Revoke a role ID from the user (repeatable)

**Examples:**

```bash
# Update user email
rackd user update --id user-123 --email newemail@example.com

# Deactivate a user
rackd user update --id user-123 --inactive

# Rename a user and update RBAC role assignments
rackd user update --id user-123 --username jsmith --add-role-id <role-id> --remove-role-id <old-role-id>
```

#### user delete

Delete a user.

```bash
rackd user delete --id <user-id>
```

#### user password

Change a user's password.

```bash
rackd user password --id <user-id>
```

You will be prompted for the old and new passwords.

### role

Manage roles and permissions.

#### role list

List roles.

```bash
rackd role list [options]
```

**Options:**
- `--name <name>` - Filter by name

#### role permissions

List available permissions.

```bash
rackd role permissions [options]
```

**Options:**
- `--resource <resource>` - Filter by resource
- `--action <action>` - Filter by action

#### role create

Create a new role.

```bash
rackd role create [options]
```

**Options:**
- `--name <name>` - Role name (required)
- `--description <desc>` - Role description

**Examples:**

```bash
rackd role create --name "Network Admin" --description "Full network management"
```

#### role delete

Delete a role.

```bash
rackd role delete --id <role-id>
```

#### role assign

Assign a role to a user.

```bash
rackd role assign --user-id <user-id> --role-id <role-id>
```

#### role revoke

Revoke a role from a user.

```bash
rackd role revoke --user-id <user-id> --role-id <role-id>
```

### apikey

Manage API keys.

#### apikey list

List API keys.

```bash
rackd apikey list
```

#### apikey create

Create a new API key.

```bash
rackd apikey create [options]
```

**Options:**
- `--name <name>` - API key name (required)
- `--description <desc>` - Description
- `--expires <date>` - Expiration date (YYYY-MM-DD)

**Examples:**

```bash
# Create a basic API key
rackd apikey create --name "CI/CD Pipeline"

# Create with expiration
rackd apikey create --name "Temp Key" --expires 2024-12-31
```

**Output:**

```
API Key created successfully!

ID:   key-123
Name: CI/CD Pipeline
Key:  rak_live_xxxxxxxxxxxx

⚠️  Save this key securely - it will not be shown again!
```

#### apikey delete

Delete an API key.

```bash
rackd apikey delete --id <key-id>
```

#### apikey generate

Generate a random API key (offline, for manual use).

```bash
rackd apikey generate
```

### credentials

Manage credentials and encryption.

#### credentials rotate-key

Rotate the master encryption key used for credentials.

```bash
rackd credentials rotate-key [options]
```

**Options:**
- `--data-dir <dir>` - Data directory (default: ./data)
- `--new-key <key>` - New 32-byte hex-encoded encryption key (required)

**Examples:**

```bash
# Generate a new key
openssl rand -hex 32

# Rotate the key (requires ENCRYPTION_KEY env var to be set)
rackd credentials rotate-key --new-key <new-64-char-hex-string>
```

### dns

DNS management commands.

#### dns provider

Manage DNS providers.

##### dns provider list

List DNS providers.

```bash
rackd dns provider list [options]
```

**Options:**
- `--type <type>` - Filter by provider type (technitium, powerdns, bind)
- `--output <format>` - Output format (table/json/yaml)

##### dns provider create

Create a DNS provider.

```bash
rackd dns provider create [options]
```

**Options:**
- `--name <name>` - Provider name (required)
- `--type <type>` - Provider type: technitium, powerdns, bind (required)
- `--endpoint <url>` - API endpoint URL
- `--token-env <var>` - Environment variable containing API token
- `--token-file <path>` - File containing API token
- `--description <desc>` - Description

**Examples:**

```bash
# Create Technitium provider
rackd dns provider create \
  --name "Main DNS" \
  --type technitium \
  --endpoint http://dns.example.com:5380 \
  --token-env TECHNITIUM_TOKEN

# Create PowerDNS provider
rackd dns provider create \
  --name "PowerDNS" \
  --type powerdns \
  --endpoint http://pdns.example.com:8081 \
  --token-env POWERDNS_API_KEY
```

##### dns provider test

Test DNS provider connection.

```bash
rackd dns provider test --id <provider-id>
```

##### dns provider update

Update a DNS provider.

```bash
rackd dns provider update --id <id> [options]
```

##### dns provider delete

Delete a DNS provider.

```bash
rackd dns provider delete --id <id> [--force]
```

#### dns zone

Manage DNS zones.

##### dns zone list

List DNS zones.

```bash
rackd dns zone list [options]
```

**Options:**
- `--provider <id>` - Filter by provider ID
- `--network <id>` - Filter by network ID
- `--output <format>` - Output format (table/json/yaml)

##### dns zone create

Create a DNS zone.

```bash
rackd dns zone create [options]
```

**Options:**
- `--name <name>` - Zone name, e.g., example.com (required)
- `--provider <id>` - Provider ID (required)
- `--network <id>` - Network ID (for auto PTR records)
- `--enable-auto-sync` - Enable automatic sync
- `--create-ptr` - Create PTR records
- `--ttl <seconds>` - Default TTL

**Examples:**

```bash
rackd dns zone create \
  --name example.com \
  --provider prov-123 \
  --network net-456 \
  --create-ptr \
  --enable-auto-sync
```

##### dns zone update

Update a DNS zone.

```bash
rackd dns zone update --id <id> [options]
```

##### dns zone delete

Delete a DNS zone.

```bash
rackd dns zone delete --id <id> [--force]
```

#### dns records

List DNS records for a zone.

```bash
rackd dns records --zone <zone-id> [options]
```

**Options:**
- `--zone <id>` - Zone ID (required)
- `--type <type>` - Filter by record type (A, AAAA, CNAME, MX, TXT, PTR, NS, SRV)
- `--device <id>` - Filter by device ID
- `--name <name>` - Filter by record name
- `--output <format>` - Output format (table/json/yaml)

#### dns sync

Sync a DNS zone to the provider.

```bash
rackd dns sync --zone <zone-id> [options]
```

**Options:**
- `--zone <id>` - Zone ID (required)
- `--force` - Force sync even if unchanged

#### dns import

Import DNS records from provider.

```bash
rackd dns import --zone <zone-id> [options]
```

**Options:**
- `--zone <id>` - Zone ID (required)
- `--delete` - Delete local records not found on provider

### audit

Audit log management.

#### audit list

List audit logs.

```bash
rackd audit list [options]
```

**Options:**
- `--resource <type>` - Filter by resource type
- `--resource-id <id>` - Filter by resource ID
- `--action <action>` - Filter by action
- `--limit <n>` - Limit number of results (default: 50)

**Examples:**

```bash
# List recent audit logs
rackd audit list

# Filter by resource type
rackd audit list --resource devices --limit 100

# Filter by specific device
rackd audit list --resource-id dev-123
```

#### audit export

Export audit logs.

```bash
rackd audit export [options]
```

**Options:**
- `--format <format>` - Export format (json/csv, default: json)
- `--output <file>` - Output file (default: stdout)
- `--resource <type>` - Filter by resource type
- `--resource-id <id>` - Filter by resource ID

### backup

Backup the Rackd database.

```bash
rackd backup [options]
```

**Options:**
- `--data-dir <dir>` - Data directory (default: ./data)
- `--output <file>` - Output file (default: rackd-backup-<timestamp>.db)

**Examples:**

```bash
# Create backup with auto-generated filename
rackd backup

# Create backup with specific filename
rackd backup --output /backups/rackd-$(date +%Y%m%d).db
```

### import

Import data from CSV or JSON files.

#### import devices

Import devices from file.

```bash
rackd import devices --file <path> [options]
```

**Options:**
- `--file <path>` - Input file (JSON or CSV, required)
- `--format <format>` - Input format (json/csv, auto-detected if omitted)
- `--dry-run` - Validate without importing

**Examples:**

```bash
# Import from JSON
rackd import devices --file devices.json

# Import from CSV with dry run
rackd import devices --file devices.csv --dry-run
```

#### import networks

Import networks from file.

```bash
rackd import networks --file <path> [options]
```

#### import datacenters

Import datacenters from file.

```bash
rackd import datacenters --file <path> [options]
```

### export

Export data to CSV or JSON.

#### export devices

Export devices.

```bash
rackd export devices [options]
```

**Options:**
- `--format <format>` - Output format (json/csv, default: json)
- `--output <file>` - Output file (default: stdout)

**Examples:**

```bash
# Export to JSON file
rackd export devices --output devices.json

# Export to CSV
rackd export devices --format csv --output devices.csv
```

#### export networks

Export networks.

```bash
rackd export networks [options]
```

#### export datacenters

Export datacenters.

```bash
rackd export datacenters [options]
```

#### export all

Export all data (devices, networks, datacenters).

```bash
rackd export all [options]
```

**Options:**
- `--format <format>` - Output format (json only)
- `--output <file>` - Output file (default: stdout)

### migrate

Database migration management.

#### migrate status

Show migration status.

```bash
rackd migrate status [options]
```

**Options:**
- `--data-dir <dir>` - Data directory (default: ./data)

**Output:**

```
VERSION     NAME                    STATUS      APPLIED AT
001         initial_schema          applied     2024-01-15 10:30:00
002         add_rbac_tables         applied     2024-01-15 10:30:00
003         add_dns_tables          pending     -

Total: 3 migrations, 1 pending
```

#### migrate run

Run pending migrations.

```bash
rackd migrate run [options]
```

**Options:**
- `--data-dir <dir>` - Data directory (default: ./data)

**Examples:**

```bash
rackd migrate run
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
