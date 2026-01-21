# Performance and Optimization

This document defines performance targets, optimization techniques, and monitoring for Rackd.

## 1. Performance Targets and SLAs

### 1.1 API Response Time Targets

| Endpoint Type | p50 | p95 | p99 | Max |
|---------------|------|------|------|------|
| Simple reads (GET /api/devices) | 50ms | 100ms | 200ms | 500ms |
| Complex reads (GET /api/devices/{id}) | 30ms | 80ms | 150ms | 400ms |
| Writes (POST/PUT) | 100ms | 300ms | 500ms | 2s |
| Search queries | 100ms | 300ms | 500ms | 2s |
| List with pagination (100 items) | 50ms | 150ms | 300ms | 1s |
| Discovery scan trigger | 200ms | 500ms | 1s | 5s |

### 1.2 Database Operation Latency Targets

| Operation | Target | Warning | Critical |
|-----------|---------|----------|-----------|
| Device GET by ID | 10ms | 30ms | 100ms |
| Device LIST (limit 100) | 20ms | 50ms | 150ms |
| Device CREATE | 30ms | 100ms | 300ms |
| Device UPDATE | 25ms | 80ms | 250ms |
| Device DELETE | 20ms | 70ms | 200ms |
| Network GET by ID | 10ms | 30ms | 100ms |
| Network LIST (limit 100) | 15ms | 40ms | 120ms |
| Full-text search | 50ms | 150ms | 300ms | 1s |
| Relationship query | 15ms | 50ms | 150ms | 400ms |

### 1.3 Discovery Scan Throughput Targets

| Scan Type | Hosts/Second | Total Scan Time (/24 network) | Max Concurrent |
|-----------|---------------|--------------------------------|-----------------|
| Quick (ping only) | 100 | ~2.5 minutes | 500 |
| Full (common ports) | 20 | ~12 minutes | 100 |
| Deep (1000 ports) | 5 | ~50 minutes | 50 |

### 1.4 Concurrency Limits

| Component | Max Concurrent | Queue Size | Timeout |
|-----------|---------------|------------|----------|
| API requests | 1000 | 500 | 30s |
| Database connections | 50 | 10 | 10s |
| Discovery scans | 3 | 5 | 1h |
| Background jobs | 10 | 20 | 5m |

---

## 2. Optimization Techniques

### 2.1 Query Optimization

**Index Usage:**

```go
// Create indexes for common query patterns
func CreateIndexes(db *sql.DB) error {
    // Index for device name searches
    if _, err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_devices_name ON devices(name)
    `); err != nil {
        return err
    }

    // Index for device datacenter filtering
    if _, err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_devices_datacenter ON devices(datacenter_id)
    `); err != nil {
        return err
    }

    // Index for device tag filtering
    if _, err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_tags_device_tag ON tags(device_id, tag)
    `); err != nil {
        return err
    }

    // Composite index for network + datacenter queries
    if _, err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_networks_dc ON networks(datacenter_id, name)
    `); err != nil {
        return err
    }

    // Covering index for device with datacenter and tags
    if _, err := db.Exec(`
        CREATE INDEX IF NOT EXISTS idx_devices_covering
        ON devices(datacenter_id, name)
        INCLUDE (description, make_model)
    `); err != nil {
        return err
    }

    return nil
}
```

**Query Planning:**

```go
func (s *SQLiteStorage) ListDevices(filter *DeviceFilter) ([]*Device, error) {
    query := `
        SELECT d.id, d.name, d.description, d.make_model, d.os,
               d.datacenter_id, d.username, d.location,
               d.created_at, d.updated_at
        FROM devices d
        WHERE 1=1
    `
    args := []interface{}{}
    argPos := 1

    // Build query with proper parameterization
    if filter.DatacenterID != "" {
        query += fmt.Sprintf(" AND d.datacenter_id = $%d", argPos)
        args = append(args, filter.DatacenterID)
        argPos++
    }

    if len(filter.Tags) > 0 {
        query += fmt.Sprintf(" AND EXISTS (
            SELECT 1 FROM tags t
            WHERE t.device_id = d.id AND t.tag IN (%s)
        )", strings.Repeat(",$?", len(filter.Tags)))
        for _, tag := range filter.Tags {
            args = append(args, tag)
            argPos++
        }
    }

    // Use EXPLAIN QUERY PLAN to analyze
    if logLevel >= LogLevelDebug {
        explainQuery := "EXPLAIN QUERY PLAN " + query
        rows, _ := s.db.Query(explainQuery, args...)
        for rows.Next() {
            var plan string
            rows.Scan(&plan)
            log.Debug("Query plan", "plan", plan)
        }
    }

    // Execute query with LIMIT for pagination
    query += fmt.Sprintf(" ORDER BY d.created_at DESC LIMIT $%d", argPos)
    args = append(args, filter.Limit)

    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    devices := []*Device{}
    for rows.Next() {
        device := &Device{}
        if err := rows.Scan(
            &device.ID, &device.Name, &device.Description,
            &device.MakeModel, &device.OS, &device.DatacenterID,
            &device.Username, &device.Location,
            &device.CreatedAt, &device.UpdatedAt,
        ); err != nil {
            return nil, err
        }
        devices = append(devices, device)
    }

    return devices, nil
}
```

