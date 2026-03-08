package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

// listCircuits returns all circuits
func (h *Handler) listCircuits(w http.ResponseWriter, r *http.Request) {
	filter := &model.CircuitFilter{
		Provider:     r.URL.Query().Get("provider"),
		DatacenterID: r.URL.Query().Get("datacenter_id"),
		Type:         r.URL.Query().Get("type"),
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = model.CircuitStatus(status)
	}

	circuits, err := h.svc.Circuits.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Ensure we return an empty array, not null
	if circuits == nil {
		circuits = []model.Circuit{}
	}
	h.writeJSON(w, http.StatusOK, circuits)
}

// getCircuit returns a single circuit by ID
func (h *Handler) getCircuit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	circuit, err := h.svc.Circuits.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, circuit)
}

// createCircuit creates a new circuit
func (h *Handler) createCircuit(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCircuitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	circuit, err := h.svc.Circuits.Create(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, circuit)
}

// updateCircuit updates an existing circuit
func (h *Handler) updateCircuit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req model.UpdateCircuitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	circuit, err := h.svc.Circuits.Update(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, circuit)
}

// deleteCircuit deletes a circuit
func (h *Handler) deleteCircuit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Circuits.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
