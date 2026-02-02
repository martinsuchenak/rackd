# Discovery

Rackd's discovery system provides automated network scanning and device detection capabilities. It can discover devices across networks using multiple scanning techniques including basic port scanning, SSH, and SNMP.

## Overview

The discovery system consists of:
- **Network Scanning**: Automated discovery of devices on networks
- **Scan Types**: Basic, advanced, and custom scanning profiles
- **Credentials Management**: Secure storage of SSH and SNMP credentials
- **Scheduled Scans**: Automated recurring discovery
- **Device Promotion**: Converting discovered devices to managed inventory

## Scan Types

### Basic Scan Types

#### Quick Scan
- **Purpose**: Fast network discovery for basic connectivity
- **Method**: TCP port scanning on common ports (22, 80, 443, 3389)
- **Speed**: Fastest option
- **Use Case**: Initial network discovery, connectivity checks

#### Full Scan  
- **Purpose**: Comprehensive port scanning
- **Method**: Scans top 100 most common ports
- **Speed**: Moderate
- **Use Case**: Detailed service discovery

#### Deep Scan
- **Purpose**: Extensive network analysis
- **Method**: Extended port range scanning with service detection
- **Speed**: Slowest but most thorough
- **Use Case**: Security audits, complete network mapping

### Advanced Scanning

Advanced scans use scan profiles with customizable parameters:

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

## Scan Profiles

Scan profiles define scanning behavior and can be customized for different network environments.

### Creating Scan Profiles

```bash
# Create a custom scan profile
rackd profile create \
  --name "Production Network Scan" \
  --scan-type custom \
  --ports 22,80,443,8080,9090 \
  --enable-snmp \
  --enable-ssh \
  --timeout 5 \
  --max-workers 20 \
  --description "Custom profile for production networks"
```

### Profile Parameters

- **scan_type**: `quick`, `full`, `deep`, or `custom`
- **ports**: Custom port list (for custom scan type)
- **enable_snmp**: Enable SNMP discovery
- **enable_ssh**: Enable SSH-based discovery
- **timeout_sec**: Connection timeout (1-60 seconds)
- **max_workers**: Concurrent scan workers (1-100)

### Built-in Profiles

Rackd includes several pre-configured profiles:

- **Quick Discovery**: Fast connectivity scan
- **Standard Network**: Balanced speed and coverage
- **Security Audit**: Comprehensive port and service scan
- **Infrastructure**: Focus on network infrastructure devices

## Credentials Management

Discovery supports multiple credential types for authenticated scanning.

### SSH Credentials

#### Password Authentication
```bash
rackd credential create \
  --type ssh_password \
  --name "SSH Admin" \
  --ssh-username admin \
  --ssh-password "secure_password"
```

#### Key-based Authentication
```bash
rackd credential create \
  --type ssh_key \
  --name "SSH Key Auth" \
  --ssh-username root \
  --ssh-key-file ~/.ssh/id_rsa
```

### SNMP Credentials

#### SNMPv2c (Community String)
```bash
rackd credential create \
  --type snmp_v2c \
  --name "SNMP Public" \
  --snmp-community public
```

**Security Warning**: SNMPv2c transmits community strings in cleartext. Use only on trusted networks.

#### SNMPv3 (Secure)
```bash
rackd credential create \
  --type snmp_v3 \
  --name "SNMP Secure" \
  --snmpv3-user admin \
  --snmpv3-auth-pass "auth_password" \
  --snmpv3-priv-pass "priv_password"
```

## Network Scanning

### Manual Scans

#### Basic Network Scan
```bash
# Quick scan of a network
rackd discovery scan --network-id <network-id> --type quick

# Full port scan
rackd discovery scan --network-id <network-id> --type full
```

#### Advanced Scan with Profile
```bash
# Scan using custom profile with credentials
rackd discovery scan \
  --network-id <network-id> \
  --profile-id <profile-id> \
  --ssh-credential-id <ssh-cred-id> \
  --snmp-credential-id <snmp-cred-id>
```

### Scan Status and Monitoring

```bash
# Check scan status
rackd discovery status <scan-id>

# List all scans
rackd discovery list

# Cancel running scan
rackd discovery cancel <scan-id>
```

### Scan Limitations

- **Maximum subnet size**: /16 (65,536 hosts)
- **Concurrent scans**: Configurable per profile (default: 10)
- **Timeout**: 1-60 seconds per host
- **Rate limiting**: Prevents network flooding

## Scheduled Scans

Automated recurring scans using cron expressions.

### Creating Scheduled Scans

```bash
rackd scheduled create \
  --name "Nightly Network Scan" \
  --network-id <network-id> \
  --profile-id <profile-id> \
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
rackd scheduled list

# Enable/disable schedule
rackd scheduled enable <schedule-id>
rackd scheduled disable <schedule-id>

# Update schedule
rackd scheduled update <schedule-id> --cron "0 3 * * *"

# Delete schedule
rackd scheduled delete <schedule-id>
```

### Minimum Interval

Scheduled scans have a minimum interval of 5 minutes to prevent system overload.

## Discovered Devices

Scan results are stored as discovered devices with detailed information.

### Device Information

```go
type DiscoveredDevice struct {
    ID                 string        `json:"id"`
    IP                 string        `json:"ip"`
    MACAddress         string        `json:"mac_address"`
    Hostname           string        `json:"hostname"`
    NetworkID          string        `json:"network_id"`
    Status             string        `json:"status"`
    OSGuess            string        `json:"os_guess"`
    Vendor             string        `json:"vendor"`
    OpenPorts          []int         `json:"open_ports"`
    Services           []ServiceInfo `json:"services"`
    FirstSeen          time.Time     `json:"first_seen"`
    LastSeen           time.Time     `json:"last_seen"`
}
```

