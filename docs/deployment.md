# Deployment Guide

This guide covers various deployment methods for Rackd in production environments.

## Docker Deployment

Each release publishes a multi-arch image (`linux/amd64`, `linux/arm64`) to `ghcr.io/martinsuchenak/rackd`, tagged with the version (`0.6.0`), the minor version (`0.6`) and `latest`. Images are signed with cosign; see [Verifying the Image](#verifying-the-image). The image keeps its database in `/data`, so mount a volume there.

### Basic Docker Run

```bash
# Create data directory
mkdir -p /opt/rackd/data

# Run container
docker run -d \
  --name rackd \
  -p 8080:8080 \
  -v /opt/rackd/data:/data \
  -e ENCRYPTION_KEY="$(openssl rand -hex 32)" \
  ghcr.io/martinsuchenak/rackd:0.6
```

Keep the `ENCRYPTION_KEY` you start with: credentials, scan profiles and DNS provider tokens are encrypted with it. Pin a version tag (`0.6` follows patch releases) rather than `latest`, so an upgrade is a choice you make.

### Docker with Environment Variables

```bash
docker run -d \
  --name rackd \
  -p 8080:8080 \
  -v /opt/rackd/data:/data \
  -e ENCRYPTION_KEY=your-64-hex-character-key \
  -e LOG_LEVEL=info \
  -e LOG_FORMAT=json \
  -e BACKUP_INTERVAL=24h \
  ghcr.io/martinsuchenak/rackd:0.6
```

See the [configuration reference](configuration-reference.md) for every setting.

### Verifying the Image

```bash
cosign verify ghcr.io/martinsuchenak/rackd:0.6.0 \
  --certificate-identity-regexp 'https://github.com/martinsuchenak/rackd/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  rackd:
    image: ghcr.io/martinsuchenak/rackd:0.6
    container_name: rackd
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - LOG_LEVEL=info
      - LOG_FORMAT=json
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 5s
      retries: 3

  nginx:
    image: nginx:alpine
    container_name: rackd-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - rackd
```

Create `.env` file:

```bash
ENCRYPTION_KEY=your-64-hex-character-key   # openssl rand -hex 32
```

Deploy:

```bash
docker-compose up -d
```

## Nomad Deployment

The Nomad job file is available at [`deploy/nomad.hcl`](../deploy/nomad.hcl).

### Prerequisites

1. Configure host volume in Nomad client:

```hcl
# nomad.hcl
client {
  host_volume "rackd-data" {
    path      = "/opt/nomad/rackd-data"
    read_only = false
  }
}
```

2. Set Nomad variables:

```bash
nomad var put nomad/jobs/rackd \
  api_auth_token="your-secure-token" \
  mcp_auth_token="your-mcp-token"
```

### Deploy

```bash
nomad job run deploy/nomad.hcl
```

## Systemd Service

### Binary Installation

```bash
# Download and install
curl -LO https://github.com/martinsuchenak/rackd/releases/latest/download/rackd-linux-amd64
sudo mv rackd-linux-amd64 /usr/local/bin/rackd
sudo chmod +x /usr/local/bin/rackd

# Create user and directories
sudo useradd -r -s /bin/false rackd
sudo mkdir -p /var/lib/rackd /etc/rackd
sudo chown rackd:rackd /var/lib/rackd
```

### Service Configuration

Create `/etc/systemd/system/rackd.service`:

```ini
[Unit]
Description=Rackd IPAM and Device Inventory
After=network.target
Wants=network.target

[Service]
Type=simple
User=rackd
Group=rackd
ExecStart=/usr/local/bin/rackd server
Restart=always
RestartSec=5
Environment=DATA_DIR=/var/lib/rackd
Environment=LISTEN_ADDR=:8080
Environment=LOG_LEVEL=info
EnvironmentFile=-/etc/rackd/rackd.env

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/rackd

[Install]
WantedBy=multi-user.target
```

Create `/etc/rackd/rackd.env` (mode `0600`, since it holds the key):

```bash
ENCRYPTION_KEY=your-64-hex-character-key   # openssl rand -hex 32
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable rackd
sudo systemctl start rackd
```

## Reverse Proxy Setup

### Nginx

Create `/etc/nginx/sites-available/rackd`:

```nginx
server {
    listen 80;
    server_name rackd.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name rackd.example.com;

    ssl_certificate /etc/ssl/certs/rackd.crt;
    ssl_certificate_key /etc/ssl/private/rackd.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Traefik

Create `traefik.yml`:

```yaml
api:
  dashboard: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"

certificatesResolvers:
  letsencrypt:
    acme:
      email: admin@example.com
      storage: acme.json
      httpChallenge:
        entryPoint: web

providers:
  docker:
    exposedByDefault: false
```

Update Docker Compose with Traefik labels:

```yaml
services:
  rackd:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.rackd.rule=Host(`rackd.example.com`)"
      - "traefik.http.routers.rackd.entrypoints=websecure"
      - "traefik.http.routers.rackd.tls.certresolver=letsencrypt"
```

## TLS/HTTPS

### Let's Encrypt with Certbot

```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d rackd.example.com

# Auto-renewal
sudo crontab -e
# Add: 0 12 * * * /usr/bin/certbot renew --quiet
```

### Self-Signed Certificate

```bash
# Generate certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout rackd.key -out rackd.crt \
  -subj "/CN=rackd.example.com"

# Install
sudo cp rackd.crt /etc/ssl/certs/
sudo cp rackd.key /etc/ssl/private/
sudo chmod 600 /etc/ssl/private/rackd.key
```

## Production Considerations

### Security

- Use strong authentication tokens
- Enable HTTPS in production
- Restrict network access with firewall rules
- Run with non-root user
- Regular security updates

### Performance

- Allocate sufficient memory (minimum 256MB)
- Use SSD storage for database
- Keep `DATA_DIR` on a local disk, not a network share (NFS, SMB/CIFS, or a
  Docker volume backed by one). SQLite is slow there and its locking is
  unreliable, which shows up as pages that take a long time to load and
  "context deadline exceeded" errors. Rackd logs a warning at start on Linux
  when it detects one. To keep a copy on the share, write backups there
  (`rackd backup`) instead.
- Monitor disk space usage
- Configure log rotation

### Backup Strategy

```bash
#!/bin/bash
# backup-rackd.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/rackd"
DATA_DIR="/var/lib/rackd"

mkdir -p $BACKUP_DIR
sqlite3 $DATA_DIR/rackd.db ".backup $BACKUP_DIR/rackd_$DATE.db"
gzip $BACKUP_DIR/rackd_$DATE.db

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -name "*.gz" -mtime +30 -delete
```

### Log Management

Configure log rotation in `/etc/logrotate.d/rackd`:

```
/var/log/rackd/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 rackd rackd
    postrotate
        systemctl reload rackd
    endscript
}
```

## Monitoring

### Health Checks

Rackd provides unauthenticated health check endpoints, `/healthz` (liveness)
and `/readyz` (readiness):

```bash
curl http://localhost:8080/healthz
```

### Prometheus Metrics

Add monitoring labels to Docker Compose:

```yaml
services:
  rackd:
    labels:
      - "prometheus.io/scrape=true"
      - "prometheus.io/port=8080"
      - "prometheus.io/path=/metrics"
