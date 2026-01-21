# Error Handling and Recovery

This document defines the comprehensive error handling strategy for Rackd, including standardized error responses, retry policies, circuit breakers, and graceful degradation.

## 1. Standardized Error Format

### 1.1 Error Response Structure

All API errors follow a consistent JSON format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {
    "field": "Additional context or validation details"
  },
  "request_id": "unique-request-id-for-tracing"
}
```

### 1.2 HTTP Status Code Mapping

| HTTP Status | Error Code Prefix | Description |
|-------------|-----------------|-------------|
| 400 | `INVALID_` | Bad request, validation errors |
| 401 | `UNAUTHORIZED` | Authentication failed or missing |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT_` | Resource conflicts (duplicates, IP conflicts) |
| 429 | `RATE_LIMITED` | Rate limit exceeded |
| 500 | `INTERNAL_ERROR` | Server-side errors |
| 503 | `SERVICE_UNAVAILABLE` | Service unavailable (degraded mode) |

### 1.3 Error Code Reference

| Error Code | HTTP Status | Description | Retryable |
|-----------|-------------|-------------|------------|
| `DEVICE_NOT_FOUND` | 404 | Device with specified ID does not exist | No |
| `DATACENTER_NOT_FOUND` | 404 | Datacenter with specified ID does not exist | No |
| `NETWORK_NOT_FOUND` | 404 | Network with specified ID does not exist | No |
| `POOL_NOT_FOUND` | 404 | Network pool with specified ID does not exist | No |
| `DISCOVERY_NOT_FOUND` | 404 | Discovery scan or device not found | No |
| `INVALID_ID` | 400 | Provided ID is invalid format | No |
| `INVALID_CIDR` | 400 | Invalid CIDR notation for subnet | No |
| `INVALID_IP_ADDRESS` | 400 | Invalid IP address format | No |
| `INVALID_INPUT` | 400 | General input validation error | No |
| `MISSING_REQUIRED_FIELD` | 400 | Required field is missing | No |
| `INVALID_DEVICE_NAME` | 400 | Device name doesn't meet requirements | No |
| `INVALID_VLAN_ID` | 400 | VLAN ID out of valid range (0-4095) | No |
| `CONFLICT_DEVICE_NAME` | 409 | Device with this name already exists | No |
| `IP_NOT_AVAILABLE` | 409 | No available IP addresses in pool | No |
| `IP_CONFLICT` | 409 | IP address already in use | No |
| `NETWORK_IN_USE` | 409 | Cannot delete network with assigned devices | No |
| `DATACENTER_IN_USE` | 409 | Cannot delete datacenter with assigned devices | No |
| `UNAUTHORIZED` | 401 | Invalid or missing authentication token | No |
| `FORBIDDEN` | 403 | User lacks permission for this action | No |
| `RATE_LIMITED` | 429 | Request rate limit exceeded | Yes |
| `DATABASE_LOCKED` | 503 | Database is locked (SQLite) | Yes |
| `DATABASE_BUSY` | 503 | Database is busy with other operations | Yes |
| `DATABASE_ERROR` | 500 | Generic database operation failed | No |
| `DISCOVERY_SCAN_FAILED` | 500 | Discovery scan encountered error | No |
| `NETWORK_TIMEOUT` | 503 | Network operation timeout | Yes |
| `INTERNAL_ERROR` | 500 | Unexpected server error | No |
| `SERVICE_DEGRADED` | 503 | Operating in degraded mode | No |
| `MIGRATION_PENDING` | 503 | Database migrations need to be applied | No |
| `BACKUP_IN_PROGRESS` | 503 | Backup operation in progress | No |
| `RESTORE_IN_PROGRESS` | 503 | Restore operation in progress | No |

---

## 2. Retry Policies

### 2.1 Retry Strategy Overview

Rackd implements an exponential backoff with jitter algorithm for retryable operations. This prevents thundering herd problems and reduces server load during transient failures.

### 2.2 Retryable vs Non-Retryable Errors

**Retryable Errors:**
- `RATE_LIMITED` (429) - Wait for rate limit reset
- `DATABASE_LOCKED` (503) - Wait for lock release
- `DATABASE_BUSY` (503) - Wait for database availability
- `NETWORK_TIMEOUT` (503) - Retry on network timeouts

