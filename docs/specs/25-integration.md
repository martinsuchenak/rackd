# Integration and Extensibility

This document defines webhook system, external service integrations, API rate limiting, and bulk operations for Rackd.

## 1. Webhook System

### 1.1 Webhook Configuration API

**Webhook Model:**

```go
package model

import "time"

type Webhook struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    URL         string    `json:"url"`
    Events      []string  `json:"events"` // List of event types
    Secret      string    `json:"secret"` // HMAC secret
    ContentType string    `json:"content_type"` // application/json, application/x-www-form-urlencoded
    Enabled     bool      `json:"enabled"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type WebhookEvent struct {
    ID        string                 `json:"id"`
    Type      string                 `json:"type"`     // device.created, device.updated, etc.
    Timestamp time.Time               `json:"timestamp"`
    Data      map[string]interface{} `json:"data"`
}

type WebhookDelivery struct {
    ID        string    `json:"id"`
    WebhookID string    `json:"webhook_id"`
    Event     *WebhookEvent
    Success   bool      `json:"success"`
    Status    string    `json:"status"`   // success, failed, retrying
    Attempts  int       `json:"attempts"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 1.2 Event Types and Payloads

**Device Events:**

```json
// device.created
{
  "id": "evt_abc123",
  "type": "device.created",
  "timestamp": "2024-01-20T08:00:00Z",
  "data": {
    "device": {
      "id": "dev_123",
      "name": "server-01",
      "description": "Web server",
      "make_model": "Dell R740",
      "os": "Ubuntu 22.04",
      "datacenter_id": "dc_456",
      "username": "admin",
      "location": "Rack A1",
      "tags": ["web", "prod"],
      "created_at": "2024-01-20T08:00:00Z"
    }
  }
}

// device.updated
{
  "id": "evt_def456",
  "type": "device.updated",
  "timestamp": "2024-01-20T09:00:00Z",
  "data": {
    "device": { /* updated device object */ },
    "changes": {
      "name": {"old": "server-01", "new": "web-server-01"},
      "description": {"old": "Web server", "new": "Primary web server"}
    }
  }
}

// device.deleted
{
  "id": "evt_ghi789",
  "type": "device.deleted",
  "timestamp": "2024-01-20T10:00:00Z",
  "data": {
    "device": {
      "id": "dev_123",
      "name": "web-server-01"
    }
  }
}
```

**Network Events:**

```json
// network.created
{
  "type": "network.created",
  "data": {
    "network": {
      "id": "net_123",
      "name": "production",
      "subnet": "192.168.1.0/24",
      "vlan_id": 100,
      "datacenter_id": "dc_456",
      "description": "Production network"
    }
  }
}

// network.utilization.high
{
  "type": "network.utilization.high",
  "data": {
    "network": { /* network object */ },
    "utilization": {
      "total_ips": 254,
      "used_ips": 229,
      "utilization_percent": 90.2,
      "threshold": 80
    }
  }
}
```

**Discovery Events:**

```json
// discovery.scan.started
{
  "type": "discovery.scan.started",
  "data": {
    "scan": {
      "id": "scan_123",
      "network_id": "net_456",
      "scan_type": "full",
      "started_at": "2024-01-20T11:00:00Z"
    }
  }
}

// discovery.scan.completed
{
  "type": "discovery.scan.completed",
  "data": {
    "scan": {
      "id": "scan_123",
      "network_id": "net_456",
      "scan_type": "full",
      "status": "completed",
      "total_hosts": 254,
      "scanned_hosts": 254,
      "found_hosts": 42,
      "started_at": "2024-01-20T11:00:00Z",
      "completed_at": "2024-01-20T11:12:34Z"
    }
  }
}

// discovery.device.found
{
  "type": "discovery.device.found",
  "data": {
    "discovered_device": {
      "id": "disc_789",
      "ip": "192.168.1.50",
      "mac_address": "00:11:22:33:44:55",
      "hostname": "unknown-host",
      "status": "online",
      "confidence": 95,
      "first_seen": "2024-01-20T11:05:00Z"
    }
  }
}
```

**All Event Types:**

| Category | Event Type | Description |
|----------|-------------|-------------|
| Device | `device.created` | New device created |
| Device | `device.updated` | Device modified |
| Device | `device.deleted` | Device deleted |
| Network | `network.created` | New network created |
| Network | `network.updated` | Network modified |
| Network | `network.deleted` | Network deleted |
| Network | `network.utilization.high` | Network utilization exceeds threshold |
| Datacenter | `datacenter.created` | New datacenter created |
| Datacenter | `datacenter.updated` | Datacenter modified |
| Datacenter | `datacenter.deleted` | Datacenter deleted |
| Discovery | `discovery.scan.started` | Discovery scan started |
| Discovery | `discovery.scan.completed` | Discovery scan completed |
| Discovery | `discovery.scan.failed` | Discovery scan failed |
| Discovery | `discovery.device.found` | New device discovered |
| Discovery | `discovery.device.promoted` | Discovered device promoted |

### 1.3 Delivery Mechanism

```go
package webhook

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "time"
)

type WebhookDeliverer struct {
    client    *http.Client
    storage   storage.WebhookStorage
    retry     *RetryPolicy
}

type RetryPolicy struct {
    MaxAttempts   int           `json:"max_attempts"`
    InitialDelay time.Duration `json:"initial_delay"`
    MaxDelay     time.Duration `json:"max_delay"`
    BackoffFactor float64       `json:"backoff_factor"`
}

func NewWebhookDeliverer(storage storage.WebhookStorage) *WebhookDeliverer {
    return &WebhookDeliverer{
        client:  &http.Client{Timeout: 30 * time.Second},
        storage: storage,
        retry: &RetryPolicy{
            MaxAttempts:   5,
            InitialDelay: 1 * time.Second,
            MaxDelay:     5 * time.Minute,
            BackoffFactor: 2.0,
        },
    }
}

