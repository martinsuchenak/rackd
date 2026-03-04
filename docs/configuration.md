# Configuration

Rackd can be configured through environment variables. All configuration options have sensible defaults and can be overridden as needed.

## Environment Variables

### Server Options

| Variable | Default | Description |
|----------|---------|-------------|
| `DATA_DIR` | `./data` | Directory for SQLite database and application data |
| `LISTEN_ADDR` | `:8080` | Address and port for the HTTP server (e.g., `:8080`, `127.0.0.1:3000`) |

### Security Options

| Variable | Default | Description |
|----------|---------|-------------|
| `API_AUTH_TOKEN` | _(empty)_ | Bearer token for API authentication. If empty, API is unauthenticated |
| `MCP_AUTH_TOKEN` | _(empty)_ | Bearer token for MCP server authentication. If empty, MCP is unauthenticated |
| `SESSION_TTL` | `24h` | Duration for which a user session is valid |
| `SESSION_STORE_TYPE` | `sqlite` | The backend storage for sessions (`sqlite`, `valkey`, `redis`) |
| `VALKEY_URL` | `redis://localhost:6379/0` | The URL for Valkey/Redis if `SESSION_STORE_TYPE` is `valkey` or `redis` |

### Rate Limiting Options

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | `false` | Enable API rate limiting |
| `RATE_LIMIT_REQUESTS` | `100` | Maximum requests per window |
| `RATE_LIMIT_WINDOW` | `1m` | Time window for rate limiting (Go duration format) |

See [Rate Limiting](ratelimit.md) for detailed documentation.

### Audit Logging Options

| Variable | Default | Description |
|----------|---------|-------------|
| `AUDIT_ENABLED` | `false` | Enable audit logging for all API changes |
| `AUDIT_RETENTION_DAYS` | `90` | Days to keep audit logs before automatic cleanup |

See [Audit Trail](audit.md) for detailed documentation.

### Logging Options

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_FORMAT` | `text` | Log output format: `text` or `json` |
| `LOG_LEVEL` | `info` | Log level: `trace`, `debug`, `info`, `warn`, or `error` |

### Discovery Options

| Variable | Default | Description |
|----------|---------|-------------|
| `DISCOVERY_INTERVAL` | `24h` | How often to run automatic network discovery (Go duration format) |
| `DISCOVERY_MAX_CONCURRENT` | `10` | Maximum number of concurrent discovery operations |
| `DISCOVERY_TIMEOUT` | `5s` | Timeout for individual device discovery attempts |
| `DISCOVERY_CLEANUP_DAYS` | `30` | Days to keep discovery history before cleanup |
| `DISCOVERY_SCAN_ON_STARTUP` | `false` | Whether to run discovery scan immediately on startup |
| `DISCOVERY_SNMPV2C_ENABLED` | `false` | If false, prevents SNMPv2c discovery scans across the infrastructure. Enable if SNMPv2c required. |

## Configuration Examples

### Basic Setup

```bash
# Minimal configuration for development
export DATA_DIR="./data"
export LISTEN_ADDR=":8080"
export LOG_LEVEL="debug"

./rackd server
```

### Production Setup

```bash
# Production configuration with authentication, rate limiting, and audit logging
export DATA_DIR="/var/lib/rackd"
export LISTEN_ADDR=":8080"
export API_AUTH_TOKEN="your-secure-api-token"
export MCP_AUTH_TOKEN="your-secure-mcp-token"
export LOG_FORMAT="json"
export LOG_LEVEL="info"
export DISCOVERY_INTERVAL="12h"
export DISCOVERY_MAX_CONCURRENT="20"
export RATE_LIMIT_ENABLED="true"
export RATE_LIMIT_REQUESTS="100"
export RATE_LIMIT_WINDOW="1m"
export AUDIT_ENABLED="true"
export AUDIT_RETENTION_DAYS="90"

./rackd server
```

### Docker Environment

```bash
# Docker configuration
docker run -d \
  -p 8080:8080 \
  -v /var/lib/rackd:/data \
  -e DATA_DIR="/data" \
  -e API_AUTH_TOKEN="your-token" \
  -e LOG_FORMAT="json" \
  rackd:latest
```

## Duration Format

Duration values (like `DISCOVERY_INTERVAL` and `DISCOVERY_TIMEOUT`) use Go's duration format:

- `s` - seconds
- `m` - minutes  
- `h` - hours
- `d` - days (24h)

Examples:
- `30s` - 30 seconds
- `5m` - 5 minutes
- `2h` - 2 hours
- `24h` - 24 hours
- `1h30m` - 1 hour 30 minutes

## Security Considerations

### Authentication Tokens

- Use strong, randomly generated tokens for production
- Store tokens securely (environment variables, secrets management)
- Rotate tokens regularly
- Never commit tokens to version control

### Network Security

- Bind to specific interfaces in production (`127.0.0.1:8080` vs `:8080`)
- Use reverse proxy with TLS termination
- Implement firewall rules to restrict access

### File Permissions

Ensure proper permissions on the data directory:

```bash
# Create data directory with restricted permissions
mkdir -p /var/lib/rackd
chown rackd:rackd /var/lib/rackd
chmod 750 /var/lib/rackd
```

## Validation

Rackd validates configuration on startup and will exit with an error if invalid values are provided:

- `LOG_LEVEL` must be one of: `trace`, `debug`, `info`, `warn`, `error`
- `LOG_FORMAT` must be either `text` or `json`
- Duration values must be positive
- Numeric values must be positive integers

## Environment File

You can use a `.env` file for configuration:

```bash
# .env file
DATA_DIR=./data
LISTEN_ADDR=:8080
LOG_LEVEL=debug
API_AUTH_TOKEN=dev-token-123
DISCOVERY_INTERVAL=1h
```

Load with:
```bash
source .env
./rackd server
```

## Systemd Service

Example systemd service file with environment configuration:

```ini
[Unit]
Description=Rackd IPAM Server
After=network.target

[Service]
Type=simple
User=rackd
Group=rackd
WorkingDirectory=/opt/rackd
ExecStart=/opt/rackd/rackd server
Environment=DATA_DIR=/var/lib/rackd
Environment=LISTEN_ADDR=:8080
Environment=LOG_FORMAT=json
Environment=LOG_LEVEL=info
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```