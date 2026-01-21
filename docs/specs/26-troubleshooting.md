# Troubleshooting Guide

This document provides comprehensive troubleshooting guide for Rackd, including common issues, debug techniques, and recovery procedures.

## 1. Common Issues and Solutions

### 1.1 Database Lock Issues

**Symptoms:**
- "database is locked" errors
- Operations timeout
- Cannot write to database

**Causes:**
- Multiple processes accessing SQLite database
- Long-running transaction blocking writes
- Concurrent backup and write operations

**Solutions:**

```bash
# Check for multiple Rackd processes
ps aux | grep rackd

# Kill duplicate processes
pkill rackd

# Check for database locks
sqlite3 /data/rackd.db "PRAGMA lock_status;"

# If locks persist, restart Rackd
rackd server stop
sleep 5
rackd server start
```

**Prevention:**
- Ensure only one Rackd instance runs
- Use WAL (Write-Ahead Log) mode
- Avoid long-running transactions

```bash
# Enable WAL mode
rackd server --db-mode wal
```

### 1.2 Connection Refused Errors

**Symptoms:**
- "connection refused" when accessing API
- Cannot connect to MCP server
- CLI commands fail with connection errors

**Causes:**
- Server not running
- Wrong port specified
- Firewall blocking connections
- Server not binding to correct interface

**Solutions:**

```bash
# Check if server is running
ps aux | grep rackd

# Check server status
systemctl status rackd
# Or
service rackd status

# Verify server is listening
netstat -tuln | grep 8080
# Or
ss -tuln | grep 8080

# Check firewall
sudo ufw status
sudo iptables -L

# Test connection locally
curl -v http://localhost:8080/api/datacenters

# Test connection from remote machine
curl -v http://server-ip:8080/api/datacenters
```

**Configuration Check:**

```yaml
# config/config.yaml
server:
  listen_addr: ":8080"  # Ensure correct address
  bind_interface: "eth0"  # Optional: bind to specific interface
```

### 1.3 Discovery Scan Failures

**Symptoms:**
- Discovery scans stuck at "running"
- Scans fail with timeout errors
- No devices discovered

**Causes:**
- Network firewall blocking scans
- Incorrect network configuration
- Insufficient permissions
- Network connectivity issues

**Solutions:**

```bash
# Check discovery status
rackd discovery list --status running

# View scan details
rackd discovery list --verbose

# Check scan logs
journalctl -u rackd -n 100 | grep discovery

# Test network connectivity
ping -c 5 192.168.1.1

# Test port connectivity
nc -zv 192.168.1.1 22
nc -zv 192.168.1.1 80
nc -zv 192.168.1.1 443

# Check firewall rules
sudo iptables -L -n | grep 192.168.1.0/24
```

**Discovery Configuration:**

```yaml
# config/discovery.yaml
discovery:
  enabled: true
  max_concurrent: 10
  timeout: 5s
  exclude_ips: "192.168.1.1,192.168.1.254"
  scan_type: full
```

**Test Discovery:**

```bash
# Run quick scan
rackd discovery scan --network net_123 --type quick

# Run full scan
rackd discovery scan --network net_123 --type full

# Run deep scan
rackd discovery scan --network net_123 --type deep
```

### 1.4 Memory Exhaustion

**Symptoms:**
- "out of memory" errors
- Process killed by OOM killer
- System becomes unresponsive
- Swap usage at 100%

**Causes:**
- Large dataset in memory
- Memory leak in application
- Insufficient system RAM
- Concurrent operations consuming too much memory

**Solutions:**

```bash
# Check memory usage
free -h

# Check Rackd process memory
ps aux | grep rackd | awk '{print $6}'

# Check for memory leaks
rackd debug profile memory --duration 5m

# Reduce concurrency
rackd server --discovery-max-concurrent 5

# Enable memory profiling
rackd server --profile-memory --profile-dir /tmp/rackd-profiles
```

**Configuration Optimizations:**

```yaml
# config/performance.yaml
performance:
  memory:
    max_concurrent_discovery: 5
    batch_size: 100
    cache_enabled: true
    cache_ttl: 5m
    cache_max_entries: 1000
```

**System Tuning:**

```bash
# Reduce swap usage (if memory is sufficient)
sudo sysctl vm.swappiness=10

# Set OOM killer behavior
sudo sysctl vm.overcommit_memory=1

# View OOM killer logs
sudo dmesg | grep -i "killed process"
```

### 1.5 High CPU Usage

**Symptoms:**
- CPU usage at 100%
- Slow response times
- System unresponsive

**Causes:**
- Infinite loop in application
- Discovery scan running without limits
- Database query not using indexes
- CPU-bound operations

**Solutions:**

```bash
# Check CPU usage
top -p $(pidof rackd)

# Check which threads are using CPU
ps -T -p $(pidof rackd)

# Generate CPU profile
rackd debug profile cpu --duration 30s

# Check for high CPU operations
journalctl -u rackd | grep "high cpu"
```

**CPU Profile Analysis:**

```bash
# Analyze CPU profile
go tool pprof /tmp/rackd-profiles/cpu.prof

# View top functions
(pprof) top

# View graph
(pprof) web

# View flamegraph
go tool pprof -http=:8080 /tmp/rackd-profiles/cpu.prof
```

### 1.6 Slow API Responses

**Symptoms:**
- API requests take > 1 second
- Timeouts on API calls
- Poor user experience

**Causes:**
- Database queries not optimized
- Missing indexes
- Network latency
- Server overloaded
- Blocking operations

**Solutions:**

```bash
# Measure API response time
time curl http://localhost:8080/api/devices

# Check API response times
rackd metrics api --duration 1m

# Check database query performance
rackd debug profile db --duration 5m

# Check database indexes
rackd db analyze
```

**Database Optimization:**

```sql
-- Check for missing indexes
SELECT name FROM sqlite_master
WHERE type='index'
  AND tbl_name = 'devices'
  AND sql NOT LIKE '%_name%';

-- Analyze query plans
EXPLAIN QUERY PLAN
SELECT * FROM devices WHERE name LIKE '%server%';

-- Vacuum database
VACUUM;

-- Analyze statistics
ANALYZE;
```

**Configuration Tuning:**

```yaml
# config/api.yaml
api:
  timeout: 30s
  max_connections: 100
  connection_pool:
    max_open: 50
    max_idle: 10
    max_lifetime: 5m
```

### 1.7 Stuck Background Jobs

**Symptoms:**
- Discovery scans never complete
- Backup jobs stuck
- Scheduled tasks not running

**Causes:**
- Job deadlocked
- Exception not caught
- Background worker stopped
- Job queue blocked

**Solutions:**

```bash
# Check job status
rackd jobs list

# Check worker logs
journalctl -u rackd -n 100 | grep worker

# Restart worker
rackd worker restart

# Force job cancellation
rackd jobs cancel <job-id>

# Clear stuck jobs
rackd jobs clear-stuck
```

**Worker Diagnostics:**

```go
// Debug stuck jobs
func debugStuckJobs(db *sql.DB) {
    // Find jobs stuck for > 1 hour
    stuckJobs := `
        SELECT id, type, status, created_at, updated_at
        FROM jobs
        WHERE status IN ('running', 'pending')
          AND updated_at < datetime('now', '-1 hour')
    `

    rows, err := db.Query(stuckJobs)
    if err != nil {
        log.Error("Failed to query stuck jobs", "error", err)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var id, jobType, status string
        var createdAt, updatedAt time.Time
        rows.Scan(&id, &jobType, &status, &createdAt, &updatedAt)

        log.Warn("Stuck job detected",
            "id", id,
            "type", jobType,
            "status", status,
            "created_at", createdAt,
            "updated_at", updatedAt,
            "stuck_for", time.Since(updatedAt),
        )
    }
}
```

### 1.8 Web UI Not Loading

**Symptoms:**
- Blank white screen
- JavaScript errors in console
- Resources not loading
- Navigation not working

**Causes:**
- Asset serving errors
- JavaScript syntax errors
- Network issues
- Browser compatibility

**Solutions:**

```bash
# Check asset files
ls -la /data/rackd/ui/assets/

# Check if assets are embedded
rackd server check --assets

# Check HTTP responses
curl -I http://localhost:8080/app.js
curl -I http://localhost:8080/output.css

# Check server logs
journalctl -u rackd -n 100 | grep "GET /"

# Rebuild assets
rackd build ui
```

**Browser Console Debugging:**

```javascript
// Check for errors
console.error('Errors:', window.__rackdErrors__)

// Check API connectivity
fetch('/api/datacenters')
  .then(response => console.log('API OK'))
  .catch(error => console.error('API Error:', error))

// Check configuration
console.log('Config:', window.rackdConfig)
```

**Asset Verification:**

```bash
# Verify asset checksums
rackd assets verify --checksum-file checksums.json

# Rebuild if checksums don't match
rackd build ui --force
```

### 1.9 MCP Connection Failures

**Symptoms:**
- MCP clients cannot connect
- Authentication failures
- Tool execution errors

**Causes:**
- Wrong MCP port
- Missing or invalid authentication token
- Network firewall blocking connections
- MCP server not running

**Solutions:**

```bash
# Check MCP server status
rackd server status --component mcp

# Test MCP connection
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-01-20"}}'

# Check MCP authentication
MCP_TOKEN=your-token curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $MCP_TOKEN" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'

# Check MCP logs
journalctl -u rackd | grep MCP
```