**Non-Retryable Errors:**
- All 4xx errors (except 429)
- `DATABASE_ERROR` (500) - Indicates data issues
- `INTERNAL_ERROR` (500) - Application logic errors
- `DISCOVERY_SCAN_FAILED` (500) - Scan-specific errors

### 2.3 Exponential Backoff with Jitter

**Algorithm:**

```go
package retry

import (
    "math/rand"
    "time"
)

type RetryConfig struct {
    MaxAttempts    int           // Maximum number of retry attempts
    BaseDelay      time.Duration // Initial delay
    MaxDelay       time.Duration // Maximum delay
    JitterFactor  float64       // Jitter factor (0.0-1.0)
}

func CalculateBackoff(attempt int, config RetryConfig) time.Duration {
    // Exponential backoff: delay = baseDelay * 2^attempt
    exponentialDelay := time.Duration(float64(config.BaseDelay) * math.Pow(2, float64(attempt)))

    // Apply jitter to prevent synchronized retries
    jitter := rand.Float64() * config.JitterFactor * float64(exponentialDelay)

    // Cap at max delay
    totalDelay := time.Duration(float64(exponentialDelay) + jitter)
    if totalDelay > config.MaxDelay {
        return config.MaxDelay
    }

    return totalDelay
}

// Example usage:
func RetryableOperation(fn func() error, config RetryConfig) error {
    var lastErr error
    for attempt := 0; attempt < config.MaxAttempts; attempt++ {
        err := fn()
        if err == nil {
            return nil
        }

        lastErr = err
        if !IsRetryableError(err) {
            break
        }

        delay := CalculateBackoff(attempt, config)
        time.Sleep(delay)
    }
    return lastErr
}
```

**Configuration:**

| Setting | Default Value | Description |
|---------|---------------|-------------|
| `MaxAttempts` | 5 | Maximum retry attempts |
| `BaseDelay` | 100ms | Initial delay |
| `MaxDelay` | 30s | Maximum delay between retries |
| `JitterFactor` | 0.25 | Jitter factor (25% variation) |

### 2.4 Idempotent Operation Requirements

Operations that are retried must be idempotent:

- **GET** requests - inherently idempotent
- **PUT** requests with stable state transitions
- **DELETE** requests - can be retried safely
- **POST** requests must implement idempotency keys for retry

**Idempotency Key Pattern:**

```go
func WithIdempotency(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        idempotencyKey := r.Header.Get("Idempotency-Key")
        if idempotencyKey == "" {
            next(w, r)
            return
        }

        // Check if request was already processed
        cached := checkIdempotencyCache(idempotencyKey)
        if cached != nil {
            w.Header().Set("Idempotency-Replayed", "true")
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode(cached)
            return
        }

        // Process request and cache response
        responseRecorder := &responseRecorder{}
        next(responseRecorder, r)

        cacheIdempotency(idempotencyKey, responseRecorder.Body, responseRecorder.StatusCode)
    }
}
```

---

## 3. Circuit Breaker Pattern

### 3.1 Circuit Breaker State Machine

```
    ┌─────────────────────────────────────────┐
    │                                     │
    ▼                                     │
┌─────────────────┐  Success Threshold     │
│      CLOSED      │<──────────────────────┤
└────────┬────────┘                      │
         │                              │
         │ Failure Count > Threshold      │
         │                              │
         ▼                              │
┌─────────────────┐  Recovery Timeout     │
│      OPEN       │──────────────────────┤
└────────┬────────┘                      │
         │                              │
         │ Recovery Timeout Elapsed        │
         │                              │
         ▼                              │
┌─────────────────┐  Success            │
│   HALF-OPEN    │──────────────────────┤
└────────┬────────┘                      │
         │                              │
         │ Success -> CLOSED              │
         │ Failure -> OPEN               │
         └──────────────────────────────┘
```

### 3.2 Circuit Breaker Implementation

