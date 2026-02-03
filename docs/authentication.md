# Authentication

Rackd supports API key authentication for securing access to the REST API and MCP server.

## Overview

Authentication in Rackd is **optional by default**. When no authentication is configured, the API is open to all requests. This is suitable for:
- Development and testing environments
- Single-user deployments
- Trusted network environments

For production deployments or multi-user environments, you can enable authentication by creating API keys.

## API Keys

API keys provide a secure way to authenticate API requests without requiring user accounts or passwords.

### Features

- **Secure**: 256-bit random keys with timing-safe comparison
- **Expirable**: Optional expiration dates
- **Trackable**: Last-used timestamps for auditing
- **Manageable**: Full CRUD operations via CLI and API
- **Flexible**: Works with both REST API and MCP server

### Creating API Keys

Use the CLI to create API keys:

```bash
# Create a basic API key
rackd apikey create --name "my-app"

# Create with description
rackd apikey create --name "ci-pipeline" --description "CI/CD automation"

# Create with expiration date
rackd apikey create --name "temp-key" --expires "2026-12-31"
```

**Output:**
```
API Key created successfully!

ID:   550e8400-e29b-41d4-a716-446655440000
Name: my-app
Key:  dGhpcyBpcyBhIHNhbXBsZSBrZXkgZm9yIGRvY3VtZW50YXRpb24=

⚠️  Save this key securely - it will not be shown again!
```

**Important:** The actual key is only shown once during creation. Store it securely.

### Listing API Keys

```bash
rackd apikey list
```

**Output:**
```
ID                                    NAME         DESCRIPTION        CREATED              LAST USED            EXPIRES
550e8400-e29b-41d4-a716-446655440000  my-app       My application     2026-02-03 13:00     2026-02-03 13:15     never
660e8400-e29b-41d4-a716-446655440001  ci-pipeline  CI/CD automation   2026-02-03 12:00     never                2026-12-31
```

### Deleting API Keys

```bash
rackd apikey delete --id 550e8400-e29b-41d4-a716-446655440000
```

### Generating Keys Offline

Generate a random key without creating it in the database:

```bash
rackd apikey generate
```

This is useful for testing or when you want to generate a key before creating it via the API.

## Using API Keys

### REST API

Include the API key in the `Authorization` header as a Bearer token:

```bash
curl -H "Authorization: Bearer YOUR_API_KEY" \
  http://localhost:8080/api/devices
```

### CLI

Configure the CLI to use an API key by editing `~/.rackd/config.yaml`:

```yaml
api_url: http://localhost:8080
api_token: YOUR_API_KEY
timeout: 30s
```

Or set via environment variable:

```bash
export RACKD_API_TOKEN=YOUR_API_KEY
rackd device list
```

### MCP Server

Pass the API key in the Authorization header when connecting to the MCP endpoint:

```bash
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"method":"tools/list"}'
```

### Web UI

The Web UI currently does not require authentication. When full user management is implemented, the Web UI will support login with API keys or user credentials.

## API Key Management API

API keys can also be managed via the REST API (requires authentication if enabled):

### List API Keys

```bash
GET /api/keys
```

**Response:**
```json
[
  {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "name": "my-app",
    "description": "My application",
    "created_at": "2026-02-03T13:00:00Z",
    "last_used_at": "2026-02-03T13:15:00Z",
    "expires_at": null
  }
]
```

### Create API Key

```bash
POST /api/keys
Content-Type: application/json

{
  "name": "my-app",
  "description": "My application",
  "expires_at": "2026-12-31T23:59:59Z"
}
```