func (wd *WebhookDeliverer) Deliver(ctx context.Context, event *model.WebhookEvent) error {
    // Get webhooks for this event type
    webhooks, err := wd.storage.GetWebhooksForEvent(event.Type)
    if err != nil {
        return err
    }

    // Deliver to each webhook
    for _, webhook := range webhooks {
        if !webhook.Enabled {
            continue
        }

        go wd.deliverWithRetry(ctx, webhook, event)
    }

    return nil
}

func (wd *WebhookDeliverer) deliverWithRetry(ctx context.Context, webhook *model.Webhook, event *model.WebhookEvent) {
    var lastErr error
    delay := wd.retry.InitialDelay

    for attempt := 1; attempt <= wd.retry.MaxAttempts; attempt++ {
        err := wd.deliverSingle(ctx, webhook, event)
        if err == nil {
            // Success - record delivery
            wd.recordDelivery(webhook.ID, event.ID, true, attempt)
            return
        }

        lastErr = err
        log.Warn("Webhook delivery failed",
            "webhook", webhook.ID,
            "url", webhook.URL,
            "attempt", attempt,
            "error", err)

        // Record failed delivery
        wd.recordDelivery(webhook.ID, event.ID, false, attempt)

        // Backoff before retry
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
            delay = time.Duration(float64(delay) * wd.retry.BackoffFactor)
            if delay > wd.retry.MaxDelay {
                delay = wd.retry.MaxDelay
            }
        }
    }

    log.Error("Webhook delivery failed after all retries",
        "webhook", webhook.ID,
        "url", webhook.URL,
        "attempts", wd.retry.MaxAttempts,
        "error", lastErr)
}

func (wd *WebhookDeliverer) deliverSingle(ctx context.Context, webhook *model.Webhook, event *model.WebhookEvent) error {
    // Prepare payload
    payload, err := json.Marshal(event)
    if err != nil {
        return err
    }

    // Create request
    req, err := http.NewRequestWithContext(ctx, "POST", webhook.URL, bytes.NewReader(payload))
    if err != nil {
        return err
    }

    // Set headers
    req.Header.Set("Content-Type", webhook.ContentType)
    req.Header.Set("X-Rackd-Webhook-Event", event.Type)
    req.Header.Set("X-Rackd-Webhook-ID", event.ID)

    // Add signature if secret provided
    if webhook.Secret != "" {
        signature := wd.calculateSignature(payload, webhook.Secret)
        req.Header.Set("X-Rackd-Webhook-Signature", signature)
    }

    // Send request
    resp, err := wd.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Check response
    if resp.StatusCode >= 200 && resp.StatusCode < 300 {
        return nil // Success
    }

    return fmt.Errorf("webhook returned status %d", resp.StatusCode)
}

func (wd *WebhookDeliverer) calculateSignature(payload []byte, secret string) string {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(payload)
    return hex.EncodeToString(h.Sum(nil))
}

func (wd *WebhookDeliverer) recordDelivery(webhookID, eventID string, success bool, attempt int) {
    delivery := &model.WebhookDelivery{
        ID:        uuid.New().String(),
        WebhookID: webhookID,
        Event:     &model.WebhookEvent{ID: eventID},
        Success:   success,
        Status:    map[bool]string{true: "success", false: "failed"}[success],
        Attempts:  attempt,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := wd.storage.CreateWebhookDelivery(delivery); err != nil {
        log.Error("Failed to record webhook delivery", "error", err)
    }
}
```

### 1.4 Retry Policy

**Exponential Backoff with Jitter:**

| Attempt | Delay Range |
|---------|-------------|
| 1 | 0-1s |
| 2 | 1-3s |
| 3 | 3-7s |
| 4 | 7-15s |
| 5 | 15-30s |

**Configuration:**

```yaml
# config/webhook.yaml
retry:
  max_attempts: 5
  initial_delay: 1s
  max_delay: 5m
  backoff_factor: 2.0
  jitter: 0.25 # 25% randomness
```

### 1.5 Signature Verification

```go
// Webhook signature verification in consumer
func VerifyWebhookSignature(payload []byte, signature string, secret string) bool {
    expectedSignature := calculateSignature(payload, secret)
    return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

func calculateSignature(payload []byte, secret string) string {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write(payload)
    return hex.EncodeToString(h.Sum(nil))
}
```

### 1.6 Webhook Management CLI Commands

```bash
# List webhooks
rackd webhook list

# Create webhook
rackd webhook create \
  --name "slack-notifications" \
  --url https://hooks.slack.com/services/XXX/YYY \
  --events device.created,device.updated,device.deleted \
  --content-type application/json \
  --secret my-webhook-secret

# Update webhook
rackd webhook update <webhook-id> \
  --events device.created,device.updated,device.deleted,device.promoted

# Enable/disable webhook
rackd webhook enable <webhook-id>
rackd webhook disable <webhook-id>

# Delete webhook
rackd webhook delete <webhook-id>

# Test webhook
rackd webhook test <webhook-id>

# View webhook deliveries
rackd webhook deliveries <webhook-id>

# Redeliver failed webhooks
rackd webhook redeliver <webhook-id>
```

---

## 2. External Service Integrations

### 2.1 DNS Server Integration

**DNS Provider Interface:**

```go
package dns

import "context"

type DNSProvider interface {
    // Record operations
    CreateARecord(ctx context.Context, name string, ip string, ttl int) error
    UpdateARecord(ctx context.Context, name string, ip string, ttl int) error
    DeleteARecord(ctx context.Context, name string) error

    CreatePTRRecord(ctx context.Context, ip string, name string, ttl int) error
    UpdatePTRRecord(ctx context.Context, ip string, name string, ttl int) error
    DeletePTRRecord(ctx context.Context, ip string) error

    // Zone operations
    ListRecords(ctx context.Context, zone string) ([]DNSRecord, error)
    ZoneExists(ctx context.Context, zone string) (bool, error)
}

type DNSRecord struct {
    Name string
    Type string // A, AAAA, PTR, CNAME
    Value string
    TTL  int
}
```

**BIND Integration:**

```go
package bind

import (
    "context"
    "fmt"
    "os/exec"
)

type BINDProvider struct {
    configFile string
    zoneDir    string
    reloadCmd string
}

func NewBINDProvider(config BINDConfig) (*BINDProvider, error) {
    return &BINDProvider{
        configFile: config.ConfigFile,
        zoneDir:    config.ZoneDir,
        reloadCmd:  config.ReloadCommand,
    }, nil
}

func (b *BINDProvider) CreateARecord(ctx context.Context, name string, ip string, ttl int) error {
    // Append to zone file
    zoneFile := b.getZoneFile(name)
    record := fmt.Sprintf("%s  IN  A  %s", name, ip)
    if ttl > 0 {
        record = fmt.Sprintf("%s  %d  IN  A  %s", name, ttl, ip)
    }

    if err := appendToFile(zoneFile, record); err != nil {
        return err
    }

    // Reload BIND
    return b.reload(ctx)
}

func (b *BINDProvider) reload(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "sh", "-c", b.reloadCmd)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to reload BIND: %w", err)
    }

    log.Info("BIND reloaded successfully")
    return nil
}

