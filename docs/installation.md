# Installation

This guide covers different methods to install and run Rackd.

## System Requirements

### Minimum Requirements
- **OS**: Linux, macOS, or Windows
- **RAM**: 256 MB
- **Disk**: 100 MB (plus data storage)
- **CPU**: Any modern CPU (x86_64 or ARM64)

### Recommended Requirements
- **RAM**: 512 MB or more
- **Disk**: 1 GB or more for data
- **CPU**: 2+ cores for discovery scanning

## Installation Methods

### 1. Download Pre-built Binary (Recommended)

Download the latest release for your platform:

```bash
# Linux (x86_64)
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-linux-amd64
chmod +x rackd-linux-amd64
sudo mv rackd-linux-amd64 /usr/local/bin/rackd

# Linux (ARM64)
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-linux-arm64
chmod +x rackd-linux-arm64
sudo mv rackd-linux-arm64 /usr/local/bin/rackd

# macOS (Intel)
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-darwin-amd64
chmod +x rackd-darwin-amd64
sudo mv rackd-darwin-amd64 /usr/local/bin/rackd

# macOS (Apple Silicon)
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-darwin-arm64
chmod +x rackd-darwin-arm64
sudo mv rackd-darwin-arm64 /usr/local/bin/rackd

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-windows-amd64.exe" -OutFile "rackd.exe"
```

Verify installation:

```bash
rackd version
```

### 2. Build from Source

#### Prerequisites
- Go 1.25 or later
- Bun (for building the web UI)
- Make

#### Build Steps

```bash
# Clone the repository
git clone https://github.com/martinsuchenak/rackd.git
cd rackd

# Install Go dependencies
go mod download

# Install frontend dependencies
cd webui
bun install
cd ..

# Build everything
make build

# The binary will be in ./build/rackd
./build/rackd version
```

See [Development Guide](development.md) for more details.

### 3. Docker

#### Using Docker Hub

```bash
# Pull the image
docker pull martinsuchenak/rackd:latest

# Run the container
docker run -d \
  --name rackd \
  -p 8080:8080 \
  -v rackd-data:/data \
  martinsuchenak/rackd:latest
```

#### Using Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  rackd:
    image: martinsuchenak/rackd:latest
    container_name: rackd
    ports:
      - "8080:8080"
    volumes:
      - rackd-data:/data
    environment:
      - RACKD_LOG_LEVEL=info
      - RACKD_API_AUTH_TOKEN=your-secret-token
    restart: unless-stopped

volumes:
  rackd-data:
```

Start the service:

```bash
docker-compose up -d
```

#### Build Your Own Image

```bash
# Build the image
docker build -t rackd:local .

# Run it
docker run -d -p 8080:8080 -v rackd-data:/data rackd:local
```

### 4. Nomad Deployment

For HashiCorp Nomad deployments, see [Deployment Guide](deployment.md#nomad).

## Post-Installation

### 1. Verify Installation

```bash
# Check version
rackd version

# Test server startup
rackd server --help
```

### 2. Create Data Directory

```bash
# Create data directory
mkdir -p /var/lib/rackd

# Set permissions (if running as non-root)
chown -R rackd:rackd /var/lib/rackd
```

### 3. Configure Environment

Create `/etc/rackd/config.env`:

```bash
# Server configuration
RACKD_LISTEN_ADDR=:8080
RACKD_DATA_DIR=/var/lib/rackd

# Authentication (recommended for production)
RACKD_API_AUTH_TOKEN=your-secret-api-token
RACKD_MCP_AUTH_TOKEN=your-secret-mcp-token

# Logging
RACKD_LOG_LEVEL=info
RACKD_LOG_FORMAT=json

# Discovery
RACKD_DISCOVERY_ENABLED=true
RACKD_DISCOVERY_INTERVAL=24h

# Credentials encryption
RACKD_ENCRYPTION_KEY=your-32-byte-encryption-key-here
```

See [Configuration Guide](configuration.md) for all options.

### 4. Start the Server

```bash
# Start with default settings
rackd server

# Start with custom data directory
rackd server --data-dir /var/lib/rackd

