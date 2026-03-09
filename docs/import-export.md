# Import/Export

Rackd supports importing and exporting data in JSON and CSV formats. This is useful for bulk operations, migrations, backups, and integration with external systems.

## Overview

The import/export functionality supports:

- **Devices** - Import/export device inventory
- **Networks** - Import/export network definitions
- **Datacenters** - Import/export datacenter configurations
- **Full Export** - Export all data in a single JSON file

## File Formats

### JSON Format

JSON files support nested objects and arrays, making them ideal for complex data structures.

**Example devices.json:**

```json
[
  {
    "name": "web-server-01",
    "description": "Production web server",
    "make_model": "Dell PowerEdge R640",
    "os": "Ubuntu 22.04",
    "datacenter_id": "dc-001",
    "username": "admin",
    "location": "Rack 5, U10",
    "tags": ["production", "web"],
    "domains": ["web-01.example.com"],
    "addresses": [
      {"type": "management", "address": "10.0.1.10"},
      {"type": "primary", "address": "192.168.1.10"}
    ]
  },
  {
    "name": "db-server-01",
    "description": "PostgreSQL database server",
    "make_model": "Dell PowerEdge R740",
    "os": "Ubuntu 22.04",
    "datacenter_id": "dc-001",
    "tags": ["production", "database"],
    "addresses": [
      {"type": "primary", "address": "192.168.1.20"}
    ]
  }
]
```

**Example networks.json:**

```json
[
  {
    "name": "Production LAN",
    "cidr": "192.168.1.0/24",
    "vlan": 100,
    "datacenter_id": "dc-001",
    "gateway": "192.168.1.1",
    "description": "Production network"
  },
  {
    "name": "Management Network",
    "cidr": "10.0.1.0/24",
    "vlan": 10,
    "datacenter_id": "dc-001",
    "gateway": "10.0.1.1",
    "description": "Out-of-band management"
  }
]
```

**Example datacenters.json:**

```json
[
  {
    "name": "Primary DC",
    "location": "New York, NY",
    "description": "Main datacenter"
  },
  {
    "name": "Secondary DC",
    "location": "Chicago, IL",
    "description": "Disaster recovery site"
  }
]
```

### CSV Format

CSV files are simpler and work well with spreadsheets. Complex fields (like addresses and tags) use special formatting.

**Example devices.csv:**

```csv
name,description,make_model,os,datacenter_id,username,location,tags,domains,addresses
web-server-01,Production web server,Dell PowerEdge R640,Ubuntu 22.04,dc-001,admin,"Rack 5, U10",production;web,web-01.example.com,management:10.0.1.10;primary:192.168.1.10
db-server-01,PostgreSQL database server,Dell PowerEdge R740,Ubuntu 22.04,dc-001,,,"Rack 6, U5",production;database,,primary:192.168.1.20
```

**CSV Field Formats:**

| Field | Format | Example |
|-------|--------|---------|
| tags | Semicolon-separated | `production;web;critical` |
| domains | Semicolon-separated | `server1.example.com;server1.local` |
| addresses | Semicolon-separated `type:ip` pairs | `management:10.0.1.10;primary:192.168.1.10` |

## Importing Data

### Import Devices

Import devices from a JSON or CSV file:

```bash
# Import from JSON (auto-detected by extension)
rackd import devices --file devices.json

# Import from CSV
rackd import devices --file devices.csv

# Validate without importing (dry run)
rackd import devices --file devices.json --dry-run

# Explicitly specify format
rackd import devices --file data.txt --format json
```

**Options:**

| Flag | Description |
|------|-------------|
| `--file <path>` | Input file path (required) |
| `--format <fmt>` | Input format: `json` or `csv` (auto-detected if omitted) |
| `--dry-run` | Validate file without importing |

**Import Results:**

```
Parsed 50 devices from devices.json

Import complete:
  Total:   50
  Created: 48
  Failed:  2

Errors:
  - web-server-05: IP address 192.168.1.10 already in use
  - db-server-03: Invalid CIDR format
```

### Import Networks

```bash
rackd import networks --file networks.json
rackd import networks --file networks.csv --dry-run
```