**MCP Configuration:**

```yaml
# config/mcp.yaml
mcp:
  enabled: true
  listen_addr: ":8080"
  auth_token: "${MCP_AUTH_TOKEN}"
  max_connections: 100
  timeout: 30s
  allowed_tools: []  # Empty = all tools
```

### 1.10 Backup Verification Failures

**Symptoms:**
- Backup verification fails
- Checksum mismatch
- Database integrity errors

**Causes:**
- Backup corrupted during transfer
- Storage disk full
- Database locked during backup
- Checksum calculation error

**Solutions:**

```bash
# Check backup file
ls -lh /backups/rackd/

# Verify backup integrity
rackd backup verify backup_20240120_083000

# Check disk space
df -h /backups

# Checksum manually
sha256sum /backups/rackd/backup_20240120_083000.db

# Verify database integrity
sqlite3 /backups/rackd/backup_20240120_083000.db "PRAGMA integrity_check;"
```

**Backup Repair:**

```bash
# If backup is corrupted, try to restore from different backup
rackd backup list --sort date

# Use the most recent valid backup
rackd backup restore backup_20240119_083000

# If no valid backup, recreate from current database
rackd backup create --type offline --name recovery_backup
```

---

## 2. Debug Techniques

### 2.1 Debug Log Configuration

**Enabling Debug Logging:**

```bash
# Enable debug logging
rackd server --log-level debug

# Enable component-specific logging
rackd server --log-level debug --log-components api,storage,discovery

# Enable verbose logging
rackd server --log-level trace

# Log to file
rackd server --log-file /var/log/rackd/rackd.log

# Log in JSON format
rackd server --log-format json
```

**Log Levels:**

| Level | Description | When to Use |
|-------|-------------|-------------|
| trace | Very fine-grained | Troubleshooting specific issues |
| debug | Detailed information | Development and debugging |
| info | General operational | Production monitoring |
| warn | Potentially problematic | Production warnings |
| error | Runtime errors | Error tracking |
| fatal | Critical errors | Critical failures |

### 2.2 Request Tracing

**Adding Request IDs:**

```go
package middleware

import (
    "context"
    "github.com/google/uuid"
    "net/http"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Generate request ID
        requestID := uuid.New().String()

        // Add to context
        ctx := context.WithValue(r.Context(), "request_id", requestID)

        // Add to response header
        w.Header().Set("X-Request-ID", requestID)

        // Log request ID
        log.Info("Request started",
            "request_id", requestID,
            "method", r.Method,
            "path", r.URL.Path,
        )

        // Pass to next handler
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Tracing Request Flow:**

```bash
# Find all log entries for a request
journalctl -u rackd | grep "request-id=abc123"

# Extract request flow
journalctl -u rackd | grep "request-id=abc123" | jq

# View request timeline
journalctl -u rackd | grep "request-id=abc123" | \
  awk '{print $1 " " $6 " " $7}'
```

### 2.3 Database Query Logging

**Enabling Query Logging:**

```go
package storage

import (
    "database/sql"
    "log"
)

type QueryLogger struct {
    db *sql.DB
}

func NewQueryLogger(db *sql.DB) *QueryLogger {
    return &QueryLogger{db: db}
}

func (ql *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
    log.Debug("Executing query",
        "query", query,
        "args", args,
    )

    rows, err := ql.db.Query(query, args...)

    if err != nil {
        log.Error("Query failed",
            "query", query,
            "args", args,
            "error", err,
        )
    }

    return rows, err
}

func (ql *QueryLogger) QueryRow(query string, args ...interface{}) *sql.Row {
    log.Debug("Executing query",
        "query", query,
        "args", args,
    )

    return ql.db.QueryRow(query, args...)
}
```

**Slow Query Detection:**

```go
func (ql *QueryLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
    start := time.Now()

    rows, err := ql.db.Query(query, args...)

    duration := time.Since(start)

    if duration > 100*time.Millisecond {
        log.Warn("Slow query detected",
            "query", query,
            "duration_ms", duration.Milliseconds(),
            "args", args,
        )
    }

    return rows, err
}
```

### 2.4 Goroutine Dump Analysis

**Capturing Goroutine Dumps:**

```bash
# Send SIGQUIT to dump goroutines
kill -QUIT $(pidof rackd)

# Dump will be printed to stdout

# Or use profile endpoint
curl http://localhost:8080/debug/goroutines
```

**Analyzing Goroutine Dump:**

```bash
# Save goroutine dump to file
curl http://localhost:8080/debug/goroutines > goroutine-dump.txt

# Analyze goroutine dump
# Look for:
# - Large number of goroutines (> 1000)
# - Goroutines stuck in I/O
# - Deadlocks
# - Memory leaks