### 2.2 Preventing N+1 Queries

```go
// BAD: N+1 queries
func (s *SQLiteStorage) GetDevice(id string) (*Device, error) {
    // Query 1: Get device
    device := queryDevice(id)

    // Query 2-N: Get addresses for each device
    for _, device := range devices {
        device.Addresses = getAddresses(device.ID)
        // This causes N+1 queries!
    }
}

// GOOD: Eager loading with JOIN
func (s *SQLiteStorage) GetDeviceWithAddresses(id string) (*Device, error) {
    query := `
        SELECT d.id, d.name, d.description,
               a.id as address_id, a.ip, a.port, a.type,
               a.label, a.network_id, a.switch_port, a.pool_id
        FROM devices d
        LEFT JOIN addresses a ON d.id = a.device_id
        WHERE d.id = ?
        ORDER BY a.id
    `

    rows, err := s.db.Query(query, id)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    device := &Device{Addresses: []Address{}}
    first := true
    for rows.Next() {
        if first {
            rows.Scan(&device.ID, &device.Name, &device.Description)
            first = false
        }

        var address Address
        var addressID sql.NullString
        rows.Scan(&addressID, &address.IP, &address.Port, &address.Type,
            &address.Label, &address.NetworkID, &address.SwitchPort, &address.PoolID)

        if addressID.Valid {
            device.Addresses = append(device.Addresses, address)
        }
    }

    return device, nil
}
```

### 2.3 Connection Pooling Configuration

```go
package storage

import (
    "database/sql"
    "time"
)

type PoolConfig struct {
    MaxOpenConns    int           `json:"max_open_conns"`
    MaxIdleConns    int           `json:"max_idle_conns"`
    ConnMaxLifetime  time.Duration `json:"conn_max_lifetime"`
    ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

func DefaultPoolConfig() PoolConfig {
    return PoolConfig{
        MaxOpenConns:    50,   // Maximum open connections
        MaxIdleConns:    10,   // Maximum idle connections
        ConnMaxLifetime:  5 * time.Minute,  // Maximum connection lifetime
        ConnMaxIdleTime: 1 * time.Minute,  // Maximum idle time
    }
}

func ConfigurePool(db *sql.DB, config PoolConfig) error {
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

    log.Info("Database pool configured",
        "max_open", config.MaxOpenConns,
        "max_idle", config.MaxIdleConns,
        "max_lifetime", config.ConnMaxLifetime,
        "max_idle_time", config.ConnMaxIdleTime,
    )

    return nil
}

func (s *SQLiteStorage) MonitorPool(ctx context.Context) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            stats := s.db.Stats()
            log.Info("Database pool stats",
                "open", stats.OpenConnections,
                "in_use", stats.InUse,
                "idle", stats.Idle,
                "wait_count", stats.WaitCount,
                "wait_duration", stats.WaitDuration,
                "max_idle_closed", stats.MaxIdleClosed,
            )

            // Alert on high connection usage
            if stats.InUse > int(float64(s.poolConfig.MaxOpenConns)*0.8) {
                log.Warn("High database connection usage",
                    "in_use", stats.InUse,
                    "max", s.poolConfig.MaxOpenConns,
                )
            }
        }
    }
}
```

### 2.4 Batch Operations

