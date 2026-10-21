# Discovery

Rackd's discovery system scans networks for hosts and lets you promote what it finds into the device inventory. Scans probe TCP ports and, for full and deep scans, also listen for NetBIOS, mDNS and LLDP announcements.

## Overview

The discovery system consists of:
- **Network Scanning**: Discovery of hosts on a network
- **Scan Types**: Quick, full and deep scans, plus custom port lists in scan profiles
- **Credentials**: Encrypted storage of SSH and SNMP credentials
- **Scheduled Scans**: Recurring scans driven by cron expressions
- **Device Promotion**: Converting discovered hosts to managed inventory

## Scan Types

#### Quick Scan
- **Ports**: 22, 80, 443, 3389
- **Use Case**: Initial network discovery, connectivity checks

#### Full Scan
- **Ports**: 21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 8080
- **Also**: NetBIOS, mDNS and LLDP listeners
- **Use Case**: Detailed service discovery

#### Deep Scan
- **Ports**: The top 100 most common ports
- **Also**: NetBIOS, mDNS and LLDP listeners, and OS fingerprinting
- **Use Case**: Complete network mapping

Every scan grabs service banners from the open ports it finds. Rackd picks the number of workers and the per-host timeout itself, based on the subnet size and scan type.

## Scan Profiles