# Count goroutines by function
grep "created by" goroutine-dump.txt | sort | uniq -c | sort -rn | head -20
```

### 2.5 Memory Profiling

**Capturing Memory Profile:**

```bash
# Generate memory profile
curl http://localhost:8080/debug/profile/memory > heap.prof

# Or use CLI
rackd debug profile memory --output heap.prof

# Generate heap profile with GC
curl http://localhost:8080/debug/profile/memory?gc=1 > heap-gc.prof
```

**Analyzing Memory Profile:**

```bash
# View top memory allocations
go tool pprof -top heap.prof

# View memory graph
go tool pprof heap.prof

# Find memory leaks
go tool pprof -base base.prof -sample_index=1 heap.prof

# Compare profiles
go tool pprof -base baseline.prof -sample_index=inuse_space heap.prof
```

**Memory Leak Detection:**

```go
// Monitor memory growth
func MonitorMemoryGrowth(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    var lastStats runtime.MemStats

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            var stats runtime.MemStats
            runtime.ReadMemStats(&stats)

            allocDiff := stats.Alloc - lastStats.Alloc
            allocPercent := float64(allocDiff) / float64(lastStats.Alloc) * 100

            if allocPercent > 10 { // 10% growth
                log.Warn("Possible memory leak detected",
                    "alloc_diff_mb", allocDiff/(1024*1024),
                    "alloc_current_mb", stats.Alloc/(1024*1024),
                    "growth_percent", allocPercent,
                )

                // Capture profile
                CaptureMemoryProfile()
            }

            lastStats = stats
        }
    }
}
```

---

## 3. Log Analysis

### 3.1 Key Log Patterns

**Error Patterns:**

```
# Database errors
ERROR:.*database is locked
ERROR:.*database disk image is malformed
ERROR:.*no such table
ERROR:.*foreign key constraint failed

# Network errors
ERROR:.*connection refused
ERROR:.*timeout
ERROR:.*no route to host
ERROR:.*network is unreachable

# Discovery errors
ERROR:.*discovery scan failed
ERROR:.*scan timeout
ERROR:.*no hosts found

# API errors
ERROR:.*internal server error
ERROR:.*request validation failed
ERROR:.*unauthorized
```

**Warning Patterns:**

```
# Performance warnings
WARN:.*slow query
WARN:.*high memory usage
WARN:.*high cpu usage

# Resource warnings
WARN:.*disk space low
WARN:.*connection pool exhausted
WARN:.*rate limit approaching

# Operational warnings
WARN:.*retrying operation
WARN:.*degraded mode
WARN:.*stuck job detected
```

### 3.2 Log Aggregation Queries

**Using journalctl:**

```bash
# View recent logs
journalctl -u rackd -n 100

# View logs since specific time
journalctl -u rackd --since "1 hour ago"

# View logs with specific level
journalctl -u rackd | grep ERROR

# View logs for specific component
journalctl -u rackd | grep api

# Follow logs in real-time
journalctl -u rackd -f

# Export logs to file
journalctl -u rackd --since "1 hour ago" > rackd-logs.txt
```

**Log Analysis with jq:**

```bash
# Parse JSON logs
journalctl -u rackd --output json | \
  jq 'select(.level == "ERROR")'

# Count errors by type
journalctl -u rackd --output json | \
  jq 'group_by(.code) | map({code: .[0].code, count: length})'

# Find slow requests
journalctl -u rackd --output json | \
  jq 'select(.duration_ms > 1000)'
```

### 3.3 Error Rate Analysis

```bash
# Count errors in last hour
journalctl -u rackd --since "1 hour ago" | grep ERROR | wc -l

# Count errors by type
journalctl -u rackd --since "1 hour ago" --output json | \
  jq -r '.code' | sort | uniq -c | sort -rn | head -10

# Calculate error rate
total_logs=$(journalctl -u rackd --since "1 hour ago" | wc -l)
error_logs=$(journalctl -u rackd --since "1 hour ago" | grep ERROR | wc -l)
error_rate=$(echo "scale=2; $error_logs * 100 / $total_logs" | bc)

echo "Error rate: ${error_rate}%"
```

**Error Rate Alerting:**

```bash
# Alert if error rate > 10%
error_rate=15  # percent
if [ $error_rate -gt 10 ]; then
    echo "WARNING: High error rate detected: ${error_rate}%"
    # Send alert
    send_alert "High error rate: ${error_rate}%"
fi
```

### 3.4 Event Correlation

**Correlating Related Events:**

```bash
# Find all events for a request
REQUEST_ID="req-abc123"
journalctl -u rackd --output json | \
  jq "select(.request_id == \"$REQUEST_ID\")"