```go
func (s *SQLiteStorage) CreateDevicesBatch(devices []*Device) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    stmt, err := tx.Prepare(`
        INSERT INTO devices (id, name, description, make_model, os,
                        datacenter_id, username, location, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    // Batch insert
    for _, device := range devices {
        _, err := stmt.Exec(
            device.ID, device.Name, device.Description,
            device.MakeModel, device.OS,
            device.DatacenterID, device.Username,
            device.Location,
            device.CreatedAt, device.UpdatedAt,
        )
        if err != nil {
            return err
        }
    }

    // Batch insert tags
    tagStmt, err := tx.Prepare(`
        INSERT INTO tags (device_id, tag) VALUES (?, ?)
    `)
    if err != nil {
        return err
    }
    defer tagStmt.Close()

    for _, device := range devices {
        for _, tag := range device.Tags {
            _, err := tagStmt.Exec(device.ID, tag)
            if err != nil {
                return err
            }
        }
    }

    return tx.Commit()
}
```

### 2.5 Efficient Pagination

```go
func (s *SQLiteStorage) ListDevicesPaginated(filter *DeviceFilter, page, pageSize int) ([]*Device, error) {
    offset := (page - 1) * pageSize

    query := `
        SELECT id, name, description, make_model, os,
               datacenter_id, username, location,
               created_at, updated_at
        FROM devices
        WHERE 1=1
    `
    args := []interface{}{}

    // Build WHERE clause
    if filter.DatacenterID != "" {
        query += " AND datacenter_id = ?"
        args = append(args, filter.DatacenterID)
    }

    // Use cursor-based pagination for large datasets
    if filter.LastID != "" {
        query += " AND id > ?"
        args = append(args, filter.LastID)
    }

    // Sort and limit
    query += " ORDER BY id LIMIT ?"
    args = append(args, pageSize+1) // +1 to check if more pages

    rows, err := s.db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    devices := []*Device{}
    for rows.Next() {
        device := &Device{}
        if err := rows.Scan(
            &device.ID, &device.Name, &device.Description,
            &device.MakeModel, &device.OS, &device.DatacenterID,
            &device.Username, &device.Location,
            &device.CreatedAt, &device.UpdatedAt,
        ); err != nil {
            return nil, err
        }

        devices = append(devices, device)
    }

    // Check if there are more results
    if len(devices) > pageSize {
        devices = devices[:pageSize]
    }

    return devices, nil
}
```

---

## 3. Caching Architecture

### 3.1 In-Memory Cache Implementation (OSS)

```go
package cache

import (
    "container/list"
    "sync"
    "time"
)

type CacheEntry struct {
    Key       string
    Value     interface{}
    ExpiresAt time.Time
}

type LRUCache struct {
    maxEntries int
    ttl        time.Duration
    entries    map[string]*list.Element
    lruList    *list.List
    mu         sync.RWMutex
}

func NewLRUCache(maxEntries int, ttl time.Duration) *LRUCache {
    return &LRUCache{
        maxEntries: maxEntries,
        ttl:        ttl,
        entries:    make(map[string]*list.Element),
        lruList:    list.New(),
    }
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    elem, ok := c.entries[key]
    if !ok {
        return nil, false
    }

    entry := elem.Value.(*CacheEntry)

    // Check expiration
    if time.Now().After(entry.ExpiresAt) {
        c.removeElement(elem)
        return nil, false
    }

    // Move to front (most recently used)
    c.lruList.MoveToFront(elem)

    return entry.Value, true
}

func (c *LRUCache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // Check if entry exists
    if elem, ok := c.entries[key]; ok {
        // Update existing entry
        entry := elem.Value.(*CacheEntry)
        entry.Value = value
        entry.ExpiresAt = time.Now().Add(c.ttl)
        c.lruList.MoveToFront(elem)
        return
    }

    // Create new entry
    entry := &CacheEntry{
        Key:       key,
        Value:     value,
        ExpiresAt: time.Now().Add(c.ttl),
    }

    elem := c.lruList.PushFront(entry)
    c.entries[key] = elem

    // Evict oldest if at capacity
    if c.lruList.Len() > c.maxEntries {
        c.evictOldest()
    }
}

func (c *LRUCache) removeElement(elem *list.Element) {
    entry := elem.Value.(*CacheEntry)
    delete(c.entries, entry.Key)
    c.lruList.Remove(elem)
}

func (c *LRUCache) evictOldest() {
    elem := c.lruList.Back()
    if elem != nil {
        c.removeElement(elem)
    }
}

func (c *LRUCache) Invalidate(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if elem, ok := c.entries[key]; ok {
        c.removeElement(elem)
    }
}

func (c *LRUCache) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.entries = make(map[string]*list.Element)
    c.lruList.Init()
}
```

### 3.2 Cache Patterns

**Device Cache:**

```go
package storage

import (
    "github.com/martinsuchenak/rackd/internal/cache"
)

type CachedStorage struct {
    storage   ExtendedStorage
    deviceCache *cache.LRUCache
    networkCache *cache.LRUCache
}

func NewCachedStorage(storage ExtendedStorage) *CachedStorage {
    return &CachedStorage{
        storage: storage,
        deviceCache: cache.NewLRUCache(1000, 5*time.Minute),
        networkCache: cache.NewLRUCache(500, 10*time.Minute),
    }
}

