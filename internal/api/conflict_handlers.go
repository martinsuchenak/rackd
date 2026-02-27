package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listConflicts(w http.ResponseWriter, r *http.Request) {
	filter := &model.ConflictFilter{}
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		filter.Type = model.ConflictType(typeStr)
	}
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		filter.Status = model.ConflictStatus(statusStr)
	}

	conflicts, err := h.svc.Conflicts.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, conflicts)
}

func (h *Handler) getConflict(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	conflict, err := h.svc.Conflicts.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, conflict)
}

func (h *Handler) resolveConflict(w http.ResponseWriter, r *http.Request) {
	var resolution model.ConflictResolution
	if err := json.NewDecoder(r.Body).Decode(&resolution); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	if resolution.ConflictID == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "conflict_id is required")
		return
	}

	if err := h.svc.Conflicts.Resolve(r.Context(), &resolution); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": "Conflict resolved successfully",
	})
}

func (h *Handler) detectConflicts(w http.ResponseWriter, r *http.Request) {
	conflictType := r.URL.Query().Get("type")

	var conflicts []model.Conflict

	if conflictType == "duplicate_ip" || conflictType == "" {
		// Detect duplicate IPs
		dupIPs, ipErr := h.svc.Conflicts.DetectDuplicateIPs(r.Context())
		if ipErr != nil {
			h.handleServiceError(w, ipErr)
			return
		}
		conflicts = append(conflicts, dupIPs...)
	}

	if conflictType == "overlapping_subnet" || conflictType == "" {
		// Detect overlapping subnets
		overlapSubnets, subnetErr := h.svc.Conflicts.DetectOverlappingSubnets(r.Context())
		if subnetErr != nil {
			h.handleServiceError(w, subnetErr)
			return
		}
		conflicts = append(conflicts, overlapSubnets...)
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"conflicts": conflicts,
		"count":     len(conflicts),
	})
}

func (h *Handler) getConflictSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.Conflicts.GetSummary(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"duplicate_ips":      summary["duplicate_ip"],
		"overlapping_subnets": summary["overlapping_subnet"],
		"total_active":       0, // Calculated from summary
	})
}

func (h *Handler) deleteConflict(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Conflicts.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": "Conflict deleted successfully",
	})
}

func (h *Handler) getDeviceConflicts(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("id")

	conflicts, err := h.svc.Conflicts.GetConflictsByDevice(r.Context(), deviceID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, conflicts)
}
