# Datacenter Management

Rackd's datacenter management system provides organization and tracking of physical locations where your infrastructure is deployed.

## Datacenter Model

A datacenter in Rackd represents a physical location containing infrastructure devices:

```go
type Datacenter struct {
    ID          string    `json:"id"`          // Unique identifier
    Name        string    `json:"name"`        // Datacenter name (required)
    Location    string    `json:"location"`    // Geographic location
    Description string    `json:"description"` // Datacenter description
    CreatedAt   time.Time `json:"created_at"`  // Creation timestamp
    UpdatedAt   time.Time `json:"updated_at"`  // Last update timestamp
}
```

### Validation Rules

- **Name**: Required, maximum 255 characters
- **Location**: Optional, maximum 255 characters
- **Description**: Optional, maximum 4096 characters

## CRUD Operations

### Create Datacenter

**CLI:**
```bash
# Basic datacenter creation
rackd datacenter add --name "Primary DC" --location "New York" --description "Main production datacenter"

# Minimal creation (only name required)
rackd datacenter add --name "DR Site"

# JSON output
rackd datacenter add --name "Edge DC" --output json
```

**API:**
```bash
# Create datacenter
curl -X POST http://localhost:8080/api/datacenters \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Primary DC",
    "location": "New York",
    "description": "Main production datacenter"
  }'
```

**Response:**
```json
{
  "id": "dc-123e4567-e89b-12d3-a456-426614174000",
  "name": "Primary DC",
  "location": "New York", 
  "description": "Main production datacenter",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

### List Datacenters

**CLI:**
```bash
# List all datacenters (table format)
rackd datacenter list

# JSON output
rackd datacenter list --output json

# YAML output
rackd datacenter list --output yaml
```

**API:**
```bash
# List all datacenters
curl http://localhost:8080/api/datacenters

# Filter by name
curl "http://localhost:8080/api/datacenters?name=Primary"
```

**Response:**
```json
[
  {
    "id": "dc-123e4567-e89b-12d3-a456-426614174000",
    "name": "Primary DC",
    "location": "New York",
    "description": "Main production datacenter",
    "created_at": "2024-01-15T10:30:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
]
```

### Get Datacenter

**CLI:**
```bash
# Get datacenter details
rackd datacenter get --id dc-123e4567-e89b-12d3-a456-426614174000

# JSON output
rackd datacenter get --id dc-123e4567-e89b-12d3-a456-426614174000 --output json
```

**API:**
```bash
# Get specific datacenter
curl http://localhost:8080/api/datacenters/dc-123e4567-e89b-12d3-a456-426614174000
```

### Update Datacenter

**CLI:**
```bash
# Update datacenter
rackd datacenter update --id dc-123e4567-e89b-12d3-a456-426614174000 \
  --name "Updated DC Name" \
  --location "Updated Location"
```

**API:**
```bash
# Update datacenter (partial update)
curl -X PUT http://localhost:8080/api/datacenters/dc-123e4567-e89b-12d3-a456-426614174000 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated DC Name",
    "location": "Updated Location"
  }'
```

### Delete Datacenter

**CLI:**
```bash
# Delete datacenter
rackd datacenter delete --id dc-123e4567-e89b-12d3-a456-426614174000
```

**API:**
```bash
# Delete datacenter
curl -X DELETE http://localhost:8080/api/datacenters/dc-123e4567-e89b-12d3-a456-426614174000
```

## Device Associations

Devices can be associated with datacenters through the `datacenter_id` field. This enables physical location tracking and organization.

### Get Datacenter Devices

**API:**
```bash
# Get all devices in a datacenter
curl http://localhost:8080/api/datacenters/dc-123e4567-e89b-12d3-a456-426614174000/devices
```

**Response:**
```json
[
  {
    "id": "dev-456e7890-e89b-12d3-a456-426614174001",
    "name": "web-server-01",
    "datacenter_id": "dc-123e4567-e89b-12d3-a456-426614174000",
    "addresses": [
      {
        "ip": "192.168.1.10",
        "type": "management"
      }
    ]
  }
]
```

### Associate Device with Datacenter

**CLI:**
```bash
# Create device with datacenter association
rackd device add --name "web-server-01" \
  --datacenter-id dc-123e4567-e89b-12d3-a456-426614174000

# Update existing device
rackd device update --id dev-456e7890-e89b-12d3-a456-426614174001 \
  --datacenter-id dc-123e4567-e89b-12d3-a456-426614174000
```

**API:**
```bash
# Create device with datacenter
curl -X POST http://localhost:8080/api/devices \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-server-01",
    "datacenter_id": "dc-123e4567-e89b-12d3-a456-426614174000"
  }'
```

## Filtering

### Name Filter

Filter datacenters by name using partial matching:

**API:**
```bash
# Find datacenters with "Primary" in the name
curl "http://localhost:8080/api/datacenters?name=Primary"
```

The filter supports partial matching, so searching for "DC" would match "Primary DC", "Secondary DC", etc.

## Web UI Examples

### Datacenter List View

The web interface provides a table view of all datacenters with:
- Name and location
- Device count per datacenter
- Creation and update timestamps
- Actions (view, edit, delete)

### Datacenter Detail View

Individual datacenter pages show:
- Complete datacenter information
- List of associated devices
- Device statistics and summaries
- Quick actions for device management

### Creating Datacenters

The web form includes:
- Name field (required)
- Location field (optional)
- Description textarea (optional)
- Real-time validation feedback

## Error Handling

### Validation Errors

**Response (400 Bad Request):**
```json
{
  "error": "VALIDATION_ERROR",
  "message": "Validation failed",
  "details": [
    {
      "field": "name",
      "message": "name is required"
    }
  ]
}
```

### Not Found Errors

**Response (404 Not Found):**
```json
{
  "error": "DATACENTER_NOT_FOUND",
  "message": "Datacenter not found"
}
```

## Best Practices

### Naming Conventions

- Use descriptive names: "Primary DC", "DR Site", "Edge Location"
- Include geographic indicators: "NYC-DC1", "LON-Primary"
- Avoid special characters in names

### Organization

- Create datacenters before adding devices
- Use consistent location formatting
- Include relevant details in descriptions
- Tag devices appropriately for datacenter-specific filtering

### Maintenance

- Regularly audit device-datacenter associations
- Update location information when facilities change
- Use meaningful descriptions for operational context