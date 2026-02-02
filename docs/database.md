# Database Schema

Rackd uses SQLite as its embedded database with a comprehensive schema for managing devices, networks, IP addresses, and datacenter resources. This document provides a complete reference of all tables, columns, indexes, foreign keys, and migrations.

## Database Configuration

- **Engine**: SQLite with WAL (Write-Ahead Logging) mode
- **Foreign Keys**: Enabled (`PRAGMA foreign_keys=1`)
- **Connection Pool**: Single writer (SQLite limitation)
- **File Location**: `{dataDir}/rackd.db` or `:memory:` for testing

## Schema Overview

The database consists of 12 core tables organized into logical groups:

- **Infrastructure**: `datacenters`, `networks`, `network_pools`
- **Devices**: `devices`, `addresses`, `tags`, `domains`
- **Relationships**: `device_relationships`
- **Discovery**: `discovered_devices`, `discovery_scans`, `discovery_rules`
- **System**: `schema_migrations`, `pool_tags`

## Tables

### schema_migrations

Tracks applied database migrations.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| version | TEXT | PRIMARY KEY | Migration version (timestamp format) |
| name | TEXT | NOT NULL | Migration name |
| applied_at | TIMESTAMP | NOT NULL, DEFAULT CURRENT_TIMESTAMP | When migration was applied |
| checksum | TEXT | NOT NULL | Migration checksum for integrity |
| execution_time_ms | INTEGER | | Migration execution time in milliseconds |
| success | INTEGER | NOT NULL, DEFAULT 1 | Success flag (1=success, 0=failure) |

### datacenters

Physical datacenter locations.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| name | TEXT | NOT NULL | Datacenter name |
| location | TEXT | | Physical location/address |
| description | TEXT | | Datacenter description |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

### networks

Network subnets and VLANs.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| name | TEXT | NOT NULL | Network name |
| subnet | TEXT | NOT NULL | CIDR subnet (e.g., "192.168.1.0/24") |
| vlan_id | INTEGER | | VLAN ID (optional) |
| datacenter_id | TEXT | FOREIGN KEY → datacenters(id) | Associated datacenter |
| description | TEXT | | Network description |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

### network_pools

IP address pools within networks.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| network_id | TEXT | NOT NULL, FOREIGN KEY → networks(id) ON DELETE CASCADE | Parent network |
| name | TEXT | NOT NULL | Pool name |
| start_ip | TEXT | NOT NULL | Starting IP address |
| end_ip | TEXT | NOT NULL | Ending IP address |
| description | TEXT | | Pool description |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

### pool_tags

Tags for network pools (many-to-many).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| pool_id | TEXT | PRIMARY KEY, FOREIGN KEY → network_pools(id) ON DELETE CASCADE | Pool identifier |
| tag | TEXT | PRIMARY KEY | Tag name |

**Composite Primary Key**: (pool_id, tag)

### devices

Physical and virtual devices.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| name | TEXT | NOT NULL | Device name |
| hostname | TEXT | DEFAULT '' | Device hostname |
| description | TEXT | | Device description |
| make_model | TEXT | | Manufacturer and model |
| os | TEXT | | Operating system |
| datacenter_id | TEXT | FOREIGN KEY → datacenters(id) | Associated datacenter |
| username | TEXT | | Default username for access |
| location | TEXT | | Physical location within datacenter |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

### addresses

IP addresses assigned to devices.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| device_id | TEXT | NOT NULL, FOREIGN KEY → devices(id) ON DELETE CASCADE | Associated device |
| ip | TEXT | NOT NULL | IP address |
| port | INTEGER | | Port number (optional) |
| type | TEXT | DEFAULT 'ipv4' | Address type (ipv4, ipv6, etc.) |
| label | TEXT | | Address label/description |
| network_id | TEXT | FOREIGN KEY → networks(id) | Associated network |
| switch_port | TEXT | | Physical switch port |
| pool_id | TEXT | FOREIGN KEY → network_pools(id) | Associated IP pool |