### Viewing Discovered Devices

```bash
# List discovered devices
rackd discovery devices --network-id <network-id>

# Get device details
rackd discovery device <device-id>

# Filter by status
rackd discovery devices --status online
```

## SSH Scanning

SSH scanning provides detailed system information through authenticated connections.

### Capabilities

- **OS Detection**: Operating system and version
- **System Information**: Hostname, kernel version
- **Package Inventory**: Installed software packages
- **Service Discovery**: Running services and processes
- **Host Key Management**: Trust-on-first-use (TOFU) verification

### SSH Discovery Process

1. **Connection**: Establish SSH connection using credentials
2. **Authentication**: Password or key-based authentication
3. **System Query**: Execute system information commands
4. **Data Collection**: Gather OS, packages, and services
5. **Host Key Storage**: Store host keys for future verification

### Security Features

- **Host Key Verification**: TOFU model prevents MITM attacks
- **Credential Encryption**: SSH credentials encrypted at rest
- **Connection Timeout**: Prevents hanging connections
- **Error Handling**: Graceful failure on authentication errors

## SNMP Scanning

SNMP scanning discovers network infrastructure devices and detailed system information.

### SNMP Information Gathered

- **System Information**: sysDescr, sysName, sysLocation, sysContact
- **Network Interfaces**: Interface details, status, and statistics
- **ARP Tables**: Network neighbor information
- **Device Identification**: Vendor and model information

### SNMP Discovery Process

1. **Connection**: Connect to SNMP agent on port 161
2. **Authentication**: Community string (v2c) or user credentials (v3)
3. **System Query**: Retrieve system MIB information
4. **Interface Walk**: Discover network interfaces
5. **ARP Discovery**: Map network topology

### SNMP Versions

#### SNMPv2c
- **Security**: Community string authentication
- **Encryption**: None (cleartext transmission)
- **Use Case**: Internal networks only

#### SNMPv3
- **Security**: User-based authentication and encryption
- **Protocols**: SHA authentication, AES encryption
- **Use Case**: Production and secure environments

## Device Promotion

Convert discovered devices to managed inventory items.

### Promotion Process

```bash
# Promote discovered device to inventory
rackd discovery promote <discovered-device-id> \
  --name "Web Server 01" \
  --type server \
  --datacenter-id <datacenter-id>
```

### Automatic Promotion Rules

Configure rules for automatic device promotion:

```bash
# Create promotion rule
rackd discovery rule create \
  --network-id <network-id> \
  --condition "port:22,80" \
  --device-type server \
  --auto-promote
```

### Promotion Benefits

- **Inventory Management**: Track devices in centralized inventory
- **Relationship Mapping**: Define device dependencies
- **Monitoring Integration**: Enable monitoring and alerting
- **Asset Tracking**: Maintain device lifecycle information

## Configuration

### Environment Variables

```bash
# Discovery settings
RACKD_DISCOVERY_MAX_CONCURRENT=10    # Max concurrent scans
RACKD_DISCOVERY_TIMEOUT=5s           # Per-host timeout
RACKD_DISCOVERY_CLEANUP_INTERVAL=1h  # Cleanup completed scans

# Credential encryption
RACKD_CREDENTIAL_KEY=<32-byte-key>   # Encryption key for credentials
```

### Performance Tuning

- **Concurrent Workers**: Balance speed vs. network load
- **Timeout Values**: Adjust for network latency
- **Scan Intervals**: Consider network size and change frequency
- **Cleanup Settings**: Manage storage of scan history

## API Integration

### REST API Endpoints

```bash
# Start scan
POST /api/v1/discovery/scans
{
  "network_id": "net-123",
  "scan_type": "full",
  "profile_id": "profile-456"
}

# Get scan status
GET /api/v1/discovery/scans/{scan-id}

# List discovered devices
GET /api/v1/discovery/devices?network_id=net-123

# Promote device
POST /api/v1/discovery/devices/{device-id}/promote
{
  "name": "Server 01",
  "type": "server",
  "datacenter_id": "dc-789"
}
```

## Troubleshooting

### Common Issues

#### Scan Failures
- **Network unreachable**: Check network configuration and routing
- **Permission denied**: Verify SSH/SNMP credentials
- **Timeout errors**: Increase timeout values or reduce concurrent workers

#### SSH Connection Issues
- **Authentication failed**: Verify username/password or SSH keys
- **Host key mismatch**: Clear stored host keys if infrastructure changed
- **Connection refused**: Ensure SSH service is running on target

#### SNMP Discovery Problems
- **No response**: Verify SNMP agent is enabled and accessible
- **Authentication error**: Check community string or SNMPv3 credentials
- **Timeout**: Increase SNMP timeout or check network connectivity

### Debug Mode

Enable debug logging for detailed scan information:

```bash
RACKD_LOG_LEVEL=debug rackd server
```

### Performance Monitoring

Monitor scan performance and resource usage:

```bash
# Check active scans
rackd discovery list --status running

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
- **Rotation**: Regularly rotate SSH and SNMP credentials
- **Audit Logging**: Monitor credential usage and access

### Best Practices
- **Least Privilege**: Use minimal required permissions for scanning
- **Network Segmentation**: Scan from appropriate network segments
- **Change Management**: Coordinate scans with network changes
- **Documentation**: Maintain records of scanning activities