func (b *BINDProvider) getZoneFile(name string) string {
    // Extract zone from FQDN
    parts := strings.Split(name, ".")
    if len(parts) < 2 {
        return fmt.Sprintf("%s/default.zone", b.zoneDir)
    }

    zone := strings.Join(parts[len(parts)-2:], ".")
    return fmt.Sprintf("%s/%s.zone", b.zoneDir, zone)
}
```

**PowerDNS Integration:**

```go
package powerdns

import (
    "context"
    "net/http"
)

type PowerDNSProvider struct {
    apiURL    string
    apiKey    string
    client    *http.Client
}

func NewPowerDNSProvider(config PowerDNSConfig) (*PowerDNSProvider, error) {
    return &PowerDNSProvider{
        apiURL: config.APIURL,
        apiKey: config.APIKey,
        client: &http.Client{Timeout: 30 * time.Second},
    }, nil
}

func (p *PowerDNSProvider) CreateARecord(ctx context.Context, name string, ip string, ttl int) error {
    payload := map[string]interface{}{
        "rrsets": []map[string]interface{}{
            {
                "name":  name,
                "type":  "A",
                "ttl":   ttl,
                "records": []map[string]string{
                    {"content": ip},
                },
            },
        },
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "PATCH",
        fmt.Sprintf("%s/api/v1/servers/localhost/zones/localhost", p.apiURL),
        bytes.NewReader(data),
    )
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-API-Key", p.apiKey)

    resp, err := p.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("PowerDNS API error: %d", resp.StatusCode)
    }

    return nil
}
```

### 2.2 DHCP Server Integration

**DHCP Manager Interface:**

```go
package dhcp

import "context"

type DHCPManager interface {
    // Lease operations
    CreateLease(ctx context.Context, lease *DHCPLease) error
    UpdateLease(ctx context.Context, lease *DHCPLease) error
    DeleteLease(ctx context.Context, ip string) error

    GetLease(ctx context.Context, ip string) (*DHCPLease, error)
    ListLeases(ctx context.Context) ([]DHCPLease, error)

    // Pool operations
    CreatePool(ctx context.Context, pool *DHCPPool) error
    UpdatePool(ctx context.Context, pool *DHCPPool) error
    DeletePool(ctx context.Context, name string) error

    GetPool(ctx context.Context, name string) (*DHCPPool, error)
    ListPools(ctx context.Context) ([]DHCPPool, error)
}

type DHCPLease struct {
    IP       string
    MAC      string
    Hostname string
    Start    time.Time
    End      time.Time
}

type DHCPPool struct {
    Name      string
    Network   string
    RangeStart string
    RangeEnd string
}
```

**ISC DHCP Integration:**

```go
package iscdhcp

import (
    "context"
    "fmt"
    "os/exec"
)

type ISCDHCPManager struct {
    configFile string
    reloadCmd string
}

func NewISCDHCPManager(config ISCDHCPConfig) (*ISCDHCPManager, error) {
    return &ISCDHCPManager{
        configFile: config.ConfigFile,
        reloadCmd:  config.ReloadCommand,
    }, nil
}

func (i *ISCDHCPManager) CreateLease(ctx context.Context, lease *DHCPLease) error {
    // Write lease to leases file
    leaseFile := i.configFile + ".leases"
    leaseLine := fmt.Sprintf(
        "lease %s { starts %d; ends %d; hardware ethernet %s; hostname %s; }\n",
        lease.IP,
        lease.Start.Unix(),
        lease.End.Unix(),
        lease.MAC,
        lease.Hostname,
    )

    if err := appendToFile(leaseFile, leaseLine); err != nil {
        return err
    }

    return nil
}

func (i *ISCDHCPManager) reload(ctx context.Context) error {
    cmd := exec.CommandContext(ctx, "systemctl", "reload", "isc-dhcp-server")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to reload ISC DHCP: %w", err)
    }

    log.Info("ISC DHCP server reloaded")
    return nil
}
```

**Kea Integration:**

```go
package kea

import (
    "context"
    "net/http"
)

type KeaManager struct {
    controlURL string
    client     *http.Client
}

func NewKeaManager(config KeaConfig) (*KeaManager, error) {
    return &KeaManager{
        controlURL: config.ControlURL,
        client:     &http.Client{Timeout: 30 * time.Second},
    }, nil
}

