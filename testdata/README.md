# Rackd Test Data

Realistic test data for a medium-sized company with multi-region infrastructure. Covers all major features of the application.

## Overview

**3 Datacenters:**
- US-East-DC1 (Virginia, USA) — Primary (ID: `dc-us-east-1`)
- EU-West-DC1 (Dublin, Ireland) — GDPR compliant (ID: `dc-eu-west-1`)
- APAC-SG-DC1 (Singapore) — Asia-Pacific (ID: `dc-apac-sg-1`)

**9 Networks** (3 per datacenter):
- Production network (/16)
- DMZ network (/24)
- Management network (/24)

**33 Devices** (all lifecycle statuses represented):
- US East (20): 2 firewalls (HA), 1 core switch, 1 load balancer, 2 web servers, 2 app servers, 2 database servers, 1 cache, 1 monitoring, 1 bastion, 1 storage, 1 decommissioned, 1 planned, 1 maintenance
- EU West (10): 1 firewall, 1 core switch, 1 load balancer, 2 web servers, 1 app server, 2 database servers, 1 cache, 1 replica
- APAC Singapore (5): 1 firewall, 1 core switch, 1 load balancer, 1 web server, 1 app server, 2 database servers

**6 Network Pools** (IP address ranges):
- 3 pools in US-East (Web, App, DB)
- 2 pools in EU-West (Web, App)
- 1 pool in APAC (Web)

**20+ Device Relationships:**
- Load balancers depend on web servers
- Web servers depend on app servers
- App servers depend on databases and cache
- Database replication (primary → replica)
- Firewall HA pairs

**8 Circuits** (WAN links, internet uplinks, cross-connects):
- US-East ↔ EU-West primary (AT&T MPLS 10G)
- US-East ↔ EU-West backup (Lumen 1G)
- US-East ↔ APAC primary (NTT MPLS 5G)
- EU-West ↔ APAC primary (Telia MPLS 2G, maintenance)
- 3 internet uplinks (one per datacenter)
- 1 internal cross-connect

**10 NAT Mappings:**
- HTTPS/HTTP for each datacenter's load balancer
- SSH jump host access
- UDP syslog forwarding
- VPN passthrough
- Disabled DB backup port (tests disabled state)

**8 IP Reservations** (across multiple pools):
- Planned server expansions
- Staging environments
- Canary deployment targets

**8 Custom Field Definitions:**
- Environment (select: production/staging/development/dr)
- Cost Center (text)
- Owner (text)
- Warranty Expiry (text)
- Rack Unit (number)
- Monitoring Enabled (boolean)
- Backup Policy (select)
- Criticality (select)

**8 Credentials:**
- SNMP v2c (public, private)
- SNMP v3 (US East, EU West)
- SSH keys (admin, dbadmin, EU admin, APAC admin)

**6 Webhooks:**
- Slack alerts (device events, conflicts, pool utilization)
- PagerDuty incidents (critical events)
- CMDB sync (all device/network changes)
- SIEM audit feed (all change events)
- Discovery notifications (disabled — too noisy)
- Capacity planning alerts (pool utilization)

**6 Scan Profiles:**
- Quick Ping Sweep (ICMP only, 50 workers)
- Standard Network Scan (common ports + SNMP)
- Deep Security Scan (extended ports, SNMP + SSH)
- Network Device Scan (SNMP, NETCONF, SSH)
- Web Server Scan (HTTP/HTTPS ports)
- Database Server Scan (all DB ports)

**5 Scheduled Scans:**
- US Prod nightly (standard, 2 AM daily)
- US DMZ weekly deep scan (Sundays 3 AM)
- EU Prod nightly (standard, 2 AM daily)
- APAC hourly ping sweep
- US Mgmt weekly network device scan

**7 Discovery Rules** (per-network auto-scan configuration)

## Loading Test Data

```bash
# API mode (default) — server must be running
./testdata/load-testdata.sh api

# CLI mode — uses rackd import commands
./testdata/load-testdata.sh cli
```

Set `RACKD_API_URL` to override the default `http://localhost:8080`.

### Manual Import

```bash
# Core infrastructure
./build/rackd import datacenters --file testdata/datacenters.json
./build/rackd import networks    --file testdata/networks.json
./build/rackd import devices     --file testdata/devices.json

# Additional features (API only)
curl -s -X POST http://localhost:8080/api/circuits \
  -H "Content-Type: application/json" -d @testdata/circuits.json
```

## Files

| File | Contents |
|------|----------|
| `datacenters.json` | 3 datacenters |
| `networks.json` | 9 networks |
| `pools.json` | 6 IP pools |
| `devices.json` | 24 core devices |
| `devices_extra.json` | 9 extra devices (varied statuses) |
| `relationships.json` | 20 device relationships |
| `circuits.json` | 8 WAN circuits |
| `nat.json` | 10 NAT mappings |
| `reservations.json` | 8 IP reservations |
| `custom_fields.json` | 8 custom field definitions |
| `credentials.json` | 8 SNMP/SSH credentials |
| `webhooks.json` | 6 webhook endpoints |
| `scan_profiles.json` | 6 scan profiles |
| `scheduled_scans.json` | 5 scheduled scans |
| `discovery_rules.json` | 7 discovery rules |
| `load-testdata.sh` | Load script |

## Network Topology

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

- US East: `10.10.0.0/16` (Prod), `10.10.100.0/24` (DMZ), `10.10.200.0/24` (Mgmt)
- EU West: `10.20.0.0/16` (Prod), `10.20.100.0/24` (DMZ), `10.20.200.0/24` (Mgmt)
- APAC SG: `10.30.0.0/16` (Prod), `10.30.100.0/24` (DMZ), `10.30.200.0/24` (Mgmt)