```go
package circuitbreaker

import (
    "sync"
    "time"
)

type State string

const (
    StateClosed   State = "closed"
    StateOpen     State = "open"
    StateHalfOpen State = "half_open"
)

type CircuitBreaker struct {
    mu                sync.RWMutex
    state             State
    failureCount      int
    successCount      int
    failureThreshold  int
    successThreshold  int
    recoveryTimeout   time.Duration
    lastFailureTime  time.Time
    lastSuccessTime  time.Time
}

type Config struct {
    FailureThreshold int           // Failures before opening
    SuccessThreshold int           // Successes to close
    RecoveryTimeout  time.Duration // Time in OPEN before HALF-OPEN
}

func NewCircuitBreaker(config Config) *CircuitBreaker {
    return &CircuitBreaker{
        state:            StateClosed,
        failureThreshold: config.FailureThreshold,
        successThreshold: config.SuccessThreshold,
        recoveryTimeout:  config.RecoveryTimeout,
    }
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
    if !cb.AllowRequest() {
        return &CircuitOpenError{
            LastFailureTime: cb.lastFailureTime,
        }
    }

    err := fn()
    cb.RecordResult(err)
    return err
}

func (cb *CircuitBreaker) AllowRequest() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    switch cb.state {
    case StateClosed:
        return true
    case StateOpen:
        if time.Since(cb.lastFailureTime) > cb.recoveryTimeout {
            return true
        }
        return false
    case StateHalfOpen:
        return true
    default:
        return false
    }
}

func (cb *CircuitBreaker) RecordResult(err error) {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    if err == nil {
        cb.onSuccess()
    } else {
        cb.onFailure()
    }
}

func (cb *CircuitBreaker) onSuccess() {
    cb.successCount++
    cb.lastSuccessTime = time.Now()

    switch cb.state {
    case StateClosed:
        // Reset on success
        cb.failureCount = 0
    case StateHalfOpen:
        // Close circuit if threshold reached
        if cb.successCount >= cb.successThreshold {
            cb.state = StateClosed
            cb.successCount = 0
            cb.failureCount = 0
        }
    }
}

func (cb *CircuitBreaker) onFailure() {
    cb.failureCount++
    cb.lastFailureTime = time.Now()

    switch cb.state {
    case StateClosed:
        // Open circuit if threshold exceeded
        if cb.failureCount >= cb.failureThreshold {
            cb.state = StateOpen
        }
    case StateHalfOpen:
        // Reopen circuit on failure
        cb.state = StateOpen
        cb.successCount = 0
    }
}
```

### 3.3 Threshold Configuration

| Component | Failure Threshold | Success Threshold | Recovery Timeout |
|-----------|-------------------|-------------------|------------------|
| Storage operations | 5 | 3 | 30s |
| API handlers | 10 | 5 | 60s |
| Discovery scanner | 3 | 2 | 5min |
| External API calls | 5 | 3 | 1min |

### 3.4 Recovery Strategies

**HALF-OPEN Testing:**

```go
func (cb *CircuitBreaker) AllowRequest() bool {
    cb.mu.RLock()
    defer cb.mu.RUnlock()

    if cb.state == StateHalfOpen {
        // Allow only one request to test if circuit is healthy
        if cb.successCount == 0 {
            return true
        }
        return false
    }
    // ... rest of logic
}
```

---

## 4. Graceful Degradation

### 4.1 Feature Flags for Degraded Mode

```go
package degradation

type FeatureFlags struct {
    // Core features (never disabled)
    CoreDatabase     bool `json:"core_database"`
    CoreAPI          bool `json:"core_api"`

    // Optional features (can be disabled)
    DiscoveryEnabled bool `json:"discovery_enabled"`
    SearchEnabled   bool `json:"search_enabled"`
    MetricsEnabled  bool `json:"metrics_enabled"`

    // Resource-intensive features
    BackgroundJobs  bool `json:"background_jobs"`
    DeepScanning   bool `json:"deep_scanning"`
    FullTextSearch  bool `json:"full_text_search"`
}

type DegradationManager struct {
    flags        FeatureFlags
    monitor      *ResourceMonitor
    autoAdjust   bool
}

func (dm *DegradationManager) CheckResources() {
    cpu := dm.monitor.CPUUsage()
    memory := dm.monitor.MemoryUsage()
    disk := dm.monitor.DiskUsage()

    // Auto-adjust based on resource pressure
    if cpu > 90 || memory > 90 {
        dm.flags.DiscoveryEnabled = false
        dm.flags.DeepScanning = false
    }

    if memory > 95 {
        dm.flags.FullTextSearch = false
    }

    if disk > 98 {
        dm.flags.BackgroundJobs = false
    }
}
```

