# Full-Text Search Implementation

## Overview

Rackd includes SQLite FTS5 (Full-Text Search) for fast, efficient searching across devices, networks, and datacenters via a unified search API.

## Features

- **Unified search endpoint** - Single `/api/search` endpoint returns all matching results
- **FTS5-powered search** for devices, networks, and datacenters
- **Comprehensive device search** across:
  - Device fields (name, hostname, description, make/model, OS, location)
  - Tags
  - Domains
  - IP addresses
- **Network search** across name, subnet, and description
- **Datacenter search** across name, location, and description
- **Automatic index maintenance** via SQLite triggers
- **Special character handling** for safe query execution

## API Endpoint

### Unified Search
```
GET /api/search?q=<query>
```

Returns:
```json
{
  "results": [
    {
      "type": "device",
      "device": { ... }
    },
    {
      "type": "network",
      "network": { ... }
    },
    {
      "type": "datacenter",
      "datacenter": { ... }
    }
  ]
}
```

## Web UI Integration

The global search bar uses the unified `/api/search` endpoint to search across all entity types simultaneously. Results are displayed in a dropdown with keyboard navigation support.

## Implementation Details

### Database Schema

Three FTS5 virtual tables are created:
- `devices_fts` - Indexes device name, hostname, description, make_model, os, location
- `networks_fts` - Indexes network name, subnet, description
- `datacenters_fts` - Indexes datacenter name, location, description

### Triggers

Automatic triggers keep FTS tables synchronized with source tables:
- `*_fts_insert` - Populates FTS table on INSERT
- `*_fts_update` - Updates FTS table on UPDATE
- `*_fts_delete` - Removes from FTS table on DELETE

### Search Strategy

Device search uses a UNION query to combine results from:
1. FTS5 search on main device fields
2. LIKE search on tags
3. LIKE search on domains
4. LIKE search on IP addresses

This hybrid approach provides comprehensive search coverage while leveraging FTS5 performance for text fields.

### Query Escaping

Special characters in search queries are automatically escaped to prevent FTS5 syntax errors. Queries are wrapped in quotes for phrase matching.

## Migration

The FTS implementation is added via migration `20260203110000_add_fts_search`:
- Creates FTS5 virtual tables
- Creates synchronization triggers
- Populates FTS tables with existing data

## Performance

FTS5 provides significant performance improvements over LIKE queries:
- Indexed token-based search
- Efficient prefix matching
- Scales well with large datasets
- Server-side filtering reduces network traffic

## Storage Interfaces

Methods added to storage interfaces:
```go
type DeviceStorage interface {
    SearchDevices(query string) ([]model.Device, error)
}

type NetworkStorage interface {
    SearchNetworks(query string) ([]model.Network, error)
}

type DatacenterStorage interface {
    SearchDatacenters(query string) ([]model.Datacenter, error)
}
```

## Testing

Comprehensive tests cover:
- Basic search functionality
- Special character handling
- Multiple match scenarios
- Empty query handling
- Tag/domain/address search

All tests pass successfully.