func (cs *CachedStorage) GetDevice(id string) (*Device, error) {
    // Try cache first
    if cached, ok := cs.deviceCache.Get(id); ok {
        metrics.CacheHit("device", id)
        return cached.(*Device), nil
    }
    metrics.CacheMiss("device", id)

    // Cache miss - fetch from storage
    device, err := cs.storage.GetDevice(id)
    if err != nil {
        return nil, err
    }

    // Store in cache
    cs.deviceCache.Set(id, device)

    return device, nil
}

func (cs *CachedStorage) UpdateDevice(device *Device) error {
    // Update storage
    if err := cs.storage.UpdateDevice(device); err != nil {
        return err
    }

    // Invalidate cache
    cs.deviceCache.Invalidate(device.ID)

    return nil
}

func (cs *CachedStorage) DeleteDevice(id string) error {
    // Delete from storage
    if err := cs.storage.DeleteDevice(id); err != nil {
        return err
    }

    // Invalidate cache
    cs.deviceCache.Invalidate(id)

    return nil
}
```

### 3.3 Redis Integration (Enterprise)

```go
package cache

import (
    "context"
    "encoding/json"
    "time"

    "github.com/redis/go-redis/v9"
)

type RedisCache struct {
    client *redis.Client
    prefix string
    ttl    time.Duration
}

func NewRedisCache(client *redis.Client, prefix string, ttl time.Duration) *RedisCache {
    return &RedisCache{
        client: client,
        prefix: prefix,
        ttl:    ttl,
    }
}

func (rc *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
    fullKey := rc.fullKey(key)

    data, err := rc.client.Get(ctx, fullKey).Bytes()
    if err != nil {
        if err == redis.Nil {
            metrics.CacheMiss(key)
            return ErrCacheMiss
        }
        return err
    }

    metrics.CacheHit(key)

    return json.Unmarshal(data, dest)
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
    fullKey := rc.fullKey(key)

    data, err := json.Marshal(value)
    if err != nil {
        return err
    }

    return rc.client.Set(ctx, fullKey, data, rc.ttl).Err()
}

func (rc *RedisCache) Invalidate(ctx context.Context, key string) error {
    fullKey := rc.fullKey(key)
    return rc.client.Del(ctx, fullKey).Err()
}