### 4.2 Fallback Mechanisms

**Database Fallback:**

```go
func WithDatabaseFallback(primary, fallback storage.ExtendedStorage) storage.ExtendedStorage {
    return &fallbackStorage{
        primary: primary,
        fallback: fallback,
    }
}

type fallbackStorage struct {
    primary   storage.ExtendedStorage
    fallback  storage.ExtendedStorage
    usePrimary bool
}

func (fs *fallbackStorage) GetDevice(id string) (*model.Device, error) {
    if fs.usePrimary {
        device, err := fs.primary.GetDevice(id)
        if err == nil {
            return device, nil
        }

        // Fall back to secondary on error
        log.Warn("Primary storage failed, using fallback", "error", err)
        return fs.fallback.GetDevice(id)
    }

    return fs.fallback.GetDevice(id)
}
```

**Search Fallback:**

```go
func (h *Handler) searchDevices(w http.ResponseWriter, r *http.Request) {
    query := r.URL.Query().Get("q")

    // Try full-text search first
    if flags.FullTextSearch {
        devices, err := h.storage.FullTextSearch(query)
        if err == nil {
            h.writeJSON(w, http.StatusOK, devices)
            return
        }
    }

    // Fallback to basic LIKE search
    devices, err := h.storage.SearchDevices(query)
    if err != nil {
        h.internalError(w, err)
        return
    }

    h.writeJSON(w, http.StatusOK, devices)
}
```

### 4.3 Degradation Response Headers

```go
func DegradationMiddleware(flags *FeatureFlags) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Add degradation headers
            if !flags.DiscoveryEnabled {
                w.Header().Set("X-Rackd-Feature-Discovery", "disabled")
            }
            if !flags.SearchEnabled {
                w.Header().Set("X-Rackd-Feature-Search", "disabled")
            }

            // Overall degradation status
            if flags.IsAnyDisabled() {
                w.Header().Set("X-Rackd-Status", "degraded")
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## 5. Error Handling Implementation

### 5.1 API Handler Error Handling

```go
package api

import (
    "errors"
    "net/http"
)

type APIError struct {
    Code    string                 `json:"code"`
    Message string                 `json:"error"`
    Details map[string]interface{} `json:"details,omitempty"`
    Status  int                    `json:"-"`
}

func (e *APIError) Error() string {
    return e.Message
}

func (e *APIError) WithDetails(key string, value interface{}) *APIError {
    if e.Details == nil {
        e.Details = make(map[string]interface{})
    }
    e.Details[key] = value
    return e
}

// Predefined errors
var (
    ErrDeviceNotFound = &APIError{
        Code:    "DEVICE_NOT_FOUND",
        Message: "Device not found",
        Status:  http.StatusNotFound,
    }
    ErrInvalidInput = &APIError{
        Code:    "INVALID_INPUT",
        Message: "Invalid input provided",
        Status:  http.StatusBadRequest,
    }
)

func (h *Handler) createDevice(w http.ResponseWriter, r *http.Request) {
    var device model.Device
    if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
        h.writeError(w, ErrInvalidInput.WithDetails("parse_error", err.Error()))
        return
    }

    // Validate device
    if device.Name == "" {
        h.writeError(w, &APIError{
            Code:    "MISSING_REQUIRED_FIELD",
            Message: "Device name is required",
            Status:  http.StatusBadRequest,
        }.WithDetails("field", "name"))
        return
    }

    // Create device
    if err := h.storage.CreateDevice(&device); err != nil {
        if errors.Is(err, storage.ErrDeviceNotFound) {
            h.writeError(w, ErrDeviceNotFound)
            return
        }
        h.internalError(w, err)
        return
    }

    h.writeJSON(w, http.StatusCreated, device)
}

func (h *Handler) writeError(w http.ResponseWriter, err *APIError) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(err.Status)
    json.NewEncoder(w).Encode(err)
}
```

### 5.2 Storage Operation Error Handling

```go
package storage

import (
    "database/sql"
    "errors"
)

