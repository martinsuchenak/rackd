package api

import (
	"context"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/metrics"
)

// metricsHandler serves Prometheus-compatible metrics
func (h *Handler) metricsHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Authorize(r.Context(), "metrics", "read"); err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Update current counts
	h.updateMetricsCounts(r.Context())

	// Export metrics
	m := metrics.Get()
	output := m.Export()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(output))
}

// updateMetricsCounts updates gauge metrics with current counts
func (h *Handler) updateMetricsCounts(ctx context.Context) {
	m := metrics.Get()

	// Counts only: listing the records would load every device on each scrape.
	if counts, err := h.store.CountInventory(ctx); err == nil {
		m.SetDeviceCount(counts.Devices)
		m.SetNetworkCount(counts.Networks)
		m.SetDatacenterCount(counts.Datacenters)
	}

	// Get database connection stats
	if db := h.store.DB(); db != nil {
		stats := db.Stats()
		m.SetDBConnectionsOpen(int64(stats.OpenConnections))
	}
}
