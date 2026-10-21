# Rackd Documentation

Complete documentation for Rackd - Open-source IP Address Management (IPAM) and Device Inventory System.

## Getting Started

Start here if you're new to Rackd:

- **[Installation](installation.md)** - Install Rackd on your system
- **[Quick Start Guide](quickstart.md)** - Get up and running in 5 minutes
- **[Configuration Reference](configuration-reference.md)** - Every setting and its default

## Core Concepts

Understand the fundamentals:

- **[Architecture](architecture.md)** - System design and technology stack
- **[Database Schema](database.md)** - Database structure and relationships

## User Guides

Learn how to use Rackd:

### Interfaces

- **[Web UI](webui.md)** - Web interface guide
- **[CLI Reference](cli.md)** - Command-line interface
- **[CLI Command Reference](cli-reference.md)** - Every command and flag
- **[API Reference](api.md)** - REST API documentation
- **[MCP Server](mcp.md)** - Model Context Protocol for AI tools

### Features

- **[Device Management](devices.md)** - Track and manage devices
- **[Network Management](networks.md)** - IPAM, subnets, and IP pools
- **[Datacenter Management](datacenters.md)** - Physical location tracking
- **[Discovery](discovery.md)** - Network scanning and auto-discovery
- **[Relationships](relationships.md)** - Device dependencies and connections
- **[DNS Integration](dns.md)** - DNS providers, zones and record sync
- **[Import/Export](import-export.md)** - Bulk data in JSON and CSV

### Advanced Features

- **[Dashboard & Trends](dashboard.md)** - Reporting, utilization trends, and activity feeds
- **[Device Lifecycle](lifecycle.md)** - Status tracking and decommission planning
- **[IP Conflict Detection](conflicts.md)** - Detect and resolve IP conflicts
- **[IP Reservations](reservations.md)** - Reserve IPs for planning
- **[Webhooks](webhooks.md)** - Event notifications for automation
- **[Circuit Management](circuits.md)** - Track network circuits
- **[NAT Tracking](nat.md)** - Document NAT mappings
- **[Custom Fields](custom-fields.md)** - User-defined device metadata

## Access Control

- **[Authentication](authentication.md)** - Sessions, API keys and OAuth for MCP
- **[User Authentication](user-authentication.md)** - User accounts, sign-in and sessions
- **[RBAC](rbac.md)** - Roles and permissions
- **[Audit Trail](audit.md)** - Change and sign-in history

## Operations

Deploy and maintain Rackd in production:

- **[Deployment](deployment.md)** - Docker, Nomad, systemd, and production setup
- **[Backup & Restore](backup.md)** - Data backup and recovery strategies
- **[Security](security.md)** - Security best practices and hardening
- **[Rate Limiting](ratelimit.md)** - Request limits and sign-in lockout
- **[Monitoring](monitoring.md)** - Metrics, health checks and observability
- **[Performance Testing](performance.md)** - Load-testing a deployment
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

## Development

Contribute to Rackd:

- **[Development Guide](development.md)** - Building and contributing
- **[Testing](testing.md)** - Testing strategy and guidelines
- **[Extensions](extensions.md)** - Building features on `pkg/extension`
- **[Full-Text Search](fts.md)** - How search is implemented

## Quick Links

### Common Tasks

