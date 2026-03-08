package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listScheduledScans(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")
	scans, err := h.svc.ScheduledScans.List(r.Context(), networkID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scans)
}

func (h *Handler) createScheduledScan(w http.ResponseWriter, r *http.Request) {
	var scan model.ScheduledScan
	if err := json.NewDecoder(r.Body).Decode(&scan); err != nil {
		h.invalidJSON(w)
		return
	}
	if err := h.svc.ScheduledScans.Create(r.Context(), &scan); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, scan)
}

func (h *Handler) getScheduledScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scan, err := h.svc.ScheduledScans.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) updateScheduledScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var scan model.ScheduledScan
	if err := json.NewDecoder(r.Body).Decode(&scan); err != nil {
		h.invalidJSON(w)
		return
	}
	if err := h.svc.ScheduledScans.Update(r.Context(), id, &scan); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) deleteScheduledScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.ScheduledScans.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
