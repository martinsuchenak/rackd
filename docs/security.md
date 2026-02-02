# Security

This document covers security considerations and best practices for deploying and operating Rackd in production environments.

## Authentication

### Default Authentication
Rackd uses session-based authentication with secure cookies:

```bash
# Set authentication credentials
export RACKD_AUTH_USERNAME="admin"
export RACKD_AUTH_PASSWORD="secure-password-here"
```

### Session Security
- Sessions expire after 24 hours of inactivity
- Secure, HttpOnly cookies prevent XSS attacks
- CSRF protection on all state-changing operations
- Session tokens are cryptographically random

## API Tokens

### Token Generation
```bash
# Generate API token via CLI
rackd auth token create --name "automation" --expires "30d"

# Generate token via API
curl -X POST http://localhost:8080/api/v1/auth/tokens \
  -H "Content-Type: application/json" \
  -d '{"name": "integration", "expires_at": "2024-12-31T23:59:59Z"}'
```

### Token Security
- Tokens are SHA-256 hashed in database
- Support expiration dates
- Can be revoked individually
- Rate limited to prevent brute force attacks

### Using Tokens
```bash
# CLI with token
export RACKD_API_TOKEN="your-token-here"
rackd devices list

# HTTP requests
curl -H "Authorization: Bearer your-token-here" \
  http://localhost:8080/api/v1/devices
```

## Credential Encryption

### Database Encryption
Sensitive fields are encrypted at rest using AES-256-GCM:

```bash
# Set encryption key (32 bytes, base64 encoded)
export RACKD_ENCRYPTION_KEY="your-32-byte-key-base64-encoded"
```

### Encrypted Fields
- Device passwords and SSH keys
- SNMP community strings
- API credentials for external systems
- Certificate private keys

### Key Management
```bash
# Generate encryption key
openssl rand -base64 32

# Rotate encryption key (requires restart)
export RACKD_ENCRYPTION_KEY_OLD="old-key"
export RACKD_ENCRYPTION_KEY="new-key"
rackd server --rotate-keys
```

## TLS/HTTPS

### Enable TLS
```bash
# Using certificate files
export RACKD_TLS_CERT="/path/to/cert.pem"
export RACKD_TLS_KEY="/path/to/key.pem"
rackd server --port 8443

# Using Let's Encrypt
export RACKD_ACME_DOMAIN="rackd.example.com"
export RACKD_ACME_EMAIL="admin@example.com"
rackd server --acme
```

### Certificate Requirements
- TLS 1.2 minimum (TLS 1.3 preferred)
- RSA 2048-bit or ECDSA P-256 minimum
- Valid certificate chain
- Proper SAN entries for all hostnames

### Reverse Proxy Configuration
```nginx
# Nginx configuration
server {
    listen 443 ssl http2;
    server_name rackd.example.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_private_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Security Headers

### Default Headers
Rackd automatically sets security headers:

```
Strict-Transport-Security: max-age=31536000; includeSubDomains
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'
```

### Custom Headers
```bash
# Additional security headers
export RACKD_SECURITY_HEADERS='{"X-Custom-Header": "value"}'
```

## Network Security

### Firewall Configuration
```bash
# Allow only necessary ports
ufw allow 22/tcp    # SSH
ufw allow 8080/tcp  # Rackd HTTP
ufw allow 8443/tcp  # Rackd HTTPS
ufw deny incoming
ufw enable
```

### Network Isolation
- Run Rackd in isolated network segment
- Use VPN for remote access
- Implement network monitoring
- Regular security scanning

### Discovery Security
```bash
# Limit discovery networks
export RACKD_DISCOVERY_NETWORKS="10.0.0.0/8,192.168.0.0/16"

# Discovery credentials (encrypted)
export RACKD_SNMP_COMMUNITY="encrypted-community-string"
export RACKD_SSH_KEY="/path/to/discovery-key"
```

## Access Control

### Role-Based Access
```bash
# Create user with specific role
rackd users create --username "operator" --role "readonly"

# Available roles
# - admin: Full access
# - operator: Read/write devices and networks
# - readonly: View-only access
# - discovery: Discovery operations only
```

### API Permissions
```json
{
  "token": "abc123",
  "permissions": [
    "devices:read",
    "devices:write",
    "networks:read",
    "discovery:run"
  ]
}
```

### Network-Based Restrictions
```bash
# Restrict access by IP/network
export RACKD_ALLOWED_NETWORKS="10.0.0.0/8,192.168.1.0/24"
export RACKD_BLOCKED_IPS="192.168.1.100,10.0.0.50"
```

## Security Best Practices

### Deployment Security
1. **Run as non-root user**
   ```bash
   useradd -r -s /bin/false rackd
   sudo -u rackd ./rackd server
   ```

2. **File permissions**
   ```bash
   chmod 600 /etc/rackd/config.yaml
   chmod 600 /var/lib/rackd/rackd.db
   chown rackd:rackd /var/lib/rackd/
   ```

3. **System hardening**
   ```bash
   # Disable unnecessary services
   systemctl disable apache2 nginx
   
   # Update system regularly
   apt update && apt upgrade -y
   
   # Configure fail2ban
   apt install fail2ban
   ```

### Operational Security
1. **Regular backups**
   ```bash
   # Automated encrypted backups
   rackd backup --encrypt --output /backup/rackd-$(date +%Y%m%d).db.enc
   ```

2. **Log monitoring**
   ```bash
   # Monitor authentication failures
   tail -f /var/log/rackd/access.log | grep "401\|403"
   
   # Set up log rotation
   logrotate /etc/logrotate.d/rackd
   ```

3. **Security scanning**
   ```bash
   # Regular vulnerability scans
   nmap -sV -sC localhost
   
   # Dependency scanning
   go list -json -m all | nancy sleuth
   ```

### Configuration Security
```yaml
# /etc/rackd/config.yaml
server:
  bind: "127.0.0.1:8080"  # Bind to localhost only
  read_timeout: "30s"
  write_timeout: "30s"
  
security:
  session_timeout: "24h"
  max_login_attempts: 5
  lockout_duration: "15m"
  password_min_length: 12
  require_https: true
  
database:
  backup_interval: "6h"
  backup_retention: "30d"
```

### Incident Response
1. **Security incident checklist**
   - Isolate affected systems
   - Preserve logs and evidence
   - Rotate all credentials
   - Update security measures
   - Document lessons learned

2. **Emergency procedures**
   ```bash
   # Disable all API tokens
   rackd auth tokens revoke --all
   
   # Force logout all sessions
   rackd auth sessions clear
   
   # Enable maintenance mode
   rackd server --maintenance
   ```

## Compliance Considerations

### Data Protection
- Encrypt sensitive data at rest and in transit
- Implement data retention policies
- Provide data export/deletion capabilities
- Maintain audit logs

### Audit Logging
```bash
# Enable comprehensive audit logging
export RACKD_AUDIT_LOG="/var/log/rackd/audit.log"
export RACKD_AUDIT_LEVEL="info"

# Log format includes:
# - Timestamp
# - User/token identification
# - Action performed
# - Resource affected
# - Source IP address
```

### Regular Security Reviews
- Monthly access reviews
- Quarterly security assessments
- Annual penetration testing
- Continuous vulnerability monitoring

## Security Updates

Stay informed about security updates:
- Monitor [GitHub Security Advisories](https://github.com/martinsuchenak/rackd/security/advisories)
- Subscribe to release notifications
- Test updates in staging environment
- Maintain update schedule

For security issues, contact: security@rackd.dev