### Import Datacenters

```bash
rackd import datacenters --file datacenters.json
```

### Import Tips

1. **Import order matters** - Import datacenters first, then networks, then devices
2. **Use dry-run first** - Validate your file before importing
3. **Check references** - Ensure `datacenter_id` values exist before importing
4. **Bulk imports** - For large imports, JSON is more reliable

## Exporting Data

### Export Devices

```bash
# Export to JSON (default)
rackd export devices

# Export to file
rackd export devices --output devices.json

# Export to CSV
rackd export devices --format csv --output devices.csv
```

**Options:**

| Flag | Description |
|------|-------------|
| `--format <fmt>` | Output format: `json` or `csv` (default: `json`) |
| `--output <path>` | Output file (default: stdout) |

### Export Networks

```bash
rackd export networks --output networks.json
rackd export networks --format csv --output networks.csv
```

### Export Datacenters

```bash
rackd export datacenters --output datacenters.json
```

### Export All

Export all data in a single JSON file:

```bash
# Export to stdout
rackd export all

# Export to file
rackd export all --output rackd-backup.json
```

**Output structure:**

```json
{
  "devices": [...],
  "networks": [...],
  "datacenters": [...],
  "exported_at": "2024-01-20T10:30:00Z"
}
```

## API Reference

### Import Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/devices/bulk` | Bulk create devices |
| POST | `/api/networks/bulk` | Bulk create networks |
| POST | `/api/datacenters` | Create datacenter |

### Export Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/devices` | List all devices |
| GET | `/api/networks` | List all networks |
| GET | `/api/datacenters` | List all datacenters |

## Migration Scenarios

### Migrating from Another IPAM

1. Export data from your current system to CSV or JSON
2. Transform the data to match Rackd's schema
3. Import datacenters first
4. Import networks
5. Import devices
6. Verify with `rackd device list`

### Disaster Recovery

```bash
# Regular backup exports
rackd export all --output /backups/rackd-$(date +%Y%m%d).json

# Restore from backup
# (Import order: datacenters → networks → devices)
rackd import datacenters --file backup.json  # Requires splitting the file
```

### Bulk Updates

For bulk updates, export, modify, and re-import:

```bash
# Export current data
rackd export devices --output devices.json

# Edit devices.json in your editor or script
# Then update devices via the API or CLI
```

## Field Reference

### Device Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Device hostname |
| `description` | string | No | Description |
| `make_model` | string | No | Hardware make/model |
| `os` | string | No | Operating system |
| `datacenter_id` | string | No | Associated datacenter ID |
| `username` | string | No | Login username |
| `location` | string | No | Physical location |
| `tags` | array | No | List of tags |
| `domains` | array | No | List of domain names |
| `addresses` | array | No | IP address objects |

### Network Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Network name |
| `cidr` | string | Yes | Network CIDR |
| `vlan` | integer | No | VLAN ID |
| `datacenter_id` | string | No | Associated datacenter ID |
| `gateway` | string | No | Gateway IP |
| `description` | string | No | Description |

### Datacenter Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Datacenter name |
| `location` | string | No | Physical location |
| `description` | string | No | Description |

## Troubleshooting

### Import Fails with "Invalid format"

- Ensure JSON is valid (use `jq` to validate)
- Check CSV has correct headers
- Verify special characters are properly escaped

### "Reference not found" Errors

- Import datacenters before networks
- Import networks before devices
- Ensure IDs in references exist

### Partial Import Success

- Check error messages for failed records
- Fix the issues in your import file
- Re-import (duplicate names/IPs will be skipped)

### Large File Imports Timeout

- Split large files into smaller batches
- Use the API directly with smaller payloads
- Increase CLI timeout: `--timeout 5m`

## Best Practices

1. **Use dry-run first** - Always validate before importing
2. **Keep backups** - Regular exports for disaster recovery
3. **Version control** - Store import files in git
4. **Document mappings** - When migrating, document field mappings
5. **Test restores** - Periodically test restoring from exports

## Related Documentation

- [CLI Reference](cli.md) - Import/export commands
- [API Reference](api.md) - REST API endpoints
- [Backup](cli.md#backup) - Database backup command
