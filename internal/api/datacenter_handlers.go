package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func (h *Handler) listDatacenters(w http.ResponseWriter, r *http.Request) {
	filter := &model.DatacenterFilter{
		Name: r.URL.Query().Get("name"),
	}
	datacenters, err := h.storage.ListDatacenters(filter)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, datacenters)
}

func (h *Handler) createDatacenter(w http.ResponseWriter, r *http.Request) {
	var dc model.Datacenter
	if err := json.NewDecoder(r.Body).Decode(&dc); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}
	if dc.Name == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Name is required")
		return
	}
	if err := h.storage.CreateDatacenter(&dc); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, dc)
}

func (h *Handler) getDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dc, err := h.storage.GetDatacenter(id)
	if err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Datacenter not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, dc)
}

func (h *Handler) updateDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	dc, err := h.storage.GetDatacenter(id)
	if err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Datacenter not found")
			return
		}
		h.internalError(w, err)
		return
	}

	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	if name, ok := updates["name"].(string); ok {
		dc.Name = name
	}
	if location, ok := updates["location"].(string); ok {
		dc.Location = location
	}
	if description, ok := updates["description"].(string); ok {
		dc.Description = description
	}

	if err := h.storage.UpdateDatacenter(dc); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, dc)
}

func (h *Handler) deleteDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.storage.DeleteDatacenter(id); err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Datacenter not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getDatacenterDevices(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := h.storage.GetDatacenter(id); err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Datacenter not found")
			return
		}
		h.internalError(w, err)
		return
	}
	devices, err := h.storage.GetDatacenterDevices(id)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}
