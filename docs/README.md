# Rackd Documentation

Complete documentation for Rackd - Open-source IP Address Management (IPAM) and Device Inventory System.

## Getting Started

Start here if you're new to Rackd:

- **[Installation](installation.md)** - Install Rackd on your system
- **[Quick Start Guide](quickstart.md)** - Get up and running in 5 minutes
- **[Configuration](configuration.md)** - Configure Rackd for your environment

## Core Concepts

Understand the fundamentals:

- **[Architecture](architecture.md)** - System design and technology stack
- **[Database Schema](database.md)** - Database structure and relationships

## User Guides

Learn how to use Rackd:

### Interfaces

- **[Web UI](webui.md)** - Web interface guide
- **[CLI Reference](cli.md)** - Command-line interface
- **[API Reference](api.md)** - REST API documentation
- **[MCP Server](mcp.md)** - Model Context Protocol for AI tools

### Features

- **[Device Management](devices.md)** - Track and manage devices
- **[Network Management](networks.md)** - IPAM, subnets, and IP pools
- **[Datacenter Management](datacenters.md)** - Physical location tracking
- **[Discovery](discovery.md)** - Network scanning and auto-discovery
- **[Relationships](relationships.md)** - Device dependencies and connections

## Operations

Deploy and maintain Rackd in production:

- **[Deployment](deployment.md)** - Docker, Nomad, systemd, and production setup
- **[Backup & Restore](backup.md)** - Data backup and recovery strategies
- **[Security](security.md)** - Security best practices and hardening
- **[Troubleshooting](troubleshooting.md)** - Common issues and solutions

## Development

Contribute to Rackd:

- **[Development Guide](development.md)** - Building and contributing
- **[Testing](testing.md)** - Testing strategy and guidelines

## Quick Links

### Common Tasks

| Task | Documentation |
|------|---------------|
| Install Rackd | [Installation](installation.md) |
| First-time setup | [Quick Start](quickstart.md) |
| Add a device | [Devices](devices.md#adding-devices) |
| Create a network | [Networks](networks.md#creating-networks) |
| Run a discovery scan | [Discovery](discovery.md#running-scans) |
| Use the CLI | [CLI Reference](cli.md) |
| Call the API | [API Reference](api.md) |
| Deploy with Docker | [Deployment](deployment.md#docker) |
| Backup database | [Backup](backup.md#manual-backup) |
| Troubleshoot issues | [Troubleshooting](troubleshooting.md) |

### By Role

**System Administrator**
1. [Installation](installation.md)
2. [Configuration](configuration.md)
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
├── quickstart.md             # 5-minute getting started
├── configuration.md          # Configuration reference
├── architecture.md           # System architecture
├── database.md               # Database schema
├── cli.md                    # CLI reference
├── api.md                    # REST API reference
├── mcp.md                    # MCP server reference
├── webui.md                  # Web UI guide
├── devices.md                # Device management
├── networks.md               # Network management
├── datacenters.md            # Datacenter management
├── discovery.md              # Network discovery
├── relationships.md          # Device relationships
├── deployment.md             # Deployment guide
├── backup.md                 # Backup and restore
├── security.md               # Security guide
├── testing.md                # Testing guide
├── development.md            # Development guide
└── troubleshooting.md        # Troubleshooting guide
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
curl -X POST http://localhost:8080/api/devices \
  -H "Authorization: Bearer token" \
  -H "Content-Type: application/json" \
  -d '{"name": "web-01"}'
```

Configuration files:
```yaml
api_url: http://localhost:8080
api_token: your-secret-token
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

Documentation is licensed under MIT License, same as the Rackd project.