func (k *KeaManager) CreateLease(ctx context.Context, lease *DHCPLease) error {
    payload := map[string]interface{}{
        "command":    "lease4-add",
        "arguments": map[string]interface{}{
            "ip-address":  lease.IP,
            "hw-address":  lease.MAC,
            "hostname":    lease.Hostname,
            "valid-lft":  lease.End.Sub(lease.Start).Seconds(),
        },
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "POST", k.controlURL, bytes.NewReader(data))
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")

    resp, err := k.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var response struct {
        Result int `json:"result"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return err
    }

    if response.Result != 0 {
        return fmt.Errorf("Kea API error: %d", response.Result)
    }

    return nil
}
```

### 2.3 Monitoring System Integration

**Monitoring Backend Interface:**

```go
package monitoring

import "context"

type MonitoringBackend interface {
    // Metrics
    RecordCounter(ctx context.Context, name string, value float64, tags map[string]string) error
    RecordGauge(ctx context.Context, name string, value float64, tags map[string]string) error
    RecordHistogram(ctx context.Context, name string, value float64, tags map[string]string) error

    // Events
    RecordEvent(ctx context.Context, event string, properties map[string]interface{}) error

    // Health checks
    SendHeartbeat(ctx context.Context) error
}
```

**Prometheus Integration:**

```go
package prometheus

import (
    "context"
    "fmt"
    "net/http"
)

type PrometheusBackend struct {
    pushgatewayURL string
    job          string
    client       *http.Client
}

func NewPrometheusBackend(config PrometheusConfig) (*PrometheusBackend, error) {
    return &PrometheusBackend{
        pushgatewayURL: config.PushGatewayURL,
        job:          config.JobName,
        client:       &http.Client{Timeout: 30 * time.Second},
    }, nil
}

func (p *PrometheusBackend) RecordCounter(ctx context.Context, name string, value float64, tags map[string]string) error {
    metric := prometheus.Metric{
        Name:  name,
        Type:  "counter",
        Value: value,
        Labels: tags,
    }

    data, err := p.formatMetrics([]prometheus.Metric{metric})
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("%s/metrics/job/%s", p.pushgatewayURL, p.job),
        bytes.NewReader(data),
    )
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "text/plain")

    resp, err := p.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("Prometheus pushgateway error: %d", resp.StatusCode)
    }

    return nil
}

func (p *PrometheusBackend) formatMetrics(metrics []prometheus.Metric) ([]byte, error) {
    var builder strings.Builder
    for _, metric := range metrics {
        // Format labels
        labels := ""
        for k, v := range metric.Labels {
            labels += fmt.Sprintf('%s="%s"', k, v)
        }

        // Format metric line
        builder.WriteString(fmt.Sprintf("%s{%s} %f\n", metric.Name, labels, metric.Value))
    }

    return []byte(builder.String()), nil
}
```

**Datadog Integration:**

```go
package datadog

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
)

type DatadogBackend struct {
    apiKey string
    site   string
    client *http.Client
}

func NewDatadogBackend(config DatadogConfig) (*DatadogBackend, error) {
    return &DatadogBackend{
        apiKey: config.APIKey,
        site:   config.Site,
        client: &http.Client{Timeout: 30 * time.Second},
    }, nil
}

func (d *DatadogBackend) RecordCounter(ctx context.Context, name string, value float64, tags map[string]string) error {
    metric := map[string]interface{}{
        "metric": name,
        "points": []map[string]interface{}{
            {
                "value": value,
                "tags":  d.formatTags(tags),
            },
        },
    }

    data, err := json.Marshal([]map[string]interface{}{metric})
    if err != nil {
        return err
    }

    req, err := http.NewRequestWithContext(ctx, "POST",
        fmt.Sprintf("https://api.%s/datadoghq.com/api/v1/series", d.site),
        bytes.NewReader(data),
    )
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("DD-API-KEY", d.apiKey)

    resp, err := d.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("Datadog API error: %d", resp.StatusCode)
    }

    return nil
}

func (d *DatadogBackend) formatTags(tags map[string]string) []string {
    formatted := make([]string, 0, len(tags))
    for k, v := range tags {
        formatted = append(formatted, fmt.Sprintf("%s:%s", k, v))
    }
    return formatted
}
```

### 2.4 CMDB Integration

**CMDB Provider Interface:**

```go
package cmdb

import "context"

type CMDBProvider interface {
    // Asset operations
    CreateAsset(ctx context.Context, asset *Asset) error
    UpdateAsset(ctx context.Context, asset *Asset) error
    DeleteAsset(ctx context.Context, id string) error

    GetAsset(ctx context.Context, id string) (*Asset, error)
    SearchAssets(ctx context.Context, query string) ([]Asset, error)

    // Synchronization
    SyncDevices(ctx context.Context, devices []Device) error
}

type Asset struct {
    ID          string
    Name        string
    Type        string
    Status      string
    Location    string
    Attributes  map[string]interface{}
}
```

**ServiceNow Integration:**

```go
package servicenow

import (
    "context"
    "net/http"
)

type ServiceNowProvider struct {
    instanceURL string
    username    string
    password    string
    client      *http.Client
}

func NewServiceNowProvider(config ServiceNowConfig) (*ServiceNowProvider, error) {
    return &ServiceNowProvider{
        instanceURL: config.InstanceURL,
        username:    config.Username,
        password:    config.Password,
        client:      &http.Client{Timeout: 30 * time.Second},
    }, nil
}

func (s *ServiceNowProvider) CreateAsset(ctx context.Context, asset *Asset) error {
    payload := map[string]interface{}{
        "cmdb_ci_computer": map[string]interface{}{
            "name":        asset.Name,
            "asset_tag":   asset.ID,
            "status":      asset.Status,
            "location":    asset.Location,
            "serial_number": asset.Attributes["serial_number"],
            "manufacturer": asset.Attributes["manufacturer"],
            "model":        asset.Attributes["model"],
        },
    }

    data, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    url := fmt.Sprintf("%s/api/now/table/cmdb_ci_computer", s.instanceURL)
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
    if err != nil {
        return err
    }

    // Basic auth
    req.SetBasicAuth(s.username, s.password)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    resp, err := s.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("ServiceNow API error: %d", resp.StatusCode)
    }

    return nil
}
```

---

## 3. API Rate Limiting

### 3.1 Token Bucket Algorithm

```go
package ratelimit

import (
    "sync"
    "time"
)

type TokenBucket struct {
    capacity    int64
    tokens      int64
    refillRate  int64  // tokens per second
    lastRefill  time.Time
    mu          sync.Mutex
}

func NewTokenBucket(capacity int64, refillRate int64) *TokenBucket {
    return &TokenBucket{
        capacity:   capacity,
        tokens:     capacity,
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

func (tb *TokenBucket) Consume(tokens int64) bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    // Refill tokens
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    refill := int64(elapsed * float64(tb.refillRate))

    tb.tokens += refill
    if tb.tokens > tb.capacity {
        tb.tokens = tb.capacity
    }
    tb.lastRefill = now

    // Check if we have enough tokens
    if tb.tokens >= tokens {
        tb.tokens -= tokens
        return true
    }

    return false
}

func (tb *TokenBucket) Available() int64 {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    // Refill tokens
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()
    refill := int64(elapsed * float64(tb.refillRate))

    tb.tokens += refill
    if tb.tokens > tb.capacity {
        tb.tokens = tb.capacity
    }
    tb.lastRefill = now

    return tb.tokens
}
```

### 3.2 Configuration Options

**Rate Limit Configuration:**

```yaml
# config/ratelimit.yaml
global:
  enabled: true
  strategy: token_bucket

tokens:
  # Default limits
  default:
    requests_per_second: 100
    burst: 200

  # Per-token limits
  token_specific:
    admin_token:
      requests_per_second: 1000
      burst: 2000

    read_only_token:
      requests_per_second: 500
      burst: 1000

# Per-IP limits
ip_based:
  enabled: true
  default:
    requests_per_second: 50
    burst: 100

  ip_specific:
    "192.168.1.100":
      requests_per_second: 200
      burst: 400

# Per-endpoint limits
endpoint_based:
  enabled: true

  "/api/devices":
    requests_per_second: 200
    burst: 400

  "/api/discovery":
    requests_per_second: 10
    burst: 20
```

### 3.3 Rate Limiting Middleware

```go
package middleware

import (
    "net/http"
    "strings"
)

type RateLimitMiddleware struct {
    limiter          *TokenBucket
    perTokenLimits   map[string]*TokenBucket
    perIPLimits     map[string]*TokenBucket
    perEndpointLimits map[string]*TokenBucket
    ipExtractor     IPExtractor
}

type IPExtractor interface {
    Extract(r *http.Request) string
}

func NewRateLimitMiddleware(config RateLimitConfig) *RateLimitMiddleware {
    return &RateLimitMiddleware{
        limiter:          NewTokenBucket(config.Default.Burst, config.Default.RequestsPerSecond),
        perTokenLimits:   initTokenBuckets(config.TokenSpecific),
        perIPLimits:     initIPBuckets(config.IPSpecific),
        perEndpointLimits: initEndpointBuckets(config.EndpointSpecific),
        ipExtractor:     NewRealIPExtractor(),
    }
}

func (rl *RateLimitMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Extract token
    token := extractToken(r)

    // Extract IP
    ip := rl.ipExtractor.Extract(r)

    // Get appropriate limiter
    limiter := rl.getLimiter(token, ip, r.URL.Path)

    // Check rate limit
    if !limiter.Consume(1) {
        rl.writeRateLimitResponse(w, r, limiter)
        return
    }

    // Add rate limit headers
    rl.addRateLimitHeaders(w, limiter)

    // Continue to next handler
    rl.next.ServeHTTP(w, r)
}

func (rl *RateLimitMiddleware) getLimiter(token, ip, path string) *TokenBucket {
    // Check per-token limit
    if token != "" {
        if limiter, ok := rl.perTokenLimits[token]; ok {
            return limiter
        }
    }

    // Check per-IP limit
    if limiter, ok := rl.perIPLimits[ip]; ok {
        return limiter
    }

    // Check per-endpoint limit
    if limiter, ok := rl.perEndpointLimits[path]; ok {
        return limiter
    }

    // Use default limiter
    return rl.limiter
}

func (rl *RateLimitMiddleware) writeRateLimitResponse(w http.ResponseWriter, r *http.Request, limiter *TokenBucket) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Retry-After", "60") // Suggest retry after 60 seconds

    w.WriteHeader(http.StatusTooManyRequests)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": "Rate limit exceeded",
        "code":  "RATE_LIMITED",
        "details": map[string]interface{}{
            "available_tokens": limiter.Available(),
            "retry_after":       60,
        },
    })
}

func (rl *RateLimitMiddleware) addRateLimitHeaders(w http.ResponseWriter, limiter *TokenBucket) {
    w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.capacity))
    w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.Available()))
    w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Unix()+60))
}

func extractToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return strings.TrimPrefix(auth, "Bearer ")
    }
    return ""
}
```

### 3.4 Rate Limit Headers

**Response Headers:**

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1705776000
X-RateLimit-Retry-After: 60
```

**Header Descriptions:**

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum requests per time window |
| `X-RateLimit-Remaining` | Remaining requests in current window |
| `X-RateLimit-Reset` | Unix timestamp when rate limit resets |
| `X-RateLimit-Retry-After` | Seconds to wait before retry |

### 3.5 Implementation Pattern

```go
// Register rate limiting middleware
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
    rateLimiter := NewRateLimitMiddleware(h.config.RateLimit)

    // Wrap handlers with rate limiting
    mux.HandleFunc("GET /api/devices", rateLimiter.With(h.listDevices))
    mux.HandleFunc("POST /api/devices", rateLimiter.With(h.createDevice))
    mux.HandleFunc("PUT /api/devices/{id}", rateLimiter.With(h.updateDevice))
    mux.HandleFunc("DELETE /api/devices/{id}", rateLimiter.With(h.deleteDevice))
}

// Alternative: Apply globally
func (h *Handler) RegisterRoutes(mux *http.ServeMux, opts ...HandlerOption) {
    rateLimiter := NewRateLimitMiddleware(h.config.RateLimit)

    // Wrap entire ServeMux
    wrappedMux := http.NewServeMux()
    wrappedMux.Handle("/", rateLimiter.Wrap(mux))

    // Register routes
    h.RegisterRoutes(wrappedMux, opts...)
}
```

---

## 4. Bulk Import/Export

### 4.1 CSV Import Format

**Device CSV Format:**

```csv
id,name,description,make_model,os,datacenter_id,username,location,tags,addresses
dev_001,server-01,Primary web server,Dell R740,Ubuntu 22.04,dc_001,admin,Rack A1,"web|prod",ip=192.168.1.10,port=22,type=ipv4,label=management
dev_002,server-02,Database server,HP DL380,RHEL 9,dc_001,dbadmin,Rack B2,"database|prod",ip=192.168.1.11,port=22,type=ipv4,label=management;ip=192.168.1.12,port=5432,type=ipv4,label=data
```

**Network CSV Format:**

```csv
id,name,subnet,vlan_id,datacenter_id,description
net_001,production,192.168.1.0/24,100,dc_001,Primary production network
net_002,development,192.168.2.0/24,200,dc_002,Development network
```

**Datacenter CSV Format:**

```csv
id,name,location,description
dc_001,Data Center West,123 Main St,San Francisco CA
dc_002,Data Center East,456 Oak Ave,New York NY
```

### 4.2 JSON Import Format

**Device JSON Format:**

```json
{
  "devices": [
    {
      "id": "dev_001",
      "name": "server-01",
      "description": "Primary web server",
      "make_model": "Dell R740",
      "os": "Ubuntu 22.04",
      "datacenter_id": "dc_001",
      "username": "admin",
      "location": "Rack A1",
      "tags": ["web", "prod"],
      "addresses": [
        {
          "ip": "192.168.1.10",
          "port": 22,
          "type": "ipv4",
          "label": "management",
          "network_id": "net_001"
        },
        {
          "ip": "192.168.1.11",
          "port": 80,
          "type": "ipv4",
          "label": "data",
          "network_id": "net_001"
        }
      ]
    }
  ]
}
```

**Bulk Device Import:**

```json
{
  "devices": [
    {
      "name": "server-01",
      "description": "Primary web server",
      "make_model": "Dell R740",
      "os": "Ubuntu 22.04",
      "datacenter_id": "dc_001",
      "tags": ["web", "prod"],
      "addresses": [
        {
          "ip": "192.168.1.10",
          "port": 22,
          "type": "ipv4",
          "label": "management"
        }
      ]
    },
    {
      "name": "server-02",
      "description": "Database server",
      "make_model": "HP DL380",
      "os": "RHEL 9",
      "datacenter_id": "dc_001",
      "tags": ["database", "prod"],
      "addresses": [
        {
          "ip": "192.168.1.11",
          "port": 22,
          "type": "ipv4",
          "label": "management"
        }
      ]
    }
  ]
}
```

### 4.3 Validation Rules

**Device Validation:**

```go
package import

import (
    "net"
)

type DeviceValidator struct {
    errors []ValidationError
}

type ValidationError struct {
    Row    int
    Field   string
    Message string
    Value   string
}

func (v *DeviceValidator) ValidateDevice(device *Device, row int) {
    // Required fields
    if device.Name == "" {
        v.errors = append(v.errors, ValidationError{
            Row:    row,
            Field:   "name",
            Message: "Device name is required",
        })
    }

    // IP address validation
    for _, addr := range device.Addresses {
        if net.ParseIP(addr.IP) == nil {
            v.errors = append(v.errors, ValidationError{
                Row:     row,
                Field:    "ip",
                Message: "Invalid IP address",
                Value:    addr.IP,
            })
        }

        if addr.Port < 1 || addr.Port > 65535 {
            v.errors = append(v.errors, ValidationError{
                Row:     row,
                Field:    "port",
                Message: "Port must be between 1 and 65535",
                Value:    fmt.Sprintf("%d", addr.Port),
            })
        }

        // Type validation
        if addr.Type != "ipv4" && addr.Type != "ipv6" {
            v.errors = append(v.errors, ValidationError{
                Row:     row,
                Field:    "type",
                Message: "Type must be 'ipv4' or 'ipv6'",
                Value:    addr.Type,
            })
        }
    }

    // Datacenter validation
    if device.DatacenterID != "" {
        // Verify datacenter exists
        exists := v.checkDatacenterExists(device.DatacenterID)
        if !exists {
            v.errors = append(v.errors, ValidationError{
                Row:     row,
                Field:    "datacenter_id",
                Message: "Datacenter does not exist",
                Value:    device.DatacenterID,
            })
        }
    }

    // Tag validation
    for _, tag := range device.Tags {
        if len(tag) > 100 {
            v.errors = append(v.errors, ValidationError{
                Row:     row,
                Field:    "tags",
                Message: "Tag length must be <= 100 characters",
                Value:    tag,
            })
        }
    }
}

func (v *DeviceValidator) Validate() []ValidationError {
    return v.errors
}
```

**Network Validation:**

```go
func (v *NetworkValidator) ValidateNetwork(network *Network, row int) {
    // Required fields
    if network.Name == "" {
        v.errors = append(v.errors, ValidationError{
            Row:    row,
            Field:   "name",
            Message: "Network name is required",
        })
    }

    // CIDR validation
    _, _, err := net.ParseCIDR(network.Subnet)
    if err != nil {
        v.errors = append(v.errors, ValidationError{
            Row:     row,
            Field:   "subnet",
            Message: "Invalid CIDR notation",
            Value:    network.Subnet,
        })
    }

    // VLAN validation
    if network.VLANID < 0 || network.VLANID > 4095 {
        v.errors = append(v.errors, ValidationError{
            Row:     row,
            Field:   "vlan_id",
            Message: "VLAN ID must be between 0 and 4095",
            Value:    fmt.Sprintf("%d", network.VLANID),
        })
    }

    // Datacenter validation
    if network.DatacenterID != "" {
        exists := v.checkDatacenterExists(network.DatacenterID)
        if !exists {
            v.errors = append(v.errors, ValidationError{
                Row:     row,
                Field:   "datacenter_id",
                Message: "Datacenter does not exist",
                Value:    network.DatacenterID,
            })
        }
    }
}
```

### 4.4 Conflict Resolution

**Strategies:**

1. **Skip on Conflict** - Skip conflicting records
2. **Update on Conflict** - Update existing records
3. **Error on Conflict** - Stop import on first conflict
4. **Generate New IDs** - Generate new IDs for conflicting records

**Configuration:**

```bash
rackd import devices --input devices.csv \
  --on-conflict skip \
  --fail-on-error

rackd import devices --input devices.json \
  --on-conflict update \
  --conflict-field name \
  --continue-on-error
```

### 4.5 Export Templates

**Device Export Template:**

```json
{
  "version": "1.0",
  "exported_at": "2024-01-20T12:00:00Z",
  "filter": {
    "datacenter_id": "dc_001",
    "tags": ["web", "prod"]
  },
  "devices": [
    {
      "id": "dev_001",
      "name": "server-01",
      "description": "Primary web server",
      "make_model": "Dell R740",
      "os": "Ubuntu 22.04",
      "datacenter_id": "dc_001",
      "datacenter_name": "Data Center West",
      "username": "admin",
      "location": "Rack A1",
      "tags": ["web", "prod"],
      "addresses": [
        {
          "ip": "192.168.1.10",
          "port": 22,
          "type": "ipv4",
          "label": "management",
          "network_id": "net_001",
          "network_name": "production",
          "switch_port": "Gi1/0/1"
        }
      ],
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-15T14:30:00Z"
    }
  ]
}
```

**Network Export Template:**

```json
{
  "version": "1.0",
  "exported_at": "2024-01-20T12:00:00Z",
  "networks": [
    {
      "id": "net_001",
      "name": "production",
      "subnet": "192.168.1.0/24",
      "vlan_id": 100,
      "datacenter_id": "dc_001",
      "datacenter_name": "Data Center West",
      "description": "Primary production network",
      "utilization": {
        "total_ips": 254,
        "used_ips": 180,
        "available_ips": 74,
        "utilization_percent": 70.87
      },
      "device_count": 42,
      "pool_count": 3,
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-10T11:00:00Z"
    }
  ]
}
```

### 4.6 Import CLI Commands

```bash
# Import from CSV
rackd import devices --input devices.csv \
  --format csv \
  --skip-rows 1 \
  --dry-run

# Import from JSON
rackd import devices --input devices.json \
  --format json \
  --on-conflict update \
  --batch-size 100

# Import networks
rackd import networks --input networks.csv \
  --format csv

# Import datacenters
rackd import datacenters --input datacenters.json \
  --format json
```

### 4.7 Export CLI Commands

```bash
# Export all devices
rackd export devices --output devices.json \
  --format json

# Export with filter
rackd export devices --output web-servers.json \
  --format json \
  --filter "tags=web,prod"

# Export to CSV
rackd export devices --output devices.csv \
  --format csv \
  --include addresses,tags

# Export networks
rackd export networks --output networks.json \
  --format json

# Export datacenters
rackd export datacenters --output datacenters.json \
  --format json
```

---

## 5. Integration Code Examples

### 5.1 Webhook Delivery Implementation

```go
package api

func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
    // ... create device logic ...

    // Emit webhook event
    event := &model.WebhookEvent{
        ID:        uuid.New().String(),
        Type:      "device.created",
        Timestamp: time.Now(),
        Data: map[string]interface{}{
            "device": device,
        },
    }

    if err := h.webhookDeliverer.Deliver(r.Context(), event); err != nil {
        log.Error("Failed to deliver webhook", "error", err)
        // Don't fail request due to webhook delivery failure
    }

    h.writeJSON(w, http.StatusCreated, device)
}
```

### 5.2 DNS Client Integration

```go
package api

