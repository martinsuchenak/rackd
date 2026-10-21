# Authentication

Rackd supports multiple authentication methods: user sessions (Web UI), API keys (REST API and MCP), and OAuth 2.1 with PKCE (MCP).

## Overview

All API endpoints require authentication. There is no anonymous access mode. The first admin user comes from environment variables; see [Initial Admin User](#initial-admin-user).

## Initial Admin User

When the database has no users, the server creates an admin from these variables at startup:

```bash
INITIAL_ADMIN_USERNAME=admin              # Required
INITIAL_ADMIN_PASSWORD='<a strong password>'  # Required, at least 8 characters
INITIAL_ADMIN_EMAIL=admin@example.com     # Optional (default: admin@localhost)
INITIAL_ADMIN_FULL_NAME="Admin User"      # Optional (default: System Administrator)
```

They have no effect once any user exists. If they are unset when there are no users, the server starts without users and logs a warning; set them and restart. `rackd user create` can't create the first admin, because it calls the API and needs an API key.

Docker:

```bash
docker run -d \
  -e INITIAL_ADMIN_USERNAME=admin \
  -e INITIAL_ADMIN_PASSWORD='<a strong password>' \
  -e INITIAL_ADMIN_EMAIL=admin@example.com \
  -p 8080:8080 \
  ghcr.io/martinsuchenak/rackd:0.6
```

Kubernetes, with the credentials in a Secret:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rackd-admin
type: Opaque
stringData:
  username: admin
  password: <a strong password>
---
# In the rackd container spec:
env:
- name: INITIAL_ADMIN_USERNAME
  valueFrom:
    secretKeyRef:
      name: rackd-admin
      key: username
- name: INITIAL_ADMIN_PASSWORD
  valueFrom:
    secretKeyRef:
      name: rackd-admin
      key: password
```

Change the initial password after the first sign-in. See [Deployment](deployment.md) for a full container setup, including `ENCRYPTION_KEY`.

## Authentication Methods

### 1. Session Authentication (Web UI)

The Web UI uses session-based authentication with secure cookies.

```
POST /api/v1/auth/login
Content-Type: application/json

{"username": "admin", "password": "your-password"}
```

Login requires `Content-Type: application/json` (other types get `415`), so a page on another site can't submit a form that signs a visitor into a different account.

Session properties:
- Expire 24 hours after last use (`SESSION_TTL`), and 7 days after sign-in however often they are used (`SESSION_MAX_LIFETIME`)
- HttpOnly, SameSite=Strict cookies
- `Secure` flag enabled by default (`COOKIE_SECURE=true`)
- CSRF protection via `X-Requested-With: XMLHttpRequest` header on state-changing requests
- End when the user changes or has their password reset, is deactivated or deleted; every request checks that the user still exists and is active
- Stored in SQLite or Valkey (`SESSION_STORE_TYPE`)

Sign-in is limited per address (`LOGIN_RATE_LIMIT_REQUESTS` per `LOGIN_RATE_LIMIT_WINDOW`) and per account: after `LOGIN_ACCOUNT_MAX_FAILURES` failed attempts (default 10) within `LOGIN_ACCOUNT_WINDOW` (default 15 minutes), further attempts on that account get `429 TOO_MANY_ATTEMPTS`, even with the right password, until the window passes. Unknown usernames are counted the same way. Each Rackd instance counts separately. Like any account lockout, this lets someone who knows a username keep that account locked by failing on purpose; if that is a bigger risk for you than password guessing, set `LOGIN_ACCOUNT_MAX_FAILURES=0` and rely on the per-address limit.

Related endpoints:
- `POST /api/v1/auth/logout` — destroy session
- `GET /api/v1/auth/me` — get current user info and permissions

### 2. API Keys (REST API and MCP)

An API key belongs to the user who creates it. By default it can do everything
that user can; give it **scopes** to limit it to some of their permissions, so
an integration that only reads inventory can't change anything. Scopes are
`resource:action` permissions (as in [RBAC](rbac.md)), each one you hold
yourself, and a scoped key never gains a permission its owner loses. Use the
resource name, such as `devices:list`, not the permission name `device:list`.

#### Creating API Keys

```bash
# A key with all your permissions
rackd apikey create --name "ci-pipeline"

# A read-only inventory key that expires
rackd apikey create --name "inventory-export" --scopes devices:list,devices:read --expires "2026-12-31"
```

Or via the API:

```
POST /api/v1/keys
Authorization: Bearer EXISTING_KEY
Content-Type: application/json

{"name": "inventory-export", "scopes": ["devices:list", "devices:read"], "expires_at": "2026-12-31T23:59:59Z"}
```

The request accepts only `name`, `description`, `expires_at` and `scopes`;
other fields are refused with `400` rather than ignored. A scope you don't hold,
or an empty `scopes` list, is refused too. The web UI's Create API Key dialog
offers the same choice, and the API Keys page marks keys with full access.

The key value is returned only once on creation. Store it securely.

#### Using API Keys

Include the key as a Bearer token:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" http://localhost:8080/api/v1/devices
```

For the CLI, set it in `~/.config/rackd/config.json` (or `$XDG_CONFIG_HOME/rackd/config.json`):

```json
{
  "server_url": "http://localhost:8080",
  "token": "YOUR_API_KEY"
}
```

Or via environment variable:

```bash
export RACKD_TOKEN=YOUR_API_KEY
```

#### Key Requirements

- Every API key belongs to a user (`user_id`), and can do at most what that user can
- A scoped key is checked against its scopes as well as its owner's permissions, on the REST API and over MCP
- Legacy keys without a user association are rejected with 401 `LEGACY_API_KEY_UNSUPPORTED`
- Expired keys are rejected with a specific `EXPIRED_KEY` error code
- Keys are SHA-256 hashed in the database; comparison is constant-time

#### Managing API Keys

```bash
rackd apikey list              # List all keys
rackd apikey delete --id ID    # Delete a key
rackd apikey generate          # Generate a random key offline
```

API endpoints:
- `GET /api/v1/keys` — list keys (key values are never returned)
- `POST /api/v1/keys` — create key (returns key value once)
- `GET /api/v1/keys/{id}` — get key metadata
- `DELETE /api/v1/keys/{id}` — delete key

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
- `GET /api/v1/oauth/clients` — list registered OAuth clients
- `DELETE /api/v1/oauth/clients/{id}` — delete a client

Token properties:
- Access token TTL: 1 hour (configurable via `MCP_OAUTH_ACCESS_TOKEN_TTL`)
- Refresh token TTL: 30 days (configurable via `MCP_OAUTH_REFRESH_TOKEN_TTL`)
- Refresh token rotation with replay detection: a refresh token works once. If it is used again, or twice at the same moment, every token issued from it is revoked, including those of whoever used it first.
- A password change or reset, deactivation or deletion revokes the user's OAuth tokens. API keys are kept; delete them separately if they may be compromised.
- Clients register themselves, so the consent page warns that their name isn't verified and shows the host the approval sends access to.
- The consent page lists only the scopes the client and the signed-in user can grant, and the approval is bound to that screen: a changed scope, redirect URI or PKCE challenge is refused.

## RBAC

All authenticated requests go through role-based access control. Permissions are checked at the service layer using the authenticated user's roles.

Built-in roles:
- `admin` — full access to all resources
- `operator` — create, read and update most inventory, IPAM, discovery and DNS resources; no deletes, and no user, role, API key or webhook management
- `viewer` — read-only access to inventory, IPAM, discovery and DNS resources

See [RBAC](rbac.md#default-roles) for the exact permissions.

Manage roles and permissions:
- `GET /api/v1/roles` — list roles
- `GET /api/v1/permissions` — list all permissions
- `POST /api/v1/users/grant-role` — assign role to user
- `POST /api/v1/users/revoke-role` — remove role from user
- `GET /api/v1/users/{id}/permissions` — get effective permissions

## User Management

```
GET    /api/v1/users              — list users
POST   /api/v1/users              — create user
GET    /api/v1/users/{id}         — get user
PUT    /api/v1/users/{id}         — update user
DELETE /api/v1/users/{id}         — delete user
POST   /api/v1/users/{id}/password       — change password (requires current password)
POST   /api/v1/users/{id}/reset-password — admin password reset
```

Password requirements:
- Minimum 8 characters
- Hashed with bcrypt (cost 14)
- Password change invalidates all existing sessions

### Recovering a lost admin password

If the only admin's password is lost and no other admin or API key exists,
reset it offline with the CLI — it writes to the database directly, so the
server must be stopped first:

```bash
rackd user reset-password --data-dir ./data --username admin
```

See [`rackd user reset-password`](cli-reference.md#rackd-user-reset-password)
for its flags. The command prompts for the new password (hidden input),
applies the same bcrypt hashing as the server, and invalidates all active
sessions for the user.

## Troubleshooting

### "No initial admin configured"

The server found no users and no `INITIAL_ADMIN_USERNAME`/`INITIAL_ADMIN_PASSWORD`. Set both and restart.

### Cannot sign in

1. Check the username and password
2. Check the account is active (`rackd user list`, as another admin)
3. After repeated failures the account is locked for `LOGIN_ACCOUNT_WINDOW` (`429 TOO_MANY_ATTEMPTS`); wait, or see [Recovering a lost admin password](#recovering-a-lost-admin-password)
4. Check the server log

### Password rejected

Passwords must be at least 8 characters.

## Security Notes

- Rate limiting is enabled by default (100 req/min API, 5 req/min login). Login, OAuth endpoints, API key and user management, password changes and bulk operations each have their own per-address limit with the login settings, so one can't exhaust another
- Session cookies are `Secure` by default — use `--dev-mode` for local HTTP
- API key last-used timestamps are updated asynchronously for auditing, at most once a minute per key, so they are accurate to about a minute
- Sign-ins (successful, failed and refused, with the address and reason), sign-outs, password changes and resets, and user, role and API key changes are recorded in the [audit log](audit.md)

## See Also

- [Configuration Reference](configuration-reference.md) — all environment variables
- [Security](security.md) — security hardening guide
- [MCP](mcp.md) — MCP server integration
