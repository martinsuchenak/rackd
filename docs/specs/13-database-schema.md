# Database Schema Reference

This document covers the SQLite database schema and entity relationships.

## Entity Relationship Diagram

```
                        ┌──────────────────┐
                        │   datacenters    │
                        ├──────────────────┤
                        │ id (PK)          │◄───────────────────┐
                        │ name             │                    │
                        │ location         │                    │
                        │ description      │                    │
                        └──────────────────┘                    │
                                                                │
                        ┌──────────────────┐                    │
                        │     networks     │                    │
                        ├──────────────────┤                    │
                        │ id (PK)          │◄──────┐            │
                        │ name             │       │            │
                        │ subnet           │       │            │
                        │ vlan_id          │       │            │
                        │ datacenter_id    │───────┼────────────┘
                        │ description      │       │
                        └──────────────────┘       │
                           │                       │
                           ▼                       │
                ┌──────────────────┐               │
                │  network_pools   │               │
                ├──────────────────┤               │
                │ id (PK)          │               │
                │ network_id (FK)  │               │
                │ name             │               │
                │ start_ip         │               │
                │ end_ip           │               │
                │ description      │               │
                └──────────────────┘               │
                                                   │
                        ┌──────────────────┐       │
                        │     devices      │       │
                        ├──────────────────┤       │
                        │ id (PK)          │       │
                        │ name             │       │
                        │ description      │       │
                        │ make_model       │       │
                        │ os               │       │
                        │ datacenter_id    │───────┘
                        │ username         │
                        │ location         │
                        └──────────────────┘
                           │
        ┌──────────────────┼──────────────────┬──────────────────┐
        │                  │                  │                  │
        ▼                  ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│   addresses   │  │     tags      │  │   domains     │  │  device_      │
├───────────────┤  ├───────────────┤  ├───────────────┤  │  relationships│
│ id (PK)       │  │ device_id     │  │ device_id     │  ├───────────────┤
│ device_id (FK)│  │ tag           │  │ domain        │  │ parent_id     │
│ ip            │  └───────────────┘  └───────────────┘  │ child_id      │
│ port          │                                        │ type          │
│ type          │                                        └───────────────┘
│ label         │
│ network_id    │
│ switch_port   │
│ pool_id       │
└───────────────┘

                        ┌──────────────────┐
                        │ discovered_      │
                        │ devices          │
                        ├──────────────────┤
                        │ id (PK)          │
                        │ ip               │
                        │ mac_address      │
                        │ hostname         │
                        │ network_id       │
                        │ status           │
                        │ confidence       │
                        │ promoted_to_id   │
                        └──────────────────┘

                        ┌──────────────────┐
                        │ discovery_scans  │
                        ├──────────────────┤
                        │ id (PK)          │
                        │ network_id       │
                        │ status           │
                        │ scan_type        │
                        │ progress         │
                        └──────────────────┘
```

## Tables

### datacenters

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| name | TEXT | NOT NULL |
| location | TEXT | |
| description | TEXT | |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### networks

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| name | TEXT | NOT NULL |
| subnet | TEXT | NOT NULL |
| vlan_id | INTEGER | |
| datacenter_id | TEXT | REFERENCES datacenters(id) |
| description | TEXT | |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### network_pools

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| network_id | TEXT | REFERENCES networks(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| start_ip | TEXT | NOT NULL |
| end_ip | TEXT | NOT NULL |
| description | TEXT | |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### devices

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| name | TEXT | NOT NULL |
| description | TEXT | |
| make_model | TEXT | |
| os | TEXT | |
| datacenter_id | TEXT | REFERENCES datacenters(id) |
| username | TEXT | |
| location | TEXT | |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### addresses

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| device_id | TEXT | REFERENCES devices(id) ON DELETE CASCADE |
| ip | TEXT | NOT NULL |
| port | INTEGER | |
| type | TEXT | DEFAULT 'ipv4' |
| label | TEXT | |
| network_id | TEXT | REFERENCES networks(id) |
| switch_port | TEXT | |
| pool_id | TEXT | REFERENCES network_pools(id) |

### tags

| Column | Type | Constraints |
|--------|------|-------------|
| device_id | TEXT | REFERENCES devices(id) ON DELETE CASCADE |
| tag | TEXT | NOT NULL |
| | | PRIMARY KEY (device_id, tag) |

### domains

| Column | Type | Constraints |
|--------|------|-------------|
| device_id | TEXT | REFERENCES devices(id) ON DELETE CASCADE |
| domain | TEXT | NOT NULL |
| | | PRIMARY KEY (device_id, domain) |

### device_relationships

| Column | Type | Constraints |
|--------|------|-------------|
| parent_id | TEXT | REFERENCES devices(id) ON DELETE CASCADE |
| child_id | TEXT | REFERENCES devices(id) ON DELETE CASCADE |
| type | TEXT | NOT NULL |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| | | PRIMARY KEY (parent_id, child_id, type) |

### discovered_devices

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| ip | TEXT | NOT NULL |
| mac_address | TEXT | |
| hostname | TEXT | |
| network_id | TEXT | REFERENCES networks(id) |
| status | TEXT | DEFAULT 'unknown' |
| confidence | INTEGER | DEFAULT 0 |
| os_guess | TEXT | |
| vendor | TEXT | |
| open_ports | TEXT | JSON array |
| services | TEXT | JSON array |
| first_seen | TIMESTAMP | |
| last_seen | TIMESTAMP | |
| promoted_to_device_id | TEXT | REFERENCES devices(id) |
| promoted_at | TIMESTAMP | |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### discovery_scans

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| network_id | TEXT | REFERENCES networks(id) |
| status | TEXT | DEFAULT 'pending' |
| scan_type | TEXT | DEFAULT 'full' |
| total_hosts | INTEGER | DEFAULT 0 |
| scanned_hosts | INTEGER | DEFAULT 0 |
| found_hosts | INTEGER | DEFAULT 0 |
| progress_percent | REAL | DEFAULT 0 |
| error_message | TEXT | |
| started_at | TIMESTAMP | |
| completed_at | TIMESTAMP | |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |

### discovery_rules

| Column | Type | Constraints |
|--------|------|-------------|
| id | TEXT | PRIMARY KEY |
| network_id | TEXT | REFERENCES networks(id) UNIQUE |
| enabled | INTEGER | DEFAULT 1 |
| scan_type | TEXT | DEFAULT 'full' |
| interval_hours | INTEGER | DEFAULT 24 |
| exclude_ips | TEXT | Comma-separated |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
| updated_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP |
