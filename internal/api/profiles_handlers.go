package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.svc.ScanProfiles.List(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, profiles)
}

func (h *Handler) createProfile(w http.ResponseWriter, r *http.Request) {
	var profile model.ScanProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if err := h.svc.ScanProfiles.Create(r.Context(), &profile); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, profile)
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	profile, err := h.svc.ScanProfiles.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var profile model.ScanProfile
	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if err := h.svc.ScanProfiles.Update(r.Context(), id, &profile); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) deleteProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.ScanProfiles.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
