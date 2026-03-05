# Webhook System

Rackd provides a webhook system for event-driven automation and external integrations.

## Overview

Webhooks allow external systems to receive real-time notifications when events occur in Rackd. This enables:

- Automated workflows triggered by device changes
- Integration with monitoring and ticketing systems
- Real-time sync with CMDBs and asset management tools
- Custom alerting and notification pipelines

## Webhook Model

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Descriptive name |
| `url` | string | Target URL for webhook delivery |
| `secret` | string | Secret for HMAC signature (write-only) |
| `events` | []string | List of subscribed events |
| `enabled` | bool | Whether webhook is active |
| `description` | string | Optional description |
| `headers` | map | Custom HTTP headers |
| `timeout_seconds` | int | Request timeout (default: 10) |
| `retry_count` | int | Number of retries on failure (default: 3) |
| `created_at` | timestamp | Creation timestamp |
| `updated_at` | timestamp | Last update timestamp |

## Supported Events

### Device Events
| Event | Description |
|-------|-------------|
| `device.created` | New device added |
| `device.updated` | Device properties changed |
| `device.deleted` | Device removed |
| `device.promoted` | Device promoted from discovered |

### Network Events
| Event | Description |
|-------|-------------|
| `network.created` | New network created |
| `network.updated` | Network properties changed |
| `network.deleted` | Network removed |

### Discovery Events
| Event | Description |
|-------|-------------|
| `discovery.started` | Discovery scan started |
| `discovery.completed` | Discovery scan completed |
| `discovery.device_found` | New device discovered |

### Conflict Events
| Event | Description |
|-------|-------------|
| `conflict.detected` | IP conflict detected |
| `conflict.resolved` | Conflict resolved |

### Pool Events
| Event | Description |
|-------|-------------|
| `pool.utilization_high` | Pool utilization exceeds threshold |

## Webhook Delivery

### Request Format

Webhooks are delivered as HTTP POST requests with JSON payloads:

```http
POST /webhook-endpoint HTTP/1.1
Content-Type: application/json
X-Rackd-Event: device.created
X-Rackd-Delivery: del-abc123
X-Rackd-Signature: sha256=abc123...
X-Rackd-Timestamp: 1709000000

{
  "event": "device.created",
  "delivery_id": "del-abc123",
  "timestamp": "2024-02-27T10:00:00Z",
  "data": {
    "id": "dev-123",
    "hostname": "server01.example.com",
    "ip_address": "192.168.1.10",
    "status": "active"
  }
}
```

### Signature Verification

Each webhook includes an HMAC-SHA256 signature in the `X-Rackd-Signature` header:

```python
import hmac
import hashlib

def verify_signature(payload, signature, secret):
    expected = hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

### Retry Policy

Failed deliveries are retried with exponential backoff:
- Retry intervals: 1m, 5m, 15m, 1h, 6h
- Maximum retry count configurable per webhook
- After max retries, delivery is marked as failed

### Security & SSRF Protection

Rackd implements robust protection against Server-Side Request Forgery (SSRF) and DNS rebinding attacks to ensure the webhook system cannot be exploited to access restricted internal endpoints:
- **Loopback Blocking**: Connections to `127.0.0.0/8` and `::1` are blocked
- **Unspecified Blocking**: Connections to `0.0.0.0` and `::` are blocked 
- **Cloud Metadata Blocking**: Connections to the standard `169.254.x.x` link-local range used by AWS/GCP metadata services are blocked
- **DNS Rebinding Prevention**: Hostnames are resolved first, and Rackd connects strictly to the pre-verified IP address

Note: Standard private intranet subnets (e.g. `10.x.x.x`, `192.168.x.x`) are **permitted** to allow legitimate communication with other internal services on your network.

## API Endpoints

### List Webhooks

```http
GET /api/webhooks
```

### Get Webhook

```http
GET /api/webhooks/{id}
```

### Create Webhook

```http
POST /api/webhooks
```

**Request body:**
```json
{
  "name": "Slack Notifications",
  "url": "https://hooks.slack.com/services/XXX/YYY/ZZZ",
  "secret": "my-webhook-secret",
  "events": ["device.created", "device.deleted", "conflict.detected"],
  "enabled": true,
  "headers": {
    "X-Custom-Header": "value"
  }
}
```

### Update Webhook

```http
PUT /api/webhooks/{id}
```

### Delete Webhook

```http
DELETE /api/webhooks/{id}
```

### Ping Webhook

```http
POST /api/webhooks/{id}/ping
```

Sends a test delivery to verify webhook configuration.

### List Deliveries

```http
GET /api/webhooks/{id}/deliveries
```

Returns delivery history with status and response details.

## CLI Commands

### List Webhooks

```bash
rackd webhook list
```

### Get Webhook

```bash
rackd webhook get --id wh-abc123
```

### Create Webhook

```bash
rackd webhook create \
  --name "Slack Notifications" \
  --url "https://hooks.slack.com/services/XXX/YYY/ZZZ" \
  --secret "my-secret" \
  --events "device.created,device.deleted" \
  --enabled
