# Monitoring and Observability Specification

This document defines the approach to monitoring, logging, and tracing for Rackd, ensuring operational visibility and efficient troubleshooting for both OSS and Enterprise editions.

## 1. Core Principles

- **Visibility**: Provide clear insight into the application's health, performance, and behavior.
- **Troubleshooting**: Enable quick identification and diagnosis of issues.
- **Proactive Alerting**: Detect and notify operators of potential problems before they impact users.
- **Efficiency**: Minimize overhead on the application while maximizing information gained.

## 2. Logging

### 2.1. Structured Logging

- **Mechanism**: All application logs will use `paularlott/logger` (JSON format recommended for production).
- **Content**: Log entries should include:
    - `timestamp`: UTC ISO 8601 format.
    - `level`: (debug, info, warn, error, fatal).
    - `message`: Human-readable description of the event.
    - `component`: e.g., `api`, `storage`, `discovery`, `worker`.
    - `trace_id`: (if applicable, for distributed tracing).
    - `span_id`: (if applicable).
    - `error`: (if log level is warn/error/fatal, include error details).
    - `user_id`: (if an authenticated user context is available).
    - `request_id`: (for API requests).
    - `resource_id`: (e.g., `device_id`, `network_id` when performing CRUD operations).
- **Sensitive Data**: **MUST NOT** log sensitive information (e.g., passwords, API tokens, PII). Redaction should be implemented where necessary.

### 2.2. Log Levels

- `TRACE`: Very fine-grained diagnostic information, typically for development.
- `DEBUG`: Detailed information on internal execution flow.
- `INFO`: General operational messages, significant events.
- `WARN`: Potentially problematic situations, but not an error.
- `ERROR`: Runtime errors or unexpected conditions.
- `FATAL`: Critical errors causing application termination.

## 3. Metrics

### 3.1. Standard Metrics (OSS)

The application will expose core operational metrics via a `/metrics` HTTP endpoint in Prometheus text format.

**Key Metrics to Expose:**

- **HTTP Request Metrics:**
    - `http_requests_total`: Counter for total HTTP requests, labeled by `method`, `path`, `status_code`.
    - `http_request_duration_seconds`: Histogram for HTTP request durations, labeled by `method`, `path`.
- **Go Runtime Metrics:**
    - `go_memstats_alloc_bytes`, `go_memstats_sys_bytes`, etc.
    - `go_goroutines`: Number of current goroutines.
    - `go_gc_duration_seconds`: Garbage collection duration.
- **Application-Specific Counters:**
    - `rackd_device_total`: Gauge for total number of devices.
    - `rackd_network_total`: Gauge for total number of networks.
    - `rackd_discovery_scans_total`: Counter for initiated discovery scans.
    - `rackd_discovery_hosts_found_total`: Counter for hosts found during discovery.
    - `rackd_storage_errors_total`: Counter for storage operation errors, labeled by `operation`, `entity`.
- **Discovery Metrics:**
    - `rackd_discovery_scan_duration_seconds`: Histogram for discovery scan durations.
    - `rackd_discovery_scan_status`: Gauge for the status of the last scan (e.g., 0=pending, 1=running, 2=completed, 3=failed).

### 3.2. Monitoring Backend (Enterprise)

- **Mechanism**: `MonitoringBackend` interface as defined in `03-feature-matrix.md`.
- **Purpose**: Allows integration with external monitoring systems (e.g., Datadog, Prometheus Pushgateway, OpenTelemetry collectors) for advanced metrics collection and custom event reporting.

## 4. Health Checks

### 4.1. Liveness Probe (`/healthz`)

- **Purpose**: Indicates if the application is still running.
- **Response**: Returns `200 OK` if the process is alive.

### 4.2. Readiness Probe (`/readyz`)

- **Purpose**: Indicates if the application is ready to serve traffic.
- **Response**: Returns `200 OK` if:
    - The process is alive.
    - The database connection is healthy.
    - Essential configuration is loaded.
    - (Optional) Critical background workers (e.g., discovery scheduler) are operational.

## 5. Alerting

- **Integration**: Leverage monitoring systems (e.g., Prometheus Alertmanager, Grafana) to define alerts based on the exposed metrics and log patterns.
- **Key Alerting Areas**:
    - High error rates (`http_requests_total` with `status_code >= 500`).
    - API latency exceeding thresholds (`http_request_duration_seconds`).
    - Service unavailability (health check failures).
    - Database connectivity issues.
    - Critical log messages (e.g., `FATAL` level logs).
    - Discovery scan failures.

## 6. Tracing (Future Consideration)

- **Purpose**: For complex environments, distributed tracing (e.g., OpenTelemetry, Jaeger) can provide end-to-end visibility across services.
- **Mechanism**: Inject `trace_id` and `span_id` into log contexts and HTTP headers.
- **Initial Scope**: Not part of initial OSS scope, but interfaces should allow for future integration.
