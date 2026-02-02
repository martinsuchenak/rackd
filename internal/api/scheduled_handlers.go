package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func (h *Handler) listScheduledScans(w http.ResponseWriter, r *http.Request) {
	networkID := r.URL.Query().Get("network_id")
	scans, err := h.scheduledStore.List(networkID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scans)
}

func (h *Handler) createScheduledScan(w http.ResponseWriter, r *http.Request) {
	var scan model.ScheduledScan
	if err := json.NewDecoder(r.Body).Decode(&scan); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if err := h.scheduledStore.Create(&scan); err != nil {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, scan)
}

func (h *Handler) getScheduledScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	scan, err := h.scheduledStore.Get(id)
	if err == storage.ErrScheduledScanNotFound {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "scheduled scan not found")
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) updateScheduledScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var scan model.ScheduledScan
	if err := json.NewDecoder(r.Body).Decode(&scan); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	scan.ID = id
	if err := h.scheduledStore.Update(&scan); err != nil {
		if err == storage.ErrScheduledScanNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "scheduled scan not found")
			return
		}
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, scan)
}

func (h *Handler) deleteScheduledScan(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.scheduledStore.Delete(id); err != nil {
		if err == storage.ErrScheduledScanNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "scheduled scan not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
