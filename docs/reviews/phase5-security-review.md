# Phase 5 Security Review: Discovery System

**Date:** 2026-01-21  
**Reviewer:** AI Security Review  
**Status:** PASSED with recommendations  
**Files Reviewed:**
- `internal/discovery/interfaces.go`
- `internal/discovery/scanner.go`
- `internal/discovery/scanner_test.go`
- `internal/worker/scheduler.go`
- `internal/worker/scheduler_test.go`

---

## Summary

Phase 5 implements the network discovery system including a scanner for CIDR-based host discovery and a background scheduler for automated scans. The implementation handles concurrent network operations with proper resource management.

**Overall Assessment:** PASSED

---

## Findings

### HIGH Priority

None identified.

### MEDIUM Priority

#### SEC-P5-001: No Subnet Size Limit for Scans

**Location:** `scanner.go:32-57` (`Scan` method)

**Status:** ✓ RESOLVED

**Fix Applied:** Added `MaxSubnetBits` constant (16) and `ErrSubnetTooLarge` error. Scan now rejects subnets larger than /16:
```go
ones, bits := ipNet.Mask.Size()
if bits-ones > MaxSubnetBits {
    return nil, ErrSubnetTooLarge
}
```

---

#### SEC-P5-002: Unbounded In-Memory Scan Cache

**Location:** `scanner.go:17-22` (`DefaultScanner` struct)

**Status:** ✓ RESOLVED

**Fix Applied:** Added `cleanupCompletedScans()` method that removes completed/failed scans older than 1 hour. Called automatically after each scan completes.

---

### LOW Priority

#### SEC-P5-003: Network Scanning May Trigger Security Alerts

**Location:** `scanner.go:137-147` (`isHostAlive`), `scanner.go:174-193` (`scanPorts`)

**Issue:** TCP connection attempts to multiple ports can trigger intrusion detection systems (IDS) or firewall alerts on the target network. This is expected behavior for a discovery tool but should be documented.

**Recommendation:** Add documentation warning users that:
- Discovery scans generate network traffic that may be flagged by security tools
- Users should coordinate with network security teams before enabling discovery
- Consider adding a "stealth mode" option with longer delays between probes

---

#### SEC-P5-004: Reverse DNS Lookup Information Disclosure

**Location:** `scanner.go:163-166`
```go
names, err := net.LookupAddr(ip)
if err == nil && len(names) > 0 {
    device.Hostname = names[0]
}
```

**Issue:** Reverse DNS lookups are performed against the system's configured DNS servers. In some environments, this could leak information about which IPs are being scanned to external DNS providers.

**Recommendation:** Consider making DNS lookups configurable or using internal DNS only:
```go
// Add to config
DiscoveryUseDNS bool `env:"DISCOVERY_USE_DNS" default:"true"`
```

---

#### SEC-P5-005: No Rate Limiting Between Hosts

**Location:** `scanner.go:69-135` (`runScan`)

**Issue:** While concurrency is limited via semaphore, there's no delay between connection attempts. Rapid scanning could overwhelm network devices or trigger rate limiting on target hosts.

**Recommendation:** Add configurable delay between scan attempts:
```go
// In config
DiscoveryScanDelay time.Duration `env:"DISCOVERY_SCAN_DELAY" default:"0"`

// In runScan, after each host
if s.config.DiscoveryScanDelay > 0 {
    time.Sleep(s.config.DiscoveryScanDelay)
}
```

---

#### SEC-P5-006: Scheduler Runs All Rules Sequentially

**Location:** `scheduler.go:85-116` (`runScheduledScans`)

**Issue:** All enabled discovery rules are executed sequentially in a single goroutine. If one scan hangs or takes very long, subsequent scans are delayed.

**Recommendation:** Consider running scans in parallel with a limit, or adding per-scan timeouts:
```go
// Add scan timeout
scanCtx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
defer cancel()
_, err = s.scanner.Scan(scanCtx, network, rule.ScanType)
```

---

## Positive Findings

### Concurrency Control ✓

- Semaphore pattern correctly limits concurrent goroutines (`DiscoveryMaxConcurrent`)
- Proper mutex usage for shared state (`scans` map, progress counters)
- WaitGroup ensures all goroutines complete before marking scan done

### Context Cancellation ✓

- Scan respects context cancellation for graceful shutdown
- Scheduler properly propagates cancellation to running scans
- Clean shutdown via `Stop()` method with WaitGroup

### Input Validation ✓

- CIDR parsing validates network format before scanning
- Invalid CIDR returns error immediately
- Scan type defaults handled appropriately

### Resource Cleanup ✓

- TCP connections are properly closed after use
- Scheduler ticker is stopped on shutdown
- Discovery cleanup job removes old records

### Test Coverage ✓

- Scanner tests cover CIDR parsing, IP enumeration, cancellation
- Scheduler tests verify start/stop lifecycle, rule filtering
- Mock scanner used for scheduler tests (good isolation)

### No SQL Injection Risk ✓

- All database operations use parameterized queries via storage layer
- No raw SQL construction in discovery code

---

## Compliance with Security Spec

| Requirement | Status | Notes |
|-------------|--------|-------|
| 4. Input Validation | ✓ | CIDR validated, scan types checked |
| 5. Secret Management | N/A | No secrets handled in discovery |
| 7. Logging | ✓ | Scan start/completion logged |
| 9. Error Handling | ✓ | Errors logged, not exposed to users |

---

## Recommendations Summary

| ID | Priority | Issue | Status |
|----|----------|-------|--------|
| SEC-P5-001 | MEDIUM | Add subnet size limit | ✓ RESOLVED |
| SEC-P5-002 | MEDIUM | Implement scan cache eviction | ✓ RESOLVED |
| SEC-P5-003 | LOW | Document IDS/firewall implications | Open |
| SEC-P5-004 | LOW | Make DNS lookups configurable | Open |
| SEC-P5-005 | LOW | Add scan delay option | Open |
| SEC-P5-006 | LOW | Add per-scan timeout | Open |

---

## Conclusion

The Phase 5 discovery system implementation is secure for its intended use case as an internal network discovery tool. The concurrency controls and context cancellation are well-implemented. Both medium-priority findings have been resolved:
- SEC-P5-001: Subnet size now limited to /16 maximum
- SEC-P5-002: Scan cache now auto-evicts completed scans after 1 hour

**Result:** PASSED