func (rc *RedisCache) fullKey(key string) string {
    return fmt.Sprintf("%s:%s", rc.prefix, key)
}
```

### 3.4 Cache Key Design

**Key Naming Conventions:**

```
device:<device_id>
network:<network_id>
datacenter:<datacenter_id>
devices:filter:<hash_of_filter>
networks:datacenter:<dc_id>
discovery:network:<network_id>
```

**Hashing Complex Keys:**

```go
func cacheKey(prefix string, params map[string]interface{}) string {
    // Sort params for consistent hashing
    keys := make([]string, 0, len(params))
    for k := range params {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    // Build key string
    var builder strings.Builder
    builder.WriteString(prefix)
    builder.WriteString(":")
    for _, k := range keys {
        builder.WriteString(k)
        builder.WriteString("=")
        builder.WriteString(fmt.Sprintf("%v", params[k]))
        builder.WriteString(",")
    }

    // Hash for shorter keys
    h := sha256.New()
    h.Write([]byte(builder.String()))
    return hex.EncodeToString(h.Sum(nil))
}
```

### 3.5 Cache Invalidation

**Strategies:**

1. **Time-based:** TTL expiration
2. **Event-based:** Invalidate on write operations
3. **Pattern-based:** Invalidate matching patterns

```go
func (cs *CachedStorage) InvalidatePattern(pattern string) {
    cs.deviceCache.Clear() // Simple approach

    // Or use pattern matching
    for key := range cs.deviceCache.entries {
        if strings.Contains(key, pattern) {
            cs.deviceCache.Invalidate(key)
        }
    }
}

// Example: Invalidate all devices in datacenter
func (cs *CachedStorage) InvalidateDatacenterDevices(dcID string) {
    pattern := fmt.Sprintf("device:*:datacenter=%s", dcID)
    cs.InvalidatePattern(pattern)
}
```

---

## 4. Code Optimization Examples

### 4.1 Optimized Storage Queries

```go
// BAD: Multiple queries
func GetDevicesWithNetworks(deviceIDs []string) (map[string]*Device, error) {
    result := make(map[string]*Device)

    for _, id := range deviceIDs {
        device, err := storage.GetDevice(id)
        if err != nil {
            return nil, err
        }
        result[id] = device

        // N+1: Get network for each device
        network, err := storage.GetNetwork(device.NetworkID)
        if err != nil {
            return nil, err
        }
        device.Network = network
    }

    return result, nil
}

// GOOD: Single query with JOIN
func GetDevicesWithNetworksOptimized(deviceIDs []string) (map[string]*Device, error) {
    query := `
        SELECT d.id, d.name, d.description, d.make_model, d.os,
               n.id as network_id, n.name as network_name, n.subnet
        FROM devices d
        INNER JOIN networks n ON d.network_id = n.id
        WHERE d.id IN (%s)
    `, placeholders(len(deviceIDs))

    args := make([]interface{}, len(deviceIDs))
    for i, id := range deviceIDs {
        args[i] = id
    }

    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    result := make(map[string]*Device)
    for rows.Next() {
        device := &Device{Network: &Network{}}
        rows.Scan(
            &device.ID, &device.Name, &device.Description,
            &device.MakeModel, &device.OS,
            &device.Network.ID, &device.Network.Name,
            &device.Network.Subnet,
        )
        result[device.ID] = device
    }

    return result, nil
}
```

### 4.2 Efficient Concurrent Scanning

```go
func (s *DefaultScanner) runScanConcurrent(ctx context.Context, ips []string, scanType string) ([]*DiscoveredDevice, error) {
    // Use semaphore to limit concurrency
    semaphore := make(chan struct{}, s.config.MaxConcurrent)

    var wg sync.WaitGroup
    results := make(chan *DiscoveredDevice, len(ips))
    errors := make(chan error, len(ips))

    // Scan concurrently
    for _, ip := range ips {
        wg.Add(1)
        go func(ip string) {
            defer wg.Done()

            // Acquire semaphore
            semaphore <- struct{}{}
            defer func() { <-semaphore }()

            // Scan host
            device, err := s.discoverHost(ctx, ip, scanType)
            if err != nil {
                errors <- err
                return
            }

            if device != nil {
                results <- device
            }
        }(ip)
    }

    // Wait for all goroutines
    go func() {
        wg.Wait()
        close(results)
        close(errors)
    }()

    // Collect results
    devices := make([]*DiscoveredDevice, 0)
    for device := range results {
        devices = append(devices, device)
    }

    // Check for errors
    for err := range errors {
        log.Warn("Discovery error", "error", err)
    }

    return devices, nil
}
```

### 4.3 Bulk Operations

```go
func (s *SQLiteStorage) BulkUpdateDevices(updates []*Device) error {
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Prepare update statement
    stmt, err := tx.Prepare(`
        UPDATE devices SET
            name = ?,
            description = ?,
            make_model = ?,
            os = ?,
            updated_at = ?
        WHERE id = ?
    `)
    if err != nil {
        return err
    }
    defer stmt.Close()

    // Execute bulk updates
    now := time.Now()
    for _, device := range updates {
        _, err := stmt.Exec(
            device.Name, device.Description,
            device.MakeModel, device.OS,
            now, device.ID,
        )
        if err != nil {
            return err
        }
    }

    return tx.Commit()
}
```

### 4.4 Memory-Efficient Data Structures

```go
// Use slices instead of maps for ordered iteration
type DeviceList struct {
    devices []Device
    index   map[string]int
}

func NewDeviceList() *DeviceList {
    return &DeviceList{
        devices: make([]Device, 0),
        index:   make(map[string]int),
    }
}

func (dl *DeviceList) Add(device Device) {
    dl.devices = append(dl.devices, device)
    dl.index[device.ID] = len(dl.devices) - 1
}

func (dl *DeviceList) Get(id string) *Device {
    idx, ok := dl.index[id]
    if !ok {
        return nil
    }
    return &dl.devices[idx]
}

func (dl *DeviceList) Filter(predicate func(*Device) bool) []*Device {
    result := make([]*Device, 0)
    for i := range dl.devices {
        if predicate(&dl.devices[i]) {
            result = append(result, &dl.devices[i])
        }
    }
    return result
}

// Pre-allocate slice capacity
func (s *SQLiteStorage) ListDevices(filter *DeviceFilter) ([]*Device, error) {
    // Get count first
    count := s.getDeviceCount(filter)

    // Pre-allocate slice
    devices := make([]*Device, 0, count)

    rows, err := s.db.Query(buildQuery(filter), buildArgs(filter)...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    for rows.Next() {
        device := &Device{}
        if err := rows.Scan(&device.ID, &device.Name, ...); err != nil {
            return nil, err
        }
        devices = append(devices, device)
    }

    return devices, nil
}
```

---

## 5. Scalability Considerations

### 5.1 SQLite Scaling Limitations

| Limitation | Value | Impact |
|-----------|-------|--------|
| Concurrent writes | 1 | Single write transaction at a time |
| Database size | 140 TB | Theoretical limit, practical ~10GB |
| Row size | 1 GB | Per row limit |
| Column count | 2000 | Practical limit ~100 |
| Network access | Local only | No network file locking (NFS) |

### 5.2 Vertical Scaling Requirements

**Resource Scaling:**

| Metric | Small | Medium | Large | Enterprise |
|--------|--------|---------|-------------|
| Devices | < 1,000 | 1,000-10,000 | 10,000-100,000 | > 100,000 |
| Networks | < 100 | 100-1,000 | 1,000-10,000 | > 10,000 |
| RAM | 256 MB | 512 MB - 1 GB | 2-4 GB | 8-16 GB |
| CPU | 1 core | 2 cores | 4 cores | 8+ cores |
| Storage | 10 GB | 50 GB | 200 GB | 1+ TB |

### 5.3 Database Migration Path to Postgres

**Migration Triggers:**

- Device count > 10,000
- Need for high availability
- Need for multi-server deployment
- Need for advanced features (full-text search, etc.)

**Migration Approach:**

```go
package postgres

import (
    "database/sql"

    _ "github.com/lib/pq"
)

type PostgresStorage struct {
    db *sql.DB
}

func MigrateFromSQLite(sqliteDB *sql.DB, pgDSN string) error {
    // Connect to Postgres
    pgDB, err := sql.Open("postgres", pgDSN)
    if err != nil {
        return err
    }
    defer pgDB.Close()

    // Create schema
    if err := createSchema(pgDB); err != nil {
        return err
    }

    // Migrate data in batches
    batchSize := 1000
    offset := 0

    for {
        devices, err := readBatch(sqliteDB, batchSize, offset)
        if err != nil {
            return err
        }

        if len(devices) == 0 {
            break
        }

        if err := writeBatch(pgDB, devices); err != nil {
            return err
        }

        offset += batchSize
        log.Info("Migrated batch", "count", len(devices), "offset", offset)
    }

    return nil
}
```

### 5.4 Read-Only Replica Considerations

```go
type ReplicaStorage struct {
    primary  ExtendedStorage
    replicas []ExtendedStorage
    strategy ReadStrategy
}

type ReadStrategy interface {
    Select() ExtendedStorage
}

type RoundRobinStrategy struct {
    current int
    replicas []ExtendedStorage
    mu       sync.Mutex
}

func (rr *RoundRobinStrategy) Select() ExtendedStorage {
    rr.mu.Lock()
    defer rr.mu.Unlock()

    selected := rr.replicas[rr.current]
    rr.current = (rr.current + 1) % len(rr.replicas)
    return selected
}

func (rs *ReplicaStorage) GetDevice(id string) (*Device, error) {
    // Read from replica
    replica := rs.strategy.Select()
    return replica.GetDevice(id)
}

func (rs *ReplicaStorage) UpdateDevice(device *Device) error {
    // Write to primary
    return rs.primary.UpdateDevice(device)
}
```

---

## 6. Performance Profiling

### 6.1 Using Go Pprof

**CPU Profiling:**

```go
import (
    "os"
    "runtime/pprof"
)

func StartCPUProfile(path string) (*os.File, error) {
    f, err := os.Create(path)
    if err != nil {
        return nil, err
    }

    if err := pprof.StartCPUProfile(f); err != nil {
        f.Close()
        return nil, err
    }

    return f, nil
}

func StopCPUProfile(f *os.File) {
    pprof.StopCPUProfile()
    f.Close()
}

// Usage:
func main() {
    cpuProfile, err := StartCPUProfile("cpu.prof")
    if err != nil {
        log.Fatal("Failed to start CPU profile", "error", err)
    }
    defer StopCPUProfile(cpuProfile)

    // Application code here
}
```

**Memory Profiling:**

```go
func DumpMemoryProfile(path string) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()

    runtime.GC() // Force GC before profiling

    if err := pprof.WriteHeapProfile(f); err != nil {
        return err
    }

    return nil
}

// Usage:
go func() {
    for {
        time.Sleep(1 * time.Minute)
        if err := DumpMemoryProfile(fmt.Sprintf("heap.%d.prof", time.Now().Unix())); err != nil {
            log.Error("Failed to write memory profile", "error", err)
        }
    }
}()
```

**Goroutine Profiling:**

```go
func DumpGoroutineProfile(path string) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }
    defer f.Close()

    if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
        return err
    }

    return nil
}
```

### 6.2 Database Query Analysis

**SQLite:**

```sql
-- Explain query plan
EXPLAIN QUERY PLAN
SELECT * FROM devices WHERE name = 'server-01';

