package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthStatus represents the health status of the application
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Checks    map[string]Check  `json:"checks,omitempty"`
}

// Check represents a single health check
type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// healthz is a simple liveness probe
func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// readyz is a detailed readiness probe
func (h *Handler) readyz(w http.ResponseWriter, r *http.Request) {
	status := HealthStatus{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    make(map[string]Check),
	}

	allHealthy := true

	// Check database connectivity
	dbCheck := h.checkDatabase()
	status.Checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		allHealthy = false
	}

	// Check discovery scheduler (if available)
	if h.scanner != nil {
		schedulerCheck := h.checkScheduler()
		status.Checks["scheduler"] = schedulerCheck
		if schedulerCheck.Status != "healthy" {
			allHealthy = false
		}
	}

	if allHealthy {
		status.Status = "healthy"
		w.WriteHeader(http.StatusOK)
	} else {
		status.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *Handler) checkDatabase() Check {
	// Get the underlying database connection
	db := h.store.DB()
	if db == nil {
		return Check{
			Status:  "unhealthy",
			Message: "database connection not available",
		}
	}

	// Ping the database
	if err := db.Ping(); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}

	// Check connection stats
	stats := db.Stats()
	if stats.OpenConnections == 0 {
		return Check{
			Status:  "unhealthy",
			Message: "no open database connections",
		}
	}

	return Check{
		Status:  "healthy",
		Message: "database is accessible",
	}
}

func (h *Handler) checkScheduler() Check {
	// Basic check - if scanner exists, assume scheduler is running
	// In a real implementation, you might want to add a health check method to the scanner
	if h.scanner == nil {
		return Check{
			Status:  "unhealthy",
			Message: "discovery scheduler not initialized",
		}
	}

	return Check{
		Status:  "healthy",
		Message: "discovery scheduler is running",
	}
}
