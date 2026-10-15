package api

import (
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/pkg/model"
)

func (h *Handler) listLogs(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	filter := &model.LogFilter{
		Pagination: pg,
		Level:      r.URL.Query().Get("level"),
		Source:     r.URL.Query().Get("source"),
		Query:      r.URL.Query().Get("query"),
	}

	var errs ValidationErrors
	if filter.StartTime, filter.EndTime, errs = timeRangeParams(r); errs != nil {
		h.writeValidationErrors(w, errs)
		return
	}

	entries, err := h.svc.Logs.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) getLogEntry(w http.ResponseWriter, r *http.Request) {
	entry, err := h.svc.Logs.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, entry)
}

func (h *Handler) exportLogs(w http.ResponseWriter, r *http.Request) {
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}

	filter := &model.LogFilter{
		Level:  r.URL.Query().Get("level"),
		Source: r.URL.Query().Get("source"),
		Query:  r.URL.Query().Get("query"),
	}

	var errs ValidationErrors
	if filter.StartTime, filter.EndTime, errs = timeRangeParams(r); errs != nil {
		h.writeValidationErrors(w, errs)
		return
	}

	data, err := h.svc.Logs.Export(r.Context(), filter, format)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=recent-logs.csv")
	default:
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=recent-logs.json")
	}
	_, _ = w.Write(data)
}
