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
| `url` | string | Target URL (`http` or `https`) |
| `secret` | string | Secret for the HMAC signature; write-only |
| `has_secret` | bool | Whether a secret is set |
| `events` | []string | Subscribed event types |
| `active` | bool | Whether deliveries are sent |
| `description` | string | Optional description |
| `created_by` | string | ID of the user who created it |
| `created_at` | timestamp | Creation time |
| `updated_at` | timestamp | Last update time |

## Supported Events

Events are published after the change is stored. `data` is the record as it was at that moment; for deletions, as it was just before.

| Event | Published when | `data` |
|-------|----------------|--------|
| `device.created` | A device is created, including by bulk create and by promoting a discovered device | Device |
| `device.updated` | A device is updated, including by bulk update and bulk tag changes | Device |
| `device.deleted` | A device is deleted, including by bulk delete | Device, as it was |
| `device.promoted` | A discovered device is promoted to a device (`device.created` is published too) | Device |
| `network.created` | A network is created, including by bulk create | Network |
| `network.updated` | A network is updated | Network |
| `network.deleted` | A network is deleted, including by bulk delete | Network, as it was |
| `discovery.started` | A discovery scan starts running | Scan |
| `discovery.completed` | A discovery scan completes; failed and cancelled scans publish nothing | Scan |
| `discovery.device_found` | A scan finds a host it hasn't seen before | Discovered device |
| `conflict.detected` | Conflict detection records a new conflict | Conflict |
| `conflict.resolved` | A conflict is resolved | Conflict |
| `pool.utilization_high` | Not published yet; subscribing is accepted | |

Extensions can add their own event types; `GET /api/v1/webhooks/events` lists every type the server accepts.

## Webhook Delivery

### Request Format

Each event is sent as a JSON `POST` to every active webhook subscribed to it:

```http
POST /webhook-endpoint HTTP/1.1
Content-Type: application/json
User-Agent: Rackd-Webhook/0.2.0
X-Rackd-Event: device.created
X-Rackd-Delivery: 01a13a74-5c2e-7c11-9d0e-3f1b2c4d5e6f
X-Rackd-Timestamp: 1792152000
X-Rackd-Signature: sha256=5d41402abc4b2a76b9719d911017c592...

{
  "event": "device.created",
  "delivery_id": "01a13a74-5c2e-7c11-9d0e-3f1b2c4d5e6f",
  "timestamp": "2026-10-14T12:00:00Z",
  "data": {
    "id": "01a13a74-1b2c-7d3e-8f40-516273849a0b",
    "name": "server01",
    "hostname": "server01.example.com",
    "status": "active",
    "addresses": [{"ip": "192.168.1.10", "type": "ipv4"}]
  }
}
```

| Header | Value |
|--------|-------|
| `X-Rackd-Event` | The event type |
| `X-Rackd-Delivery` | The delivery ID, also `delivery_id` in the body. It stays the same across retries, so use it to ignore duplicates. |
| `X-Rackd-Timestamp` | When this attempt was sent, in Unix seconds. Each retry has a new one. |
| `X-Rackd-Signature` | `sha256=` and the signature, when the webhook has a secret |

The body's `timestamp` is when the event happened. The test ping (`POST /api/v1/webhooks/{id}/ping`) sends a `webhook.ping` event in the same format.

### Signature Verification

The signature is the hex HMAC-SHA256, keyed with the webhook's secret, of the timestamp header, a `.`, and the raw request body:

```text
X-Rackd-Signature = "sha256=" + hex(HMAC-SHA256(secret, X-Rackd-Timestamp + "." + body))
```

To verify a request:

1. Compute the signature from the raw body bytes, before parsing the JSON.
2. Compare it with `X-Rackd-Signature` in constant time.
3. Reject requests whose `X-Rackd-Timestamp` is more than 5 minutes from your clock, so a captured request can't be replayed later.

```python
import hashlib
import hmac
import time

def verify(secret: str, headers, body: bytes, tolerance: int = 300) -> bool:
    timestamp = headers.get("X-Rackd-Timestamp", "")
    if not timestamp.isdigit() or abs(time.time() - int(timestamp)) > tolerance:
        return False
    expected = hmac.new(secret.encode(), timestamp.encode() + b"." + body, hashlib.sha256).hexdigest()
    return hmac.compare_digest("sha256=" + expected, headers.get("X-Rackd-Signature", ""))
```

