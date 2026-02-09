package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

// listAuditLogs handles GET /api/audit
func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	filter := &model.AuditFilter{
		Resource:   r.URL.Query().Get("resource"),
		ResourceID: r.URL.Query().Get("resource_id"),
		UserID:     r.URL.Query().Get("user_id"),
		Action:     r.URL.Query().Get("action"),
		Limit:      100,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	if start := r.URL.Query().Get("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			filter.StartTime = &t
		}
	}

	if end := r.URL.Query().Get("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			filter.EndTime = &t
		}
	}

	logs, err := h.svc.Audit.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, logs)
}

// getAuditLog handles GET /api/audit/{id}
func (h *Handler) getAuditLog(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	log, err := h.svc.Audit.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, log)
}

// exportAuditLogs handles GET /api/audit/export
func (h *Handler) exportAuditLogs(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	filter := &model.AuditFilter{
		Resource:   r.URL.Query().Get("resource"),
		ResourceID: r.URL.Query().Get("resource_id"),
		UserID:     r.URL.Query().Get("user_id"),
		Action:     r.URL.Query().Get("action"),
	}

	if start := r.URL.Query().Get("start_time"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			filter.StartTime = &t
		}
	}

	if end := r.URL.Query().Get("end_time"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			filter.EndTime = &t
		}
	}

	data, err := h.svc.Audit.Export(r.Context(), filter, format)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	var contentType string
	var filename string

	switch format {
	case "csv":
		contentType = "text/csv"
		filename = "audit-logs.csv"
	default:
		contentType = "application/json"
		filename = "audit-logs.json"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Write(data)
}
