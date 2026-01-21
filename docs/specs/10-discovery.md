# Discovery System

This document covers the network discovery implementation including the scanner and scheduler.

## Scanner Interface

```go
// internal/discovery/scanner.go
package discovery

// Scanner interface for network discovery
type Scanner interface {
    Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error)
    GetScanStatus(scanID string) (*model.DiscoveryScan, error)
}
```

## Default Scanner Implementation

**File**: `internal/discovery/scanner.go`

```go
package discovery

import (
    "context"
    "net"
    "sync"
    "time"

    "github.com/martinsuchenak/rackd/internal/config"
    "github.com/martinsuchenak/rackd/internal/log"
    "github.com/martinsuchenak/rackd/internal/model"
    "github.com/martinsuchenak/rackd/internal/storage"
)

type DefaultScanner struct {
    storage storage.DiscoveryStorage
    config  *config.Config
    scans   map[string]*model.DiscoveryScan
    mu      sync.RWMutex
}

func NewScanner(store storage.DiscoveryStorage, cfg *config.Config) *DefaultScanner {
    return &DefaultScanner{
        storage: store,
        config:  cfg,
        scans:   make(map[string]*model.DiscoveryScan),
    }
}

func (s *DefaultScanner) Scan(ctx context.Context, network *model.Network, scanType string) (*model.DiscoveryScan, error) {
    // Parse CIDR
    _, ipNet, err := net.ParseCIDR(network.Subnet)
    if err != nil {
        return nil, err
    }

    // Create scan record
    scan := &model.DiscoveryScan{
        NetworkID:  network.ID,
        Status:     model.ScanStatusPending,
        ScanType:   scanType,
        TotalHosts: countHosts(ipNet),
    }

    if err := s.storage.CreateDiscoveryScan(scan); err != nil {
        return nil, err
    }

    // Start scan in background
    go s.runScan(ctx, scan, network, ipNet, scanType)

    return scan, nil
}

func (s *DefaultScanner) runScan(ctx context.Context, scan *model.DiscoveryScan, network *model.Network, ipNet *net.IPNet, scanType string) {
    now := time.Now()
    scan.Status = model.ScanStatusRunning
    scan.StartedAt = &now
    s.storage.UpdateDiscoveryScan(scan)

    // Get all IPs in subnet
    ips := expandCIDR(ipNet)
    scan.TotalHosts = len(ips)

    // Scan with concurrency limit
    semaphore := make(chan struct{}, s.config.DiscoveryMaxConcurrent)
    var wg sync.WaitGroup
    var foundCount int
    var mu sync.Mutex

    for i, ip := range ips {
        select {
        case <-ctx.Done():
            scan.Status = model.ScanStatusFailed
            scan.ErrorMessage = "scan cancelled"
            s.storage.UpdateDiscoveryScan(scan)
            return
        default:
        }

        wg.Add(1)
        semaphore <- struct{}{}

        go func(ip string, index int) {
            defer wg.Done()
            defer func() { <-semaphore }()

            if s.isHostAlive(ip) {
                device := s.discoverHost(ip, network.ID, scanType)
                if device != nil {
                    // Check if already exists
                    existing, _ := s.storage.GetDiscoveredDeviceByIP(network.ID, ip)
                    if existing != nil {
                        device.ID = existing.ID
                        device.FirstSeen = existing.FirstSeen
                        s.storage.UpdateDiscoveredDevice(device)
                    } else {
                        s.storage.CreateDiscoveredDevice(device)
                    }

                    mu.Lock()
                    foundCount++
                    mu.Unlock()
                }
            }

            // Update progress
            mu.Lock()
            scan.ScannedHosts = index + 1
            scan.FoundHosts = foundCount
            scan.ProgressPercent = float64(scan.ScannedHosts) / float64(scan.TotalHosts) * 100
            mu.Unlock()
            s.storage.UpdateDiscoveryScan(scan)
        }(ip, i)
    }

    wg.Wait()

    // Mark completed
    completedAt := time.Now()
    scan.Status = model.ScanStatusCompleted
    scan.CompletedAt = &completedAt
    s.storage.UpdateDiscoveryScan(scan)

    log.Info("Discovery scan completed",
        "network", network.Name,
        "found", scan.FoundHosts,
        "scanned", scan.ScannedHosts,
    )
}

func (s *DefaultScanner) isHostAlive(ip string) bool {
    // TCP ping on common ports
    ports := []int{22, 80, 443, 3389}
    for _, port := range ports {
        conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), s.config.DiscoveryTimeout)
        if err == nil {
            conn.Close()
            return true
        }
    }
    return false
}

func (s *DefaultScanner) discoverHost(ip string, networkID string, scanType string) *model.DiscoveredDevice {
    now := time.Now()
    device := &model.DiscoveredDevice{
        IP:        ip,
        NetworkID: networkID,
        Status:    "online",
        FirstSeen: now,
        LastSeen:  now,
    }

    // Reverse DNS lookup
    names, err := net.LookupAddr(ip)
    if err == nil && len(names) > 0 {
        device.Hostname = names[0]
    }

    // Port scan for full/deep scans
    if scanType != model.ScanTypeQuick {
        device.OpenPorts = s.scanPorts(ip, scanType)
    }

    return device
}

func (s *DefaultScanner) scanPorts(ip string, scanType string) []int {
    var ports []int
    var portsToScan []int

    if scanType == model.ScanTypeFull {
        // Common ports
        portsToScan = []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 8080}
    } else {
        // Deep scan - top 1000 ports
        portsToScan = getTop1000Ports()
    }

    for _, port := range portsToScan {
        conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Second)
        if err == nil {
            conn.Close()
            ports = append(ports, port)
        }
    }

    return ports
}

// Helper functions
func countHosts(ipNet *net.IPNet) int {
    ones, bits := ipNet.Mask.Size()
    return 1 << (bits - ones)
}

func expandCIDR(ipNet *net.IPNet) []string {
    var ips []string
    for ip := ipNet.IP.Mask(ipNet.Mask); ipNet.Contains(ip); incrementIP(ip) {
        ips = append(ips, ip.String())
    }
    // Remove network and broadcast addresses
    if len(ips) > 2 {
        ips = ips[1 : len(ips)-1]
    }
    return ips
}

func incrementIP(ip net.IP) {
    for i := len(ip) - 1; i >= 0; i-- {
        ip[i]++
        if ip[i] > 0 {
            break
        }
    }
}
```

