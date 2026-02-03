# Rackd Test Data

Realistic test data for a medium-sized company with multi-region infrastructure.

## Overview

**3 Datacenters** (with predictable IDs):
- US-East-DC1 (Virginia, USA) - Primary (ID: `dc-us-east-1`)
- EU-West-DC1 (Dublin, Ireland) - GDPR compliant (ID: `dc-eu-west-1`)
- APAC-SG-DC1 (Singapore) - Asia-Pacific (ID: `dc-apac-sg-1`)

**9 Networks** (3 per datacenter, linked to datacenters):
- Production network (/16)
- DMZ network (/24)
- Management network (/24)

**24 Devices** (linked to datacenters, addresses linked to networks):
- **US East** (11 devices): 2 firewalls (HA), 1 core switch, 1 load balancer, 2 web servers, 2 app servers, 2 database servers (primary + replica), 1 cache server
- **EU West** (8 devices): 1 firewall, 1 core switch, 1 load balancer, 2 web servers, 1 app server, 1 database server
- **APAC Singapore** (5 devices): 1 firewall, 1 core switch, 1 load balancer, 1 web server, 1 app server, 1 database server

**6 Network Pools** (IP address ranges):
- 3 pools in US-East (Web, App, DB)
- 2 pools in EU-West (Web, App)
- 1 pool in APAC (Web)

**20 Device Relationships**:
- Load balancers depend on web servers
- Web servers depend on app servers
- App servers depend on databases and cache
- Database replication (primary → replica)
- Firewall HA pairs

## Network Topology

Each datacenter follows a standard 3-tier architecture:

```
Internet
    ↓
[Firewall] ← DMZ Network (10.X.100.0/24)
    ↓
[Core Switch] ← Management Network (10.X.200.0/24)
    ↓
[Load Balancer]
    ↓
[Web Tier] ← Production Network (10.X.0.0/16)
    ↓
[App Tier]
    ↓
[Database Tier]
```

## IP Addressing Scheme

- **US East**: 10.10.0.0/16
  - Production: 10.10.0.0/16
  - DMZ: 10.10.100.0/24
  - Management: 10.10.200.0/24

- **EU West**: 10.20.0.0/16
  - Production: 10.20.0.0/16
  - DMZ: 10.20.100.0/24
  - Management: 10.20.200.0/24

- **APAC Singapore**: 10.30.0.0/16
  - Production: 10.30.0.0/16
  - DMZ: 10.30.100.0/24
  - Management: 10.30.200.0/24

## Loading Test Data

### Option 1: CLI (using import commands)

```bash
./testdata/load-testdata.sh cli
```

This uses the `rackd import` commands with bulk endpoints for better performance.

### Option 2: API (direct HTTP calls)

```bash
# Start the server first
./build/rackd server

# In another terminal
./testdata/load-testdata.sh api
```

This uses curl to POST directly to the bulk API endpoints.

### Manual Import

```bash
# Import datacenters
./build/rackd import datacenters --file testdata/datacenters.json

# Import networks (uses bulk endpoint)
./build/rackd import networks --file testdata/networks.json

# Import devices (uses bulk endpoint)
./build/rackd import devices --file testdata/devices.json
```

## Testing Queries

After loading the data, try these queries:

```bash
# List all devices
./build/rackd device list

# Filter by tags
./build/rackd device list --tags firewall
./build/rackd device list --tags database
./build/rackd device list --tags us-east

# List networks
./build/rackd network list

# Search
curl http://localhost:8080/api/search?q=firewall | jq

# Get devices by datacenter
curl http://localhost:8080/api/datacenters | jq -r '.[0].id' | \
  xargs -I {} curl http://localhost:8080/api/datacenters/{}/devices | jq
```

## Files

- `datacenters.json` - 3 datacenters with predictable IDs
- `networks.json` - 9 networks linked to datacenters
- `devices.json` - 24 devices linked to datacenters, addresses linked to networks
- `pools.json` - 6 network IP pools
- `relationships.json` - 20 device relationships
- `load-testdata.sh` - Script to load all data (CLI or API mode)
- `README.md` - This file

## Tags Used

- **By Region**: `us-east`, `eu-west`, `apac`
- **By Function**: `firewall`, `switch`, `load-balancer`, `web`, `app`, `database`, `cache`
- **By Technology**: `nginx`, `nodejs`, `postgresql`, `redis`
- **By Role**: `production`, `security`, `core`, `primary`, `replica`, `ha`
