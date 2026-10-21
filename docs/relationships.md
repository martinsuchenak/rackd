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

## Direction

A relationship is stored from a parent device to a child device. Both devices see it:

- `GET /devices/{id}/relationships` returns relationships where the device is the parent or the child
- `GET /devices/{id}/related?type=<type>` returns the devices on the other side, in either direction

Updating or removing a relationship uses the parent's ID in the path.

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
# Server in rack (rack-01 is the parent)
curl -H "Authorization: Bearer $RACKD_TOKEN" -X POST http://localhost:8080/api/v1/devices/rack-01/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "child_id": "server-web-01",
    "type": "contains"
  }'

# Network connection, with notes
curl -H "Authorization: Bearer $RACKD_TOKEN" -X POST http://localhost:8080/api/v1/devices/server-web-01/relationships \
  -H "Content-Type: application/json" \
  -d '{
    "child_id": "switch-core-01",
    "type": "connected_to",
    "notes": "eth0 to Gi1/0/12"
  }'
```

Adding a relationship that already exists replaces its notes.

### Query Relationships
```bash
# Get all relationships for a device
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/devices/server-web-01/relationships

# Get the devices related by one type
curl -H "Authorization: Bearer $RACKD_TOKEN" "http://localhost:8080/api/v1/devices/server-web-01/related?type=connected_to"

# Get every relationship
curl -H "Authorization: Bearer $RACKD_TOKEN" http://localhost:8080/api/v1/relationships
```

### Update Notes and Remove
```bash
# Update notes
curl -H "Authorization: Bearer $RACKD_TOKEN" -X PATCH http://localhost:8080/api/v1/devices/server-web-01/relationships/switch-core-01/connected_to \
  -H "Content-Type: application/json" \
  -d '{"notes": "eth1 to Gi1/0/13"}'

# Remove
curl -H "Authorization: Bearer $RACKD_TOKEN" -X DELETE http://localhost:8080/api/v1/devices/server-web-01/relationships/switch-core-01/connected_to
```

## Other Interfaces

There is no `rackd` CLI command for relationships. Besides the API:

- **Web UI**: the device detail page lists a device's relationships, adds and removes them, and edits notes; `/devices/graph` shows the relationship graph
- **MCP**: `device_add_relationship` and `device_get_relationships`

## Implementation

Relationships are handled by the relationship service and stored in the `device_relationships` table. The relationship model supports:

- Querying from either device
- Relationship type validation
- Cascade deletion when devices are removed

See the relationship handlers in `internal/api/relationship_handlers.go` and the relationship model in `pkg/model/relationship.go` for implementation details.