func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
    // ... create device logic ...

    // Integrate with DNS (if configured)
    if h.dnsProvider != nil && len(device.Addresses) > 0 {
        for _, addr := range device.Addresses {
            // Create A record for management IP
            if addr.Type == "ipv4" && addr.Label == "management" {
                if err := h.dnsProvider.CreateARecord(r.Context(),
                    device.Name+"-mgmt",
                    addr.IP,
                    300); err != nil {
                    log.Error("Failed to create DNS record", "error", err)
                    // Continue anyway - device is created
                }
            }

            // Create PTR record
            if err := h.dnsProvider.CreatePTRRecord(r.Context(),
                addr.IP,
                device.Name,
                300); err != nil {
                log.Error("Failed to create PTR record", "error", err)
            }
        }
    }

    h.writeJSON(w, http.StatusCreated, device)
}
```

### 5.3 DHCP Client Integration

```go
package api

func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
    // ... create device logic ...

    // Integrate with DHCP (if configured)
    if h.dhcpManager != nil && len(device.Addresses) > 0 {
        for _, addr := range device.Addresses {
            // Create DHCP lease
            lease := &DHCPLease{
                IP:       addr.IP,
                MAC:      device.MACAddress, // Would need to be in device model
                Hostname: device.Name,
                Start:    time.Now(),
                End:      time.Now().AddDate(0, 0, 7), // 7 days
            }

            if err := h.dhcpManager.CreateLease(r.Context(), lease); err != nil {
                log.Error("Failed to create DHCP lease", "error", err)
            }
        }
    }

    h.writeJSON(w, http.StatusCreated, device)
}
```

### 5.4 Batch Import Processing

```go
package import

