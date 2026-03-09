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

- Minimum length: 12 characters (enforced server-side)
- Hashing: bcrypt with cost factor 14
- Password changes invalidate all active sessions for the user
- Login rate limiting prevents brute-force attacks

## API Key Security

- Keys are SHA-256 hashed before storage — plaintext is never persisted
- Token comparison uses constant-time comparison to prevent timing attacks
- Every key must be associated with a user account for RBAC enforcement
- Legacy keys (no user association) are rejected at the auth boundary
- Expired keys return a specific `EXPIRED_KEY` error code
- Last-used timestamps are tracked for auditing

## Session Security

- Sessions use cryptographically random tokens
- Cookies are `HttpOnly`, `SameSite=Lax`, and `Secure` (by default)
- 24-hour TTL with sliding expiration
- Sessions are invalidated on password change
- Session store supports SQLite (default) or Valkey for distributed deployments

## OAuth 2.1 Security

- PKCE required (no plain code challenge)
- Refresh token rotation with replay detection
- Dynamic client registration
- Token revocation endpoint

## Credential Encryption

Device credentials (SSH passwords, SNMP community strings) are encrypted at rest using AES-256-GCM. The encryption key is derived from the server's internal key material.

## Webhook Security

- HMAC-SHA256 signatures on webhook payloads (when a secret is configured)
- SSRF protection: loopback, link-local (169.254.x.x), and unspecified addresses are blocked
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
- `Content-Security-Policy` (see above)

## Audit Logging

When `AUDIT_ENABLED=true`, Rackd logs all API operations including:

- Timestamp, action, resource type, resource ID
- User ID and username (from API key or session)
- Client IP address
- Success/failure status
- Change details (before/after for updates)

Audit logs are retained for 90 days by default (`AUDIT_RETENTION_DAYS`).

Query audit logs:
- `GET /api/audit` — list with filters (action, resource, user, date range)
- `GET /api/audit/export` — export as CSV
- `GET /api/audit/{id}` — get single entry

## Rate Limiting

Two rate limiters protect against abuse:

1. General API: 100 requests/minute per IP (configurable)
2. Login/sensitive endpoints: 5 requests/minute per IP (configurable)

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

Set `TRUST_PROXY=true` when behind a reverse proxy so Rackd reads the real client IP from forwarded headers.

### File Permissions

```bash
chmod 600 /var/lib/rackd/rackd.db
chmod 600 /etc/rackd/.env
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
