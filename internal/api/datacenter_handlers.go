package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listDatacenters(w http.ResponseWriter, r *http.Request) {
	filter := &model.DatacenterFilter{
		Name: r.URL.Query().Get("name"),
	}

	dcs, err := h.svc.Datacenters.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, dcs)
}

func (h *Handler) createDatacenter(w http.ResponseWriter, r *http.Request) {
	var dc model.Datacenter
	if err := json.NewDecoder(r.Body).Decode(&dc); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	if errs := ValidateDatacenter(&dc); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}

	if err := h.svc.Datacenters.Create(r.Context(), &dc); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, dc)
}

func (h *Handler) getDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	dc, err := h.svc.Datacenters.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, dc)
}

func (h *Handler) updateDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	dc, err := h.svc.Datacenters.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
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

	if errs := ValidateDatacenter(dc); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}

	if err := h.svc.Datacenters.Update(r.Context(), dc); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, dc)
}

func (h *Handler) deleteDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	if err := h.svc.Datacenters.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getDatacenterDevices(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	devices, err := h.svc.Datacenters.GetDevices(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) searchDatacenters(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_QUERY", "query parameter 'q' is required")
		return
	}

	dcs, err := h.svc.Datacenters.Search(r.Context(), query)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, dcs)
}
