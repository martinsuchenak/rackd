package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

type addRelationshipRequest struct {
	ChildID string `json:"child_id"`
	Type    string `json:"type"`
	Notes   string `json:"notes"`
}

func (h *Handler) listAllRelationships(w http.ResponseWriter, r *http.Request) {
	rels, err := h.svc.Relationships.ListAll(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rels)
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

	if err := h.svc.Relationships.Add(r.Context(), parentID, req.ChildID, req.Type, req.Notes); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (h *Handler) getRelationships(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")

	rels, err := h.svc.Relationships.Get(r.Context(), deviceID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rels)
}

func (h *Handler) getRelatedDevices(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")
	relType := r.URL.Query().Get("type")

	devices, err := h.svc.Relationships.GetRelated(r.Context(), deviceID, relType)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, devices)
}

func (h *Handler) removeRelationship(w http.ResponseWriter, r *http.Request) {
	parentID := r.PathValue("id")
	childID := r.PathValue("child_id")
	relType := r.PathValue("type")

	if err := h.svc.Relationships.Remove(r.Context(), parentID, childID, relType); err != nil {
		h.handleServiceError(w, err)
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

	if err := h.svc.Relationships.UpdateNotes(r.Context(), parentID, childID, relType, req.Notes); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func isValidRelationshipType(t string) bool {
	return t == model.RelationshipContains || t == model.RelationshipConnectedTo || t == model.RelationshipDependsOn
}
