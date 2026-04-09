package discovery

import (
	"context"
	"testing"
	"time"
)

func TestAdaptiveScannerDefaultAndLatencyHelpers(t *testing.T) {
	scanner := NewAdaptiveScanner(1500*time.Millisecond, 12)

	params := scanner.CalculateParameters("bad-cidr", "quick")
	if params.Timeout != 1500*time.Millisecond || params.Workers != 12 {
		t.Fatalf("unexpected default parameters: %+v", params)
	}

	if adjusted := scanner.AdjustTimeoutByLatency(time.Second, 50*time.Millisecond); adjusted != time.Second {
		t.Fatalf("expected low latency to keep timeout, got %v", adjusted)
	}
	if adjusted := scanner.AdjustTimeoutByLatency(time.Second, 2*time.Second); adjusted > scanner.maxTimeout {
		t.Fatalf("expected adjusted timeout to cap at maxTimeout, got %v", adjusted)
	}

	latency := scanner.MeasureLatency(context.Background(), "203.0.113.1")
	if latency <= 0 {
		t.Fatalf("expected positive latency result, got %v", latency)
	}
}

func TestMetricsCollectorAndResultCache(t *testing.T) {
	collector := NewMetricsCollector()
	for i := 0; i < 105; i++ {
		collector.RecordLatency(10 * time.Millisecond)
	}
	if avg := collector.AverageLatency(); avg != 10*time.Millisecond {
		t.Fatalf("unexpected average latency: %v", avg)
	}

	start := time.Now().Add(-10 * time.Second)
	end := time.Now()
	metrics := collector.CalculateMetrics(50, 40, 5, start, end)
	if metrics.TotalHosts != 50 || metrics.ScannedHosts != 40 || metrics.AverageLatency == 0 {
		t.Fatalf("unexpected calculated metrics: %+v", metrics)
	}

	cache := NewResultCache()
	cache.Set("live", "device-1", time.Minute)
	if value, ok := cache.Get("live"); !ok || value.(string) != "device-1" {
		t.Fatalf("expected cached result, got value=%v ok=%v", value, ok)
	}

	cache.cache["expired"] = &CachedResult{
		Device:    "device-2",
		Timestamp: time.Now().Add(-2 * time.Minute),
		TTL:       time.Second,
	}
	if _, ok := cache.Get("expired"); ok {
		t.Fatal("expected expired cache entry to be evicted on get")
	}

	cache.cache["expired2"] = &CachedResult{
		Device:    "device-3",
		Timestamp: time.Now().Add(-2 * time.Minute),
		TTL:       time.Second,
	}
	cache.PurgeExpired()
	if _, ok := cache.cache["expired2"]; ok {
		t.Fatal("expected PurgeExpired to remove stale entry")
	}

	cache.Clear()
	if len(cache.cache) != 0 {
		t.Fatalf("expected cache to be cleared, got %d entries", len(cache.cache))
	}
}
