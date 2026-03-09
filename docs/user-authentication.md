# User Authentication

Rackd supports user authentication via environment variable bootstrapping and password-based login.

## Initial Admin User

On first startup, Rackd can create an initial admin user if you provide credentials via environment variables. This follows industry standards for containerized deployments.

### Environment Variables

```bash
# Required
INITIAL_ADMIN_USERNAME=admin          # Username for initial admin
INITIAL_ADMIN_PASSWORD=securepass     # Password (min 12 characters)

# Optional
INITIAL_ADMIN_EMAIL=admin@example.com  # Email (default: admin@localhost)
INITIAL_ADMIN_FULL_NAME="Admin User" # Full name (default: System Administrator)
SESSION_TTL=24h                     # Session timeout (default: 24h)
SESSION_STORE_TYPE=sqlite           # Session store backend (sqlite, valkey, redis)
VALKEY_URL=redis://localhost:6379/0 # Valkey/Redis URL if used
```

### Docker Example

```bash
docker run -d \
  -e INITIAL_ADMIN_USERNAME=admin \
  -e INITIAL_ADMIN_PASSWORD=mySecurePassword123 \
  -e INITIAL_ADMIN_EMAIL=admin@example.com \
  -p 8080:8080 \
  rackd
```

### Kubernetes Example

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rackd-admin
type: Opaque
stringData:
  username: admin
  password: mySecurePassword123
---
apiVersion: v1
kind: Pod
metadata:
  name: rackd
spec:
  containers:
  - name: rackd
    image: rackd:latest
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

## Session Management

### Session Tokens

After login, users receive a session token that must be included in the Authorization header:

```
Authorization: Bearer <session-token>
```

### Session TTL

Sessions expire after the configured TTL (default: 24 hours). The session is automatically refreshed on each authenticated request.

### Session Persistence

As of Rackd v0.x, sessions are fully persistent across server restarts. By default, they are stored directly in the `sqlite` database. You can also configure a `valkey` or `redis` store for distributed deployments using the `SESSION_STORE_TYPE` and `VALKEY_URL` environment variables.

### Session Invalidation

- All user sessions are invalidated when the user changes their password
- Individual sessions can be revoked via logout
- Expired sessions are automatically cleaned up from the session store

## API Authentication

Rackd supports multiple authentication methods:

1. **Session Tokens** - For interactive web UI use
2. **API Keys** - For programmatic access

### Using Session Tokens

```bash
# Login
TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"securepass"}' \
  | jq -r '.token')

# Use session token
curl http://localhost:8080/api/devices \
  -H "Authorization: Bearer $TOKEN"
```

### Using API Keys

```bash
# Use API key
curl http://localhost:8080/api/devices \
  -H "Authorization: Bearer <api-key>"
```

## User Management via CLI

### Create First Admin (if not using env vars)

```bash
rackd user create \
  --username admin \
  --email admin@example.com
# You'll be prompted for password interactively
```

### List Users

```bash
rackd user list
```

### Update User

```bash
rackd user update --id <user-id> --email new@example.com --full-name "New Name"
```

### Change Password

```bash
rackd user password --id <user-id>
# You'll be prompted for old and new passwords
```

## Security Best Practices

1. **Change Default Password**: After first login, change the initial admin password
2. **Use Secrets Management**: Store admin credentials in secrets managers (Kubernetes Secrets, Vault, etc.)
3. **Enable Audit Logging**: Set `AUDIT_ENABLED=true` to track all user actions
4. **Use Strong Passwords**: Minimum 12 characters with bcrypt hashing (cost factor 14)
5. **Set Appropriate Session TTL**: Adjust based on your security requirements

## Authentication Flow

### 1. First Boot

1. Server checks if users exist in database
2. If no users exist and `INITIAL_ADMIN_USERNAME`/`INITIAL_ADMIN_PASSWORD` are set:
   - Creates admin user
   - Logs success message
3. If no users exist and env vars are not set:
   - Logs warning with instructions
   - Server starts without users (admin must create user via CLI)

### 2. Login

1. Client sends POST to `/api/auth/login` with username and password
2. Server verifies password (bcrypt)
3. Server creates session token
4. Server returns session token and user info
5. Client stores token for subsequent requests

### 3. Authenticated Requests

1. Client includes session token in Authorization header
2. Middleware validates token and refreshes session
3. Request proceeds if token is valid
4. User context available for audit logging

## Troubleshooting

### "No initial admin configured"

If you see this warning at startup:

1. Set environment variables for initial admin
2. Or create admin via CLI: `rackd user create`
3. Restart server

### Cannot Login

1. Verify username/password are correct
2. Check if user is active: `rackd user list`
3. Verify session hasn't expired
4. Check server logs for errors

### Password Rejected

Passwords must be at least 8 characters long. Use a stronger password.

## Migration from API Keys

Existing API keys continue to work. You can use both authentication methods:

- Use API keys for automated scripts/CI/CD
- Use session tokens for interactive web UI use
