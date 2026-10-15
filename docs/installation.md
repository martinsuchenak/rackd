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

### 1. Download a Release (Recommended)

Releases are published as archives on [GitHub Releases](https://github.com/martinsuchenak/rackd/releases), named `rackd_<version>_<os>_<arch>.tar.gz` (`.zip` for Windows) for `linux`, `darwin` and `windows` on `amd64` and `arm64`.

```bash
VERSION=$(curl -fsSLI -o /dev/null -w '%{url_effective}' https://github.com/martinsuchenak/rackd/releases/latest | sed 's#.*/v##')   # latest release, e.g. 0.0.22
OS=linux        # linux or darwin
ARCH=amd64      # amd64 or arm64

curl -LO "https://github.com/martinsuchenak/rackd/releases/download/v${VERSION}/rackd_${VERSION}_${OS}_${ARCH}.tar.gz"
curl -LO "https://github.com/martinsuchenak/rackd/releases/download/v${VERSION}/checksums.txt"
tar -xzf "rackd_${VERSION}_${OS}_${ARCH}.tar.gz" rackd
sudo install -m 0755 rackd /usr/local/bin/rackd
```

On macOS you can also install with Homebrew:

```bash
brew install --cask martinsuchenak/tap/rackd
```

On Windows, download `rackd_<version>_windows_amd64.zip` from the release page and extract `rackd.exe`.

Verify installation:

```bash
rackd version
```

### Verifying a Release

Check that the archive matches the published checksum:

```bash
sha256sum --ignore-missing -c checksums.txt     # macOS: shasum -a 256 --ignore-missing -c checksums.txt
```

Releases after v0.0.21 also publish a [cosign](https://github.com/sigstore/cosign) signature bundle (`checksums.txt.sigstore.json`), an SPDX SBOM per archive (`*.sbom.json`, each with its own `.sigstore.json`), and a GitHub build provenance attestation. To confirm `checksums.txt` was produced by this repository's release workflow:

```bash
curl -LO "https://github.com/martinsuchenak/rackd/releases/download/v${VERSION}/checksums.txt.sigstore.json"
cosign verify-blob checksums.txt \
  --bundle checksums.txt.sigstore.json \
  --certificate-identity "https://github.com/martinsuchenak/rackd/.github/workflows/release.yml@refs/tags/v${VERSION}" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Or check the build provenance with the GitHub CLI:

```bash
gh attestation verify "rackd_${VERSION}_${OS}_${ARCH}.tar.gz" --repo martinsuchenak/rackd
```

### 2. Build from Source

#### Prerequisites
- Go 1.27 or later
- Bun (for building the web UI)
- [Task](https://taskfile.dev)

#### Build Steps

```bash
git clone https://github.com/martinsuchenak/rackd.git
cd rackd

# Builds the web UI and the binary into ./build/rackd
task build:local
./build/rackd version
```

See [Development Guide](development.md) for more details.

### 3. Docker

No pre-built image is published; build it from the repository:

```bash
git clone https://github.com/martinsuchenak/rackd.git
cd rackd
docker build -t rackd:local \
  --build-arg VERSION="$(git describe --tags --always)" \
  --build-arg COMMIT="$(git rev-parse --short HEAD)" .
docker run -d --name rackd -p 8080:8080 -v rackd-data:/data rackd:local
```

Or use the repository's `docker-compose.yml`, which builds the same image:

```bash
RACKD_VERSION="$(git describe --tags --always)" RACKD_COMMIT="$(git rev-parse --short HEAD)" \
  docker compose up -d --build
```

Without the build arguments the image still works, but `rackd version` reports `dev`.

Settings are environment variables without a prefix (for example `LOG_LEVEL`, `DATA_DIR`, `ENCRYPTION_KEY`); see [Configuration](configuration.md). The image's health check uses `/healthz`.

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
# Server
LISTEN_ADDR=:8080
DATA_DIR=/var/lib/rackd

# First admin, created only when the database has no users
INITIAL_ADMIN_USERNAME=admin
INITIAL_ADMIN_PASSWORD=change-me-now
INITIAL_ADMIN_EMAIL=admin@example.com

# Encryption for credentials, DNS provider tokens and webhook secrets.
# Generate with: openssl rand -hex 32
# Rackd refuses to start if this is set but isn't 64 hex characters.
ENCRYPTION_KEY=

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Discovery
DISCOVERY_INTERVAL=24h
```

Make the file readable only by the service user (`chmod 600 /etc/rackd/config.env`), since it holds the encryption key.

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