-- Check query performance
.timer on
SELECT * FROM devices WHERE name = 'server-01';
.timer off

-- Analyze database
ANALYZE;

-- Check index usage
PRAGMA index_info('idx_devices_name');
```

**Postgres:**

```sql
-- Explain query plan
EXPLAIN ANALYZE
SELECT * FROM devices WHERE name = 'server-01';

-- Check index usage
SELECT * FROM pg_stat_user_indexes WHERE schemaname = 'public';

-- Vacuum and analyze
VACUUM ANALYZE devices;
```

### 6.3 Memory Leak Detection

```go
func MonitorMemoryLeaks(ctx context.Context) {
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

            // Check for continuous growth
            allocDiff := stats.Alloc - lastStats.Alloc
            if allocDiff > 100*1024*1024 { // > 100 MB growth
                log.Warn("Possible memory leak detected",
                    "alloc_diff_mb", allocDiff/(1024*1024),
                    "alloc_current_mb", stats.Alloc/(1024*1024),
                )
            }

            lastStats = stats
        }
    }
}
```

### 6.4 Goroutine Leak Detection

```go
func MonitorGoroutineLeaks(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    lastCount := runtime.NumGoroutine()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            currentCount := runtime.NumGoroutine()
            diff := currentCount - lastCount

            // Check for continuous growth
            if diff > 100 { // More than 100 new goroutines
                log.Warn("Possible goroutine leak detected",
                    "diff", diff,
                    "current", currentCount,
                    "previous", lastCount,
                )

                // Dump goroutine profile
                DumpGoroutineProfile(fmt.Sprintf("goroutines.%d.prof", time.Now().Unix()))
            }

            lastCount = currentCount
        }
    }
}
```

---

## 7. Benchmark Testing

### 7.1 Benchmark Suite Design

```go
package storage_test

