# Security

Security considerations and hardening guide for production Rackd deployments.

## Authentication

Rackd requires authentication on all API endpoints. Three methods are supported:

- Session cookies (Web UI login)
- API keys (Bearer token, tied to a user account)
- OAuth 2.1 with PKCE (MCP clients)

See [Authentication](authentication.md) for details on each method.

## Secure Defaults

Rackd ships with secure defaults for production use:

| Setting | Default | Description |
|---------|---------|-------------|
| `COOKIE_SECURE` | `true` | Session cookies only sent over HTTPS |
| `RATE_LIMIT_ENABLED` | `true` | API rate limiting active |
| `RATE_LIMIT_REQUESTS` | `100` | 100 requests per minute per IP |
| `LOGIN_RATE_LIMIT_REQUESTS` | `5` | 5 login attempts per minute per IP |

For local development without TLS, use `--dev-mode` which disables `COOKIE_SECURE` and `RATE_LIMIT_ENABLED` automatically.

## CSRF Protection

State-changing requests (POST, PUT, DELETE, PATCH) from session-authenticated users must include the `X-Requested-With: XMLHttpRequest` header. Requests without it are rejected with 403. This prevents cross-origin form submissions from exploiting session cookies.

The built-in Web UI sends this header on all requests. External integrations using session auth must include it as well.

## Password Security

- Minimum length: 8 characters (enforced server-side)
- Hashing: bcrypt with cost factor 14
- Password changes invalidate all active sessions for the user
- Login rate limiting (per IP) and an account lockout after `LOGIN_ACCOUNT_MAX_FAILURES` failed sign-ins (default 10) within `LOGIN_ACCOUNT_WINDOW` (default 15m) slow down brute-force attacks

## API Key Security

- Keys are SHA-256 hashed before storage — plaintext is never persisted
- Token comparison uses constant-time comparison to prevent timing attacks
- Every key must be associated with a user account for RBAC enforcement
- Legacy keys (no user association) are rejected at the auth boundary
- Expired keys return a specific `EXPIRED_KEY` error code
- Last-used timestamps are tracked for auditing (to about a minute)

## Session Security

- Sessions use cryptographically random tokens
- Cookies are `HttpOnly`, `SameSite=Strict`, and `Secure` (by default)
- 24-hour TTL with sliding expiration (`SESSION_TTL`), capped at 7 days from sign-in (`SESSION_MAX_LIFETIME`)
- Sessions are invalidated on password change
- Session store supports SQLite (default) or Valkey for distributed deployments

## OAuth 2.1 Security

- PKCE required (no plain code challenge)
- Refresh token rotation with replay detection
- Dynamic client registration
- Token revocation endpoint

## Credential Encryption

Device credentials (SSH passwords, SNMP community strings), DNS provider tokens and webhook secrets are encrypted at rest using AES-256-GCM with the key in `ENCRYPTION_KEY` (64 hex characters, e.g. from `openssl rand -hex 32`). Without it, credentials, scan profiles, scheduled scans and DNS are disabled and webhook secrets are stored unencrypted. Change the key with `rackd credentials rotate-key`.

## Webhook Security

- HMAC-SHA256 signatures on webhook payloads (when a secret is configured)
- SSRF protection (webhooks and DNS providers): loopback, link-local (169.254.x.x, including cloud metadata), multicast and unspecified addresses are blocked. Private ranges (RFC 1918, CGNAT `100.64.0.0/10`, IPv6 ULA) are blocked too unless listed in `SSRF_ALLOWED_TARGETS` (or all allowed with `SSRF_ALLOW_PRIVATE_TARGETS=true`)
- Secure HTTP client with `SafeDialContext` prevents DNS rebinding
- URL length capped at 2048 characters
- Only HTTP and HTTPS schemes allowed

## Content Security Policy

The Web UI sets a strict CSP header:
- No `unsafe-eval` or `unsafe-inline` for scripts
- Alpine.js CSP-compatible build
- `default-src 'self'` baseline

## Security Headers

Rackd sets the following headers on all responses:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: geolocation=(), microphone=(), camera=()`
- `Content-Security-Policy` (see above)
- `Strict-Transport-Security: max-age=31536000`, when `COOKIE_SECURE=true` (the default) or the request arrived over TLS

## Audit Logging

Rackd always audits changes and sign-ins (see [Audit Trail](audit.md) for the full list). Each entry records:

- Timestamp, action, resource type, resource ID
- User ID and username (from API key or session)
- Client IP address
- Success/failure status, and why a failure failed
- The record after the change, without secrets, keys, passwords or full webhook URLs

Audit entries are kept for 90 days by default (`AUDIT_RETENTION_DAYS`; `0` keeps everything).

Query audit logs:
- `GET /api/v1/audit` — list with filters (action, resource, user, date range)
- `GET /api/v1/audit/export` — export as JSON (default) or CSV (`?format=csv`)
- `GET /api/v1/audit/{id}` — get single entry

## Rate Limiting

Two rate limiters protect against abuse:

1. General API: 100 requests/minute per IP (configurable)
2. Login/sensitive endpoints: 5 requests/minute per IP (configurable), plus the per-account lockout above

See [Rate Limiting](ratelimit.md).

Rate limit headers are included in responses:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

## RBAC

All API operations are authorized through role-based access control at the service layer. Both REST API and MCP requests go through the same RBAC checks.

Built-in system roles:
- `admin` — all permissions
- `operator` — read/write on infrastructure resources
- `viewer` — read-only access

Custom roles can be created and assigned arbitrary permission sets.

## Network Security Recommendations

### Reverse Proxy

Run Rackd behind a reverse proxy for TLS termination:

```nginx
server {
    listen 443 ssl http2;
    server_name rackd.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Set `TRUST_PROXY=true` when behind a reverse proxy so Rackd reads the real client IP from forwarded headers. You must also set `TRUSTED_PROXIES` to the reverse proxy's IP or CIDR (e.g. `TRUSTED_PROXIES=10.0.0.5`): forwarded headers are honored only when the direct peer matches a configured network. Without `TRUSTED_PROXIES`, forwarded headers are ignored and the socket address is used, so a directly reachable instance cannot have its per-IP login throttling bypassed by spoofed `X-Forwarded-For` / `X-Real-IP` values.

### File Permissions

```bash
chmod 600 /var/lib/rackd/rackd.db
chmod 600 /etc/rackd/config.env   # the EnvironmentFile holding ENCRYPTION_KEY
chown rackd:rackd /var/lib/rackd/
```

### Run as Non-Root

```bash
useradd -r -s /bin/false rackd
sudo -u rackd ./rackd server
```

### Firewall

Only expose the HTTP port. Bind to localhost if using a reverse proxy:

```bash
LISTEN_ADDR=127.0.0.1:8080
```

## Discovery Security

Network discovery scans probe hosts via ARP, TCP, and optionally SNMP. Considerations:

- Scans are limited to the configured network's subnet
- Maximum subnet size is /16 (65,534 hosts) — larger subnets may cause resource exhaustion
- SNMPv2c is disabled by default (`DISCOVERY_SNMPV2C_ENABLED=false`)
- Discovery credentials are encrypted at rest

## See Also

- [Authentication](authentication.md) — authentication methods and user management
- [Configuration Reference](configuration-reference.md) — all environment variables
- [Deployment](deployment.md) — production deployment guide
