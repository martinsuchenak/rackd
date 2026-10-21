# Custom Fields

Rackd supports user-defined custom fields for devices, allowing you to track additional metadata specific to your organization.

## Overview

Custom fields provide flexibility to extend device records with organization-specific data:

- Asset tracking numbers
- Warranty information
- Department/owner assignments
- Physical location details
- Compliance attributes
- Any custom metadata you need

## Custom Field Types

| Type | Description | Example |
|------|-------------|---------|
| `text` | Free-form text input | Serial numbers, notes |
| `number` | Numeric values | Rack units, power draw |
| `boolean` | True/false checkbox | Under maintenance |
| `select` | Dropdown with predefined options | Environment, tier |

## Custom Field Definition

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier |
| `name` | string | Field name (display label) |
| `key` | string | Unique key for API access |
| `type` | string | Field type (text/number/boolean/select) |
| `required` | bool | Whether field is mandatory |
| `description` | string | Help text for users |
| `options` | []string | Options for select type |
| `created_at` | timestamp | Creation timestamp |
| `updated_at` | timestamp | Last update timestamp |

## API Endpoints

### List Custom Field Definitions

```http
GET /api/v1/custom-fields
```

**Response:**
```json
[
  {
    "id": "cf-abc123",
    "name": "Asset Tag",
    "key": "asset_tag",
    "type": "text",
    "required": false,
    "description": "Organization asset tracking number"
  },
  {
    "id": "cf-def456",
    "name": "Environment",
    "key": "environment",
    "type": "select",
    "required": true,
    "options": ["production", "staging", "development", "testing"]
  }
]
```

### Get Custom Field Definition

```http
GET /api/v1/custom-fields/{id}
```

### Get Available Field Types

```http
GET /api/v1/custom-fields/types
```

**Response:**
```json
[
  {"value": "text", "label": "Text"},
  {"value": "number", "label": "Number"},
  {"value": "boolean", "label": "Boolean"},
  {"value": "select", "label": "Select"}
]
```

### Create Custom Field Definition

```http
POST /api/v1/custom-fields
```

**Request body:**
```json
{
  "name": "Warranty Expiry",
  "key": "warranty_expiry",
  "type": "text",
  "required": false,
  "description": "Warranty expiration date"
}
```

For select fields:
```json
{
  "name": "Environment",
  "key": "environment",
  "type": "select",
  "required": true,
  "options": ["production", "staging", "development"]
}
```

### Update Custom Field Definition

```http
PUT /api/v1/custom-fields/{id}
```

All fields, including `type`, can be changed. Changing a field to `select` needs `options`.

### Delete Custom Field Definition

```http
DELETE /api/v1/custom-fields/{id}
```

Warning: This will remove all values for this field from all devices.

## Using Custom Fields on Devices

### Create Device with Custom Fields

```http
POST /api/v1/devices
```

Values are keyed by the definition's `id`, given as `field_id`:

```json
{
  "name": "server01",
  "hostname": "server01.example.com",
  "addresses": [{"ip": "192.168.1.10"}],
  "custom_fields": [
    {"field_id": "<asset-tag-field-id>", "value": "AST-12345"},
    {"field_id": "<environment-field-id>", "value": "production"}
  ]
}
```

### Update Device Custom Fields

```http
PUT /api/v1/devices/{id}
```

```json
{
  "custom_fields": [
    {"field_id": "<asset-tag-field-id>", "value": "AST-12345-UPDATED"},
    {"field_id": "<warranty-expiry-field-id>", "value": "2025-12-31"}
  ]
}
```

## CLI Commands

### List Custom Field Definitions

```bash
rackd custom-field list
```

### Get Custom Field Definition

```bash
rackd custom-field get --id cf-abc123
```

### List Available Types

```bash
rackd custom-field types
```

### Create Custom Field Definition

```bash
# Text field
rackd custom-field create \
  --name "Asset Tag" \
  --key "asset_tag" \
  --type text \
  --description "Organization asset tracking number"

# Select field
rackd custom-field create \
  --name "Environment" \
  --key "environment" \
  --type select \
  --required \
  --options "production,staging,development,testing"
```

### Update Custom Field Definition

```bash
rackd custom-field update \
  --id cf-abc123 \
  --name "Asset ID" \
  --required
```

### Delete Custom Field Definition

```bash
rackd custom-field delete --id cf-abc123
```

## Web UI

### Custom Fields Management Page

Access at `/custom-fields` in the web interface.

Features:
- List all custom field definitions
- Create new fields with type selection
- Edit field properties
- Delete unused fields

### Device Form Integration

Custom fields appear in device forms:
- **Create/Edit Modal**: Custom Fields tab with all defined fields
- **Device Detail**: Custom fields section showing values
- **Validation**: Required fields enforced on form submission

## RBAC Permissions

| Permission | Description |
|------------|-------------|
| `custom-fields:list` | View list of custom field definitions |
| `custom-fields:read` | View field definition details |
| `custom-fields:create` | Create new field definitions |
| `custom-fields:update` | Modify field definitions |
| `custom-fields:delete` | Delete field definitions |

Note: Device-level custom field values respect device permissions.

## Common Use Cases

### Asset Management

```json
{
  "name": "Asset Tag",
  "key": "asset_tag",
  "type": "text",
  "required": true
}
```

```json
{
  "name": "Purchase Date",
  "key": "purchase_date",
  "type": "text",
  "description": "Date device was purchased (YYYY-MM-DD)"
}
```

```json
{
  "name": "Warranty Expiry",
  "key": "warranty_expiry",
  "type": "text"
}
```

### Physical Location

```json
{
  "name": "Rack Position",
  "key": "rack_position",
  "type": "text",
  "description": "Rack and U position (e.g., R10-U15)"
}
```

```json
{
  "name": "Data Center",
  "key": "dc_location",
  "type": "select",
  "options": ["DC1 - Primary", "DC2 - Secondary", "DC3 - DR"]
}
```

### Operational Status

```json
{
  "name": "Under Maintenance",
  "key": "maintenance",
  "type": "boolean"
}
```

```json
{
  "name": "Environment",
  "key": "environment",
  "type": "select",
  "required": true,
  "options": ["production", "staging", "development", "testing", "sandbox"]
}
```

### Compliance

```json
{
  "name": "PCI Scope",
  "key": "pci_scope",
  "type": "boolean",
  "description": "Is this device in PCI DSS scope?"
}
```

```json
{
  "name": "Compliance Review Date",
  "key": "compliance_review",
  "type": "text"
}
```

## Best Practices

1. **Key Naming**: Use snake_case for keys (e.g., `asset_tag`, `warranty_expiry`)
2. **Required Fields**: Only mark fields as required if truly mandatory
3. **Select Options**: Keep option lists manageable (< 20 items)
4. **Descriptions**: Add help text for fields that may be ambiguous
5. **Naming Convention**: Use clear, consistent field names
6. **Avoid Duplicates**: Check existing fields before creating new ones
7. **Type Selection**: Choose appropriate types (use `number` for numeric data, not `text`)
