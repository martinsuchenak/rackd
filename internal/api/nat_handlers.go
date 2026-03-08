package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

// listNATMappings returns all NAT mappings
func (h *Handler) listNATMappings(w http.ResponseWriter, r *http.Request) {
	filter := &model.NATFilter{
		ExternalIP:   r.URL.Query().Get("external_ip"),
		InternalIP:   r.URL.Query().Get("internal_ip"),
		DeviceID:     r.URL.Query().Get("device_id"),
		DatacenterID: r.URL.Query().Get("datacenter_id"),
		NetworkID:    r.URL.Query().Get("network_id"),
	}
	if protocol := r.URL.Query().Get("protocol"); protocol != "" {
		filter.Protocol = model.NATProtocol(protocol)
	}
	if enabled := r.URL.Query().Get("enabled"); enabled != "" {
		enabledBool := enabled == "true"
		filter.Enabled = &enabledBool
	}

	mappings, err := h.svc.NAT.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Ensure we return an empty array, not null
	if mappings == nil {
		mappings = []model.NATMapping{}
	}
	h.writeJSON(w, http.StatusOK, mappings)
}

// getNATMapping returns a single NAT mapping by ID
func (h *Handler) getNATMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	mapping, err := h.svc.NAT.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, mapping)
}

// createNATMapping creates a new NAT mapping
func (h *Handler) createNATMapping(w http.ResponseWriter, r *http.Request) {
	var req model.CreateNATRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	mapping, err := h.svc.NAT.Create(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, mapping)
}

// updateNATMapping updates an existing NAT mapping
func (h *Handler) updateNATMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req model.UpdateNATRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	mapping, err := h.svc.NAT.Update(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, mapping)
}

// deleteNATMapping deletes a NAT mapping
func (h *Handler) deleteNATMapping(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.NAT.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