# Extract timeline
journalctl -u rackd --output json | \
  jq "select(.request_id == \"$REQUEST_ID\") | .timestamp, .message"

# Correlate errors with requests
journalctl -u rackd --output json | \
  jq 'group_by(.request_id) | map({
      request_id: .[0].request_id,
      events: [.[].message]
    })'
```

---

## 4. Performance Diagnosis

### 4.1 Identifying Bottlenecks

**CPU Bottlenecks:**

```bash
# Check CPU usage
top -b -n 1 | grep rackd

# Check CPU profile
rackd debug profile cpu --duration 30s

# Identify high CPU functions
go tool pprof -top cpu.prof | head -20
```

**Memory Bottlenecks:**

```bash
# Check memory usage
ps aux | grep rackd | awk '{print $6}'

# Check memory profile
rackd debug profile memory

# Identify large allocations
go tool pprof -top heap.prof | head -20
```

**I/O Bottlenecks:**

```bash
# Check I/O wait
iostat -x 1 5 | grep rackd

# Check disk usage
df -h /data

# Check file descriptor usage
ls -la /proc/$(pidof rackd)/fd | wc -l
```

**Network Bottlenecks:**

```bash
# Check network statistics
ifstat -i eth0 1 5

# Check connection count
netstat -an | grep :8080 | wc -l

# Check network latency
ping -c 10 $(hostname -I)
```

### 4.2 Database Query Analysis

**Query Profiling:**

```sql
-- Find slow queries
SELECT * FROM query_log
WHERE duration_ms > 100
ORDER BY duration_ms DESC
LIMIT 100;

-- Find queries without indexes
SELECT sql, query_plan
FROM sqlite_master
WHERE sql LIKE '%SELECT%'
  AND sql NOT LIKE '%WHERE%'
  AND sql NOT LIKE '%JOIN%';

-- Find missing indexes
SELECT name, sql
FROM sqlite_master
WHERE type = 'table'
  AND name NOT IN (
    SELECT DISTINCT tbl_name
    FROM sqlite_master
    WHERE type = 'index'
  );
```

**Index Usage Analysis:**

```bash
# Check index statistics
rackd db analyze --table devices

# View index stats
sqlite3 /data/rackd.db "SELECT * FROM pragma_index_info('devices');"

# Check index efficiency
sqlite3 /data/rackd.db "ANALYZE;"
```

### 4.3 Network Latency Investigation

**Network Latency Testing:**

```bash
# Test local latency
time curl http://localhost:8080/api/datacenters

# Test remote latency
time curl http://remote-server:8080/api/datacenters

# Test with different payloads
time curl -X POST http://localhost:8080/api/devices \
  -H "Content-Type: application/json" \
  -d '{"name":"test"}'

# Traceroute to identify bottlenecks
traceroute remote-server
```

**Latency Optimization:**

```yaml
# config/network.yaml
network:
  tcp:
    keep_alive: true
    keep_alive_idle: 30s
    keep_alive_interval: 10s
  timeout:
    read: 30s
    write: 30s
    dial: 10s
```

### 4.4 Resource Utilization Monitoring

**Resource Monitoring Dashboard:**

```bash
# CPU, Memory, Disk, Network
while true; do
    echo "=== $(date) ==="
    echo "CPU: $(top -b -n 1 | grep rackd | awk '{print $9}')"
    echo "Memory: $(ps aux | grep rackd | awk '{print $6}')"
    echo "Disk: $(df -h /data | tail -1)"
    echo "Network: $(netstat -an | grep :8080 | wc -l)"
    sleep 60
done
```

**Resource Alerts:**

```bash
# Alert on high CPU
cpu_usage=$(top -b -n 1 | grep rackd | awk '{print $9}')
if (( $(echo "$cpu_usage > 80" | bc -l) )); then
    send_alert "High CPU usage: ${cpu_usage}%"
fi

# Alert on low disk space
disk_usage=$(df -h /data | tail -1 | awk '{print $5}' | sed 's/%//')
if [ $disk_usage -lt 10 ]; then
    send_alert "Low disk space: ${disk_usage}%"
fi

# Alert on high memory
mem_usage=$(ps aux | grep rackd | awk '{print $4}')
if [ $mem_usage -gt 80 ]; then
    send_alert "High memory usage: ${mem_usage}%"
fi
```

---

## 5. Recovery Procedures

### 5.1 Database Corruption Recovery

**Detecting Corruption:**

```bash
# Check database integrity
sqlite3 /data/rackd.db "PRAGMA integrity_check;"

# Check foreign key constraints
sqlite3 /data/rackd.db "PRAGMA foreign_key_check;"

# Check for errors
sqlite3 /data/rackd.db "SELECT * FROM pragma_database_list;"
```

**Recovery Options:**

**Option 1: Export and Import**

```bash
# Export data to SQL
sqlite3 /data/rackd.db .dump > backup.sql