```go
func verify(secret string, r *http.Request, body []byte) bool {
	ts := r.Header.Get("X-Rackd-Timestamp")
	sent, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || math.Abs(float64(time.Now().Unix()-sent)) > 300 {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(r.Header.Get("X-Rackd-Signature")))
}
```

### Retry Policy

Any response other than 2xx, a timeout (30 seconds) or a connection error counts as a failure. A failed delivery is retried with the same body and delivery ID:

- Up to 5 attempts in all.
- About 20 seconds, 40 seconds, 80 seconds and 160 seconds apart. Retries are checked every 30 seconds, so each wait can be up to 30 seconds longer.
- After the last attempt the delivery is marked `failed`. Deliveries to a webhook that was deleted or deactivated in the meantime are marked `abandoned`.
- Deliveries in progress when the server shuts down are retried after it starts again. Delivery is best-effort across a crash: an event published moments before the process is killed may not be delivered.
- Delivery records are kept for 30 days. `GET /api/v1/webhooks/{id}/deliveries` lists them.

### Security & SSRF Protection

Webhook URLs are checked when a webhook is saved, and again when each request connects, so DNS rebinding can't redirect a delivery:

- Loopback (`127.0.0.0/8`, `::1`), unspecified (`0.0.0.0`, `::`), link-local (`169.254.0.0/16`, which includes cloud metadata services) and multicast addresses are always blocked.
- Private ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, CGNAT `100.64.0.0/10`, and IPv6 ULA) are blocked unless `SSRF_ALLOW_PRIVATE_TARGETS=true`. Set it to deliver to services on your own network.

## API Endpoints

### List Webhooks

```http
GET /api/v1/webhooks
```

### Get Webhook

```http
GET /api/v1/webhooks/{id}
```

### Create Webhook

```http
POST /api/v1/webhooks
```

**Request body:**
```json
{
  "name": "Ops relay",
  "url": "https://relay.example.com/rackd-webhook",
  "secret": "my-webhook-secret",
  "events": ["device.created", "device.deleted", "conflict.detected"],
  "active": true,
  "description": "Device changes for the ops channel"
}
```

### Update Webhook

```http
PUT /api/v1/webhooks/{id}
```

### Delete Webhook

```http
DELETE /api/v1/webhooks/{id}
```

### Ping Webhook

```http
POST /api/v1/webhooks/{id}/ping
```

Sends a test delivery to verify webhook configuration.

### List Deliveries

```http
GET /api/v1/webhooks/{id}/deliveries
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
  --name "CMDB sync" \
  --url "https://cmdb.example.com/rackd-webhook" \
  --secret "my-secret" \
  --events "device.created,device.deleted"
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

### Chat Tools

Slack, Microsoft Teams and similar incoming webhooks expect their own message format, so they can't receive Rackd's events directly. Point the Rackd webhook at a small relay (such as the handler below) that verifies the signature and posts a message to the chat tool.

### Generic Webhook Handler

```python
import hashlib
import hmac
import time

from flask import Flask, abort, request

app = Flask(__name__)
WEBHOOK_SECRET = "your-secret-key"
seen_deliveries = set()  # use a persistent store in production


def verify(headers, body: bytes) -> bool:
    timestamp = headers.get("X-Rackd-Timestamp", "")
    if not timestamp.isdigit() or abs(time.time() - int(timestamp)) > 300:
        return False
    expected = hmac.new(WEBHOOK_SECRET.encode(), timestamp.encode() + b"." + body, hashlib.sha256).hexdigest()
    return hmac.compare_digest("sha256=" + expected, headers.get("X-Rackd-Signature", ""))


@app.route("/webhook", methods=["POST"])
def handle_webhook():
    if not verify(request.headers, request.get_data()):
        abort(401)

    event = request.get_json()
    if event["delivery_id"] in seen_deliveries:
        return "", 204  # a retry of something already handled
    seen_deliveries.add(event["delivery_id"])

    if event["event"] == "device.created":
        print("New device:", event["data"]["name"])
    return "", 204


if __name__ == "__main__":
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

1. **Use a secret** and verify every request's signature and timestamp.
2. **Subscribe only to the events you need.**
3. **Respond quickly**, within 30 seconds, with a 2xx status. Do slow work after responding.
4. **Handle duplicates**: a retry reuses the delivery ID, so record the IDs you've processed.
5. **Watch the delivery history** for failed and abandoned deliveries.
6. **Use HTTPS endpoints.**