func (imp *Importer) ProcessDevices(ctx context.Context, file string, config ImportConfig) error {
    // Open file
    f, err := os.Open(file)
    if err != nil {
        return err
    }
    defer f.Close()

    // Parse based on format
    var records []DeviceRecord
    if config.Format == "csv" {
        records, err = parseCSV(f)
    } else if config.Format == "json" {
        records, err = parseJSON(f)
    }

    // Validate records
    validator := &DeviceValidator{}
    for i, record := range records {
        validator.ValidateDevice(&record.Device, i+1)
    }

    if len(validator.errors) > 0 && !config.ContinueOnError {
        return fmt.Errorf("validation failed: %d errors", len(validator.errors))
    }

    // Process in batches
    batchSize := config.BatchSize
    if batchSize == 0 {
        batchSize = 100
    }

    for i := 0; i < len(records); i += batchSize {
        end := i + batchSize
        if end > len(records) {
            end = len(records)
        }

        batch := records[i:end]

        // Process batch
        if err := imp.processBatch(ctx, batch); err != nil {
            log.Error("Batch processing failed", "batch", i/batchSize, "error", err)

            if !config.ContinueOnError {
                return err
            }
        }
    }

    return nil
}

func (imp *Importer) processBatch(ctx context.Context, records []DeviceRecord) error {
    for _, record := range records {
        // Check for conflicts
        existing, err := imp.storage.GetDeviceByName(record.Device.Name)
        if err != nil && !errors.Is(err, storage.ErrDeviceNotFound) {
            return err
        }

        if existing != nil {
            switch config.OnConflict {
            case "skip":
                log.Info("Skipping existing device", "name", record.Device.Name)
                continue
            case "update":
                record.Device.ID = existing.ID
                if err := imp.storage.UpdateDevice(&record.Device); err != nil {
                    return err
                }
                continue
            case "error":
                return &ConflictError{DeviceName: record.Device.Name}
            case "generate":
                // Generate new ID (already done)
            }
        }

        // Create device
        if err := imp.storage.CreateDevice(&record.Device); err != nil {
            return err
        }
    }

    return nil
}
```

---

## 6. Integration Testing

### 6.1 Webhook Testing Patterns

```go
package webhook_test