### tags

Device tags (many-to-many).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| device_id | TEXT | PRIMARY KEY, FOREIGN KEY → devices(id) ON DELETE CASCADE | Device identifier |
| tag | TEXT | PRIMARY KEY | Tag name |

**Composite Primary Key**: (device_id, tag)

### domains

Domain names associated with devices (many-to-many).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| device_id | TEXT | PRIMARY KEY, FOREIGN KEY → devices(id) ON DELETE CASCADE | Device identifier |
| domain | TEXT | PRIMARY KEY | Domain name |

**Composite Primary Key**: (device_id, domain)

### device_relationships

Relationships between devices (dependencies, connections, etc.).

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| parent_id | TEXT | PRIMARY KEY, FOREIGN KEY → devices(id) ON DELETE CASCADE | Parent device |
| child_id | TEXT | PRIMARY KEY, FOREIGN KEY → devices(id) ON DELETE CASCADE | Child device |
| type | TEXT | PRIMARY KEY, NOT NULL | Relationship type |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |

**Composite Primary Key**: (parent_id, child_id, type)

### discovered_devices

Devices found through network discovery.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| ip | TEXT | NOT NULL | Discovered IP address |
| mac_address | TEXT | | MAC address (if available) |
| hostname | TEXT | | Discovered hostname |
| network_id | TEXT | FOREIGN KEY → networks(id) | Associated network |
| status | TEXT | DEFAULT 'unknown' | Device status |
| confidence | INTEGER | DEFAULT 0 | Discovery confidence level |
| os_guess | TEXT | | Operating system guess |
| vendor | TEXT | | Hardware vendor |
| open_ports | TEXT | | JSON array of open ports |
| services | TEXT | | JSON array of detected services |
| first_seen | TIMESTAMP | | First discovery timestamp |
| last_seen | TIMESTAMP | | Last seen timestamp |
| promoted_to_device_id | TEXT | FOREIGN KEY → devices(id) | If promoted to managed device |
| promoted_at | TIMESTAMP | | Promotion timestamp |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

### discovery_scans

Network discovery scan jobs.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| network_id | TEXT | FOREIGN KEY → networks(id) | Target network |
| status | TEXT | DEFAULT 'pending' | Scan status |
| scan_type | TEXT | DEFAULT 'full' | Type of scan |
| total_hosts | INTEGER | DEFAULT 0 | Total hosts to scan |
| scanned_hosts | INTEGER | DEFAULT 0 | Hosts scanned so far |
| found_hosts | INTEGER | DEFAULT 0 | Active hosts found |
| progress_percent | REAL | DEFAULT 0 | Scan progress percentage |
| error_message | TEXT | | Error message if failed |
| started_at | TIMESTAMP | | Scan start time |
| completed_at | TIMESTAMP | | Scan completion time |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

### discovery_rules

Automated discovery rules for networks.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | TEXT | PRIMARY KEY | UUID identifier |
| network_id | TEXT | UNIQUE, FOREIGN KEY → networks(id) | Target network |
| enabled | INTEGER | DEFAULT 1 | Rule enabled flag |
| scan_type | TEXT | DEFAULT 'full' | Type of scan to perform |
| interval_hours | INTEGER | DEFAULT 24 | Scan interval in hours |
| exclude_ips | TEXT | | JSON array of IPs to exclude |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

## Indexes

Performance indexes for common query patterns:

