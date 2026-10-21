# Configuration Reference

The server is configured with environment variables, which have no prefix. Authentication needs no settings: people sign in with user accounts, scripts and the CLI use API keys (see [Authentication](authentication.md)), and MCP clients use an API key or, with `MCP_OAUTH_ENABLED`, OAuth.

## Loading

- At startup the server reads `.env` from its working directory, if there is one. Values in `.env` override variables already set in the environment.
- A value that doesn't parse as the expected type (int, bool or duration) is ignored and the default is used; check the startup log if a setting seems to have no effect.
- A few settings also have a command-line flag: `--data-dir` (`DATA_DIR`, on every command that opens the database), and on `rackd server` `--listen-addr`, `--log-level`, `--log-format` and `--discovery-interval`. A flag you pass overrides the variable; one you leave out leaves it in charge.
- `rackd server --dev-mode` turns off `COOKIE_SECURE` and `RATE_LIMIT_ENABLED` and generates a temporary `ENCRYPTION_KEY` when none is set. Use it for local development only.

The CLI has its own settings (`RACKD_SERVER_URL`, `RACKD_TOKEN`, `RACKD_VERIFY_SSL` and a config file); see [CLI](cli.md).

## Server

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DATA_DIR` | string | `./data` | Directory for SQLite database and data files |
| `BACKUP_INTERVAL` | duration | `0` | Take a verified backup this often (at least `5m`); `0` turns scheduled backups off. See [Backup & Restore](backup.md#scheduled-backups-in-the-server) |
| `BACKUP_KEEP` | int | `7` | Scheduled backups to keep; older ones are deleted |
| `BACKUP_DIR` | string | `<DATA_DIR>/backups` | Directory for scheduled backups |
| `PRE_MIGRATION_BACKUP` | bool | `true` | Back up the database to `<DATA_DIR>/backups/` before pending migrations run at startup; the server does not start if the backup fails. Set `false` to skip it |
| `LISTEN_ADDR` | string | `:8080` | Address and port to listen on |
| `REQUEST_TIMEOUT` | duration | `30s` | HTTP request timeout |
| `LOG_FORMAT` | string | `text` | Log format: `text` or `json` |
| `LOG_LEVEL` | string | `info` | Log level: `trace`, `debug`, `info`, `warn`, `error` |
| `TRUST_PROXY` | bool | `false` | Trust `X-Forwarded-For` and `X-Real-IP` headers for client IP detection (requires `TRUSTED_PROXIES`) |
| `TRUSTED_PROXIES` | string | *(empty)* | Comma-separated IPs/CIDRs of reverse proxies whose forwarded headers may be trusted |

## Security

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ENCRYPTION_KEY` | string | _(empty)_ | 64 hex characters (32 bytes), e.g. from `openssl rand -hex 32`. Encrypts stored credentials, DNS provider tokens and webhook secrets. Without it, credentials, scan profiles, scheduled scans and DNS are disabled and webhook secrets are stored unencrypted; `--dev-mode` generates a random key instead, which doesn't survive a restart. A malformed key stops the server. Change it with `rackd credentials rotate-key` |
| `COOKIE_SECURE` | bool | `true` | Set `Secure` flag on session cookies (requires HTTPS). Disable for local dev without TLS |
| `RATE_LIMIT_ENABLED` | bool | `true` | Enable API rate limiting |
| `RATE_LIMIT_REQUESTS` | int | `100` | Max requests per window per IP |
| `RATE_LIMIT_WINDOW` | duration | `1m` | Rate limit sliding window |
| `LOGIN_RATE_LIMIT_REQUESTS` | int | `5` | Max login attempts per window per IP |
| `LOGIN_RATE_LIMIT_WINDOW` | duration | `1m` | Login rate limit sliding window |
| `LOGIN_ACCOUNT_MAX_FAILURES` | int | `10` | Failed sign-ins per account within `LOGIN_ACCOUNT_WINDOW` before further attempts are refused; `0` turns it off |
| `LOGIN_ACCOUNT_WINDOW` | duration | `15m` | Window for `LOGIN_ACCOUNT_MAX_FAILURES` |

