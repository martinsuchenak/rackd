package metrics

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds application metrics
type Metrics struct {
	// HTTP metrics
	httpRequestsTotal   atomic.Int64
	httpRequestDuration sync.Map // map[string]*histogram
	httpRequestsByCode  sync.Map // map[int]*atomic.Int64

	// Application metrics
	devicesTotal      atomic.Int64
	networksTotal     atomic.Int64
	datacentersTotal  atomic.Int64
	discoveryScans    atomic.Int64
	discoveryDuration sync.Map // map[string]*histogram

	// Database metrics
	dbQueriesTotal    atomic.Int64
	dbQueryDuration   sync.Map // map[string]*histogram
	dbConnectionsOpen atomic.Int64

	startTime time.Time
}

type histogram struct {
	sum   atomic.Int64
	count atomic.Int64
}

var globalMetrics = &Metrics{
	startTime: time.Now(),
}

// Get returns the global metrics instance
func Get() *Metrics {
	return globalMetrics
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path string, statusCode int, duration time.Duration) {
	m.httpRequestsTotal.Add(1)

	key := fmt.Sprintf("%s %s", method, path)
	h := m.getOrCreateHistogram(&m.httpRequestDuration, key)
	h.observe(duration)

	counter := m.getOrCreateCounter(&m.httpRequestsByCode, statusCode)
	counter.Add(1)
}

// RecordDiscoveryScan records a discovery scan
func (m *Metrics) RecordDiscoveryScan(scanType string, duration time.Duration, hostsFound int) {
	m.discoveryScans.Add(1)
	h := m.getOrCreateHistogram(&m.discoveryDuration, scanType)
	h.observe(duration)
}

// RecordDBQuery records a database query
func (m *Metrics) RecordDBQuery(query string, duration time.Duration) {
	m.dbQueriesTotal.Add(1)
	h := m.getOrCreateHistogram(&m.dbQueryDuration, query)
	h.observe(duration)
}

// SetDeviceCount sets the current device count
func (m *Metrics) SetDeviceCount(count int64) {
	m.devicesTotal.Store(count)
}

// SetNetworkCount sets the current network count
func (m *Metrics) SetNetworkCount(count int64) {
	m.networksTotal.Store(count)
}

// SetDatacenterCount sets the current datacenter count
func (m *Metrics) SetDatacenterCount(count int64) {
	m.datacentersTotal.Store(count)
}

// SetDBConnectionsOpen sets the current open database connections
func (m *Metrics) SetDBConnectionsOpen(count int64) {
	m.dbConnectionsOpen.Store(count)
}

func (m *Metrics) getOrCreateHistogram(store *sync.Map, key string) *histogram {
	if v, ok := store.Load(key); ok {
		return v.(*histogram)
	}
	h := &histogram{}
	actual, _ := store.LoadOrStore(key, h)
	return actual.(*histogram)
}

func (m *Metrics) getOrCreateCounter(store *sync.Map, key int) *atomic.Int64 {
	if v, ok := store.Load(key); ok {
		return v.(*atomic.Int64)
	}
	counter := &atomic.Int64{}
	actual, _ := store.LoadOrStore(key, counter)
	return actual.(*atomic.Int64)
}

func (h *histogram) observe(d time.Duration) {
	h.sum.Add(d.Milliseconds())
	h.count.Add(1)
}

func (h *histogram) avg() float64 {
	count := h.count.Load()
	if count == 0 {
		return 0
	}
	return float64(h.sum.Load()) / float64(count)
}