# Create new database
sqlite3 /data/rackd-new.db < backup.sql

# Replace corrupted database
mv /data/rackd.db /data/rackd-corrupted.db
mv /data/rackd-new.db /data/rackd.db

# Restart server
rackd server restart
```

**Option 2: Partial Recovery**

```bash
# Export tables individually
for table in devices networks datacenters; do
    sqlite3 /data/rackd.db ".dump $table" > $table.sql
done

# Rebuild database
sqlite3 /data/rackd-new.db < schema.sql
for table in devices networks datacenters; do
    sqlite3 /data/rackd-new.db < $table.sql
done
```

**Option 3: Restore from Backup**

```bash
# Find latest valid backup
rackd backup list --sort date | head -1

# Restore backup
rackd backup restore backup_20240119_083000

# Verify restored data
rackd doctor validate
```

### 5.2 Stuck Scan Recovery

**Identifying Stuck Scans:**

```bash
# Check scan status
rackd discovery list --status running

# Find scans running > 1 hour
rackd discovery list --stuck-duration 1h

# Get scan details
rackd discovery scan-info <scan-id>
```

**Recovery Actions:**

```bash
# Cancel stuck scan
rackd discovery cancel <scan-id>

# Force restart scanner
rackd discovery restart --scanner-id <scanner-id>

# Clear all stuck scans
rackd discovery clear-stuck

# Restart worker
rackd worker restart
```

**Preventing Stuck Scans:**

```yaml
# config/discovery.yaml
discovery:
  max_scan_duration: 1h  # Maximum scan duration
  scan_timeout: 30m   # Timeout per host
  heartbeat_interval: 5m # Check progress every 5 minutes
```

### 5.3 Deadlock Resolution

**Detecting Deadlocks:**

```bash
# Check for deadlocks in logs
journalctl -u rackd | grep -i deadlock

# Check goroutine dump
kill -QUIT $(pidof rackd)

# Look for goroutines waiting on locks
curl http://localhost:8080/debug/goroutines | grep "waiting on"
```

**Resolving Deadlocks:**

```bash
# Kill and restart Rackd
pkill -9 rackd
sleep 5
rackd server start

