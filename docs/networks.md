# Network Management

Rackd provides comprehensive IP Address Management (IPAM) capabilities for managing networks, subnets, VLANs, IP pools, and tracking IP allocation and utilization across your infrastructure.

## Overview

The network management system in Rackd consists of several key components:

- **Networks**: Define network segments with CIDR notation and VLAN assignments
- **IP Pools**: Manage ranges of IP addresses within networks for allocation
- **IP Allocation**: Track which IP addresses are assigned to devices
- **Utilization Tracking**: Monitor IP usage and availability across networks
- **Heatmaps**: Visual representation of IP allocation patterns

## Network Model

Networks in Rackd are defined by the following structure (see `internal/model/network.go`):

```go
type Network struct {
    ID           string    `json:"id"`
    Name         string    `json:"name"`
    Subnet       string    `json:"subnet"`        // CIDR notation (e.g., "192.168.1.0/24")
    VLANID       int       `json:"vlan_id"`       // VLAN ID (0-4094)
    DatacenterID string    `json:"datacenter_id"` // Associated datacenter
    Description  string    `json:"description"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
```

## IP Pool Model

IP pools define allocatable ranges within networks:

```go
type NetworkPool struct {
    ID          string    `json:"id"`
    NetworkID   string    `json:"network_id"`
    Name        string    `json:"name"`
    StartIP     string    `json:"start_ip"`      // First IP in range
    EndIP       string    `json:"end_ip"`        // Last IP in range
    Description string    `json:"description"`
    Tags        []string  `json:"tags"`          // Pool categorization
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

## Network Management Operations

### Creating Networks

**CLI:**
```bash
# Create a basic network
rackd network add --name "Production LAN" --subnet "192.168.1.0/24"

# Create network with VLAN and datacenter
rackd network add \
  --name "DMZ Network" \
  --subnet "10.0.100.0/24" \
  --vlan 100 \
  --datacenter "dc-east-1" \
  --description "DMZ for web servers"
```

**API:**
```bash
curl -X POST http://localhost:8080/api/networks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production LAN",
    "subnet": "192.168.1.0/24",
    "vlan_id": 10,
    "datacenter_id": "dc-east-1",
    "description": "Main production network"
  }'
```

### Listing Networks

**CLI:**
```bash
# List all networks
rackd network list

# Filter by datacenter
rackd network list --datacenter "dc-east-1"

# Filter by name
rackd network list --name "Production"

# JSON output
rackd network list --output json
```

**API:**
```bash
# List all networks
curl http://localhost:8080/api/networks

# Filter by datacenter
curl "http://localhost:8080/api/networks?datacenter_id=dc-east-1"

# Filter by VLAN
curl "http://localhost:8080/api/networks?vlan_id=100"
```

### Getting Network Details

**CLI:**
```bash
rackd network get <network-id>
```

**API:**
```bash
curl http://localhost:8080/api/networks/<network-id>
```

### Updating Networks

**API:**
```bash
curl -X PATCH http://localhost:8080/api/networks/<network-id> \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated description",
    "vlan_id": 20
  }'
```

### Deleting Networks

**CLI:**
```bash
rackd network delete <network-id>
```

**API:**
```bash
curl -X DELETE http://localhost:8080/api/networks/<network-id>
```

## IP Pool Management

### Creating IP Pools

**CLI:**
```bash
# Create an IP pool within a network
rackd network pool add \
  --network <network-id> \
  --name "DHCP Pool" \
  --start "192.168.1.100" \
  --end "192.168.1.200" \
  --description "Dynamic IP allocation pool"
```

**API:**
```bash
curl -X POST http://localhost:8080/api/networks/<network-id>/pools \
  -H "Content-Type: application/json" \
  -d '{
    "name": "DHCP Pool",
    "start_ip": "192.168.1.100",
    "end_ip": "192.168.1.200",
    "description": "Dynamic IP allocation pool",
    "tags": ["dhcp", "dynamic"]
  }'
```

### Listing IP Pools

**CLI:**
```bash
rackd network pool list --network <network-id>
```

**API:**
```bash
curl http://localhost:8080/api/networks/<network-id>/pools
```

### Managing Individual Pools

**Get Pool Details:**
```bash
curl http://localhost:8080/api/pools/<pool-id>
```

**Update Pool:**
```bash
curl -X PATCH http://localhost:8080/api/pools/<pool-id> \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated pool description",
    "tags": ["dhcp", "dynamic", "production"]
  }'
```

**Delete Pool:**
```bash
curl -X DELETE http://localhost:8080/api/pools/<pool-id>
```

## IP Allocation

### Getting Next Available IP

**API:**
```bash
curl http://localhost:8080/api/pools/<pool-id>/next-ip
```

Response:
```json
{
  "ip": "192.168.1.101"
}
```

### Allocating Specific IPs

IP allocation is handled automatically when devices are assigned IP addresses. The system tracks which IPs are in use through device address assignments.

## Utilization Tracking

### Network Utilization

Get utilization statistics for a network:

**API:**
```bash
curl http://localhost:8080/api/networks/<network-id>/utilization
```

Response:
```json
{
  "network_id": "net-123",
  "total_ips": 254,
  "used_ips": 45,
  "available_ips": 209,
  "utilization": 17.7
}
```

The utilization model (see `internal/model/network.go`):

```go
type NetworkUtilization struct {
    NetworkID    string  `json:"network_id"`
    TotalIPs     int     `json:"total_ips"`     // Total allocatable IPs
    UsedIPs      int     `json:"used_ips"`      // Currently allocated IPs
    AvailableIPs int     `json:"available_ips"` // Remaining IPs
    Utilization  float64 `json:"utilization"`   // Percentage used
}
```

## Heatmaps

### Pool Heatmap

Get visual representation of IP allocation within a pool:

**API:**
```bash
curl http://localhost:8080/api/pools/<pool-id>/heatmap
```

Response format:
```json
{
  "pool_id": "pool-123",
  "start_ip": "192.168.1.100",
  "end_ip": "192.168.1.200",
  "total_ips": 101,
  "allocated": [
    {
      "ip": "192.168.1.100",
      "device_id": "dev-456",
      "device_name": "web-server-01"
    },
    {
      "ip": "192.168.1.101",
      "device_id": "dev-789",
      "device_name": "db-server-01"
    }
  ],
  "available": [
    "192.168.1.102",
    "192.168.1.103"
  ]
}
```

## Network Devices

### Listing Devices in a Network

Get all devices that have IP addresses within a network:

**API:**
```bash
curl http://localhost:8080/api/networks/<network-id>/devices
```

## Validation Rules

The network management system enforces several validation rules (see `internal/api/validation.go`):

### Network Validation
- **Name**: Required, max 255 characters
- **Subnet**: Required, valid CIDR notation
- **VLAN ID**: 0-4094 range
- **Description**: Max 4096 characters

### IP Pool Validation
- **Name**: Required, max 255 characters
- **Start IP**: Required, valid IP address
- **End IP**: Required, valid IP address, must be >= start IP
- **Description**: Max 4096 characters
- **IP Range**: Start IP must be less than or equal to end IP

## Examples

### Complete Network Setup

1. **Create a datacenter:**
```bash
rackd datacenter add --name "East Coast DC" --location "New York"
```

2. **Create a network:**
```bash
rackd network add \
  --name "Production Network" \
  --subnet "10.0.1.0/24" \
  --vlan 100 \
  --datacenter "dc-east-1" \
  --description "Main production network"
```

3. **Create IP pools:**
```bash
# Static IP pool for servers
rackd network pool add \
  --network <network-id> \
  --name "Server Pool" \
  --start "10.0.1.10" \
  --end "10.0.1.50" \
  --description "Static IPs for servers"

# DHCP pool for workstations
rackd network pool add \
  --network <network-id> \
  --name "DHCP Pool" \
  --start "10.0.1.100" \
  --end "10.0.1.200" \
  --description "Dynamic IPs for workstations"
```

4. **Monitor utilization:**
```bash
curl http://localhost:8080/api/networks/<network-id>/utilization
```

### Multi-VLAN Environment

```bash
# Management VLAN
rackd network add \
  --name "Management" \
  --subnet "192.168.10.0/24" \
  --vlan 10 \
  --description "Management network"

# Production VLAN
rackd network add \
  --name "Production" \
  --subnet "192.168.20.0/24" \
  --vlan 20 \
  --description "Production services"

# DMZ VLAN
rackd network add \
  --name "DMZ" \
  --subnet "192.168.30.0/24" \
  --vlan 30 \
  --description "Demilitarized zone"
```

### IP Pool Strategies

**Server Pools (Static Assignment):**
```bash
rackd network pool add \
  --network <network-id> \
  --name "Database Servers" \
  --start "10.0.1.10" \
  --end "10.0.1.19" \
  --description "Database server static IPs"
```

**DHCP Pools (Dynamic Assignment):**
```bash
rackd network pool add \
  --network <network-id> \
  --name "Workstation DHCP" \
  --start "10.0.1.100" \
  --end "10.0.1.199" \
  --description "Workstation dynamic IPs"
```

**Reserved Ranges:**
```bash
# Leave gaps for network equipment
# 10.0.1.1-9: Network equipment (router, switches)
# 10.0.1.10-99: Servers (static)
# 10.0.1.100-199: Workstations (DHCP)
# 10.0.1.200-254: Future expansion
```

## Integration with Discovery

Networks integrate with the discovery system for automatic device detection:

1. **Network Scanning**: Discovery scans can be configured per network
2. **Device Promotion**: Discovered devices can be promoted and assigned to networks
3. **IP Detection**: Discovery automatically detects IP addresses and associates them with networks

See [Discovery Documentation](discovery.md) for more details on network scanning capabilities.

## Best Practices

### Network Design
- Use consistent CIDR sizing across similar network types
- Reserve VLAN 1 for management traffic
- Document network purposes in descriptions
- Use meaningful network names

### IP Pool Management
- Create separate pools for different device types
- Leave gaps between pools for future expansion
- Use descriptive pool names and tags
- Monitor utilization regularly

### VLAN Management
- Use VLAN ranges consistently (e.g., 10-99 for management, 100-199 for production)
- Document VLAN purposes
- Avoid VLAN 1 for production traffic

### Monitoring
- Set up alerts for high utilization (>80%)
- Regular utilization reviews
- Use heatmaps to identify allocation patterns
- Plan for growth based on utilization trends

## API Reference

For complete API documentation, see the [OpenAPI specification](../api/openapi.yaml). Key endpoints:

- `GET /api/networks` - List networks
- `POST /api/networks` - Create network
- `GET /api/networks/{id}` - Get network details
- `PATCH /api/networks/{id}` - Update network
- `DELETE /api/networks/{id}` - Delete network
- `GET /api/networks/{id}/devices` - List network devices
- `GET /api/networks/{id}/utilization` - Get utilization stats
- `GET /api/networks/{id}/pools` - List network pools
- `POST /api/networks/{id}/pools` - Create pool
- `GET /api/pools/{id}` - Get pool details
- `PATCH /api/pools/{id}` - Update pool
- `DELETE /api/pools/{id}` - Delete pool
- `GET /api/pools/{id}/next-ip` - Get next available IP
- `GET /api/pools/{id}/heatmap` - Get pool heatmap

## CLI Reference

For complete CLI documentation, see [CLI Reference](cli.md). Key commands:

- `rackd network list` - List networks
- `rackd network add` - Create network
- `rackd network get` - Get network details
- `rackd network delete` - Delete network
- `rackd network pool list` - List pools
- `rackd network pool add` - Create pool