# Start with environment file
export $(cat /etc/rackd/config.env | xargs)
rackd server
```

### 5. Access the Web UI

Open your browser and navigate to:

```
http://localhost:8080
```

## System Service Setup

### systemd (Linux)

Create `/etc/systemd/system/rackd.service`:

```ini
[Unit]
Description=Rackd IPAM and Device Inventory
After=network.target

[Service]
Type=simple
User=rackd
Group=rackd
EnvironmentFile=/etc/rackd/config.env
ExecStart=/usr/local/bin/rackd server
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/rackd

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
# Create user
sudo useradd -r -s /bin/false rackd

# Set permissions
sudo chown -R rackd:rackd /var/lib/rackd

# Enable service
sudo systemctl daemon-reload
sudo systemctl enable rackd
sudo systemctl start rackd

# Check status
sudo systemctl status rackd

# View logs
sudo journalctl -u rackd -f
```

### launchd (macOS)

Create `~/Library/LaunchAgents/com.rackd.server.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.rackd.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/rackd</string>
        <string>server</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>RACKD_DATA_DIR</key>
        <string>/Users/YOUR_USERNAME/.rackd</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/rackd.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/rackd.error.log</string>
</dict>
</plist>
```

Load the service:

```bash
launchctl load ~/Library/LaunchAgents/com.rackd.server.plist
launchctl start com.rackd.server
```

### Windows Service

Use [NSSM](https://nssm.cc/) to create a Windows service:

```powershell
# Download NSSM
# Install as service
nssm install rackd "C:\Program Files\rackd\rackd.exe" server

# Set data directory
nssm set rackd AppEnvironmentExtra RACKD_DATA_DIR=C:\ProgramData\rackd

# Start service
nssm start rackd
```

## Upgrading

### Binary Upgrade

```bash
# Stop the service
sudo systemctl stop rackd

# Backup database
cp /var/lib/rackd/rackd.db /var/lib/rackd/rackd.db.backup

# Download new version
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-linux-amd64
chmod +x rackd-linux-amd64
sudo mv rackd-linux-amd64 /usr/local/bin/rackd

# Start the service
sudo systemctl start rackd

# Verify
rackd version
```

### Docker Upgrade

```bash
# Pull new image
docker pull martinsuchenak/rackd:latest

# Stop and remove old container
docker stop rackd
docker rm rackd

# Start new container (data persists in volume)
docker run -d \
  --name rackd \
  -p 8080:8080 \
  -v rackd-data:/data \
  martinsuchenak/rackd:latest
```

### Database Migrations

Database migrations run automatically on startup. No manual intervention required.

## Uninstallation

### Remove Binary

```bash
# Stop service
sudo systemctl stop rackd
sudo systemctl disable rackd

# Remove binary
sudo rm /usr/local/bin/rackd

# Remove service file
sudo rm /etc/systemd/system/rackd.service
sudo systemctl daemon-reload

# Remove data (optional)
sudo rm -rf /var/lib/rackd
```

### Remove Docker

```bash
# Stop and remove container
docker stop rackd
docker rm rackd

# Remove image
docker rmi martinsuchenak/rackd:latest

# Remove volume (deletes all data)
docker volume rm rackd-data
```

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 8080
sudo lsof -i :8080

# Use a different port
rackd server --listen-addr :8081
```

### Permission Denied

```bash
# Check file permissions
ls -la /var/lib/rackd

# Fix ownership
sudo chown -R rackd:rackd /var/lib/rackd
```

### Database Locked

```bash
# Check for stale lock files
ls -la /var/lib/rackd/

# Remove WAL files if server is stopped
rm /var/lib/rackd/rackd.db-wal
rm /var/lib/rackd/rackd.db-shm
```

For more troubleshooting, see [Troubleshooting Guide](troubleshooting.md).

## Next Steps

- [Quick Start Guide](quickstart.md) - Get started with basic operations
- [Configuration](configuration.md) - Configure Rackd for your environment
- [CLI Reference](cli.md) - Learn CLI commands
- [Web UI Guide](webui.md) - Explore the web interface