| Task | Documentation |
|------|---------------|
| Install Rackd | [Installation](installation.md) |
| First-time setup | [Quick Start](quickstart.md) |
| Add a device | [Devices](devices.md#create-device) |
| Create a network | [Networks](networks.md#creating-networks) |
| Run a discovery scan | [Discovery](discovery.md#manual-scans) |
| View dashboard | [Dashboard](dashboard.md) |
| Track device lifecycle | [Lifecycle](lifecycle.md) |
| Detect IP conflicts | [Conflicts](conflicts.md) |
| Reserve IP addresses | [Reservations](reservations.md) |
| Configure webhooks | [Webhooks](webhooks.md) |
| Track NAT mappings | [NAT](nat.md) |
| Manage circuits | [Circuits](circuits.md) |
| Add custom fields | [Custom Fields](custom-fields.md) |
| Use the CLI | [CLI Reference](cli.md) |
| Call the API | [API Reference](api.md) |
| Deploy with Docker | [Deployment](deployment.md#docker-deployment) |
| Backup database | [Backup](backup.md#backing-up) |
| Troubleshoot issues | [Troubleshooting](troubleshooting.md) |

### By Role

**System Administrator**
1. [Installation](installation.md)
2. [Configuration Reference](configuration-reference.md)
3. [Deployment](deployment.md)
4. [Security](security.md)
5. [Backup & Restore](backup.md)

**Network Engineer**
1. [Quick Start](quickstart.md)
2. [Networks](networks.md)
3. [Devices](devices.md)
4. [Discovery](discovery.md)
5. [CLI Reference](cli.md)

**Developer**
1. [Architecture](architecture.md)
2. [API Reference](api.md)
3. [Database Schema](database.md)
4. [Development Guide](development.md)
5. [Testing](testing.md)

**DevOps/Automation**
1. [API Reference](api.md)
2. [MCP Server](mcp.md)
3. [CLI Reference](cli.md)
4. [Deployment](deployment.md)

## Documentation Structure

```
docs/
├── README.md                  # This file
├── installation.md            # Installation guide
├── quickstart.md              # 5-minute getting started
├── configuration.md           # Pointer to the configuration reference
├── configuration-reference.md # Every setting and its default
├── architecture.md            # System architecture
├── database.md                # Database schema
├── cli.md                     # CLI guide
├── cli-reference.md           # Every CLI command and flag (generated)
├── api.md                     # REST API reference
├── mcp.md                     # MCP server reference
├── webui.md                   # Web UI guide
├── devices.md                 # Device management
├── networks.md                # Network management
├── datacenters.md             # Datacenter management
├── discovery.md               # Network discovery
├── relationships.md           # Device relationships
├── dns.md                     # DNS integration
├── dashboard.md               # Dashboard and trends
├── lifecycle.md               # Device lifecycle tracking
├── conflicts.md               # IP conflict detection
├── reservations.md            # IP reservations
├── webhooks.md                # Webhook system
├── circuits.md                # Circuit management
├── nat.md                     # NAT tracking
├── custom-fields.md           # Custom fields
├── import-export.md           # Import and export
├── authentication.md          # Sessions, API keys, OAuth
├── user-authentication.md     # User accounts and sign-in
├── rbac.md                    # Roles and permissions
├── audit.md                   # Audit trail
├── deployment.md              # Deployment guide
├── backup.md                  # Backup and restore
├── security.md                # Security guide
├── ratelimit.md               # Rate limiting
├── monitoring.md              # Metrics and health checks
├── performance.md             # Load testing
├── troubleshooting.md         # Troubleshooting guide
├── development.md             # Development guide
├── testing.md                 # Testing guide
├── extensions.md              # Extension API
└── fts.md                     # Full-text search internals
```

## Additional Resources

### Project Links

- **GitHub Repository**: https://github.com/martinsuchenak/rackd
- **Issue Tracker**: https://github.com/martinsuchenak/rackd/issues
- **Discussions**: https://github.com/martinsuchenak/rackd/discussions
- **Releases**: https://github.com/martinsuchenak/rackd/releases

### Technology Documentation

- **Go**: https://go.dev/doc/
- **SQLite**: https://www.sqlite.org/docs.html
- **Alpine.js**: https://alpinejs.dev/
- **TailwindCSS**: https://tailwindcss.com/docs
- **Model Context Protocol**: https://modelcontextprotocol.io/

## Contributing to Documentation

Documentation improvements are welcome! To contribute:

1. Fork the repository
2. Edit documentation in `docs/` directory
3. Submit a pull request

See [Development Guide](development.md) for more details.

## Documentation Conventions

### Code Examples

Shell commands:
```bash
rackd server --listen-addr :8080
```

API requests:
```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Authorization: Bearer token" \
  -H "Content-Type: application/json" \
  -d '{"name": "web-01"}'
```

CLI configuration file (`~/.config/rackd/config.json`):
```json
{
  "server_url": "http://localhost:8080",
  "token": "your-secret-token"
}
```

### Placeholders

- `<id>` - Resource identifier
- `<name>` - Resource name
- `your-secret-token` - Authentication token
- `10.0.1.0/24` - Example network
- `dc1` - Example datacenter ID

### Symbols

- ✅ - Recommended or supported
- ⚠️ - Warning or caution
- 🔒 - Security-related
- 📝 - Note or important information

## Getting Help

If you can't find what you're looking for:

1. Check the [Troubleshooting Guide](troubleshooting.md)
2. Search [GitHub Issues](https://github.com/martinsuchenak/rackd/issues)
3. Ask in [GitHub Discussions](https://github.com/martinsuchenak/rackd/discussions)
4. Open a new issue if you've found a bug

## License

Documentation is licensed under the Apache License 2.0, the same as the Rackd project.
