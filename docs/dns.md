# DNS Integration

Rackd provides DNS integration with external DNS servers, allowing automatic management of DNS records for devices in your inventory.

## Overview

The DNS integration enables:

- Automatic A record creation for devices
- Optional PTR record generation for reverse DNS lookups
- Integration with Technitium DNS Server (PowerDNS and BIND are planned)
- Pushing records to the DNS server, and importing records from it

## Supported Providers

| Provider | Type | Status |
|----------|------|--------|
| Technitium | `technitium` | Supported |
| PowerDNS | `powerdns` | Planned; rejected when creating a provider |
| BIND (RFC 2136) | `bind` | Planned; rejected when creating a provider |

## Configuration

### Environment Variables

```bash
DNS_SYNC_INTERVAL=1h           # Interval between automatic sync operations
SSRF_ALLOWED_TARGETS=192.168.10.53  # A provider on a private address must be listed
```

### SSRF_ALLOWED_TARGETS

Provider endpoints on private addresses (RFC 1918, CGNAT, IPv6 ULA) are refused unless the address is listed here, as a comma-separated list of IPs or CIDR ranges. Loopback and link-local addresses can't be allowed. See the [configuration reference](configuration-reference.md#outbound-requests-ssrf).

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
```

#### Token Sources

For security, API tokens are not passed directly on the command line. Instead, use:

- `--token-env <VAR>` - Read token from environment variable
- `--token-file <PATH>` - Read token from file

Rackd never returns a provider's token. Changing a provider's endpoint requires entering the token again, so the stored token is never sent to a new host.

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
| `--create-ptr` | Enable PTR record creation (also needs `ptr_zone`; see [Reverse DNS](#reverse-dns-ptr-records)) |
| `--enable-auto-sync` | Enable automatic synchronization |
| `--ttl` | Default TTL for records (default: 3600) |

## DNS Records

### Automatic Record Generation

When a device is created or updated, Rackd generates records in each zone that has auto-sync enabled and is linked to a network the device has an address on:

1. **A and AAAA Records** - One per address on the zone's network: A for IPv4, AAAA for IPv6
2. **PTR Records** - If the zone has `create_ptr` and `ptr_zone` set; they go in the reverse zone (see [Reverse DNS](#reverse-dns-ptr-records))
3. **CNAME Records** - For each device domain that falls inside a zone, pointing at the device's A/AAAA record

Record names are stored relative to their zone: `web-01` in `example.com`, and `@` for the zone apex. Imported records are stored the same way.

### Record Mapping

Records are created based on:

- The device's `hostname` field (devices without a hostname get no records)
- Addresses whose `network_id` is the zone's network
- The device's domains (for CNAME records)

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
```

A sync pushes every record in the zone to the provider.

### Automatic Sync

When `--enable-auto-sync` is set on a zone:

- Zone is synced periodically based on `DNS_SYNC_INTERVAL`
- Creating or updating a device creates its records in the zone

### Import from Provider

Import existing DNS records into Rackd:

```bash
# Import records from provider
rackd dns import --zone zone-123
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

### PowerDNS and BIND

Planned. The `powerdns` and `bind` provider types are reserved, but the API rejects them until their integrations exist.

## Reverse DNS (PTR Records)

### Enabling PTR Records

PTR records need a reverse zone in Rackd as well as the forward zone:

1. Create the reverse zone (e.g. `1.168.192.in-addr.arpa`, or a `.ip6.arpa` zone for IPv6) on the same or another provider. Enable auto-sync on it if its records should be pushed as they are created; otherwise sync it with `rackd dns sync`.
2. On the forward zone, set `create_ptr: true` (`--create-ptr` on the CLI) and `ptr_zone` to the reverse zone's name. The CLI has no flag for `ptr_zone`; set it in the web UI's zone form or through the API:

```bash
curl -H "Authorization: Bearer $RACKD_TOKEN" -X PUT http://localhost:8080/api/v1/dns/zones/<zone-id> \
  -H "Content-Type: application/json" \
  -d '{"create_ptr": true, "ptr_zone": "1.168.192.in-addr.arpa"}'
```

### PTR Record Generation

For each A or AAAA record the forward zone generates, Rackd adds a PTR record to the reverse zone. Its name is the reversed address relative to that zone (`10` for `192.168.1.10` in `1.168.192.in-addr.arpa`), and its value is the device's `hostname` plus the forward zone name (e.g. `web-01.example.com`).

An address outside the reverse zone gets no PTR record, and neither does any address if no zone named `ptr_zone` exists in Rackd. Both cases are logged as warnings. Check the generated records with `rackd dns records --zone <reverse-zone-id> --type PTR`.

## API Reference

### DNS Providers

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/dns/providers` | List providers |
| GET | `/api/v1/dns/providers/{id}` | Get provider |
| POST | `/api/v1/dns/providers` | Create provider |
| PUT | `/api/v1/dns/providers/{id}` | Update provider |
| DELETE | `/api/v1/dns/providers/{id}` | Delete provider |
| POST | `/api/v1/dns/providers/{id}/test` | Test connection |

### DNS Zones

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/dns/zones` | List zones |
| GET | `/api/v1/dns/zones/{id}` | Get zone |
| POST | `/api/v1/dns/zones` | Create zone |
| PUT | `/api/v1/dns/zones/{id}` | Update zone |
| DELETE | `/api/v1/dns/zones/{id}` | Delete zone |

### DNS Records

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/dns/zones/{id}/records` | List records |
| POST | `/api/v1/dns/zones/{id}/sync` | Sync to provider |
| POST | `/api/v1/dns/zones/{id}/import` | Import from provider |

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
4. Try a manual sync with `rackd dns sync --zone <zone-id>`

### Records Not Created

1. Ensure the device has a `hostname`
2. Ensure the address has `network_id` set to the zone's network
3. Check the zone has `auto_sync` enabled
4. For CNAME records, check the device's domains fall inside the zone

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