func (s *SQLiteStorage) GetDevice(id string) (*model.Device, error) {
    device := &model.Device{}
    err := s.db.QueryRow(`
        SELECT id, name, description, make_model, os, datacenter_id,
               username, location, created_at, updated_at
        FROM devices WHERE id = ?
    `, id).Scan(&device.ID, &device.Name, &device.Description,
        &device.MakeModel, &device.OS, &device.DatacenterID,
        &device.Username, &device.Location, &device.CreatedAt, &device.UpdatedAt)

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrDeviceNotFound
        }
        if errors.Is(err, sql.ErrConnDone) {
            return nil, &StorageError{
                Code:    "DATABASE_ERROR",
                Message: "Database connection lost",
                Err:     err,
            }
        }
        return nil, err
    }

    // Load addresses
    device.Addresses, err = s.getDeviceAddresses(id)
    if err != nil {
        return nil, err
    }

    return device, nil
}

type StorageError struct {
    Code    string
    Message string
    Err     error
}

func (e *StorageError) Error() string {
    return e.Message
}

func (e *StorageError) Unwrap() error {
    return e.Err
}
```

### 5.3 Discovery Scanner Error Handling

```go
package discovery

func (s *DefaultScanner) runScan(ctx context.Context, scan *model.DiscoveryScan, network *model.Network) {
    defer func() {
        if r := recover(); r != nil {
            log.Error("Discovery scan panic", "error", r)
            scan.Status = model.ScanStatusFailed
            scan.ErrorMessage = "Scan encountered unexpected error"
            s.storage.UpdateDiscoveryScan(scan)
        }
    }()

    now := time.Now()
    scan.Status = model.ScanStatusRunning
    scan.StartedAt = &now
    s.storage.UpdateDiscoveryScan(scan)

    // Scan with error handling per host
    for i, ip := range ips {
        select {
        case <-ctx.Done():
            scan.Status = model.ScanStatusFailed
            scan.ErrorMessage = "Scan cancelled"
            s.storage.UpdateDiscoveryScan(scan)
            return
        default:
        }

        // Scan individual host with timeout
        device, err := s.scanHostWithTimeout(ctx, ip, scanType)
        if err != nil {
            log.Warn("Failed to scan host", "ip", ip, "error", err)
            // Continue scanning other hosts
        } else if device != nil {
            s.storeDiscoveredDevice(device)
        }

        // Update progress
        scan.ScannedHosts = i + 1
        scan.ProgressPercent = float64(scan.ScannedHosts) / float64(scan.TotalHosts) * 100
        s.storage.UpdateDiscoveryScan(scan)
    }

    // Mark completed
    completedAt := time.Now()
    scan.Status = model.ScanStatusCompleted
    scan.CompletedAt = &completedAt
    s.storage.UpdateDiscoveryScan(scan)
}