## Scheduler Implementation

**File**: `internal/worker/scheduler.go`

```go
package worker

import (
    "context"
    "sync"
    "time"

    "github.com/martinsuchenak/rackd/internal/config"
    "github.com/martinsuchenak/rackd/internal/discovery"
    "github.com/martinsuchenak/rackd/internal/log"
    "github.com/martinsuchenak/rackd/internal/storage"
)

type Scheduler struct {
    storage  storage.ExtendedStorage
    scanner  discovery.Scanner
    config   *config.Config
    ctx      context.Context
    cancel   context.CancelFunc
    wg       sync.WaitGroup
    running  bool
    mu       sync.Mutex
}

func NewScheduler(store storage.ExtendedStorage, scanner discovery.Scanner, cfg *config.Config) *Scheduler {
    ctx, cancel := context.WithCancel(context.Background())
    return &Scheduler{
        storage: store,
        scanner: scanner,
        config:  cfg,
        ctx:     ctx,
        cancel:  cancel,
    }
}

// ===== Worker Directory Structure =====

**File**: `internal/worker/scheduler.go`

The worker package manages background job scheduling for discovery and other periodic tasks.

```text
internal/worker/
├── scheduler.go          # Background job scheduler
├── jobs.go              # Job definitions and execution
└── worker.go            # Generic worker interface

Job Types:
- Discovery Scan: Run network discovery scans on schedule
- Cleanup Jobs: Remove old discovered devices
- Maintenance Jobs: Update network utilization stats
```

**File**: `internal/worker/jobs.go`

```go
package worker

import (
    "context"
    "time"

    "github.com/martinsuchenak/rackd/internal/config"
    "github.com/martinsuchenak/rackd/internal/storage"
)

// Job represents a background job
type Job struct {
    ID          string
    Name        string
    Schedule    func(*Scheduler) (time.Time, error)
    Execute     func(context.Context, *Scheduler) error
    NextRun     time.Time
    LastRun     *time.Time
}

// JobRunner executes background jobs
type JobRunner struct {
    jobs   map[string]*Job
    storage storage.ExtendedStorage
    config  *config.Config
    mu      sync.Mutex
}

func NewJobRunner(store storage.ExtendedStorage, cfg *config.Config) *JobRunner {
    runner := &JobRunner{
        jobs:    make(map[string]*Job),
        storage: store,
        config:  cfg,
    }

    // Register default jobs
    runner.registerDiscoveryJob()
    runner.registerCleanupJob()

    return runner
}