func TestWebhookDelivery(t *testing.T) {
    server := httptest.NewServer(webhookHandler)
    defer server.Close()

    // Create test webhook
    webhook := &model.Webhook{
        ID:   "test-webhook",
        URL:   server.URL,
        Events: []string{"device.created"},
    }

    // Test successful delivery
    event := &model.WebhookEvent{
        Type: "device.created",
        Data: map[string]interface{}{
            "device": testDevice,
        },
    }

    err := deliverer.Deliver(context.Background(), event)
    if err != nil {
        t.Errorf("Webhook delivery failed: %v", err)
    }
}

func TestWebhookSignature(t *testing.T) {
    secret := "test-secret"
    payload := []byte(`{"test": "data"}`)

    // Calculate signature
    signature := calculateSignature(payload, secret)

    // Verify signature
    if !verifyWebhookSignature(payload, signature, secret) {
        t.Error("Webhook signature verification failed")
    }
}
```

### 6.2 External Service Mocking

```go
package mockdns

import (
    "context"
)

type MockDNSProvider struct {
    records map[string]DNSRecord
}

func NewMockDNSProvider() *MockDNSProvider {
    return &MockDNSProvider{
        records: make(map[string]DNSRecord),
    }
}

func (m *MockDNSProvider) CreateARecord(ctx context.Context, name string, ip string, ttl int) error {
    m.records[name] = DNSRecord{
        Name:  name,
        Type:  "A",
        Value: ip,
        TTL:   ttl,
    }
    return nil
}