import (
    "testing"
)

func BenchmarkGetDevice(b *testing.B) {
    store := setupBenchmarkStorage(b)
    device := createTestDevice(store)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := store.GetDevice(device.ID)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkListDevices(b *testing.B) {
    store := setupBenchmarkStorage(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := store.ListDevices(&DeviceFilter{Limit: 100})
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkCreateDevice(b *testing.B) {
    store := setupBenchmarkStorage(b)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        device := &Device{
            ID:   uuid.New().String(),
            Name:  fmt.Sprintf("device-%d", i),
        }
        if err := store.CreateDevice(device); err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkSearchDevices(b *testing.B) {
    store := setupBenchmarkStorage(b)
    query := "server"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := store.SearchDevices(query)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### 7.2 Regression Testing

```go
package perf_test

import (
    "testing"
)

// Store baseline performance
var baseline PerformanceBaseline

type PerformanceBaseline struct {
    GetDevice        BenchmarkResult
    ListDevices      BenchmarkResult
    CreateDevice     BenchmarkResult
    SearchDevices    BenchmarkResult
}

type BenchmarkResult struct {
    NsPerOp  float64
    BytesPerOp float64
    AllocsPerOp float64
}

func TestPerformanceRegression(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping in short mode")
    }

    current := runPerformanceBenchmarks()
    compareBaseline(t, baseline, current)
}

func runPerformanceBenchmarks() PerformanceBaseline {
    result := testing.Benchmark(func(b *testing.B) {
        BenchmarkGetDevice(b)
        BenchmarkListDevices(b)
        BenchmarkCreateDevice(b)
        BenchmarkSearchDevices(b)
    })

    return PerformanceBaseline{
        GetDevice:      extractBenchmark(result, "BenchmarkGetDevice"),
        ListDevices:    extractBenchmark(result, "BenchmarkListDevices"),
        CreateDevice:   extractBenchmark(result, "BenchmarkCreateDevice"),
        SearchDevices:  extractBenchmark(result, "BenchmarkSearchDevices"),
    }
}

func compareBaseline(t *testing.T, baseline, current PerformanceBaseline) {
    checkRegression(t, baseline.GetDevice, current.GetDevice, "GetDevice")
    checkRegression(t, baseline.ListDevices, current.ListDevices, "ListDevices")
    checkRegression(t, baseline.CreateDevice, current.CreateDevice, "CreateDevice")
    checkRegression(t, baseline.SearchDevices, current.SearchDevices, "SearchDevices")
}

func checkRegression(t *testing.T, baseline, current BenchmarkResult, name string) {
    regression := (current.NsPerOp - baseline.NsPerOp) / baseline.NsPerOp

    if regression > 0.1 { // > 10% regression
        t.Errorf("%s: Performance regression detected: %.1f%% slower (was %.0f ns/op, now %.0f ns/op)",
            name, regression*100, baseline.NsPerOp, current.NsPerOp)
    }
}
```

---

## 8. Performance Monitoring

### 8.1 Key Metrics to Track

**API Metrics:**

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    httpRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: []float64{.01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
        },
        []string{"method", "path"},
    )

    httpRequestDurationSummary = promauto.NewSummaryVec(
        prometheus.SummaryOpts{
            Name: "http_request_duration_summary_seconds",
            Help: "HTTP request duration summary",
        },
        []string{"method", "path"},
    )
)

func RecordHTTPRequest(method, path string, status int, duration float64) {
    httpRequestsTotal.WithLabelValues(method, path, fmt.Sprintf("%d", status)).Inc()
    httpRequestDuration.WithLabelValues(method, path).Observe(duration)
    httpRequestDurationSummary.WithLabelValues(method, path).Observe(duration)
}
```

**Database Metrics:**

```go
var (
    dbQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "db_query_duration_seconds",
            Help: "Database query duration",
            Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
        },
        []string{"operation", "table"},
    )

    dbConnectionPool = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "db_connection_pool",
            Help: "Database connection pool metrics",
        },
        []string{"state"}, // open, in_use, idle
    )
)

