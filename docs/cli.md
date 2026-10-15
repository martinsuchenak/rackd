# CLI Reference

Complete command-line interface reference for Rackd.

## Overview

The Rackd CLI provides full access to all functionality via command-line commands. It communicates with the Rackd server via the REST API.

## Global Options

```bash
rackd [global options] command [command options] [arguments...]
```

### CLI Settings

| Setting | Config file key | Environment variable | Default |
|---------|-----------------|----------------------|---------|
| Server URL | `server_url` | `RACKD_SERVER_URL` | `http://localhost:8080` |
| API token | `token` | `RACKD_TOKEN` | - |
| Verify TLS certificates | `verify_ssl` | `RACKD_VERIFY_SSL` | `true` |
| Request timeout | `timeout` | - | `30s` |
| Output format (table, json, yaml) | `output` | - | `table` |

The config file is `$XDG_CONFIG_HOME/rackd/config.json` (by default `~/.config/rackd/config.json`). Environment variables override it. `--help`/`-h` is the only global flag; output and other options are per-command flags.

### Configuration File

The CLI reads `$XDG_CONFIG_HOME/rackd/config.json` (by default `~/.config/rackd/config.json`):

```json
{
  "server_url": "http://localhost:8080",
  "token": "your-secret-token",
  "timeout": "30s",
  "output": "table",
  "verify_ssl": true
}
```

## Commands

| Command | Purpose |
|---------|---------|
| `server` | Run the HTTP, MCP and web UI server |
| `device`, `network`, `datacenter` | Manage inventory |
| `discovery` | Run scans and promote discovered devices |
| `conflict`, `reservation`, `circuit`, `nat`, `dns` | Manage conflicts, IP reservations, circuits, NAT and DNS |
| `webhook`, `custom-field` | Manage webhooks and custom field definitions |
| `user`, `role`, `apikey`, `oauth` | Manage access |
| `credentials` | Rotate the encryption key |
| `scan-profile`, `scheduled-scan` | Configure discovery |
| `audit` | List and export the audit log |
| `backup`, `migrate`, `import`, `export` | Back up, restore, migrate and move data; see [Backup & Restore](backup.md) and [Database](database.md) |
| `version` | Show the version |

Every command, subcommand and flag is listed in the [CLI Reference](cli-reference.md), generated from the commands themselves. `rackd <command> --help` shows the same.

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

`rackd` exits with `0` on success and `1` on any error; the error is printed to standard error.

## Environment Variables

The CLI reads its server connection from environment variables, which override the config file:

```bash
export RACKD_SERVER_URL=http://localhost:8080
export RACKD_TOKEN=mysecret
export RACKD_VERIFY_SSL=false   # only for self-signed certificates in testing

rackd device list
```

## Examples

### Complete Workflow

```bash
# 1. Start the server (see Installation for ENCRYPTION_KEY and the first admin)
rackd server &

# 2. Point the CLI at it with an API key created in the web UI or with `rackd apikey create`
export RACKD_SERVER_URL=http://localhost:8080
export RACKD_TOKEN=<api key>

# 3. Add a datacenter and a network (IDs are printed on creation)
rackd datacenter add --name dc1 --location "New York"
rackd network add --name prod-net --subnet 10.0.1.0/24 --datacenter <datacenter id>

# 4. Scan the network, then list and promote what it found
rackd discovery scan --network <network id> --type quick
rackd discovery list --network <network id>
rackd discovery promote --discovered-id <discovered id> --name db-01
```

## Related Documentation

- [API Reference](api.md) - REST API documentation
- [Configuration](configuration.md) - Configuration options
- [Quick Start](quickstart.md) - Getting started guide
