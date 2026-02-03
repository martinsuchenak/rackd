package api

import (
	"net/http"

	"github.com/martinsuchenak/rackd/internal/metrics"
)

// metricsHandler serves Prometheus-compatible metrics
func (h *Handler) metricsHandler(w http.ResponseWriter, r *http.Request) {
	// Update current counts
	h.updateMetricsCounts()

	// Export metrics
	m := metrics.Get()
	output := m.Export()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

// updateMetricsCounts updates gauge metrics with current counts
func (h *Handler) updateMetricsCounts() {
	m := metrics.Get()

	// Get device count
	devices, err := h.store.ListDevices(nil)
	if err == nil {
		m.SetDeviceCount(int64(len(devices)))
	}

	// Get network count
	networks, err := h.store.ListNetworks(nil)
	if err == nil {
		m.SetNetworkCount(int64(len(networks)))
	}

	// Get datacenter count
	datacenters, err := h.store.ListDatacenters(nil)
	if err == nil {
		m.SetDatacenterCount(int64(len(datacenters)))
	}

	// Get database connection stats
	if db := h.store.DB(); db != nil {
		stats := db.Stats()
		m.SetDBConnectionsOpen(int64(stats.OpenConnections))
	}
}
