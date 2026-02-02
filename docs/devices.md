# Device Management

Rackd's device management system provides comprehensive tracking of servers, switches, routers, and other network devices in your infrastructure.

## Device Model

A device in Rackd represents any physical or virtual infrastructure component:

```go
type Device struct {
    ID           string    `json:"id"`           // Unique identifier
    Name         string    `json:"name"`         // Device name (required)
    Hostname     string    `json:"hostname"`     // DNS hostname
    Description  string    `json:"description"`  // Device description
    MakeModel    string    `json:"make_model"`   // Manufacturer and model
    OS           string    `json:"os"`           // Operating system
    DatacenterID string    `json:"datacenter_id"` // Associated datacenter
    Username     string    `json:"username"`     // Login username
    Location     string    `json:"location"`     // Physical location
    Tags         []string  `json:"tags"`         // Searchable tags
    Addresses    []Address `json:"addresses"`    // Network addresses
    Domains      []string  `json:"domains"`      // Associated domains
    CreatedAt    time.Time `json:"created_at"`   // Creation timestamp
    UpdatedAt    time.Time `json:"updated_at"`   // Last update timestamp
}
```

### Address Model

Each device can have multiple network addresses:

```go
type Address struct {
    IP         string `json:"ip"`           // IP address (required)
    Port       *int   `json:"port"`        // Port number (optional)
    Type       string `json:"type"`        // Address type (e.g., "management", "data")
    Label      string `json:"label"`       // Human-readable label
    NetworkID  string `json:"network_id"`  // Associated network
    SwitchPort string `json:"switch_port"` // Physical switch port
    PoolID     string `json:"pool_id"`     // IP pool assignment
}
```

## CRUD Operations

### Create Device

**CLI:**
```bash
# Basic device creation
rackd device add --name "web-server-01" --description "Production web server"

# Full device with addresses and tags
rackd device add \
  --name "db-server-01" \
  --description "Primary database server" \
  --make-model "Dell PowerEdge R740" \
  --os "Ubuntu 22.04" \
  --datacenter "dc-east-1" \
  --location "Rack 15, U10-12" \
  --tags "database,production,mysql" \
  --addresses "10.1.1.100:3306:management,192.168.1.100:22:ssh" \
  --domains "db.example.com,mysql.internal"

# From JSON file
rackd device add --input device.json
```

**API:**
```bash
curl -X POST http://localhost:8080/api/devices \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server-01",
    "description": "Production web server",
    "make_model": "Dell PowerEdge R740",
    "os": "Ubuntu 22.04",
    "datacenter_id": "dc-east-1",
    "location": "Rack 10, U5-7",
    "tags": ["web", "production", "nginx"],
    "addresses": [
      {
        "ip": "10.1.1.50",
        "port": 80,
        "type": "http",
        "label": "Web interface"
      },
      {
        "ip": "10.1.1.50",
        "port": 22,
        "type": "ssh",
        "label": "Management"
      }
    ],
    "domains": ["web.example.com"]
  }'
```

### Read Device

**CLI:**
```bash
# Get specific device
rackd device get --id "device-123"

# List all devices
rackd device list

# Filter devices
rackd device list --tags "production,web" --datacenter "dc-east-1"

# Search devices
rackd device list --query "web server"

# Output formats
rackd device get --id "device-123" --output json
rackd device list --output yaml
```

**API:**
```bash
# Get device by ID
curl http://localhost:8080/api/devices/device-123

# List all devices
curl http://localhost:8080/api/devices

# Filter by tags
curl "http://localhost:8080/api/devices?tags=production,web"

# Filter by datacenter
curl "http://localhost:8080/api/devices?datacenter_id=dc-east-1"

# Search devices
curl "http://localhost:8080/api/devices/search?q=web+server"
```

### Update Device

**CLI:**
```bash
# Update specific fields
rackd device update --id "device-123" \
  --description "Updated web server" \
  --tags "web,production,updated"

# Update from JSON file
rackd device update --id "device-123" --input updates.json
```

**API:**
```bash
curl -X PATCH http://localhost:8080/api/devices/device-123 \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated production web server",
    "tags": ["web", "production", "nginx", "updated"],
    "addresses": [
      {
        "ip": "10.1.1.51",
        "port": 80,
        "type": "http",
        "label": "Primary web interface"
      }
    ]
  }'
```

### Delete Device

**CLI:**
```bash
rackd device delete --id "device-123"
```

**API:**
```bash
curl -X DELETE http://localhost:8080/api/devices/device-123
```

## Addresses

Devices can have multiple network addresses for different purposes:

