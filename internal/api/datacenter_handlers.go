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

	if h.svc != nil && h.svc.Datacenters != nil {
		dcs, err := h.svc.Datacenters.List(r.Context(), filter)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, dcs)
		return
	}

	datacenters, err := h.store.ListDatacenters(filter)
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

	if h.svc != nil && h.svc.Datacenters != nil {
		if err := h.svc.Datacenters.Create(r.Context(), &dc); err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusCreated, dc)
		return
	}

	if errs := ValidateDatacenter(&dc); len(errs) > 0 {
		h.writeValidationErrors(w, errs)
		return
	}
	if err := h.store.CreateDatacenter(h.auditContext(r), &dc); err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, dc)
}

func (h *Handler) getDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Datacenters != nil {
		dc, err := h.svc.Datacenters.Get(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, dc)
		return
	}

	dc, err := h.store.GetDatacenter(id)
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

	// Fetch through service so RBAC is enforced on the read too
	var dc *model.Datacenter
	var err error
	if h.svc != nil && h.svc.Datacenters != nil {
		dc, err = h.svc.Datacenters.Get(r.Context(), id)
	} else {
		dc, err = h.store.GetDatacenter(id)
	}
	if err != nil {
		if h.svc != nil && h.svc.Datacenters != nil {
			h.handleServiceError(w, err)
		} else if errors.Is(err, storage.ErrDatacenterNotFound) {
			h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Datacenter not found")
		} else {
			h.internalError(w, err)
		}
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

	if h.svc != nil && h.svc.Datacenters != nil {
		if err := h.svc.Datacenters.Update(r.Context(), dc); err != nil {
			h.handleServiceError(w, err)
			return
		}
	} else {
		if errs := ValidateDatacenter(dc); len(errs) > 0 {
			h.writeValidationErrors(w, errs)
			return
		}
		if err := h.store.UpdateDatacenter(h.auditContext(r), dc); err != nil {
			h.internalError(w, err)
			return
		}
	}
	h.writeJSON(w, http.StatusOK, dc)
}

func (h *Handler) deleteDatacenter(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if h.svc != nil && h.svc.Datacenters != nil {
		if err := h.svc.Datacenters.Delete(r.Context(), id); err != nil {
			h.handleServiceError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := h.store.DeleteDatacenter(h.auditContext(r), id); err != nil {
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

	if h.svc != nil && h.svc.Datacenters != nil {
		devices, err := h.svc.Datacenters.GetDevices(r.Context(), id)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, devices)
		return
	}

	if _, err := h.store.GetDatacenter(id); err != nil {
		if errors.Is(err, storage.ErrDatacenterNotFound) {
			h.writeError(w, http.StatusNotFound, "DATACENTER_NOT_FOUND", "Datacenter not found")
			return
		}
		h.internalError(w, err)
		return
	}
	devices, err := h.store.GetDatacenterDevices(id)
	if err != nil {
		h.internalError(w, err)
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

	if h.svc != nil && h.svc.Datacenters != nil {
		dcs, err := h.svc.Datacenters.Search(r.Context(), query)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}
		h.writeJSON(w, http.StatusOK, dcs)
		return
	}

	datacenters, err := h.store.SearchDatacenters(query)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, datacenters)
}
