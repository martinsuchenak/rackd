# Troubleshooting

Common issues and solutions for Rackd.

## Server Issues

### Server Won't Start

#### Port Already in Use

**Symptom:**
```
Error: listen tcp :8080: bind: address already in use
```

**Solution:**

```bash
# Find what's using the port
sudo lsof -i :8080
# or
sudo netstat -tulpn | grep :8080

# Kill the process
sudo kill <PID>

# Or use a different port
rackd server --listen-addr :8081
```

#### Permission Denied

**Symptom:**
```
Error: open /var/lib/rackd/rackd.db: permission denied
```

**Solution:**

```bash
# Check permissions
ls -la /var/lib/rackd/

# Fix ownership
sudo chown -R rackd:rackd /var/lib/rackd/

# Or run with appropriate user
sudo -u rackd rackd server
```

#### Data Directory Not Found

**Symptom:**
```
Error: stat ./data: no such file or directory
```

**Solution:**

```bash
# Create data directory
mkdir -p ./data

# Or specify existing directory
rackd server --data-dir /var/lib/rackd
```

### Server Crashes

#### Out of Memory

**Symptom:**
```
fatal error: runtime: out of memory
```

**Solution:**

```bash
# Check memory usage
free -h

# Increase available memory
# Or reduce discovery scan size
# Limit concurrent scans in configuration
```

#### Database Corruption

**Symptom:**
```
Error: database disk image is malformed
```

**Solution:**

```bash
# Stop server
sudo systemctl stop rackd

# Backup database
cp /var/lib/rackd/rackd.db /var/lib/rackd/rackd.db.corrupt

# Try to recover
sqlite3 /var/lib/rackd/rackd.db ".recover" | sqlite3 /var/lib/rackd/rackd.db.recovered

# Replace with recovered database
mv /var/lib/rackd/rackd.db.recovered /var/lib/rackd/rackd.db

# Start server
sudo systemctl start rackd
```

## Database Issues

### Database Locked

**Symptom:**
```
Error: database is locked
```

**Solution:**

```bash
# Check for multiple instances
ps aux | grep rackd

# Kill duplicate instances
sudo killall rackd

# Remove stale lock files (only if server is stopped)
rm /var/lib/rackd/rackd.db-wal
rm /var/lib/rackd/rackd.db-shm

# Restart server
rackd server
```

### Slow Queries

**Symptom:**
- Slow API responses
- High CPU usage
- Long query times

**Solution:**

```bash
# Check database size
ls -lh /var/lib/rackd/rackd.db

# Vacuum database (requires downtime)
sudo systemctl stop rackd
sqlite3 /var/lib/rackd/rackd.db "VACUUM;"
sudo systemctl start rackd

# Analyze query performance
sqlite3 /var/lib/rackd/rackd.db "ANALYZE;"
```

### Migration Failures

**Symptom:**
```
Error: migration failed: ...
```

**Solution:**

```bash
# Check migration status
sqlite3 /var/lib/rackd/rackd.db "SELECT * FROM schema_migrations;"

# Restore from backup
sudo systemctl stop rackd
cp /var/lib/rackd/rackd.db.backup /var/lib/rackd/rackd.db
sudo systemctl start rackd

# If no backup, check logs for specific error
journalctl -u rackd -n 100
```

## API Issues

### Authentication Failures

#### 401 Unauthorized

**Symptom:**
```json
{
  "error": "unauthorized",
  "message": "Missing or invalid authentication token"
}
```

**Solution:**

```bash
# Check token is set
echo $RACKD_API_TOKEN

# Set token
export RACKD_API_TOKEN=your-secret-token

# Or pass in request
curl -H "Authorization: Bearer your-secret-token" http://localhost:8080/api/devices
```

#### Token Mismatch

**Symptom:**
- CLI works but API calls fail
- Inconsistent authentication

**Solution:**

```bash
# Ensure server and client use same token
# Server
export RACKD_API_AUTH_TOKEN=mysecret
rackd server

# Client
export RACKD_API_TOKEN=mysecret
rackd device list
```

### Request Failures

#### 400 Bad Request

**Symptom:**
```json
{
  "error": "validation_error",
  "message": "Invalid input",
  "details": [...]
}
```

**Solution:**

Check validation errors in response:
- Invalid CIDR notation
- Invalid IP address format
- Missing required fields
- Invalid field values

```bash
# Example: Fix invalid CIDR
# Wrong
rackd network add --name test --cidr 10.0.1.0

# Correct
rackd network add --name test --cidr 10.0.1.0/24
```