func (m *MockDNSProvider) ListRecords(ctx context.Context, zone string) ([]DNSRecord, error) {
    records := make([]DNSRecord, 0)
    for _, record := range m.records {
        records = append(records, record)
    }
    return records, nil
}

// Usage in tests
func TestDeviceCreationWithDNS(t *testing.T) {
    mockDNS := NewMockDNSProvider()
    storage := NewTestStorage()
    storage.SetDNSProvider(mockDNS)

    device := &Device{
        Name: "test-server",
        Addresses: []Address{
            {IP: "192.168.1.10", Type: "ipv4"},
        },
    }

    if err := storage.CreateDevice(device); err != nil {
        t.Fatal(err)
    }

    // Verify DNS record was created
    record, ok := mockDNS.records["test-server-mgmt"]
    if !ok {
        t.Error("DNS record not created")
    }

    if record.Value != "192.168.1.10" {
        t.Errorf("DNS record IP mismatch: expected 192.168.1.10, got %s", record.Value)
    }
}
```

### 6.3 Contract Testing

```go
package contract_test

import (
    "context"
    "testing"
)

// Define contract for DNS provider
type DNSProviderContract interface {
    dns.DNSProvider
    ContractTests() []string
}

func (m *MockDNSProvider) ContractTests() []string {
    return []string{
        "TestCreateARecord",
        "TestUpdateARecord",
        "TestDeleteARecord",
        "TestListRecords",
        "TestZoneExists",
    }
}

func RunContractTests(t *testing.T, provider DNSProviderContract) {
    for _, testName := range provider.ContractTests() {
        t.Run(testName, func(t *testing.T) {
            switch testName {
            case "TestCreateARecord":
                testCreateARecord(t, provider)
            case "TestUpdateARecord":
                testUpdateARecord(t, provider)
            case "TestDeleteARecord":
                testDeleteARecord(t, provider)
            case "TestListRecords":
                testListRecords(t, provider)
            case "TestZoneExists":
                testZoneExists(t, provider)
            }
        })
    }
}

func testCreateARecord(t *testing.T, provider dns.DNSProvider) {
    name := "test-record.example.com"
    ip := "192.168.1.10"
    ttl := 300

    if err := provider.CreateARecord(context.Background(), name, ip, ttl); err != nil {
        t.Errorf("CreateARecord failed: %v", err)
    }

    // Verify record was created
    records, err := provider.ListRecords(context.Background(), "example.com")
    if err != nil {
        t.Fatalf("ListRecords failed: %v", err)
    }

    found := false
    for _, record := range records {
        if record.Name == name && record.Value == ip {
            found = true
            break
        }
    }

    if !found {
        t.Error("DNS record not found after creation")
    }
}
```
