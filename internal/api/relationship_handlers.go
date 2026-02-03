package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

type addRelationshipRequest struct {
	ChildID string `json:"child_id"`
	Type    string `json:"type"`
	Notes   string `json:"notes"`
}

func (h *Handler) addRelationship(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue("id")
	var req addRelationshipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}
	if req.ChildID == "" || req.Type == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_FIELD", "child_id and type are required")
		return
	}
	if !isValidRelationshipType(req.Type) {
		h.writeError(w, http.StatusBadRequest, "INVALID_TYPE", "type must be contains, connected_to, or depends_on")
		return
	}
	if err := h.store.AddRelationship(parentID, req.ChildID, req.Type, req.Notes); err != nil {
		if errors.Is(err, storage.ErrDeviceNotFound) {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Device not found")
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (h *Handler) getRelationships(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	rels, err := h.store.GetRelationships(deviceID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rels)
}

func (h *Handler) getRelatedDevices(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	relType := r.URL.Query().Get("type")
	devices, err := h.store.GetRelatedDevices(deviceID, relType)
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) removeRelationship(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue("id")
	childID := r.PathValue("child_id")
	relType := r.PathValue("type")
	if err := h.store.RemoveRelationship(parentID, childID, relType); err != nil {
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type updateRelationshipNotesRequest struct {
	Notes string `json:"notes"`
}

func (h *Handler) updateRelationshipNotes(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue("id")
	childID := r.PathValue("child_id")
	relType := r.PathValue("type")
	
	var req updateRelationshipNotesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON body")
		return
	}
	
	if err := h.store.UpdateRelationshipNotes(parentID, childID, relType, req.Notes); err != nil {
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isValidRelationshipType(t string) bool {
	return t == model.RelationshipContains || t == model.RelationshipConnectedTo || t == model.RelationshipDependsOn
}