# Or graceful shutdown
rackd server stop
sleep 10
rackd server start
```

**Prevention:**

```go
// Use timeouts
func safeOperation() error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    resultChan := make(chan error, 1)

    go func() {
        resultChan <- longOperation(ctx)
    }()

    select {
    case err := <-resultChan:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### 5.4 Service Restart Procedures

**Graceful Restart:**

```bash
# Check server status
rackd server status

# Gracefully stop server
rackd server stop

# Wait for shutdown to complete
sleep 10

# Verify no processes running
ps aux | grep rackd

# Start server
rackd server start

# Verify server is running
curl -f http://localhost:8080/api/datacenters
```

**Force Restart:**

```bash
# Kill all Rackd processes
pkill -9 rackd

# Wait for processes to exit
sleep 5

# Start server
rackd server start

# Verify server is running
curl -f http://localhost:8080/api/datacenters
```

**Service Health Check:**

```bash
# Check server health
curl -f http://localhost:8080/healthz || echo "Server not healthy"

# Check database connectivity
rackd db check

# Check worker status
rackd worker status

# Check API endpoints
curl -f http://localhost:8080/api/devices
curl -f http://localhost:8080/api/networks
curl -f http://localhost:8080/api/datacenters
```

### 5.5 Emergency Mode Activation

**Activating Emergency Mode:**

```bash
# Start server in emergency mode
rackd server --emergency-mode

# Emergency mode features:
# - Read-only API (no writes)
# - Disabled discovery
# - Disabled background jobs
# - Reduced logging
# - Disabled webhooks
```

**Emergency Mode Configuration:**

```yaml
# config/emergency.yaml
emergency:
  enabled: true
  read_only: true
  disable_discovery: true
  disable_webhooks: true
  disable_background_jobs: true
  max_connections: 10
  log_level: error
  cache_enabled: false
```

**Exiting Emergency Mode:**

```bash
# Disable emergency mode
rackd server --emergency-mode=false

# Or update config
# Edit config/emergency.yaml: set enabled: false

# Restart server
rackd server restart
```

---

## 6. Troubleshooting CLI Commands

### 6.1 System Health Check

```bash
# Run comprehensive system check
rackd doctor

# Check specific components
rackd doctor --component database
rackd doctor --component api
rackd doctor --component discovery
rackd doctor --component storage

# Quick check
rackd doctor --quick

# Verbose output
rackd doctor --verbose

# Output format
rackd doctor --format json
rackd doctor --format yaml
```

**Doctor Checks:**

| Check | Description |
|-------|-------------|
| Database connectivity | Can connect to database |
| Database integrity | Database passes integrity checks |
| Database size | Database size and growth |
| Index usage | Index statistics and efficiency |
| API endpoints | All API endpoints responding |
| API authentication | Authentication working (if configured) |
| MCP server | MCP server accessible |
| Worker status | Background worker running |
| Discovery status | Discovery scanner status |
| Disk space | Sufficient disk space |
| Memory usage | Memory usage within limits |
| CPU usage | CPU usage within limits |
| Network connectivity | Network connections working |

**Sample Output:**

```bash
$ rackd doctor

Rackd System Health Check
=======================

✓ Database connectivity: OK
✓ Database integrity: OK
✓ API endpoints: OK
✓ Worker status: OK
✓ Disk space: 23.5 GB free (46%)
✓ Memory usage: 512 MB / 1 GB (51%)
✓ CPU usage: 15% / 100%
✓ Discovery scanner: OK

Warnings:
⚠ Database size: 2.3 GB (recommend vacuum)
⚠ High memory usage: 51% (recommend review)

Overall Status: HEALTHY
```

### 6.2 Log Retrieval

```bash
# View recent logs
rackd debug logs --tail 100

# View logs for component
rackd debug logs --component api --tail 50

# View logs for level
rackd debug logs --level ERROR --tail 100

# View logs since time
rackd debug logs --since "1 hour ago"

# View logs matching pattern
rackd debug logs --pattern "device.*not found"

# Follow logs in real-time
rackd debug logs --follow

# Export logs to file
rackd debug logs --output rackd-logs.txt
```

### 6.3 Profile Generation

```bash
# Generate CPU profile
rackd debug profile cpu --duration 30s --output cpu.prof

# Generate memory profile
rackd debug profile memory --output heap.prof

# Generate goroutine profile
rackd debug profile goroutines --output goroutines.prof

# Generate block profile
rackd debug profile block --duration 30s --output block.prof

# Generate all profiles
rackd debug profile all --duration 60s

# Interactive profiling
rackd debug profile --interactive
```

### 6.4 Runtime Statistics

```bash
# View runtime statistics
rackd debug stats

# View memory statistics
rackd debug stats --memory

# View goroutine statistics
rackd debug stats --goroutines

# View database statistics
rackd debug stats --database

# View API statistics
rackd debug stats --api

# View worker statistics
rackd debug stats --worker

# Update interval
rackd debug stats --interval 5s

# Export statistics
rackd debug stats --export stats.json
```

**Sample Output:**

```bash
$ rackd debug stats

Runtime Statistics
==================

Memory:
  Alloc: 512 MB
  Total Alloc: 2.5 GB
  Sys: 256 MB
  Num GC: 1423
  GC Pause: 2.5 ms

Goroutines:
  Total: 45
  Running: 23
  Waiting: 22

Database:
  Connections: 8/50
  Queries/sec: 125
  Avg Query Time: 3.2 ms

API:
  Requests/sec: 45
  Avg Response Time: 85 ms
  P50 Response Time: 65 ms
  P95 Response Time: 150 ms
  P99 Response Time: 250 ms

Worker:
  Active Jobs: 3
  Pending Jobs: 12
  Completed Jobs: 1423
```

### 6.5 Reset Internal State

```bash
# Reset specific component
rackd debug reset --component cache
rackd debug reset --component rate-limit
rackd debug reset --component webhooks
rackd debug reset --component scheduler

# Reset all internal state
rackd debug reset --all

# Clear stuck jobs
rackd debug reset --stuck-jobs

# Reset with confirmation
rackd debug reset --all --confirm

# Output:
# Cache cleared: 1234 entries
# Rate limiters reset: 45 entries
# Webhooks reset: 12 webhooks
# Scheduler reset: 3 jobs cancelled
```

---

## 7. Support Information

### 7.1 Required Diagnostic Information

**Bug Report Template:**

```
## Environment

- Rackd version:
- Go version:
- OS: [e.g., Ubuntu 22.04]
- Architecture: [e.g., amd64]
- Database: [e.g., SQLite 3.38.5]
- Configuration: [attach config file]

## Issue Description

[Detailed description of the issue]

## Steps to Reproduce

1.
2.
3.

## Expected Behavior

[What you expected to happen]

## Actual Behavior

[What actually happened]

## Logs

[Attach relevant logs]

## Error Messages

[Copy-paste error messages]

## Additional Information

[Any other relevant information]
```

**System Information Collection:**

```bash
# Collect system info
rackd doctor --format json > system-info.json

# Collect logs
journalctl -u rackd --since "1 hour ago" > recent-logs.txt

# Collect configuration
cp /etc/rackd/config.yaml config-backup.yaml

# Collect database schema
sqlite3 /data/rackd.db ".schema" > schema.sql

# Collect runtime stats
rackd debug stats --format json > runtime-stats.json

# Package everything
tar -czf rackd-support-$(date +%Y%m%d).tar.gz \
  system-info.json \
  recent-logs.txt \
  config-backup.yaml \
  schema.sql \
  runtime-stats.json
```

### 7.2 Log Collection Procedure

```bash
# Collect last N hours of logs
HOURS=24

# Export to file
journalctl -u rackd --since "$HOURS hours ago" > rackd-logs.txt

# Compress logs
gzip rackd-logs.txt

# Generate log summary
journalctl -u rackd --since "$HOURS hours ago" --output json | \
  jq '{error_count: [. | select(.level == "ERROR") | length],
        warning_count: [. | select(.level == "WARN") | length],
        info_count: [. | select(.level == "INFO") | length]}'
```

### 7.3 System Information Template

**Template:**

```json
{
  "rackd": {
    "version": "v1.2.0",
    "build": "20240120120000",
    "commit": "abcdef123456"
  },
  "system": {
    "os": "Ubuntu 22.04",
    "kernel": "5.15.0-91-generic",
    "architecture": "amd64",
    "hostname": "rackd-server-01"
  },
  "go": {
    "version": "go1.21.6"
  },
  "database": {
    "type": "SQLite",
    "version": "3.38.5",
    "size_mb": 2345,
    "wal_mode": true,
    "foreign_keys": true
  },
  "configuration": {
    "listen_addr": ":8080",
    "data_dir": "/data/rackd",
    "log_level": "info",
    "discovery_enabled": true,
    "api_auth_token": "***"
  },
  "resources": {
    "cpu_cores": 4,
    "memory_mb": 8192,
    "disk_gb": 500,
    "disk_free_gb": 234.5
  }
}
```

---

## 8. Preventive Measures

### 8.1 Regular Maintenance Tasks

**Daily:**

```bash
# Check system health
rackd doctor

# Review logs for errors
journalctl -u rackd --since "1 day ago" | grep ERROR

# Check disk space
df -h /data
```

**Weekly:**

```bash
# Vacuum database
sqlite3 /data/rackd.db "VACUUM;"

# Analyze database
sqlite3 /data/rackd.db "ANALYZE;"

# Clean up old discoveries
rackd discovery cleanup --older-than 30d

# Check backup status
rackd backup list
```

**Monthly:**

```bash
# Create backup
rackd backup create --type online --name monthly

# Update configuration
rackd config update

# Review performance metrics
rackd metrics summary --duration 30d

# Review security logs
rackd logs security --duration 30d
```

### 8.2 Health Monitoring Setup

**Monitoring Script:**

```bash
#!/bin/bash
# monitor-rackd.sh

# Check server health
if ! curl -f http://localhost:8080/healthz; then
    send_alert "Server health check failed"
fi

# Check error rate
error_count=$(journalctl -u rackd --since "5 minutes ago" | grep ERROR | wc -l)
if [ $error_count -gt 10 ]; then
    send_alert "High error rate: $error_count errors in 5 minutes"
fi

# Check disk space
disk_percent=$(df -h /data | tail -1 | awk '{print $5}' | sed 's/%//')
if [ $disk_percent -lt 10 ]; then
    send_alert "Low disk space: ${disk_percent}% remaining"
fi

# Check memory usage
mem_percent=$(ps aux | grep rackd | awk '{print $4}')
if [ $mem_percent -gt 90 ]; then
    send_alert "High memory usage: ${mem_percent}%"
fi
```

### 8.3 Performance Baseline Establishment

**Baseline Recording:**

```bash
# Establish performance baseline
rackd benchmark --duration 1h --name baseline

# Save baseline metrics
rackd metrics baseline save --name baseline

# Compare against current performance
rackd metrics baseline compare baseline --threshold 20%
```

**Alert Thresholds:**

```yaml
# config/monitoring.yaml
alerts:
  error_rate:
    warning: 5%  # 5% error rate
    critical: 10% # 10% error rate

  response_time:
    warning_ms: 500  # P95 > 500ms
    critical_ms: 1000 # P95 > 1s

  resource_usage:
    cpu_warning: 80%
    cpu_critical: 95%
    memory_warning: 80%
    memory_critical: 95%
    disk_warning: 20%
    disk_critical: 10%
```

### 8.4 Backup Verification Schedule

**Automated Verification:**

```bash
# Verify latest backup daily
0 2 * * * * rackd backup verify latest

# Check backup integrity weekly
0 3 * * * 0 rackd backup check --older-than 7d

# Test restore monthly
0 4 1 * * * 1 rackd backup test-restore latest

# Report backup status
0 5 * * * * 1 rackd backup report
```