func (s *DefaultScanner) scanHostWithTimeout(ctx context.Context, ip string, scanType string) (*model.DiscoveredDevice, error) {
    ctx, cancel := context.WithTimeout(ctx, s.config.DiscoveryTimeout)
    defer cancel()

    resultChan := make(chan *model.DiscoveredDevice, 1)
    errChan := make(chan error, 1)

    go func() {
        device, err := s.discoverHost(ip, scanType)
        if err != nil {
            errChan <- err
        } else {
            resultChan <- device
        }
    }()

    select {
    case device := <-resultChan:
        return device, nil
    case err := <-errChan:
        return nil, err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

### 5.4 MCP Server Error Handling

```go
package mcp

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Authentication
    if s.bearerToken != "" {
        if !s.authenticate(r) {
            s.writeError(w, &MCPError{
                Code:    "UNAUTHORIZED",
                Message: "Invalid or missing authentication token",
            })
            return
        }
    }

    // Parse request
    var req MCPRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.writeError(w, &MCPError{
            Code:    "INVALID_REQUEST",
            Message: "Invalid JSON request",
            Details: map[string]interface{}{"error": err.Error()},
        })
        return
    }

    // Execute tool
    result, err := s.executeTool(req)
    if err != nil {
        s.writeError(w, err)
        return
    }

    // Write response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}

func (s *Server) executeTool(req MCPRequest) (interface{}, error) {
    tool, ok := s.mcpServer.GetTool(req.Tool)
    if !ok {
        return nil, &MCPError{
            Code:    "TOOL_NOT_FOUND",
            Message: fmt.Sprintf("Tool '%s' not found", req.Tool),
        }
    }

    result, err := tool.Execute(req.Params)
    if err != nil {
        return nil, &MCPError{
            Code:    "TOOL_EXECUTION_ERROR",
            Message: fmt.Sprintf("Tool execution failed: %v", err),
        }
    }

    return result, nil
}

type MCPError struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Details map[string]interface{} `json:"details,omitempty"`
}

func (e *MCPError) Error() string {
    return e.Message
}
```

---

## 6. Error Logging Best Practices

### 6.1 Structured Error Logging

```go
log.Error("Failed to create device",
    "device_id", device.ID,
    "device_name", device.Name,
    "error", err,
    "error_type", fmt.Sprintf("%T", err),
    "stack_trace", string(debug.Stack()),
)
```

### 6.2 Context Propagation

```go
func (h *Handler) withRequestContext(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }

        ctx := context.WithValue(r.Context(), "request_id", requestID)
        ctx = context.WithValue(ctx, "user_agent", r.Header.Get("User-Agent"))

        w.Header().Set("X-Request-ID", requestID)
        next(w, r.WithContext(ctx))
    }
}
```

### 6.3 Sensitive Data Redaction

```go
func sanitizeError(err error) error {
    errStr := err.Error()

    // Remove passwords
    re := regexp.MustCompile(`password["\s:=]+\S+`)
    errStr = re.ReplaceAllString(errStr, `password="***"`)

    // Remove tokens
    re = regexp.MustCompile(`token["\s:=]+\S+`)
    errStr = re.ReplaceAllString(errStr, `token="***"`)

    // Remove API keys
    re = regexp.MustCompile(`api[_-]?key["\s:=]+\S+`)
    errStr = re.ReplaceAllString(errStr, `api_key="***"`)

    return fmt.Errorf(errStr)
}
```

---

## 7. Error Monitoring

### 7.1 Error Rate Metrics

```go
package metrics

var (
    errorCount = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "rackd_errors_total",
        Help: "Total number of errors",
    }, []string{"code", "component", "severity"})

    errorRate = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "rackd_error_rate",
        Help: "Error rate per minute",
    }, []string{"code"})
)

func RecordError(code, component string, severity string) {
    errorCount.WithLabelValues(code, component, severity).Inc()
}

func UpdateErrorRates() {
    // Calculate error rates per minute
    // Update gauge metrics
}
```

### 7.2 Alerting Thresholds

| Metric | Threshold | Duration | Alert Level |
|---------|------------|-----------|-------------|
| Error rate (5xx) | > 5% | 5min | Warning |
| Error rate (5xx) | > 10% | 5min | Critical |
| Circuit breaker opens | Any | - | Warning |
| Storage errors | > 10/min | 5min | Warning |
| Discovery failures | > 5/scans | 30min | Warning |

### 7.3 Dashboard Visualization

Create dashboard panels for:
- Error rate over time (by error code)
- Error distribution by component
- Circuit breaker state transitions
- Top error messages (grouped by message)
- Error rate by endpoint
- Retry attempt distribution

---

## 8. CLI Error Handling

### 8.1 Exit Codes

| Exit Code | Meaning | Usage |
|-----------|----------|-------|
| 0 | Success | Command completed successfully |
| 1 | Generic Error | Unclassified error occurred |
| 2 | Invalid Usage | Wrong arguments, invalid flags |
| 3 | Network Error | Network connectivity issues |
| 4 | Authentication Error | Failed authentication |
| 5 | Server Error | Server returned 5xx error |

### 8.2 Error Display

```go
func (c *CLI) displayError(err error) {
    var apiErr *client.APIError
    if errors.As(err, &apiErr) {
        fmt.Fprintf(os.Stderr, "Error: %s (code: %s)\n", apiErr.Message, apiErr.Code)
        if apiErr.Details != nil {
            fmt.Fprintf(os.Stderr, "Details: %v\n", apiErr.Details)
        }
        os.Exit(5)
    }

    var netErr *net.OpError
    if errors.As(err, &netErr) {
        fmt.Fprintf(os.Stderr, "Network error: %v\n", netErr)
        os.Exit(3)
    }

    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```
