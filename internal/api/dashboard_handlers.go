package api

import (
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

// getDashboardStats returns aggregated dashboard statistics
func (h *Handler) getDashboardStats(w http.ResponseWriter, r *http.Request) {
	staleDays := parseIntParam(r, "stale_days", 7)
	if staleDays < 1 {
		staleDays = 1
	} else if staleDays > 365 {
		staleDays = 365
	}
	recentLimit := parseIntParam(r, "recent_limit", 10)
	if recentLimit < 1 {
		recentLimit = 1
	} else if recentLimit > 100 {
		recentLimit = 100
	}

	stats, err := h.svc.Dashboard.GetStats(r.Context(), staleDays, recentLimit)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, stats)
}

// getUtilizationTrend returns utilization trend data for charts
func (h *Handler) getUtilizationTrend(w http.ResponseWriter, r *http.Request) {
	resourceType := model.SnapshotType(r.URL.Query().Get("type"))
	if resourceType != model.SnapshotTypeNetwork && resourceType != model.SnapshotTypePool {
		resourceType = model.SnapshotTypeNetwork
	}

	resourceID := r.URL.Query().Get("resource_id")
	if resourceID == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_RESOURCE_ID", "resource_id parameter is required")
		return
	}

	days := parseIntParam(r, "days", 30)
	if days < 1 {
		days = 1
	} else if days > 365 {
		days = 365
	}

	trend, err := h.svc.Dashboard.GetUtilizationTrend(r.Context(), resourceType, resourceID, days)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, trend)
}