### Address Types
- `management` - Management interface
- `data` - Data/production traffic
- `backup` - Backup network
- `ipmi` - IPMI/BMC interface
- `ssh` - SSH access
- `http`/`https` - Web interfaces
- Custom types as needed

### Examples

**Adding multiple addresses:**
```json
{
  "addresses": [
    {
      "ip": "10.1.1.100",
      "port": 22,
      "type": "ssh",
      "label": "SSH Management",
      "network_id": "mgmt-network"
    },
    {
      "ip": "192.168.1.100",
      "type": "ipmi",
      "label": "IPMI Interface",
      "switch_port": "Gi1/0/24"
    },
    {
      "ip": "10.2.1.100",
      "port": 80,
      "type": "http",
      "label": "Web Interface",
      "pool_id": "web-pool-1"
    }
  ]
}
```

## Tags

Tags provide flexible categorization and filtering:

### Common Tag Patterns
- **Environment**: `production`, `staging`, `development`
- **Function**: `web`, `database`, `storage`, `compute`
- **Technology**: `nginx`, `mysql`, `docker`, `kubernetes`
- **Location**: `rack-15`, `floor-2`, `building-a`
- **Status**: `active`, `maintenance`, `decommissioned`

### Tag Usage
```bash
# Filter by multiple tags
rackd device list --tags "production,database"

# Search includes tags
rackd device list --query "mysql"
```

## Domains

Associate DNS domains with devices:

```json
{
  "domains": [
    "web01.example.com",
    "www.example.com",
    "api.example.com"
  ]
}
```

## Relationships

Devices can have relationships with other devices:

### Relationship Types
- `contains` - Physical containment (rack contains servers)
- `connected_to` - Network/physical connections
- `depends_on` - Service dependencies

### Managing Relationships

**API:**
```bash
# Add relationship
curl -X POST http://localhost:8080/api/devices/rack-01/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "child_id": "server-01",
    "type": "contains"
  }'

# Get device relationships
curl http://localhost:8080/api/devices/server-01/relationships

# Get related devices
curl "http://localhost:8080/api/devices/server-01/related?type=depends_on"

# Remove relationship
curl -X DELETE http://localhost:8080/api/devices/rack-01/relationships/server-01/contains
```

## Search and Filtering

### Search Capabilities
- **Text search**: Name, description, hostname, make/model
- **Tag filtering**: Exact tag matches
- **Datacenter filtering**: Filter by datacenter ID
- **Network filtering**: Filter by network association

### Filter Examples

**CLI:**
```bash
# Complex filtering
rackd device list \
  --tags "production,web" \
  --datacenter "dc-east-1" \
  --network "prod-network" \
  --limit 50

# Search with query
rackd device list --query "Dell PowerEdge"
```

**API:**
```bash
# Multiple filters
curl "http://localhost:8080/api/devices?tags=production,web&datacenter_id=dc-east-1&network_id=prod-network"

# Search query
curl "http://localhost:8080/api/devices/search?q=Dell+PowerEdge"
```

## Web UI Examples

### Device List View
- Sortable columns: Name, Type, Location, Status
- Filter sidebar: Tags, Datacenter, Network
- Search bar with real-time filtering
- Bulk operations: Tag assignment, deletion

### Device Detail View
- Overview tab: Basic information, addresses, tags
- Relationships tab: Visual relationship graph
- History tab: Change audit log
- Actions: Edit, Clone, Delete

### Device Creation Form
- Step-by-step wizard
- Address management with validation
- Tag autocomplete
- Datacenter/network selection dropdowns

## Validation Rules

### Required Fields
- `name` - Device name (max 255 chars)

### Optional Field Limits
- `hostname` - Valid hostname format (max 253 chars)
- `description` - Max 4096 characters
- `tags` - Each tag max 128 characters
- `domains` - Valid domain format (max 253 chars each)

### Address Validation
- `ip` - Valid IPv4/IPv6 address (required)
- `port` - 1-65535 range
- `type` - Max 64 characters
- `label` - Max 128 characters

## Best Practices

### Naming Conventions
```bash
# Environment-function-number
web-prod-01, db-staging-02, cache-dev-01

# Location-based
dc1-web-01, rack15-switch-01
```

### Tagging Strategy
```bash
# Hierarchical tags
environment:production
function:web
stack:lamp
location:dc-east-1
```

### Address Management
- Use consistent address types across devices
- Document port purposes in labels
- Associate addresses with appropriate networks
- Track switch port connections for physical devices

### Relationship Modeling
- Model physical containment (racks → servers)
- Track network connections (switches ↔ servers)
- Document service dependencies (web → database)