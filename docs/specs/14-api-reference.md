# API Reference

This document provides a complete reference of all API endpoints.

## Error Response Format

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE"
}
```

## Common Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `DEVICE_NOT_FOUND` | 404 | Device with given ID does not exist |
| `DATACENTER_NOT_FOUND` | 404 | Datacenter with given ID does not exist |
| `NETWORK_NOT_FOUND` | 404 | Network with given ID does not exist |
| `POOL_NOT_FOUND` | 404 | Network pool with given ID does not exist |
| `INVALID_INPUT` | 400 | Request validation failed |
| `UNAUTHORIZED` | 401 | Authentication required or failed |
| `FORBIDDEN` | 403 | User lacks permission |
| `INTERNAL_ERROR` | 500 | Unexpected server error |
| `IP_NOT_AVAILABLE` | 409 | No IP addresses available in pool |
| `IP_CONFLICT` | 409 | IP address already in use |

## Route Reference

### Datacenters

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/datacenters` | List all datacenters |
| POST | `/api/datacenters` | Create datacenter |
| GET | `/api/datacenters/{id}` | Get datacenter by ID |
| PUT | `/api/datacenters/{id}` | Update datacenter |
| DELETE | `/api/datacenters/{id}` | Delete datacenter |
| GET | `/api/datacenters/{id}/devices` | Get devices in datacenter |

### Networks

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/networks` | List networks |
| POST | `/api/networks` | Create network |
| GET | `/api/networks/{id}` | Get network by ID |
| PUT | `/api/networks/{id}` | Update network |
| DELETE | `/api/networks/{id}` | Delete network |
| GET | `/api/networks/{id}/devices` | Get devices on network |
| GET | `/api/networks/{id}/utilization` | Get IP utilization stats |
| GET | `/api/networks/{id}/pools` | List network pools |
| POST | `/api/networks/{id}/pools` | Create pool in network |

### Network Pools

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/pools/{id}` | Get pool by ID |
| PUT | `/api/pools/{id}` | Update pool |
| DELETE | `/api/pools/{id}` | Delete pool |
| GET | `/api/pools/{id}/next-ip` | Get next available IP |
| GET | `/api/pools/{id}/heatmap` | Get IP utilization heatmap |

### Devices

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/devices` | List devices |
| POST | `/api/devices` | Create device |
| GET | `/api/devices/{id}` | Get device by ID |
| PUT | `/api/devices/{id}` | Update device |
| DELETE | `/api/devices/{id}` | Delete device |
| GET | `/api/devices/search?q={query}` | Search devices |

### Relationships

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/devices/{id}/relationships` | Add relationship |
| GET | `/api/devices/{id}/relationships` | Get relationships |
| GET | `/api/devices/{id}/related?type={type}` | Get related devices |
| DELETE | `/api/devices/{id}/relationships/{child_id}/{type}` | Remove relationship |

### Discovery

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/discovery/networks/{id}/scan` | Start network scan |
| GET | `/api/discovery/networks/{id}/rules` | Get discovery rules |
| POST | `/api/discovery/networks/{id}/rules` | Save discovery rule |
| GET | `/api/discovery/scans` | List all scans |
| GET | `/api/discovery/scans/{id}` | Get scan status |
| GET | `/api/discovery/scans/{id}/results` | Get scan results |
| GET | `/api/discovery/devices` | List discovered devices |
| POST | `/api/discovery/devices/{id}/promote` | Promote to inventory |
| DELETE | `/api/discovery/devices/{id}` | Delete discovered device |
| POST | `/api/discovery/arp` | Import ARP table |

### Configuration

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/config` | Get UI configuration |

### MCP

| Method | Path | Description |
|--------|------|-------------|
| POST | `/mcp` | Model Context Protocol endpoint |

### Static Files

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Serve index.html (Web UI) |
| GET | `/app.js` | Serve JavaScript bundle |
| GET | `/output.css` | Serve CSS styles |

## Authentication

When `API_AUTH_TOKEN` is configured, all API endpoints require a Bearer token:

```
Authorization: Bearer your-secret-token
```

The MCP endpoint uses `MCP_AUTH_TOKEN` separately.