func (jr *JobRunner) registerDiscoveryJob() {
    jr.mu.Lock()
    defer jr.mu.Unlock()

    jr.jobs["discovery_scan"] = &Job{
        ID:   "discovery_scan",
        Name: "Discovery Scan",
        Schedule: func(s *Scheduler) (time.Time, error) {
            // Schedule next scan based on configured interval
            rules, err := s.storage.ListDiscoveryRules()
            if err != nil {
                return time.Time{}, err
            }

            // Find the earliest next run time from all rules
            var nextRun time.Time
            for _, rule := range rules {
                if rule.Enabled {
                    runTime := time.Now().Add(time.Duration(rule.IntervalHours) * time.Hour)
                    if nextRun.IsZero() || runTime.Before(nextRun) {
                        nextRun = runTime
                    }
                }
            }

            return nextRun, nil
        },
        Execute: func(ctx context.Context, s *Scheduler) error {
            return s.runScheduledScans(ctx)
        },
    }
}

func (jr *JobRunner) registerCleanupJob() {
    jr.mu.Lock()
    defer jr.mu.Unlock()

    jr.jobs["cleanup_discovered"] = &Job{
        ID:   "cleanup_discovered",
        Name: "Cleanup Old Discoveries",
        Schedule: func(s *Scheduler) (time.Time, error) {
            // Run cleanup daily at 2 AM
            now := time.Now()
            next := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, now.Location())
            if next.Before(now) {
                next = next.AddDate(0, 0, 1)
            }
            return next, nil
        },
        Execute: func(ctx context.Context, s *Scheduler) error {
            return s.storage.CleanupOldDiscoveries(s.config.DiscoveryCleanupDays)
        },
    }
}

func (jr *JobRunner) Tick() {
    jr.mu.Lock()
    defer jr.mu.Unlock()

    now := time.Now()
    for _, job := range jr.jobs {
        if job.NextRun.IsZero() || job.NextRun.After(now) {
            continue
        }

        // Job is due to run
        go func(j *Job) {
            log.Info("Running job", "job_id", j.ID, "job_name", j.Name)

            // Calculate next run time
            nextRun, err := j.Schedule(nil)
            if err != nil {
                log.Error("Failed to schedule next run", "job_id", j.ID, "error", err)
                return
            }

            // Execute job
            if err := j.Execute(context.Background(), nil); err != nil {
                log.Error("Job execution failed", "job_id", j.ID, "error", err)
            }

            // Update job state
            jr.mu.Lock()
            j.LastRun = &now
            j.NextRun = nextRun
            jr.mu.Unlock()
        }(job)
    }
}
```

func (s *Scheduler) Start() {
    s.mu.Lock()
    if s.running {
        s.mu.Unlock()
        return
    }
    s.running = true
    s.mu.Unlock()

    s.wg.Add(1)
    go s.run()

    log.Info("Discovery scheduler started", "interval", s.config.DiscoveryInterval)
}

func (s *Scheduler) Stop() {
    s.mu.Lock()
    if !s.running {
        s.mu.Unlock()
        return
    }
    s.running = false
    s.mu.Unlock()

    s.cancel()
    s.wg.Wait()
    log.Info("Discovery scheduler stopped")
}

func (s *Scheduler) run() {
    defer s.wg.Done()

    ticker := time.NewTicker(s.config.DiscoveryInterval)
    defer ticker.Stop()

    // Run initial scan on startup (if configured)
    if s.config.DiscoveryScanOnStartup {
        s.runScheduledScans()
    }

    for {
        select {
        case <-s.ctx.Done():
            return
        case <-ticker.C:
            s.runScheduledScans()
        }
    }
}

func (s *Scheduler) runScheduledScans() {
    rules, err := s.storage.ListDiscoveryRules()
    if err != nil {
        log.Error("Failed to list discovery rules", "error", err)
        return
    }

    for _, rule := range rules {
        if !rule.Enabled {
            continue
        }

        network, err := s.storage.GetNetwork(rule.NetworkID)
        if err != nil {
            log.Error("Failed to get network for discovery", "network_id", rule.NetworkID, "error", err)
            continue
        }

        log.Info("Starting scheduled discovery scan", "network", network.Name)

        _, err = s.scanner.Scan(s.ctx, network, rule.ScanType)
        if err != nil {
            log.Error("Scheduled scan failed", "network", network.Name, "error", err)
        }
    }

    // Cleanup old discoveries
    if s.config.DiscoveryCleanupDays > 0 {
        if err := s.storage.CleanupOldDiscoveries(s.config.DiscoveryCleanupDays); err != nil {
            log.Error("Failed to cleanup old discoveries", "error", err)
        }
    }
}
```

## Scan Types

| Type | Description | Ports Scanned |
|------|-------------|---------------|
| `quick` | Ping only | None |
| `full` | Ping + common ports | 21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 8080 |
| `deep` | Full port scan + service detection | Top 1000 ports |
