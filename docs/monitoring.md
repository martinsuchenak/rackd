# Monitoring and Metrics

Rackd provides comprehensive monitoring capabilities through Prometheus-compatible metrics, health checks, and structured logging.

## Health Checks

Rackd exposes two health check endpoints for container orchestration and load balancers:

### Liveness Probe (`/healthz`)

Simple endpoint that returns `200 OK` if the application is running.

```bash
curl http://localhost:8080/healthz
# Response: ok
```

**Use case**: Kubernetes liveness probe to detect if the application needs to be restarted.

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
```

### Readiness Probe (`/readyz`)

Detailed endpoint that checks if the application is ready to serve traffic.

```bash
curl http://localhost:8080/readyz
```

**Response** (healthy):
```json
{
  "status": "healthy",
  "timestamp": "2026-02-03T12:00:00Z",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "database is accessible"
    },
    "scheduler": {
      "status": "healthy",
      "message": "discovery scheduler is running"
    }
  }
}
```

**Response** (unhealthy):
```json
{
  "status": "unhealthy",
  "timestamp": "2026-02-03T12:00:00Z",
  "checks": {
    "database": {
      "status": "unhealthy",
      "message": "no open database connections"
    }
  }
}
```

**Use case**: Kubernetes readiness probe to detect if the application can handle requests.

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Metrics

Rackd exposes Prometheus-compatible metrics at `/metrics` endpoint.

### Accessing Metrics

```bash
curl http://localhost:8080/metrics
```

**Note**: Metrics endpoint does not require authentication.

### Available Metrics

#### HTTP Metrics

- `http_requests_total` - Total number of HTTP requests (counter)
- `http_request_duration_milliseconds{route="..."}` - Average request duration by route (summary)
- `http_requests_by_code{code="..."}` - Total requests by HTTP status code (counter)

#### Application Metrics

- `devices_total` - Current number of devices (gauge)
- `networks_total` - Current number of networks (gauge)
- `datacenters_total` - Current number of datacenters (gauge)
- `discovery_scans_total` - Total number of discovery scans performed (counter)
- `discovery_scan_duration_milliseconds{type="..."}` - Average scan duration by type (summary)

#### Database Metrics

- `db_queries_total` - Total number of database queries (counter)
- `db_query_duration_milliseconds{query="..."}` - Average query duration (summary)
- `db_connections_open` - Current number of open database connections (gauge)

#### Runtime Metrics

- `process_uptime_seconds` - Process uptime in seconds (gauge)
- `go_goroutines` - Number of goroutines (gauge)
- `go_memory_alloc_bytes` - Bytes of allocated heap objects (gauge)
- `go_memory_sys_bytes` - Total bytes of memory obtained from the OS (gauge)

### Prometheus Configuration

Add Rackd to your Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'rackd'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Example Queries

**Request rate**:
```promql
rate(http_requests_total[5m])
```

**Average request duration**:
```promql
http_request_duration_milliseconds
```

**Error rate**:
```promql
rate(http_requests_by_code{code=~"5.."}[5m])
```

**Device growth**:
```promql
devices_total
```

**Discovery scan rate**:
```promql
rate(discovery_scans_total[1h])
```

## Grafana Dashboard

### Sample Dashboard Panels

**1. Request Rate**
```promql
rate(http_requests_total[5m])
```

**2. Error Rate**
```promql
rate(http_requests_by_code{code=~"5.."}[5m]) / rate(http_requests_total[5m])
```

**3. Response Time (p95)**
```promql
histogram_quantile(0.95, rate(http_request_duration_milliseconds[5m]))
```

**4. Active Resources**
```promql
devices_total
networks_total
datacenters_total
```

**5. Database Connections**
```promql
db_connections_open
```

**6. Memory Usage**
```promql
go_memory_alloc_bytes
```

## Alerting

### Sample Prometheus Alerts

```yaml
groups:
  - name: rackd
    rules:
      - alert: RackdDown
        expr: up{job="rackd"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Rackd is down"
          description: "Rackd instance {{ $labels.instance }} is down"

      - alert: RackdHighErrorRate
        expr: rate(http_requests_by_code{code=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate in Rackd"
          description: "Error rate is {{ $value | humanizePercentage }}"

      - alert: RackdHighMemory
        expr: go_memory_alloc_bytes > 1e9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Rackd high memory usage"
          description: "Memory usage is {{ $value | humanize }}B"

      - alert: RackdDatabaseUnhealthy
        expr: up{job="rackd"} == 1 and probe_success{job="rackd",probe="readyz"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Rackd database is unhealthy"
          description: "Readiness probe is failing"
```

## Logging

Rackd uses structured logging with the following levels:

- `TRACE` - Very detailed debugging information
- `DEBUG` - Debugging information (HTTP requests, queries)
- `INFO` - Informational messages (startup, shutdown)
- `WARN` - Warning messages (missing auth tokens)
- `ERROR` - Error messages (failed operations)
- `FATAL` - Fatal errors (application exits)

### Log Configuration

Set log level via environment variable:

```bash
export LOG_LEVEL=debug
./rackd server
```

Available levels: `trace`, `debug`, `info`, `warn`, `error`, `fatal`

### Log Format

Logs are output in key-value format:

```
2026-02-03T12:00:00Z INFO Starting server addr=:8080
2026-02-03T12:00:01Z DEBUG HTTP request started method=GET path=/api/devices
2026-02-03T12:00:01Z DEBUG HTTP request completed method=GET path=/api/devices status=200 duration_ms=45
```

### HTTP Request Logging

All HTTP requests are logged with:
- Method
- Path
- Query parameters
- Remote address
- Status code
- Duration

Example:
```
DEBUG HTTP request started method=GET path=/api/devices query=datacenter_id=123 remote_addr=192.168.1.100:54321
DEBUG HTTP request completed method=GET path=/api/devices status=200 duration_ms=45
```

## Monitoring Best Practices

1. **Set up health checks** in your orchestration platform (Kubernetes, Nomad, Docker Swarm)
2. **Configure Prometheus** to scrape metrics every 30 seconds
3. **Create Grafana dashboards** for visualization
4. **Set up alerts** for critical conditions (downtime, high error rate, database issues)
5. **Monitor resource usage** (memory, goroutines, database connections)
6. **Track application metrics** (device count, scan frequency)
7. **Enable debug logging** during troubleshooting, but use `info` level in production

## Integration Examples

### Docker Compose with Prometheus

```yaml
version: '3.8'
services:
  rackd:
    image: rackd:latest
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rackd
spec:
  replicas: 2
  selector:
    matchLabels:
      app: rackd
  template:
    metadata:
      labels:
        app: rackd
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: rackd
        image: rackd:latest
        ports:
        - containerPort: 8080
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

## Troubleshooting

### Metrics not appearing

1. Check that `/metrics` endpoint is accessible:
   ```bash
   curl http://localhost:8080/metrics
   ```

2. Verify Prometheus is scraping:
   ```bash
   # Check Prometheus targets
   curl http://localhost:9090/api/v1/targets
   ```

### Health check failing

1. Check readiness endpoint:
   ```bash
   curl -v http://localhost:8080/readyz
   ```

2. Look for specific check failures in the response

3. Check logs for database or scheduler issues:
   ```bash
   export LOG_LEVEL=debug
   ./rackd server
   ```

### High memory usage

1. Check current memory metrics:
   ```bash
   curl http://localhost:8080/metrics | grep go_memory
   ```

2. Monitor goroutine count:
   ```bash
   curl http://localhost:8080/metrics | grep go_goroutines
   ```

3. Consider increasing memory limits or investigating memory leaks
