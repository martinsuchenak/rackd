# DNS Integration

Rackd provides DNS integration with external DNS servers, allowing automatic management of DNS records for devices in your inventory.

## Overview

The DNS integration enables:

- Automatic A/AAAA record creation for devices
- Automatic PTR record generation for reverse DNS lookups
- Integration with multiple DNS provider types
- Bidirectional sync between Rackd and DNS servers

## Supported Providers

| Provider | Type | Description |
|----------|------|-------------|
| Technitium | `technitium` | Technitium DNS Server |
| PowerDNS | `powerdns` | PowerDNS Authoritative Server |
| BIND | `bind` | BIND DNS Server (via RFC 2136) |

## Configuration

### Environment Variables

```bash
DNS_SYNC_INTERVAL=1h           # Interval between automatic sync operations
```

### DNS_SYNC_INTERVAL

Controls how often Rackd automatically syncs DNS zones that have `auto_sync` enabled.

## Setting Up DNS Integration

### 1. Create a DNS Provider

First, configure a DNS provider in Rackd.

```bash
# Create a Technitium provider
rackd dns provider create \
  --name "Main DNS" \
  --type technitium \
  --endpoint http://dns.example.com:5380 \
  --token-env TECHNITIUM_TOKEN

# Create a PowerDNS provider
rackd dns provider create \
  --name "PowerDNS" \
  --type powerdns \
  --endpoint http://pdns.example.com:8081 \
  --token-env POWERDNS_API_KEY
```

#### Token Sources

For security, API tokens are not passed directly on the command line. Instead, use:

- `--token-env <VAR>` - Read token from environment variable
- `--token-file <PATH>` - Read token from file

### 2. Test the Provider

Verify connectivity before creating zones:

```bash
rackd dns provider test --id provider-123
```

### 3. Create a DNS Zone

Create a zone that maps to a Rackd network:

```bash
rackd dns zone create \
  --name example.com \
  --provider provider-123 \
  --network net-456 \
  --create-ptr \
  --enable-auto-sync \
  --ttl 3600
```

**Options:**

| Option | Description |
|--------|-------------|
| `--name` | Zone name (e.g., `example.com`, `10.in-addr.arpa`) |
| `--provider` | DNS provider ID |
| `--network` | Network ID for automatic record association |
| `--create-ptr` | Enable automatic PTR record creation |
| `--enable-auto-sync` | Enable automatic synchronization |
| `--ttl` | Default TTL for records (default: 3600) |

## DNS Records

### Automatic Record Generation

When a device has IP addresses assigned, Rackd can automatically generate DNS records:

1. **A Records** - For IPv4 addresses
2. **AAAA Records** - For IPv6 addresses
3. **PTR Records** - Reverse DNS (if `--create-ptr` is enabled)

### Record Mapping

Records are created based on:

- Device hostname (from device name)
- Assigned IP addresses
- Associated domains

### Viewing Records

List DNS records for a zone:

```bash
# List all records in a zone
rackd dns records --zone zone-123

# Filter by type
rackd dns records --zone zone-123 --type A

# Filter by device
rackd dns records --zone zone-123 --device dev-456
```

## Synchronization

### Manual Sync

Trigger an immediate sync:

```bash
# Sync zone to DNS provider
rackd dns sync --zone zone-123

# Force sync even if no changes detected
rackd dns sync --zone zone-123 --force
```

### Automatic Sync

When `--enable-auto-sync` is set on a zone:

- Zone is synced periodically based on `DNS_SYNC_INTERVAL`
- Changes to devices trigger incremental updates
- New devices get records automatically

### Import from Provider

Import existing DNS records into Rackd:

```bash
# Import records from provider
rackd dns import --zone zone-123

# Import and delete local records not on provider
rackd dns import --zone zone-123 --delete
```

## Provider Configuration Details

### Technitium

```bash
rackd dns provider create \
  --name "Technitium" \
  --type technitium \
  --endpoint http://technitium.example.com:5380 \
  --token-env TECHNITIUM_TOKEN
```

Requirements:

- Technitium DNS Server 10.0+
- Admin API token with zone management permissions
- HTTP/HTTPS access to the server

### PowerDNS

```bash
rackd dns provider create \
  --name "PowerDNS" \
  --type powerdns \
  --endpoint http://pdns.example.com:8081 \
  --token-env POWERDNS_API_KEY
```

Requirements:

- PowerDNS Authoritative Server 4.x+
- API key with full zone access
- HTTP API enabled

### BIND (RFC 2136)

```bash
rackd dns provider create \
  --name "BIND" \
  --type bind \
  --endpoint 192.168.1.10:53 \
  --token-env TSIG_KEY
```

Requirements:

- BIND 9.x server with RFC 2136 dynamic updates enabled
- TSIG key for authentication
- UDP/TCP access to DNS server

## Reverse DNS (PTR Records)

### Setting Up Reverse Zones

For PTR record generation, create reverse zones:

```bash
# For 10.0.0.0/8 network
rackd dns zone create \
  --name 10.in-addr.arpa \
  --provider provider-123 \
  --network net-456 \
  --create-ptr

# For 192.168.1.0/24 network
rackd dns zone create \
  --name 1.168.192.in-addr.arpa \
  --provider provider-123 \
  --network net-789 \
  --create-ptr
```

### PTR Record Generation

When `--create-ptr` is enabled:

- PTR records are automatically created for A records
- Record name is the reversed IP with `.in-addr.arpa` suffix
- Record value is the device hostname

Example:
- Device: `web-01` with IP `192.168.1.10`
- PTR: `10.1.168.192.in-addr.arpa` → `web-01.example.com`

## API Reference

### DNS Providers

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dns/providers` | List providers |
| GET | `/api/dns/providers/{id}` | Get provider |
| POST | `/api/dns/providers` | Create provider |
| PUT | `/api/dns/providers/{id}` | Update provider |
| DELETE | `/api/dns/providers/{id}` | Delete provider |
| POST | `/api/dns/providers/{id}/test` | Test connection |

### DNS Zones

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dns/zones` | List zones |
| GET | `/api/dns/zones/{id}` | Get zone |
| POST | `/api/dns/zones` | Create zone |
| PUT | `/api/dns/zones/{id}` | Update zone |
| DELETE | `/api/dns/zones/{id}` | Delete zone |

### DNS Records

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dns/zones/{id}/records` | List records |
| POST | `/api/dns/zones/{id}/sync` | Sync to provider |
| POST | `/api/dns/zones/{id}/import` | Import from provider |

## Troubleshooting

### Connection Failed

1. Verify network connectivity to DNS server
2. Check API token is valid
3. Ensure correct endpoint URL
4. Run `rackd dns provider test --id <id>`

### Sync Not Working

1. Check zone has `auto_sync` enabled
2. Verify provider connection is working
3. Check server logs for errors
4. Try manual sync with `--force` flag

### Records Not Created

1. Ensure device has IP addresses assigned
2. Verify device is associated with correct network
3. Check zone's network association
4. Verify device has domains configured

## Best Practices

1. **Use dedicated API tokens** - Create tokens with minimal required permissions
2. **Enable auto-sync** - Keeps DNS records synchronized automatically
3. **Set up reverse zones** - Enable PTR records for proper reverse DNS
4. **Test before production** - Always test provider connectivity first
5. **Monitor sync status** - Check logs for sync errors

## Related Documentation

- [CLI Reference](cli.md) - DNS command documentation
- [API Reference](api.md) - REST API endpoints
- [Configuration](configuration-reference.md) - Environment variables