#### 404 Not Found

**Symptom:**
```json
{
  "error": "not_found",
  "message": "Resource not found"
}
```

**Solution:**

```bash
# Verify resource exists
rackd device list
rackd network list

# Check ID is correct
rackd device get <correct-id>
```

#### 500 Internal Server Error

**Symptom:**
```json
{
  "error": "internal_error",
  "message": "An internal error occurred"
}
```

**Solution:**

```bash
# Check server logs
journalctl -u rackd -n 50

# Look for stack traces or error messages
# Report bug if unexpected error
```

### Connection Issues

#### Connection Refused

**Symptom:**
```
Error: dial tcp 127.0.0.1:8080: connect: connection refused
```

**Solution:**

```bash
# Check server is running
sudo systemctl status rackd

# Check listening port
sudo netstat -tulpn | grep rackd

# Start server if not running
sudo systemctl start rackd
```

#### Timeout

**Symptom:**
```
Error: context deadline exceeded
```

**Solution:**

```bash
# Increase timeout
export RACKD_TIMEOUT=60s
rackd device list

# Or in CLI
rackd device list --timeout 60s

# Check network connectivity
ping localhost
curl http://localhost:8080/api/config
```

## Discovery Issues

### Scans Not Running

**Symptom:**
- No discovered devices
- Scans stuck in "running" state

**Solution:**

```bash
# Check discovery is enabled
# In server config
export RACKD_DISCOVERY_ENABLED=true

# Check scan status
rackd discovery list

# Cancel stuck scan
curl -X DELETE http://localhost:8080/api/discovery/scans/<scan-id>

# Check logs for errors
journalctl -u rackd | grep discovery
```

### No Devices Discovered

**Symptom:**
- Scan completes but finds no devices
- Expected devices not found

**Solution:**

```bash
# Verify network is reachable
ping 10.0.1.1

# Check CIDR is correct
# Ensure network exists and is accessible

# Try manual ping
nmap -sn 10.0.1.0/24

# Check firewall rules
sudo iptables -L

# Verify server has network access
# May need to run with appropriate permissions
```

### SSH/SNMP Discovery Fails

**Symptom:**
- Basic discovery works
- Advanced discovery fails
- No SSH/SNMP data collected

**Solution:**

```bash
# Verify credentials are configured
# Check SSH key or password
# Check SNMP community string

# Test SSH manually
ssh user@10.0.1.10

# Test SNMP manually
snmpwalk -v2c -c public 10.0.1.10

# Check credential encryption key is set
export RACKD_ENCRYPTION_KEY=your-32-byte-key
```

### Scheduled Scans Not Running

**Symptom:**
- Manual scans work
- Scheduled scans don't run

**Solution:**

```bash
# Check scheduled scan configuration
# Verify cron expression is valid

# Check logs for scheduler errors
journalctl -u rackd | grep scheduler

# Verify discovery is enabled
# Check RACKD_DISCOVERY_ENABLED=true

# Restart server to reload schedules
sudo systemctl restart rackd
```

## Web UI Issues

### UI Not Loading

**Symptom:**
- Blank page
- 404 errors for assets

**Solution:**

```bash
# Check server is running
curl http://localhost:8080/

# Check assets are embedded
# Rebuild if necessary
make build

# Check browser console for errors
# Open DevTools (F12) and check Console tab

# Clear browser cache
# Hard refresh (Ctrl+Shift+R or Cmd+Shift+R)
```

### UI Shows Errors

**Symptom:**
- "Failed to load data"
- API errors in UI

**Solution:**

```bash
# Check API is accessible
curl http://localhost:8080/api/config

# Check authentication
# If API requires token, UI needs it too

# Check browser console for specific errors
# Look for CORS issues, network errors, etc.

# Check server logs
journalctl -u rackd -f
```

### Dark Mode Not Working

**Symptom:**
- Theme toggle doesn't work
- Stuck in light/dark mode

**Solution:**

```bash
# Clear browser local storage
# In browser console:
localStorage.clear()

# Refresh page
# Toggle theme again
```

## CLI Issues

### Command Not Found

**Symptom:**
```
bash: rackd: command not found
```

**Solution:**

```bash
# Check installation
which rackd

# Add to PATH
export PATH=$PATH:/usr/local/bin

# Or use full path
/usr/local/bin/rackd version

# Verify binary is executable
chmod +x /usr/local/bin/rackd
```

### Config File Not Found

