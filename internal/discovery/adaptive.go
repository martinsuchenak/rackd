package discovery

import (
	"context"
	"net"
	"time"
)

type AdaptiveScanner struct {
	defaultTimeout time.Duration
	defaultWorkers int
	minTimeout     time.Duration
	maxTimeout     time.Duration
	minWorkers     int
	maxWorkers     int
}

type ScanParameters struct {
	Timeout       time.Duration
	Workers       int
	PriorityPorts []int
	EnableCaching bool
}

func NewAdaptiveScanner(defaultTimeout time.Duration, defaultWorkers int) *AdaptiveScanner {
	return &AdaptiveScanner{
		defaultTimeout: defaultTimeout,
		defaultWorkers: defaultWorkers,
		minTimeout:     500 * time.Millisecond,
		maxTimeout:     10 * time.Second,
		minWorkers:     1,
		maxWorkers:     100,
	}
}

func (s *AdaptiveScanner) CalculateParameters(subnet string, scanType string) *ScanParameters {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return s.defaultParameters()
	}

	numHosts := countHosts(ipNet)

	timeout := s.calculateTimeout(numHosts, scanType)
	workers := s.calculateWorkers(numHosts, scanType)
	priorityPorts := s.getPriorityPorts(scanType)

	return &ScanParameters{
		Timeout:       timeout,
		Workers:       workers,
		PriorityPorts: priorityPorts,
		EnableCaching: numHosts > 1000,
	}
}

func (s *AdaptiveScanner) calculateTimeout(numHosts int, scanType string) time.Duration {
	baseTimeout := s.defaultTimeout

	switch scanType {
	case "quick":
		baseTimeout = 1 * time.Second
	case "full":
		baseTimeout = 2 * time.Second
	case "deep":
		baseTimeout = 3 * time.Second
	}

	if numHosts < 256 {
		return baseTimeout
	}

	factor := float64(numHosts) / 256.0
	if factor > 5 {
		factor = 5
	}

	timeout := time.Duration(float64(baseTimeout) * factor)

	if timeout < s.minTimeout {
		timeout = s.minTimeout
	}
	if timeout > s.maxTimeout {
		timeout = s.maxTimeout
	}

	return timeout
}

func (s *AdaptiveScanner) calculateWorkers(numHosts int, scanType string) int {
	baseWorkers := s.defaultWorkers

	switch scanType {
	case "quick":
		baseWorkers = 20
	case "full":
		baseWorkers = 50
	case "deep":
		baseWorkers = 100
	}

	if numHosts < 256 {
		return baseWorkers
	}

	factor := int(numHosts / 256)
	if factor > 10 {
		factor = 10
	}

	workers := baseWorkers * (factor + 1)

	if workers < s.minWorkers {
		workers = s.minWorkers
	}
	if workers > s.maxWorkers {
		workers = s.maxWorkers
	}

	return workers
}

func (s *AdaptiveScanner) getPriorityPorts(scanType string) []int {
	switch scanType {
	case "quick":
		return []int{22, 80, 443, 3389}
	case "full":
		return []int{21, 22, 23, 25, 53, 80, 110, 143, 443, 445, 993, 995, 3306, 3389, 5432, 8080}
	case "deep":
		return getTop100Ports()
	default:
		return []int{22, 80, 443, 3389}
	}
}

func (s *AdaptiveScanner) defaultParameters() *ScanParameters {
	return &ScanParameters{
		Timeout:       s.defaultTimeout,
		Workers:       s.defaultWorkers,
		PriorityPorts: []int{22, 80, 443, 3389},
		EnableCaching: false,
	}
}

func (s *AdaptiveScanner) MeasureLatency(ctx context.Context, ip string) time.Duration {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", ip+":80", 1*time.Second)
	if err != nil {
		return 1 * time.Second
	}
	defer conn.Close()
	latency := time.Since(start)
	return latency
}

func (s *AdaptiveScanner) AdjustTimeoutByLatency(baseTimeout time.Duration, latency time.Duration) time.Duration {
	if latency < 100*time.Millisecond {
		return baseTimeout
	}

	factor := latency.Seconds() / 0.1
	if factor > 3 {
		factor = 3
	}

	adjusted := time.Duration(float64(baseTimeout) * factor)
	if adjusted < s.minTimeout {
		adjusted = s.minTimeout
	}
	if adjusted > s.maxTimeout {
		adjusted = s.maxTimeout
	}

	return adjusted
}

type ScanMetrics struct {
	TotalHosts       int
	ScannedHosts     int
	FoundHosts       int
	StartTime        time.Time
	EndTime          time.Time
	AverageLatency   time.Duration
	PacketsPerSecond float64
}

type MetricsCollector struct {
	latencies []time.Duration
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies: []time.Duration{},
	}
}

func (m *MetricsCollector) RecordLatency(latency time.Duration) {
	m.latencies = append(m.latencies, latency)
	if len(m.latencies) > 100 {
		m.latencies = m.latencies[1:]
	}
}

func (m *MetricsCollector) AverageLatency() time.Duration {
	if len(m.latencies) == 0 {
		return 0
	}

	var sum time.Duration
	for _, l := range m.latencies {
		sum += l
	}
	return sum / time.Duration(len(m.latencies))
}

func (m *MetricsCollector) CalculateMetrics(totalHosts, scannedHosts, foundHosts int, startTime, endTime time.Time) *ScanMetrics {
	duration := endTime.Sub(startTime)
	pps := 0.0
	if duration.Seconds() > 0 {
		pps = float64(scannedHosts) / duration.Seconds()
	}

	return &ScanMetrics{
		TotalHosts:       totalHosts,
		ScannedHosts:     scannedHosts,
		FoundHosts:       foundHosts,
		StartTime:        startTime,
		EndTime:          endTime,
		AverageLatency:   m.AverageLatency(),
		PacketsPerSecond: pps,
	}
}

type ResultCache struct {
	cache map[string]*CachedResult
}

type CachedResult struct {
	Device    interface{}
	Timestamp time.Time
	TTL       time.Duration
}

func NewResultCache() *ResultCache {
	return &ResultCache{
		cache: make(map[string]*CachedResult),
	}
}

func (c *ResultCache) Get(key string) (interface{}, bool) {
	result, ok := c.cache[key]
	if !ok {
		return nil, false
	}

	if time.Since(result.Timestamp) > result.TTL {
		delete(c.cache, key)
		return nil, false
	}

	return result.Device, true
}

func (c *ResultCache) Set(key string, device interface{}, ttl time.Duration) {
	c.cache[key] = &CachedResult{
		Device:    device,
		Timestamp: time.Now(),
		TTL:       ttl,
	}
}

func (c *ResultCache) Clear() {
	c.cache = make(map[string]*CachedResult)
}

func (c *ResultCache) PurgeExpired() {
	now := time.Now()
	for key, result := range c.cache {
		if now.Sub(result.Timestamp) > result.TTL {
			delete(c.cache, key)
		}
	}
}