```sql
-- Device indexes
CREATE INDEX idx_devices_name ON devices(name);
CREATE INDEX idx_devices_datacenter ON devices(datacenter_id);

-- Address indexes
CREATE INDEX idx_addresses_device ON addresses(device_id);
CREATE INDEX idx_addresses_ip ON addresses(ip);
CREATE INDEX idx_addresses_network ON addresses(network_id);
CREATE INDEX idx_addresses_pool ON addresses(pool_id);

-- Tag indexes
CREATE INDEX idx_tags_device ON tags(device_id);
CREATE INDEX idx_pool_tags_pool ON pool_tags(pool_id);

-- Domain indexes
CREATE INDEX idx_domains_device ON domains(device_id);

-- Network indexes
CREATE INDEX idx_networks_datacenter ON networks(datacenter_id);
CREATE INDEX idx_network_pools_network ON network_pools(network_id);

-- Discovery indexes
CREATE INDEX idx_discovered_devices_network ON discovered_devices(network_id);
CREATE INDEX idx_discovered_devices_ip ON discovered_devices(ip);
CREATE INDEX idx_discovery_scans_network ON discovery_scans(network_id);

-- Relationship indexes
CREATE INDEX idx_device_relationships_parent ON device_relationships(parent_id);
CREATE INDEX idx_device_relationships_child ON device_relationships(child_id);
```

## Foreign Key Relationships

```
datacenters
├── networks (datacenter_id)
└── devices (datacenter_id)

networks
├── network_pools (network_id) [CASCADE DELETE]
├── addresses (network_id)
├── discovered_devices (network_id)
├── discovery_scans (network_id)
└── discovery_rules (network_id)

network_pools
├── pool_tags (pool_id) [CASCADE DELETE]
└── addresses (pool_id)

devices
├── addresses (device_id) [CASCADE DELETE]
├── tags (device_id) [CASCADE DELETE]
├── domains (device_id) [CASCADE DELETE]
├── device_relationships (parent_id) [CASCADE DELETE]
├── device_relationships (child_id) [CASCADE DELETE]
└── discovered_devices (promoted_to_device_id)
```

## Migrations

### Migration 20240120080000: initial_schema
- Creates all core tables
- Establishes foreign key relationships
- Creates performance indexes
- Sets up cascade delete rules

### Migration 20240121080000: add_pool_tags
- Adds `pool_tags` table for network pool tagging
- Creates index for pool tag queries

### Migration 20240122080000: add_device_hostname
- Adds `hostname` column to `devices` table
- Defaults to empty string for existing records

## Data Types and Constraints

### UUID Generation
- Primary keys use UUIDv7 (time-ordered) when available
- Falls back to UUIDv4 for compatibility
- Generated by `newUUID()` function

### Timestamps
- All timestamps stored as UTC
- Uses SQLite's `CURRENT_TIMESTAMP` for defaults
- Application manages `updated_at` fields

### NULL Handling
- Optional foreign keys use `sql.NullString`
- Optional integers use `sql.NullInt64`
- Empty strings converted to NULL for consistency

### JSON Fields
- `open_ports` and `services` in `discovered_devices`
- `exclude_ips` in `discovery_rules`
- Stored as TEXT, parsed by application

## Performance Considerations

### Connection Pool
- Single writer connection (SQLite limitation)
- WAL mode for better read concurrency
- Connection lifetime: 1 hour

### Query Optimization
- Indexes on all foreign keys
- Composite indexes for multi-column queries
- EXPLAIN QUERY PLAN used for optimization

### Data Limits
- IP pool enumeration capped at 65,536 addresses
- Large subnet calculations capped at /12 (1M addresses)
- Discovery scan results paginated

## Backup and Recovery

### WAL Mode Benefits
- Continuous backup possible
- Better crash recovery
- Reduced lock contention

### Backup Strategy
- Copy main database file + WAL file
- Use SQLite BACKUP API for consistency
- Regular VACUUM for optimization

## Security Considerations

### SQL Injection Prevention
- All queries use parameterized statements
- No dynamic SQL construction
- Input validation at application layer

### Data Integrity
- Foreign key constraints enforced
- Transaction-based operations
- Cascade deletes for cleanup

### Access Control
- File-system level permissions
- No built-in user authentication
- Application-level authorization