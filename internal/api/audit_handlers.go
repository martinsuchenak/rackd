package api

import (
	"net/http"

	"github.com/martinsuchenak/rackd/pkg/model"
)

// listAuditLogs handles GET /api/v1/audit
func (h *Handler) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	filter := &model.AuditFilter{
		Pagination: pg,
		Resource:   r.URL.Query().Get("resource"),
		ResourceID: r.URL.Query().Get("resource_id"),
		UserID:     r.URL.Query().Get("user_id"),
		Action:     r.URL.Query().Get("action"),
		Source:     r.URL.Query().Get("source"),
	}

	var errs ValidationErrors
	if filter.StartTime, filter.EndTime, errs = timeRangeParams(r); errs != nil {
		h.writeValidationErrors(w, errs)
		return
	}

	logs, err := h.svc.Audit.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, logs)
}

// getAuditLog handles GET /api/v1/audit/{id}
func (h *Handler) getAuditLog(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	log, err := h.svc.Audit.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, log)
}

// exportAuditLogs handles GET /api/v1/audit/export
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
		Source:     r.URL.Query().Get("source"),
	}

	var errs ValidationErrors
	if filter.StartTime, filter.EndTime, errs = timeRangeParams(r); errs != nil {
		h.writeValidationErrors(w, errs)
		return
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
	_, _ = w.Write(data)
}
