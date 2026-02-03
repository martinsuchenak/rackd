# Rackd

**Open-source IP Address Management (IPAM) and Device Inventory System**

Rackd is a lightweight, self-contained infrastructure management tool for tracking devices, networks, IP addresses, and datacenter resources. Built with Go and SQLite, it requires no external dependencies and runs as a single binary.

## Features

- **Device Inventory**: Track servers, switches, routers, and other network devices
- **IP Address Management (IPAM)**: Manage networks, subnets, VLANs, and IP pools
- **Network Discovery**: Automated network scanning and device discovery
- **Datacenter Management**: Organize devices by physical location
- **Device Relationships**: Track dependencies and connections between devices
- **Web UI**: Modern, responsive interface built with Alpine.js and TailwindCSS
- **CLI Tool**: Full-featured command-line interface for automation
- **MCP Server**: Model Context Protocol integration for AI/automation tools
- **REST API**: Complete HTTP API for integrations

## Quick Start

```bash
# Download the latest release
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-linux-amd64

# Make it executable
chmod +x rackd-linux-amd64

# Start the server
./rackd-linux-amd64 server

# Access the web UI at http://localhost:8080
```

## Documentation

Comprehensive documentation is available in the [docs/](docs/) directory:

### Getting Started

- [Installation](docs/installation.md) - Installation methods and requirements
- [Quick Start Guide](docs/quickstart.md) - Get up and running in 5 minutes
- [Configuration](docs/configuration.md) - Environment variables and settings

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

### Development

- [Architecture](docs/architecture.md) - System design and structure
- [Development Guide](docs/development.md) - Building and contributing
- [Database Schema](docs/database.md) - SQLite schema reference
- [Testing](docs/testing.md) - Testing strategy and guidelines

### Operations

- [Deployment](docs/deployment.md) - Docker, Nomad, and production deployment
- [Backup & Restore](docs/backup.md) - Data backup strategies
- [Security](docs/security.md) - Security best practices
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## Technology Stack

- **Backend**: Go 1.25+ (CGO-free)
- **Database**: SQLite (embedded, no external dependencies)
- **Frontend**: TypeScript, Alpine.js, TailwindCSS v4
- **Build**: Make, Bun (frontend)
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
make build

# Run
./build/rackd server
```

See [Development Guide](docs/development.md) for detailed build instructions.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Support

- **Documentation**: [docs/](docs/)
- **Issues**: [GitHub Issues](https://github.com/martinsuchenak/rackd/issues)
- **Discussions**: [GitHub Discussions](https://github.com/martinsuchenak/rackd/discussions)