**Symptom:**
```
Warning: config file not found
```

**Solution:**

```bash
# Create config directory
mkdir -p ~/.rackd

# Create config file
cat > ~/.rackd/config.yaml <<EOF
api_url: http://localhost:8080
api_token: your-secret-token
EOF

# Or use environment variables
export RACKD_API_URL=http://localhost:8080
export RACKD_API_TOKEN=your-secret-token
```

## Docker Issues

### Container Won't Start

**Symptom:**
```
Error: container exited with code 1
```

**Solution:**

```bash
# Check logs
docker logs rackd

# Check volume permissions
docker volume inspect rackd-data

# Remove and recreate container
docker stop rackd
docker rm rackd
docker run -d --name rackd -p 8080:8080 -v rackd-data:/data martinsuchenak/rackd:latest
```

### Data Not Persisting

**Symptom:**
- Data lost after container restart
- Database resets

**Solution:**

```bash
# Ensure volume is mounted
docker inspect rackd | grep Mounts

# Use named volume
docker run -d -v rackd-data:/data martinsuchenak/rackd:latest

# Or bind mount
docker run -d -v /var/lib/rackd:/data martinsuchenak/rackd:latest
```

### Network Issues

**Symptom:**
- Can't access UI
- Port not accessible

**Solution:**

```bash
# Check port mapping
docker port rackd

# Ensure port is published
docker run -d -p 8080:8080 martinsuchenak/rackd:latest

# Check firewall
sudo ufw allow 8080

# Check container is running
docker ps
```

## Performance Issues

### High Memory Usage

**Symptom:**
- Server using excessive memory
- OOM killer terminates process

**Solution:**

```bash
# Check memory usage
docker stats rackd
# or
ps aux | grep rackd

# Reduce discovery scan size
# Limit concurrent operations
# Increase available memory

# Check for memory leaks
# Monitor over time
# Report if memory grows continuously
```

### High CPU Usage

**Symptom:**
- Server using 100% CPU
- Slow response times

**Solution:**

```bash
# Check what's running
top -p $(pgrep rackd)

# Check for active scans
rackd discovery list

# Cancel unnecessary scans
# Reduce scan frequency
# Limit scan scope

# Check for slow queries
# Enable query logging if needed
```

### Slow API Responses

**Symptom:**
- Requests take several seconds
- Timeouts

**Solution:**

```bash
# Check database size
ls -lh /var/lib/rackd/rackd.db

# Vacuum database
sqlite3 /var/lib/rackd/rackd.db "VACUUM;"

# Check indexes
sqlite3 /var/lib/rackd/rackd.db ".schema"

# Reduce result set size
# Use filtering and pagination

# Check server resources
# Ensure adequate CPU/memory
```

## Logging and Debugging

### Enable Debug Logging

```bash
# Start server with debug logging
rackd server --log-level debug --log-format json

# Or set environment variable
export RACKD_LOG_LEVEL=debug
rackd server
```

### View Logs

```bash
# systemd
journalctl -u rackd -f

# Docker
docker logs -f rackd

# File
tail -f /var/log/rackd.log
```

### Increase Log Verbosity

```bash
# Trace level (very verbose)
rackd server --log-level trace

# JSON format for parsing
rackd server --log-format json | jq
```

## Getting Help

### Collect Diagnostic Information

```bash
# Version
rackd version

# System info
uname -a
cat /etc/os-release

# Server status
sudo systemctl status rackd

# Recent logs
journalctl -u rackd -n 100 --no-pager

# Database info
sqlite3 /var/lib/rackd/rackd.db "PRAGMA integrity_check;"
sqlite3 /var/lib/rackd/rackd.db "SELECT * FROM schema_migrations;"

# Configuration
env | grep RACKD
```

### Report Issues

When reporting issues, include:

1. Rackd version (`rackd version`)
2. Operating system and version
3. Deployment method (binary, Docker, etc.)
4. Error messages and logs
5. Steps to reproduce
6. Expected vs actual behavior

**GitHub Issues**: https://github.com/martinsuchenak/rackd/issues

### Community Support

- **Documentation**: https://github.com/martinsuchenak/rackd/tree/main/docs
- **Discussions**: https://github.com/martinsuchenak/rackd/discussions
- **Issues**: https://github.com/martinsuchenak/rackd/issues

## Related Documentation

- [Installation](installation.md) - Installation guide
- [Configuration](configuration.md) - Configuration options
- [Deployment](deployment.md) - Deployment strategies
- [Security](security.md) - Security best practices
