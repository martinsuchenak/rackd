package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listLogs(w http.ResponseWriter, r *http.Request) {
	pg := parsePagination(r)
	filter := &model.LogFilter{
		Pagination: pg,
		Level:      r.URL.Query().Get("level"),
		Source:     r.URL.Query().Get("source"),
		Query:      r.URL.Query().Get("query"),
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
	w.Write(data)
}