// Export exports metrics in Prometheus text format
func (m *Metrics) Export() string {
	var out string

	// HTTP metrics
	out += fmt.Sprintf("# HELP http_requests_total Total number of HTTP requests\n")
	out += fmt.Sprintf("# TYPE http_requests_total counter\n")
	out += fmt.Sprintf("http_requests_total %d\n", m.httpRequestsTotal.Load())

	out += fmt.Sprintf("# HELP http_request_duration_milliseconds HTTP request duration in milliseconds\n")
	out += fmt.Sprintf("# TYPE http_request_duration_milliseconds summary\n")
	m.httpRequestDuration.Range(func(key, value interface{}) bool {
		h := value.(*histogram)
		out += fmt.Sprintf("http_request_duration_milliseconds{route=\"%s\"} %.2f\n", key, h.avg())
		return true
	})

	out += fmt.Sprintf("# HELP http_requests_by_code Total HTTP requests by status code\n")
	out += fmt.Sprintf("# TYPE http_requests_by_code counter\n")
	m.httpRequestsByCode.Range(func(key, value interface{}) bool {
		counter := value.(*atomic.Int64)
		out += fmt.Sprintf("http_requests_by_code{code=\"%d\"} %d\n", key, counter.Load())
		return true
	})

	// Application metrics
	out += fmt.Sprintf("# HELP devices_total Total number of devices\n")
	out += fmt.Sprintf("# TYPE devices_total gauge\n")
	out += fmt.Sprintf("devices_total %d\n", m.devicesTotal.Load())

	out += fmt.Sprintf("# HELP networks_total Total number of networks\n")
	out += fmt.Sprintf("# TYPE networks_total gauge\n")
	out += fmt.Sprintf("networks_total %d\n", m.networksTotal.Load())

	out += fmt.Sprintf("# HELP datacenters_total Total number of datacenters\n")
	out += fmt.Sprintf("# TYPE datacenters_total gauge\n")
	out += fmt.Sprintf("datacenters_total %d\n", m.datacentersTotal.Load())

	out += fmt.Sprintf("# HELP discovery_scans_total Total number of discovery scans\n")
	out += fmt.Sprintf("# TYPE discovery_scans_total counter\n")
	out += fmt.Sprintf("discovery_scans_total %d\n", m.discoveryScans.Load())

	out += fmt.Sprintf("# HELP discovery_scan_duration_milliseconds Discovery scan duration in milliseconds\n")
	out += fmt.Sprintf("# TYPE discovery_scan_duration_milliseconds summary\n")
	m.discoveryDuration.Range(func(key, value interface{}) bool {
		h := value.(*histogram)
		out += fmt.Sprintf("discovery_scan_duration_milliseconds{type=\"%s\"} %.2f\n", key, h.avg())
		return true
	})

	// Database metrics
	out += fmt.Sprintf("# HELP db_queries_total Total number of database queries\n")
	out += fmt.Sprintf("# TYPE db_queries_total counter\n")
	out += fmt.Sprintf("db_queries_total %d\n", m.dbQueriesTotal.Load())

	out += fmt.Sprintf("# HELP db_query_duration_milliseconds Database query duration in milliseconds\n")
	out += fmt.Sprintf("# TYPE db_query_duration_milliseconds summary\n")
	m.dbQueryDuration.Range(func(key, value interface{}) bool {
		h := value.(*histogram)
		out += fmt.Sprintf("db_query_duration_milliseconds{query=\"%s\"} %.2f\n", key, h.avg())
		return true
	})

	out += fmt.Sprintf("# HELP db_connections_open Current number of open database connections\n")
	out += fmt.Sprintf("# TYPE db_connections_open gauge\n")
	out += fmt.Sprintf("db_connections_open %d\n", m.dbConnectionsOpen.Load())

	// Runtime metrics
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	out += fmt.Sprintf("# HELP process_uptime_seconds Process uptime in seconds\n")
	out += fmt.Sprintf("# TYPE process_uptime_seconds gauge\n")
	out += fmt.Sprintf("process_uptime_seconds %.0f\n", time.Since(m.startTime).Seconds())

	out += fmt.Sprintf("# HELP go_goroutines Number of goroutines\n")
	out += fmt.Sprintf("# TYPE go_goroutines gauge\n")
	out += fmt.Sprintf("go_goroutines %d\n", runtime.NumGoroutine())

	out += fmt.Sprintf("# HELP go_memory_alloc_bytes Bytes of allocated heap objects\n")
	out += fmt.Sprintf("# TYPE go_memory_alloc_bytes gauge\n")
	out += fmt.Sprintf("go_memory_alloc_bytes %d\n", mem.Alloc)

	out += fmt.Sprintf("# HELP go_memory_sys_bytes Total bytes of memory obtained from the OS\n")
	out += fmt.Sprintf("# TYPE go_memory_sys_bytes gauge\n")
	out += fmt.Sprintf("go_memory_sys_bytes %d\n", mem.Sys)

	return out
}
