# Device Relationships

Rackd supports tracking relationships between devices to model physical connections, dependencies, and containment hierarchies in your infrastructure.

## Relationship Types

### contains
Represents physical or logical containment relationships.

**Examples:**
- Rack contains servers
- Chassis contains blade servers
- Switch contains line cards
- Datacenter contains racks

### connected_to
Represents physical network or power connections between devices.

**Examples:**
- Server connected to switch via network cable
- Switch connected to router
- Server connected to PDU for power
- Storage array connected to SAN switch

### depends_on
Represents logical dependencies where one device relies on another for functionality.

**Examples:**
- Virtual machine depends on hypervisor
- Application server depends on database server
- Load balancer depends on backend servers
- Monitoring system depends on network infrastructure

## Bidirectional Relationships

All relationships in Rackd are bidirectional, meaning they can be viewed from either device's perspective:

- If Device A **contains** Device B, then Device B is **contained by** Device A
- If Device A is **connected_to** Device B, then Device B is **connected_to** Device A  
- If Device A **depends_on** Device B, then Device B **supports** Device A

## Use Cases

### Infrastructure Mapping
Track physical layout and connections to understand network topology and plan changes.

### Impact Analysis
Identify which devices will be affected when a device goes offline or requires maintenance.

### Capacity Planning
Understand containment relationships to track rack space, power consumption, and cooling requirements.

### Troubleshooting
Follow connection paths to isolate network issues and identify single points of failure.

### Compliance
Document device relationships for audit trails and regulatory compliance.

## API Examples

### Create Relationship
```bash
# Server in rack
curl -X POST http://localhost:8080/api/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "source_device_id": "rack-01",
    "target_device_id": "server-web-01", 
    "relationship_type": "contains"
  }'

# Network connection
curl -X POST http://localhost:8080/api/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "source_device_id": "server-web-01",
    "target_device_id": "switch-core-01",
    "relationship_type": "connected_to"
  }'
```

### Query Relationships
```bash
# Get all relationships for a device
curl http://localhost:8080/api/devices/server-web-01/relationships

# Get specific relationship type
curl http://localhost:8080/api/devices/server-web-01/relationships?type=connected_to
```

## CLI Examples

### Create Relationships
```bash
# Add server to rack
rackd relationship create --source rack-01 --target server-web-01 --type contains

# Connect server to switch
rackd relationship create --source server-web-01 --target switch-core-01 --type connected_to

# Add dependency
rackd relationship create --source app-server-01 --target db-server-01 --type depends_on
```

### List Relationships
```bash
# Show all relationships for a device
rackd relationship list --device server-web-01

# Show only dependencies
rackd relationship list --device server-web-01 --type depends_on
```

## Implementation

Relationships are handled by the relationship service and stored in the relationships table. The relationship model supports:

- Bidirectional querying
- Relationship type validation
- Cascade deletion when devices are removed
- Circular dependency detection for `depends_on` relationships

See the relationship handlers in `internal/handlers/relationship.go` and the relationship model in `internal/models/relationship.go` for implementation details.