**Response:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "my-app",
  "key": "dGhpcyBpcyBhIHNhbXBsZSBrZXkgZm9yIGRvY3VtZW50YXRpb24=",
  "description": "My application",
  "created_at": "2026-02-03T13:00:00Z",
  "expires_at": "2026-12-31T23:59:59Z"
}
```

**Note:** The `key` field is only returned on creation.

### Get API Key

```bash
GET /api/keys/{id}
```

### Delete API Key

```bash
DELETE /api/keys/{id}
```

## Security Best Practices

### Key Storage

- **Never commit keys to version control**
- Store keys in environment variables or secure configuration files
- Use different keys for different environments (dev, staging, prod)
- Rotate keys periodically

### Key Management

- Create keys with descriptive names
- Use expiration dates for temporary access
- Delete unused keys immediately
- Monitor last-used timestamps to identify inactive keys

### Access Control

- Create separate keys for different applications or users
- Use the principle of least privilege (when RBAC is implemented)
- Revoke keys immediately when no longer needed

## Authentication Flow

```
┌─────────┐                    ┌─────────┐                    ┌──────────┐
│ Client  │                    │  Rackd  │                    │ Database │
└────┬────┘                    └────┬────┘                    └────┬─────┘
     │                              │                              │
     │  GET /api/devices            │                              │
     │  Authorization: Bearer KEY   │                              │
     ├─────────────────────────────>│                              │
     │                              │                              │
     │                              │  SELECT * FROM api_keys      │
     │                              │  WHERE key = ?               │
     │                              ├─────────────────────────────>│
     │                              │                              │
     │                              │  Key found, not expired      │
     │                              │<─────────────────────────────┤
     │                              │                              │
     │                              │  UPDATE last_used_at         │
     │                              ├─────────────────────────────>│
     │                              │                              │
     │                              │  Process request             │
     │                              │                              │
     │  200 OK + Response           │                              │
     │<─────────────────────────────┤                              │
     │                              │                              │
```

## Migration from Legacy Tokens

**Note**: Legacy `API_AUTH_TOKEN` and `MCP_AUTH_TOKEN` environment variables have been removed. Use API keys instead.

If you need authentication:

1. Create an API key:
   ```bash
   rackd apikey create --name "admin"
   ```

2. Use the key in your requests:
   ```bash
   curl -H "Authorization: Bearer YOUR_KEY" http://localhost:8080/api/devices
   ```

3. Configure CLI:
   ```bash
   echo "api_token: YOUR_KEY" >> ~/.rackd/config.yaml
   ```

## Future Enhancements

The current API key system provides a foundation for future authentication features:

- **User Management**: Full user accounts with passwords
- **Role-Based Access Control (RBAC)**: Permissions and roles
- **SSO/OIDC**: Integration with identity providers
- **Session Management**: Web UI login sessions
- **Audit Logging**: Track all authentication events
- **API Key Scopes**: Limit key permissions to specific resources

When these features are implemented, API keys will become **required** for all API access.

## Troubleshooting

### "Unauthorized" Error

If you receive a 401 Unauthorized error:

1. Verify the API key is correct
2. Check the key hasn't expired
3. Ensure the Authorization header is properly formatted:
   ```
   Authorization: Bearer YOUR_API_KEY
   ```
4. Verify the key exists:
   ```bash
   rackd apikey list
   ```

### Key Not Working

If your API key isn't working:

1. Check if the key has expired:
   ```bash
   rackd apikey list
   ```
2. Verify the key was copied correctly (no extra spaces or newlines)
3. Try creating a new key and testing with that

### CLI Authentication Issues

If the CLI can't authenticate:

1. Check `~/.rackd/config.yaml` exists and contains `api_token`
2. Verify the API URL is correct
3. Test the key directly with curl:
   ```bash
   curl -H "Authorization: Bearer YOUR_KEY" http://localhost:8080/api/devices
   ```

## Examples

### Automation Script

```bash
#!/bin/bash
# Create API key for automation
KEY=$(rackd apikey create --name "automation-$(date +%Y%m%d)" | grep "Key:" | awk '{print $2}')

# Use the key
export RACKD_API_TOKEN=$KEY

# Run automation tasks
rackd device list
rackd network list

# Cleanup (optional)
# rackd apikey delete --id $KEY_ID
```

### CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
env:
  RACKD_API_TOKEN: ${{ secrets.RACKD_API_KEY }}

steps:
  - name: Update device inventory
    run: |
      rackd device create \
        --name "web-server-${{ github.run_number }}" \
        --datacenter "$DATACENTER_ID"
```

### Python Script

```python
import requests

API_KEY = "your-api-key-here"
BASE_URL = "http://localhost:8080"

headers = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}

# List devices
response = requests.get(f"{BASE_URL}/api/devices", headers=headers)
devices = response.json()

# Create device
device = {
    "name": "new-server",
    "description": "Created via API"
}
response = requests.post(f"{BASE_URL}/api/devices", json=device, headers=headers)
```

## See Also

- [API Reference](api.md) - Complete API documentation
- [CLI Reference](cli.md) - CLI command documentation
- [Security](security.md) - Security best practices
- [Deployment](deployment.md) - Production deployment guide