Scan profiles are used by [scheduled scans](#scheduled-scans). Manual scans take only a scan type.

```go
type ScanProfile struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    ScanType    string    `json:"scan_type"`
    Ports       []int     `json:"ports,omitempty"`
    EnableSNMP  bool      `json:"enable_snmp"`
    EnableSSH   bool      `json:"enable_ssh"`
    TimeoutSec  int       `json:"timeout_sec"`
    MaxWorkers  int       `json:"max_workers"`
    Description string    `json:"description,omitempty"`
}
```

### Creating Scan Profiles

```bash
rackd scan-profile create \
  --name "Production Network Scan" \
  --type custom \
  --ports 22,80,443,8080,9090 \
  --description "Custom profile for production networks"
```

### Profile Parameters

- **scan_type**: `quick`, `full`, `deep`, or `custom`
- **ports**: Port list; when set, it replaces the scan type's ports
- **timeout_sec**: Must be positive (CLI default 30)
- **max_workers**: 1-100 (CLI default 10)
- **enable_snmp**, **enable_ssh**: Stored with the profile

Scans currently use only a profile's scan type and ports. `timeout_sec`, `max_workers`, `enable_snmp` and `enable_ssh` are stored but don't change how a scan runs yet.

### Built-in Profiles

Rackd seeds three profiles:

- **Quick Scan** (`default-quick`)
- **Full Scan** (`default-full`)
- **Deep Scan** (`default-deep`)

## Credentials

Rackd stores SSH and SNMP credentials encrypted with `ENCRYPTION_KEY`. Scans don't use credentials yet: no scan, manual or scheduled, logs in over SSH or queries SNMP.

Credential types are `ssh_password`, `ssh_key`, `snmp_v2c` and `snmp_v3`. Create them on the web UI's Credentials page or through the API:

```bash
curl -H "Authorization: Bearer $RACKD_TOKEN" -X POST http://localhost:8080/api/v1/credentials \
  -H "Content-Type: application/json" \
  -d '{
    "name": "SNMP Secure",
    "type": "snmp_v3",
    "snmp_v3_user": "<user>",
    "snmp_v3_auth": "<auth-password>",
    "snmp_v3_priv": "<priv-password>"
  }'
```

Other fields: `snmp_community` (for `snmp_v2c`), `ssh_username`, `ssh_key_id`, `datacenter_id` and `description`. Secrets are never returned.

`DISCOVERY_SNMPV2C_ENABLED` (default `false`) allows SNMPv2c, which sends the community string in cleartext. It has no effect until scans use credentials.

## Network Scanning

### Manual Scans

```bash
# Quick scan of a network
rackd discovery scan --network <network-id> --type quick

# Full port scan
rackd discovery scan --network <network-id> --type full
```

### Scan Status and Monitoring

The CLI starts scans and lists discovered hosts. Follow and cancel scans with the API or the web UI's Discovery page:

```bash
# List scans (optionally ?network_id=<network-id>)
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/discovery/scans

# Check one scan
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/discovery/scans/<scan-id>

# Cancel a running scan
curl -H "Authorization: Bearer $RACKD_TOKEN" -X POST http://localhost:8080/api/v1/discovery/scans/<scan-id>/cancel
```

### Scan Limitations

- **Maximum subnet size**: /16 (65,536 hosts)
- **Addresses never scanned**: subnets that include loopback (`127.0.0.0/8`, `::1`), link-local (`169.254.0.0/16`, which includes cloud metadata services, and `fe80::/10`), unspecified (`0.0.0.0/8`, `::`), multicast, reserved (`240.0.0.0/4`) or IPv4-mapped IPv6 addresses are refused with `400`. Private ranges can be scanned.
- **Scans at once**: at most 4 scans run at the same time across the server; another is refused with `429 TOO_MANY_SCANS` until one finishes
- **Workers and timeouts**: chosen automatically per scan (1-100 workers, 0.5-10 seconds per probe)

## Scheduled Scans

Recurring scans that run a scan profile on a network, using cron expressions.

### Creating Scheduled Scans

```bash
rackd scheduled-scan create \
  --name "Nightly Network Scan" \
  --network <network-id> \
  --profile <profile-id> \
  --cron "0 2 * * *" \
  --description "Daily scan at 2 AM"
```

### Cron Expression Examples

- `0 2 * * *` - Daily at 2:00 AM
- `0 */6 * * *` - Every 6 hours
- `0 9 * * 1` - Every Monday at 9:00 AM
- `*/30 * * * *` - Every 30 minutes

### Managing Scheduled Scans

```bash
# List scheduled scans
rackd scheduled-scan list

# Enable/disable schedule
rackd scheduled-scan update --id <schedule-id> --enable
rackd scheduled-scan update --id <schedule-id> --disable

# Update schedule
rackd scheduled-scan update --id <schedule-id> --cron "0 3 * * *"

# Delete schedule
rackd scheduled-scan delete --id <schedule-id>
```

### Minimum Interval

Scheduled scans have a minimum interval of 5 minutes to prevent system overload.

## Discovered Devices

Scan results are stored as discovered devices.

### Device Information

```go
type DiscoveredDevice struct {
    ID                 string        `json:"id"`
    IP                 string        `json:"ip"`
    MACAddress         string        `json:"mac_address"`
    Hostname           string        `json:"hostname"`
    NetworkID          string        `json:"network_id"`
    Status             string        `json:"status"`
    Confidence         int           `json:"confidence"`
    OSGuess            string        `json:"os_guess"`
    Vendor             string        `json:"vendor"`
    OpenPorts          []int         `json:"open_ports"`
    Services           []ServiceInfo `json:"services"`
    FirstSeen          time.Time     `json:"first_seen"`
    LastSeen           time.Time     `json:"last_seen"`
    PromotedToDeviceID string        `json:"promoted_to_device_id,omitempty"`
    MatchedDeviceID    string        `json:"matched_device_id,omitempty"`
    MatchedDeviceName  string        `json:"matched_device_name,omitempty"`
}
```

### Viewing Discovered Devices

```bash
# List discovered devices on a network
rackd discovery list --network <network-id>

# JSON output
rackd discovery list --output json
```

## Device Promotion

Convert discovered devices to managed inventory items.

### Hosts already in the inventory

A discovered host is in the inventory when a device is already it: the device
it was promoted to, or a device with an address on the host's IP, on the
host's network or with no network set. This is worked out when the list is
read, so a device added after a scan matches straight away.

- The web UI shows "In inventory: <device>" instead of Promote, linking to the
  device, and **New only** hides those hosts.
- `rackd discovery list` shows the device in its IN INVENTORY column, and the
  API returns it as `matched_device_id` and `matched_device_name`.
- Promoting such a host answers `409 ALREADY_EXISTS`, naming the device,
  instead of creating a second device with the same address.
- The dashboard counts only hosts that aren't in the inventory, and a device a
  scan keeps finding isn't counted as stale.

### Promotion Process

```bash
# Promote discovered device to inventory
rackd discovery promote --discovered-id <discovered-device-id> --name "Web Server 01"
```

The device gets the host's IP, hostname, OS and vendor; open ports, services
and the MAC address go in its description.

### Promotion Benefits

- **Inventory Management**: Track devices in centralized inventory
- **Relationship Mapping**: Define device dependencies
- **Monitoring Integration**: Enable monitoring and alerting
- **Asset Tracking**: Maintain device lifecycle information

## Configuration

### Environment Variables

```bash
# Discovery settings
DISCOVERY_INTERVAL=24h          # How often scheduled discovery rules run
DISCOVERY_CLEANUP_DAYS=30       # Delete discovered devices not seen for this long
DISCOVERY_SCAN_ON_STARTUP=false # Run discovery rules when the server starts

# Credential encryption (64 hex characters: openssl rand -hex 32)
ENCRYPTION_KEY=<key>
```

At most 4 scans run at once. Workers and per-host timeouts are chosen automatically for each scan.

### Performance Tuning

- **Scan Type**: Quick scans probe 4 ports, deep scans 100
- **Scan Intervals**: Consider network size and change frequency
- **Cleanup Settings**: Manage storage of scan history

## API Integration

### REST API Endpoints

```bash
# Start scan (scan_type: quick, full or deep; default quick)
POST /api/v1/discovery/networks/{network-id}/scan
{
  "scan_type": "full"
}

# Get scan status
GET /api/v1/discovery/scans/{scan-id}

# List discovered devices
GET /api/v1/discovery/devices?network_id=net-123

# Promote device
POST /api/v1/discovery/devices/{device-id}/promote
{
  "name": "Server 01",
  "make_model": "Dell PowerEdge R640",
  "datacenter_id": "dc-789"
}
```

## Troubleshooting

### Common Issues

#### Scan Failures
- **Network unreachable**: Check network configuration and routing
- **Too many scans**: Wait for a running scan to finish (`429 TOO_MANY_SCANS`)
- **Subnet refused**: Subnets larger than /16 or covering the ranges listed under [Scan Limitations](#scan-limitations) can't be scanned

### Debug Mode

Enable debug logging for detailed scan information:

```bash
LOG_LEVEL=debug rackd server
```

### Performance Monitoring

Monitor scan performance and resource usage:

```bash
# Check active scans (look for "status": "running")
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/discovery/scans

# Monitor system resources
top -p $(pgrep rackd)

# Check network utilization
iftop -i eth0
```

## Security Considerations

### Network Security
- **Scan Detection**: Network scans may trigger security alerts
- **Rate Limiting**: Use appropriate scan intervals to avoid detection
- **Firewall Rules**: Ensure scanning hosts can reach target networks

### Credential Security
- **Encryption**: All credentials encrypted at rest
- **Access Control**: Limit credential access to authorized users
- **Key Rotation**: Re-encrypt stored credentials with `rackd credentials rotate-key`
- **Audit Logging**: Monitor credential usage and access

### Best Practices
- **Least Privilege**: Use minimal required permissions for scanning
- **Network Segmentation**: Scan from appropriate network segments
- **Change Management**: Coordinate scans with network changes
- **Documentation**: Maintain records of scanning activities