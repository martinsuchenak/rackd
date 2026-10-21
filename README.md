# Rackd

**Open-source IP Address Management (IPAM) and Device Inventory System**

Rackd is a lightweight, self-contained infrastructure management tool for tracking devices, networks, IP addresses, and datacenter resources. Built with Go and SQLite, it requires no external dependencies and runs as a single binary.

> **Status:** Rackd is pre-1.0.
>
> - The REST API under `/api/v1` stays backward compatible; see [Versioning and Deprecation](docs/api.md#versioning-and-deprecation).
> - Upgrades back up the database before migrating, and can be rolled back; see [Backup and Restore](docs/backup.md).
> - Only the latest release gets security fixes; see [Security](SECURITY.md).
> - The extension API (`pkg/extension`) may change in any release until 1.0.

## Features

### Core Capabilities

- **Device Inventory**: Track servers, switches, routers, and other network devices
- **IP Address Management (IPAM)**: Manage networks, subnets, VLANs, and IP pools
- **Network Discovery**: Automated network scanning and device discovery
- **Datacenter Management**: Organize devices by physical location
- **Device Relationships**: Track dependencies and connections between devices
- **Full-Text Search**: Fast FTS5-powered search across devices, networks, and datacenters

### DNS & Network Services

- **DNS Management**: Zone and record management synced to [Technitium DNS Server](https://technitium.com/dns/); PowerDNS and BIND support is planned
- **NAT Tracking**: Document NAT mappings
- **Circuit Tracking**: Document network circuits and providers

### Security & Access Control

- **RBAC**: Role-Based Access Control with fine-grained permissions
- **Audit Trail**: Complete change history for compliance and troubleshooting
- **Webhooks**: Event-driven integrations with external systems

### Extensibility

- **Custom Fields**: User-defined fields on devices
- **IP Reservations**: Reserve IPs for future use with owner tracking
- **Import/Export**: JSON/CSV bulk data operations
- **MCP Server**: Model Context Protocol integration for AI/automation tools

### Interfaces

- **Web UI**: Modern, responsive interface built with Alpine.js and TailwindCSS
- **CLI Tool**: Full-featured command-line interface for automation
- **REST API**: Versioned HTTP API under `/api/v1`, with a [deprecation policy](docs/api.md#versioning-and-deprecation)
- **Monitoring & Metrics**: Prometheus-compatible metrics and health checks

## Quick Start

```bash
# Download and unpack a release (see docs/installation.md for other platforms and verification)
VERSION=$(curl -fsSLI -o /dev/null -w '%{url_effective}' https://github.com/martinsuchenak/rackd/releases/latest | sed 's#.*/v##')
curl -LO "https://github.com/martinsuchenak/rackd/releases/download/v${VERSION}/rackd_${VERSION}_linux_amd64.tar.gz"
tar -xzf "rackd_${VERSION}_linux_amd64.tar.gz" rackd

# Start the server; the first admin is created from these on first start
INITIAL_ADMIN_USERNAME=admin INITIAL_ADMIN_PASSWORD='<choose a password>' ./rackd server

# Sign in to the web UI at http://localhost:8080
```

To use the CLI, create an API key in the web UI (**API Keys**) and export it:

```bash
export RACKD_SERVER_URL=http://localhost:8080
export RACKD_TOKEN='<api key>'
./rackd device list
```

Set `ENCRYPTION_KEY` (`openssl rand -hex 32`) to enable credentials, scan profiles, scheduled scans and DNS; see the [Configuration Reference](docs/configuration-reference.md).

## Documentation

Comprehensive documentation is available in the [docs/](docs/) directory:

### Getting Started

- [Installation](docs/installation.md) - Installation methods and requirements
- [Quick Start Guide](docs/quickstart.md) - Get up and running in 5 minutes
- [Configuration Reference](docs/configuration-reference.md) - Environment variables and their defaults
- [Authentication](docs/authentication.md) - Users, sessions, API keys and the first admin

### Core Features

- [CLI Reference](docs/cli.md) - Command-line interface documentation
- [API Reference](docs/api.md) - REST API endpoints and examples
- [MCP Server](docs/mcp.md) - Model Context Protocol integration
- [Web UI](docs/webui.md) - Web interface guide

### Modules

- [Device Management](docs/devices.md) - Device inventory and tracking
- [Network Management](docs/networks.md) - IPAM, subnets, and IP pools
- [Datacenter Management](docs/datacenters.md) - Physical location tracking
- [Discovery](docs/discovery.md) - Network scanning and auto-discovery
- [Relationships](docs/relationships.md) - Device dependencies and connections
- [DNS Management](docs/dns.md) - DNS zones, providers, and sync
- [Webhooks](docs/webhooks.md) - Event notifications and integrations
- [Custom Fields](docs/custom-fields.md) - User-defined fields
- [Circuits](docs/circuits.md) - Network circuit tracking
- [NAT](docs/nat.md) - NAT mapping tracking
- [Reservations](docs/reservations.md) - IP reservation tracking
- [Conflicts](docs/conflicts.md) - IP conflict detection and resolution

### Security & Compliance

- [RBAC](docs/rbac.md) - Role-Based Access Control
- [Security](docs/security.md) - Security best practices
- [Audit Trail](docs/audit.md) - Change history and compliance logging
- [Rate Limiting](docs/ratelimit.md) - API rate limiting configuration

### Operations

- [Deployment](docs/deployment.md) - Docker, Nomad, and production deployment
- [Monitoring](docs/monitoring.md) - Metrics, health checks, and observability
- [Performance Testing](docs/performance.md) - Load-testing a deployment with `tools/loadtest`
- [Extensions](docs/extensions.md) - Building features on `pkg/extension`
- [Backup & Restore](docs/backup.md) - Data backup strategies
- [Import/Export](docs/import-export.md) - Bulk data operations
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

### Development

- [Architecture](docs/architecture.md) - System design and structure
- [Development Guide](docs/development.md) - Building and contributing
- [Database Schema](docs/database.md) - SQLite schema reference
- [Testing](docs/testing.md) - Testing strategy and guidelines

## Technology Stack

- **Backend**: Go 1.27+ (CGO-free)
- **Database**: SQLite (embedded, no external dependencies)
- **Frontend**: TypeScript, Alpine.js, TailwindCSS v4
- **Build**: [Task](https://taskfile.dev), Bun (frontend)
- **CLI**: paularlott/cli
- **MCP**: paularlott/mcp

## Building from Source

```bash
# Clone the repository
git clone https://github.com/martinsuchenak/rackd.git
cd rackd

# Install dependencies
go mod download
cd webui && bun install && cd ..

# Build
task build:local

# Run
./build/rackd server
```

See [Development Guide](docs/development.md) for detailed build instructions.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/martinsuchenak/rackd/issues)
- **Discussions**: [GitHub Discussions](https://github.com/martinsuchenak/rackd/discussions)
- **Website**: [getrackd.dev](https://getrackd.dev)

## License

Apache License 2.0 - see [LICENSE](LICENSE) and [NOTICE](NOTICE). Contributions are accepted under the [Contributor License Agreement](CLA.md); see [CONTRIBUTING.md](CONTRIBUTING.md).
