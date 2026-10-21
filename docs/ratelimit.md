# API Rate Limiting

Rackd limits how many requests each client IP address can make, and how often sign-ins can be tried. Rate limiting is **enabled by default**. Exceeding a limit returns HTTP 429 (Too Many Requests).

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | `true` | General request limit on or off |
| `RATE_LIMIT_REQUESTS` | `100` | Requests per window per client IP |
| `RATE_LIMIT_WINDOW` | `1m` | Window length |
| `LOGIN_RATE_LIMIT_REQUESTS` | `5` | Requests per window per client IP on sign-in and other sensitive endpoints |
| `LOGIN_RATE_LIMIT_WINDOW` | `1m` | Window length for the login limit |
| `LOGIN_ACCOUNT_MAX_FAILURES` | `10` | Failed sign-ins per account within `LOGIN_ACCOUNT_WINDOW` before further attempts are refused; `0` turns it off |
| `LOGIN_ACCOUNT_WINDOW` | `15m` | Window for `LOGIN_ACCOUNT_MAX_FAILURES` |

`rackd server --dev-mode` turns the general limit off. The login limit and the account lockout stay on.

See the [Configuration Reference](configuration-reference.md) for every setting.

## How It Works

### General limit

- Covers every response the server sends: the API, MCP, the web UI and its assets, `/healthz`, `/readyz` and `/metrics`.
- Counts requests per client IP address. API keys and sessions don't get separate limits, and there is no exception for localhost.
- Uses a fixed window per IP: the window starts with that address's first request, and the full allowance comes back once the window has passed.
- Buckets for addresses that have gone quiet are dropped after twice the window.

### Login and sensitive endpoints

These have their own limit, using the `LOGIN_RATE_LIMIT_*` settings, on top of the general one. Each endpoint keeps separate buckets, so failed sign-ins from an address don't also block that address's token refreshes or password changes:

- `POST /api/v1/auth/login`
- `POST /api/v1/keys`
- `POST /api/v1/users`, `POST /api/v1/users/{id}/password`, `POST /api/v1/users/{id}/reset-password`
- The bulk endpoints (`/api/v1/devices/bulk`, `/api/v1/devices/bulk/tags`, `/api/v1/networks/bulk`)
- `POST /mcp-oauth/register`, `POST /mcp-oauth/authorize`, `POST /mcp-oauth/token`

### Account lockout

Separately from the per-IP limits, an account that has `LOGIN_ACCOUNT_MAX_FAILURES` failed sign-ins within `LOGIN_ACCOUNT_WINDOW` is refused further sign-ins for the rest of that window, whatever address they come from. The lock applies to unknown usernames too, so it doesn't reveal which accounts exist. Refused attempts return 429 with the code `TOO_MANY_ATTEMPTS`.

### Client IP behind a proxy

By default the client IP is the address of the TCP connection. Behind a reverse proxy, set `TRUST_PROXY=true` and `TRUSTED_PROXIES` to the proxy's IPs or CIDRs. Forwarded headers are then used only for requests whose direct peer is one of those proxies: the **rightmost** `X-Forwarded-For` entry (the one your proxy appended), or else `X-Real-IP`. Without `TRUSTED_PROXIES`, forwarded headers are ignored, so clients can't pick a fresh bucket by sending their own.

## Response Headers

Responses carry the state of the limit that applied:

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 42
X-RateLimit-Reset: 2026-02-03T16:45:00Z
```

`X-RateLimit-Reset` is the time the window ends (RFC 3339). On the login and sensitive endpoints, the headers describe the login limit.

When a limit is exceeded:

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 2026-02-03T16:45:00Z
Retry-After: 2026-02-03T16:45:00Z
Content-Type: application/json

{
  "error": "Rate limit exceeded",
  "code": "RATE_LIMIT_EXCEEDED"
}
```

The login limit returns the code `LOGIN_RATE_LIMIT_EXCEEDED` instead. `Retry-After` is a timestamp, not a number of seconds.

## Monitoring

At start the server logs `Rate limiting enabled` (with the requests and window) when the general limit is on, and `Login rate limiting enabled`. Requests over the general limit are logged at debug level (`Rate limit exceeded`, with the client IP and path); requests over the login limit and account lockouts are logged as warnings.

## Limitations

- Counters are kept in memory: they reset when the server restarts.
- Each Rackd instance counts separately; there is no coordination across instances.
- Clients behind the same NAT or proxy share a limit unless `TRUST_PROXY` and `TRUSTED_PROXIES` are set.

For distributed or per-route limits, use a reverse proxy (nginx, Caddy) or an API gateway as well.

## Best Practices

- Watch `X-RateLimit-Remaining` and back off until `X-RateLimit-Reset` after a 429.
- The bulk endpoints save requests, but they fall under the login limit (`LOGIN_RATE_LIMIT_REQUESTS` per window), so space out bulk calls or raise that limit for large imports.
- Raise `RATE_LIMIT_REQUESTS` for busy automation rather than turning the limit off.

## See Also

- [Authentication](authentication.md) - API key management
- [Configuration Reference](configuration-reference.md) - Environment variables
- [Monitoring](monitoring.md) - Metrics and observability
- [Security](security.md) - Security best practices