func RecordDBQuery(operation, table string, duration float64) {
    dbQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

func RecordDBConnectionStats(stats sql.DBStats) {
    dbConnectionPool.WithLabelValues("open").Set(float64(stats.OpenConnections))
    dbConnectionPool.WithLabelValues("in_use").Set(float64(stats.InUse))
    dbConnectionPool.WithLabelValues("idle").Set(float64(stats.Idle))
}
```

**Cache Metrics:**

```go
var (
    cacheHits = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_hits_total",
            Help: "Total cache hits",
        },
        []string{"cache_type", "key"},
    )

    cacheMisses = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_misses_total",
            Help: "Total cache misses",
        },
        []string{"cache_type", "key"},
    )

    cacheEvictions = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "cache_evictions_total",
            Help: "Total cache evictions",
        },
        []string{"cache_type"},
    )
)

func RecordCacheHit(cacheType, key string) {
    cacheHits.WithLabelValues(cacheType, key).Inc()
}

func RecordCacheMiss(cacheType, key string) {
    cacheMisses.WithLabelValues(cacheType, key).Inc()
}

func RecordCacheEviction(cacheType string) {
    cacheEvictions.WithLabelValues(cacheType).Inc()
}
```

### 8.2 Performance Baseline Targets

| Metric | Baseline | Warning | Critical |
|--------|-----------|----------|-----------|
| API GET /api/devices p50 | 50ms | 100ms | 200ms |
| API GET /api/devices p95 | 100ms | 200ms | 500ms |
| DB query GetDevice | 10ms | 30ms | 100ms |
| DB query ListDevices (100) | 20ms | 50ms | 150ms |
| Cache hit ratio | 80% | 60% | 40% |
| Discovery hosts/second | 20 | 10 | 5 |

### 8.3 Alert Thresholds

```go
type PerformanceAlert struct {
    Metric      string
    Threshold  float64
    Duration   time.Duration
    Severity   string
}

var performanceAlerts = []PerformanceAlert{
    {
        Metric:     "http_request_duration_seconds{path=\"/api/devices\",quantile=\"0.95\"}",
        Threshold:  0.5, // 500ms
        Duration:   5 * time.Minute,
        Severity:   "warning",
    },
    {
        Metric:     "http_request_duration_seconds{path=\"/api/devices\",quantile=\"0.99\"}",
        Threshold:  1.0, // 1s
        Duration:   5 * time.Minute,
        Severity:   "critical",
    },
    {
        Metric:     "cache_hit_ratio",
        Threshold:  0.6, // 60%
        Duration:   5 * time.Minute,
        Severity:   "warning",
    },
}

func MonitorPerformance(ctx context.Context) {
    for _, alert := range performanceAlerts {
        go monitorAlert(ctx, alert)
    }
}

func monitorAlert(ctx context.Context, alert PerformanceAlert) {
    ticker := time.NewTicker(alert.Duration)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            value := queryPrometheus(alert.Metric)
            if value > alert.Threshold {
                sendAlert(alert, value)
            }
        }
    }
}
```

### 8.4 Performance Regression Detection

```go
func DetectPerformanceRegression(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Hour)
    defer ticker.Stop()

    baseline := loadPerformanceBaseline()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            current := collectCurrentMetrics()

            if regression := detectRegression(baseline, current); regression != nil {
                alertRegression(regression)
                baseline = current
            }
        }
    }
}

func detectRegression(baseline, current map[string]float64) *Regression {
    for metric, baselineValue := range baseline {
        currentValue, ok := current[metric]
        if !ok {
            continue
        }

        regression := (currentValue - baselineValue) / baselineValue
        if regression > 0.2 { // > 20% regression
            return &Regression{
                Metric:     metric,
                Baseline:   baselineValue,
                Current:    currentValue,
                Regression: regression,
            }
        }
    }

    return nil
}
```