```

### Update Webhook

```bash
rackd webhook update \
  --id wh-abc123 \
  --events "device.created,device.deleted,device.updated"
```

### Delete Webhook

```bash
rackd webhook delete --id wh-abc123
```

### Test Webhook

```bash
rackd webhook ping --id wh-abc123
```

### List Available Events

```bash
rackd webhook events
```

## Web UI

Access webhook management at `/webhooks` in the web interface.

### Features

- **List View**: Table of all webhooks with status indicators
- **Delivery History**: View delivery attempts and responses
- **Event Selection**: Multi-select for subscribed events
- **Test Delivery**: Ping button to verify configuration
- **Secret Management**: Secure secret input (never displayed after save)

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `webhook:list` | View list of webhooks |
| `webhook:read` | View webhook details (excludes secret) |
| `webhook:create` | Create new webhooks |
| `webhook:update` | Modify existing webhooks |
| `webhook:delete` | Delete webhooks |

## Integration Examples

### Slack Integration

```bash
rackd webhook create \
  --name "Slack - Device Alerts" \
  --url "https://hooks.slack.com/services/T00/B00/XXX" \
  --events "device.created,device.deleted,conflict.detected"
```

### Generic Webhook Handler

```python
from flask import Flask, request, jsonify
import hmac
import hashlib

app = Flask(__name__)
WEBHOOK_SECRET = "your-secret-key"

@app.route('/webhook', methods=['POST'])
def handle_webhook():
    # Verify signature
    signature = request.headers.get('X-Rackd-Signature', '')
    expected = hmac.new(
        WEBHOOK_SECRET.encode(),
        request.data,
        hashlib.sha256
    ).hexdigest()

    if not hmac.compare_digest(f"sha256={expected}", signature):
        return 'Invalid signature', 401

    # Process event
    event = request.json
    print(f"Received: {event['event']}")

    # Handle based on event type
    if event['event'] == 'device.created':
        handle_new_device(event['data'])

    return jsonify({'status': 'ok'}), 200

if __name__ == '__main__':
    app.run(port=5000)
```

### PowerShell Handler

```powershell
# webhook-handler.ps1
param(
    [Parameter(Mandatory=$true)]
    [string]$Payload
)

$event = $Payload | ConvertFrom-Json

switch ($event.event) {
    "device.created" {
        Write-Host "New device: $($event.data.hostname)"
        # Add to AD, update DNS, etc.
    }
    "device.deleted" {
        Write-Host "Device removed: $($event.data.hostname)"
        # Clean up resources
    }
}
```

## Best Practices

1. **Use Secrets**: Always configure a secret for signature verification
2. **Minimal Events**: Subscribe only to events you need
3. **Timeouts**: Set appropriate timeouts for your endpoint
4. **Retries**: Configure retry count based on endpoint reliability
5. **Monitoring**: Check delivery history for failed webhooks
6. **Idempotency**: Handle duplicate deliveries using the delivery_id
7. **HTTPS**: Always use HTTPS endpoints for security
