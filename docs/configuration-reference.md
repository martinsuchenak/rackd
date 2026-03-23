# Configuration Reference

All configuration is done via environment variables. Variables can be set directly or loaded from a `.env` file in the working directory.

Use `--dev-mode` flag on `rackd server` to automatically disable `COOKIE_SECURE` and `RATE_LIMIT_ENABLED` for local development.

## Server

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DATA_DIR` | string | `./data` | Directory for SQLite database and data files |
| `LISTEN_ADDR` | string | `:8080` | Address and port to listen on |
| `REQUEST_TIMEOUT` | duration | `30s` | HTTP request timeout |
| `LOG_FORMAT` | string | `text` | Log format: `text` or `json` |
| `LOG_LEVEL` | string | `info` | Log level: `trace`, `debug`, `info`, `warn`, `error` |
| `TRUST_PROXY` | bool | `false` | Trust `X-Forwarded-For` and `X-Real-IP` headers for client IP detection |

## Security

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `COOKIE_SECURE` | bool | `true` | Set `Secure` flag on session cookies (requires HTTPS). Disable for local dev without TLS |
| `RATE_LIMIT_ENABLED` | bool | `true` | Enable API rate limiting |
| `RATE_LIMIT_REQUESTS` | int | `100` | Max requests per window per IP |
| `RATE_LIMIT_WINDOW` | duration | `1m` | Rate limit sliding window |
| `LOGIN_RATE_LIMIT_REQUESTS` | int | `5` | Max login attempts per window per IP |
| `LOGIN_RATE_LIMIT_WINDOW` | duration | `1m` | Login rate limit sliding window |

## Sessions

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SESSION_TTL` | duration | `24h` | Session expiry after last activity |
| `SESSION_STORE_TYPE` | string | `sqlite` | Session store backend: `sqlite` or `valkey` |
| `VALKEY_URL` | string | `redis://localhost:6379/0` | Valkey/Redis URL (when `SESSION_STORE_TYPE=valkey`) |

## Initial Admin

These are only used on first startup when no users exist.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `INITIAL_ADMIN_USERNAME` | string | _(empty)_ | Bootstrap admin username |
| `INITIAL_ADMIN_PASSWORD` | string | _(empty)_ | Bootstrap admin password (min 8 characters) |
| `INITIAL_ADMIN_EMAIL` | string | `admin@localhost` | Bootstrap admin email |
| `INITIAL_ADMIN_FULL_NAME` | string | `System Administrator` | Bootstrap admin display name |

## Discovery

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DISCOVERY_INTERVAL` | duration | `24h` | Interval between scheduled discovery scans |
| `DISCOVERY_MAX_CONCURRENT` | int | `10` | Max concurrent host probes during a scan |
| `DISCOVERY_TIMEOUT` | duration | `5s` | Per-host probe timeout |
| `DISCOVERY_CLEANUP_DAYS` | int | `30` | Auto-delete discovered devices older than this |
| `DISCOVERY_SCAN_ON_STARTUP` | bool | `false` | Run discovery scans immediately on server start |
| `DISCOVERY_SNMPV2C_ENABLED` | bool | `false` | Enable SNMPv2c probing during discovery |

## Audit

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AUDIT_ENABLED` | bool | `false` | Enable audit logging of API operations |
| `AUDIT_RETENTION_DAYS` | int | `90` | Days to retain audit log entries |

## OAuth 2.1 (MCP)

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `MCP_OAUTH_ENABLED` | bool | `false` | Enable OAuth 2.1 authentication for MCP endpoint |
| `MCP_OAUTH_ISSUER_URL` | string | _(empty)_ | OAuth issuer URL (required when enabled) |
| `MCP_OAUTH_ACCESS_TOKEN_TTL` | duration | `1h` | Access token lifetime |
| `MCP_OAUTH_REFRESH_TOKEN_TTL` | duration | `720h` | Refresh token lifetime (30 days) |

## Utilization Snapshots

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SNAPSHOT_INTERVAL` | duration | `1h` | Interval between network utilization snapshots |
| `SNAPSHOT_RETENTION_DAYS` | int | `90` | Days to retain utilization snapshots |

## DNS

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DNS_SYNC_INTERVAL` | duration | `1h` | Interval between DNS zone sync operations |

## Duration Format

Duration values use Go duration syntax: `30s`, `5m`, `1h`, `24h`, `720h`. Combine units: `1h30m`.