## Sessions

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SESSION_TTL` | duration | `24h` | Session expiry after last activity |
| `SESSION_MAX_LIFETIME` | duration | `168h` | Longest a session lasts from sign-in, however often it is used; `0` means no limit |
| `SESSION_STORE_TYPE` | string | `sqlite` | Session store backend: `sqlite` or `valkey` (`redis` is accepted too). Any other value keeps sessions in memory, so they are lost on restart, and logs a warning |
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
| `DISCOVERY_CLEANUP_DAYS` | int | `30` | Delete discovered devices that were never promoted and haven't been seen for this many days |
| `DISCOVERY_SCAN_ON_STARTUP` | bool | `false` | Run discovery scans immediately on server start |
| `DISCOVERY_SNMPV2C_ENABLED` | bool | `false` | Enable SNMPv2c probing during discovery |

## Audit

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `AUDIT_RETENTION_DAYS` | int | `90` | Days to keep audit entries, deleted daily; `0` keeps everything. Auditing is always on (`AUDIT_ENABLED` is ignored) |

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
| `SNAPSHOT_INTERVAL` | duration | `1h` | Interval between network and pool utilization snapshots, which feed the dashboard trend chart; `0` disables them |
| `SNAPSHOT_RETENTION_DAYS` | int | `90` | Days to retain utilization snapshots |
| `POOL_UTILIZATION_ALERT_PERCENT` | int | `90` | Pool utilization (percent) at which a `pool.utilization_high` webhook event is published, checked at each snapshot when a pool crosses it; `0` turns it off |

## DNS

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `DNS_SYNC_INTERVAL` | duration | `1h` | Interval between DNS zone sync operations |

## Outbound Requests (SSRF)

Webhooks and DNS providers can't reach loopback, link-local (including cloud metadata), multicast or reserved addresses. Private ranges (RFC 1918, CGNAT `100.64.0.0/10`, IPv6 ULA) are blocked too, unless allowed here.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `SSRF_ALLOWED_TARGETS` | string | _(empty)_ | Comma-separated private IPs or CIDR ranges that webhooks and DNS providers may reach, e.g. `192.168.10.53,10.20.0.0/24`. Each entry must be inside a private range; rackd refuses to start otherwise |
| `SSRF_ALLOW_PRIVATE_TARGETS` | bool | `false` | Allow every private range. Prefer `SSRF_ALLOWED_TARGETS`: with this set, anyone who can save a webhook or DNS provider can make rackd send requests anywhere on your network |

## Duration Format

Duration values use Go duration syntax: `30s`, `5m`, `1h`, `24h`, `720h`. Combine units: `1h30m`. There is no day unit: write 7 days as `168h`.

## Validation

The server refuses to start when:

- `LOG_LEVEL` isn't one of `trace`, `debug`, `info`, `warn`, `error`
- `LOG_FORMAT` isn't `text` or `json`
- `DISCOVERY_INTERVAL` or `DISCOVERY_CLEANUP_DAYS` isn't positive
- `RATE_LIMIT_REQUESTS` or `RATE_LIMIT_WINDOW` isn't positive while rate limiting is on
- `AUDIT_RETENTION_DAYS` is negative
- `MCP_OAUTH_ENABLED` is true without `MCP_OAUTH_ISSUER_URL`
- `SSRF_ALLOWED_TARGETS` has an entry that isn't a private IP or range
- `ENCRYPTION_KEY` is set but isn't 64 hex characters

## Examples

### Local development

```bash
./rackd server --dev-mode --log-level debug
```

### Production

```bash
export DATA_DIR=/var/lib/rackd
export ENCRYPTION_KEY='<64 hex characters>'       # generate once with openssl rand -hex 32 and keep it: a new key can't read stored secrets
export INITIAL_ADMIN_USERNAME=admin               # first start only
export INITIAL_ADMIN_PASSWORD='<a strong password>'
export LOG_FORMAT=json
export TRUST_PROXY=true                           # behind a reverse proxy that terminates TLS
export TRUSTED_PROXIES=10.0.0.5

./rackd server
```

`COOKIE_SECURE` and `RATE_LIMIT_ENABLED` are already on by default. Keep the data directory readable only by the account rackd runs as (`chmod 750`), and bind `LISTEN_ADDR` to `127.0.0.1:8080` when a reverse proxy on the same host is the only client.

### `.env` file

```bash
# .env in the server's working directory
DATA_DIR=./data
LOG_LEVEL=debug
DISCOVERY_INTERVAL=1h
```

For systemd, Docker and Nomad setups see [Installation](installation.md) and [Deployment](deployment.md).
