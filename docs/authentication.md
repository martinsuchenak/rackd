# Authentication

Rackd supports multiple authentication methods: user sessions (Web UI), API keys (REST API and MCP), and OAuth 2.1 with PKCE (MCP).

## Overview

All API endpoints require authentication. There is no anonymous access mode. On first startup, create an admin user via environment variables (see [Configuration Reference](configuration-reference.md)) or the bootstrap flow.

## Authentication Methods

### 1. Session Authentication (Web UI)

The Web UI uses session-based authentication with secure cookies.

```
POST /api/auth/login
Content-Type: application/json

{"username": "admin", "password": "your-password"}
```

Session properties:
- 24-hour TTL (configurable via `SESSION_TTL`)
- HttpOnly, SameSite=Lax cookies
- `Secure` flag enabled by default (`COOKIE_SECURE=true`)
- CSRF protection via `X-Requested-With: XMLHttpRequest` header on state-changing requests
- Sessions invalidated on password change
- Stored in SQLite or Valkey (`SESSION_STORE_TYPE`)

Related endpoints:
- `POST /api/auth/logout` — destroy session
- `GET /api/auth/me` — get current user info and permissions

### 2. API Keys (REST API and MCP)

API keys are tied to a user account and inherit that user's RBAC permissions.

#### Creating API Keys

```bash
# Via CLI
rackd apikey create --name "ci-pipeline" --user-id USER_ID

# With expiration
rackd apikey create --name "temp-key" --user-id USER_ID --expires "2026-12-31"
```

Or via the API:

```
POST /api/keys
Authorization: Bearer EXISTING_KEY
Content-Type: application/json

{"name": "my-app", "expires_at": "2026-12-31T23:59:59Z"}
```

The key value is returned only once on creation. Store it securely.

#### Using API Keys

Include the key as a Bearer token:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8080/api/devices
```

For the CLI, set in `~/.rackd/config.yaml`:

```yaml
api_url: http://localhost:8080
api_token: YOUR_API_KEY
```

Or via environment variable:

```bash
export RACKD_API_TOKEN=YOUR_API_KEY
```

#### Key Requirements

- Every API key must be associated with a user (`user_id` field)
- Legacy keys without a user association are rejected with 401
- Expired keys are rejected with a specific `EXPIRED_KEY` error code
- Keys are SHA-256 hashed in the database; comparison is constant-time

#### Managing API Keys

```bash
rackd apikey list              # List all keys
rackd apikey delete --id ID    # Delete a key
rackd apikey generate          # Generate a random key offline
```

API endpoints:
- `GET /api/keys` — list keys (key values are never returned)
- `POST /api/keys` — create key (returns key value once)
- `GET /api/keys/{id}` — get key metadata
- `DELETE /api/keys/{id}` — delete key

### 3. OAuth 2.1 with PKCE (MCP)

For MCP clients that support OAuth, Rackd implements OAuth 2.1 with PKCE (Authorization Code flow). This is the recommended auth method for AI clients connecting via MCP.

Enable with:

```bash
export MCP_OAUTH_ENABLED=true
export MCP_OAUTH_ISSUER_URL=https://rackd.example.com
```

OAuth endpoints:
- `GET /.well-known/oauth-authorization-server` — server metadata
- `GET /.well-known/oauth-protected-resource` — protected resource metadata
- `POST /mcp-oauth/register` — dynamic client registration
- `GET /mcp-oauth/authorize` — authorization endpoint
- `POST /mcp-oauth/authorize` — authorization consent submission
- `POST /mcp-oauth/token` — token exchange
- `POST /mcp-oauth/revoke` — token revocation

Client management:
- `GET /api/oauth/clients` — list registered OAuth clients
- `DELETE /api/oauth/clients/{id}` — delete a client

Token properties:
- Access token TTL: 1 hour (configurable via `MCP_OAUTH_ACCESS_TOKEN_TTL`)
- Refresh token TTL: 30 days (configurable via `MCP_OAUTH_REFRESH_TOKEN_TTL`)
- Refresh token rotation with replay detection

## RBAC

All authenticated requests go through role-based access control. Permissions are checked at the service layer using the authenticated user's roles.

Built-in roles:
- `admin` — full access to all resources
- `operator` — read/write access to devices, networks, and discovery
- `viewer` — read-only access

Manage roles and permissions:
- `GET /api/roles` — list roles
- `GET /api/permissions` — list all permissions
- `POST /api/users/grant-role` — assign role to user
- `POST /api/users/revoke-role` — remove role from user
- `GET /api/users/{id}/permissions` — get effective permissions

## User Management

```
GET    /api/users              — list users
POST   /api/users              — create user
GET    /api/users/{id}         — get user
PUT    /api/users/{id}         — update user
DELETE /api/users/{id}         — delete user
POST   /api/users/{id}/password       — change password (requires current password)
POST   /api/users/{id}/reset-password — admin password reset
```

Password requirements:
- Minimum 8 characters
- Hashed with bcrypt (cost 14)
- Password change invalidates all existing sessions

## Security Notes

- Rate limiting is enabled by default (100 req/min API, 5 req/min login)
- Session cookies are `Secure` by default — use `--dev-mode` for local HTTP
- API key last-used timestamps are updated asynchronously for auditing
- All authentication events are recorded in the audit log when `AUDIT_ENABLED=true`

## See Also

- [Configuration Reference](configuration-reference.md) — all environment variables
- [Security](security.md) — security hardening guide
- [MCP](mcp.md) — MCP server integration