```

The scrape needs an API key whose user holds `metrics:read`; see the
[monitoring guide](monitoring.md#prometheus-configuration).

### Basic Monitoring Script

```bash
#!/bin/bash
# monitor-rackd.sh
URL="http://localhost:8080/healthz"
TIMEOUT=5

if curl -f -s --max-time $TIMEOUT $URL > /dev/null; then
    echo "Rackd is healthy"
    exit 0
else
    echo "Rackd health check failed"
    exit 1
fi
```

### Uptime Monitoring

Configure external monitoring with services like:
- Pingdom
- UptimeRobot  
- StatusCake

Monitor these endpoints:
- `/healthz` - the server is up (no authentication)
- `/readyz` - the server can serve requests, database included
- `/` - Web UI availability

## Troubleshooting

### Common Issues

1. **Permission denied on data directory**
   ```bash
   sudo chown -R rackd:rackd /var/lib/rackd
   ```

2. **Port already in use**
   ```bash
   sudo netstat -tlnp | grep :8080
   ```

3. **Database locked**
   ```bash
   # Check for stale processes
   ps aux | grep rackd
   ```

### Logs

```bash
# Systemd logs
sudo journalctl -u rackd -f

# Docker logs
docker logs -f rackd

# Nomad logs
nomad alloc logs -f <alloc-id>
```