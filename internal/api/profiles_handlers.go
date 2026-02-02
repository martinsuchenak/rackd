package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func (h *Handler) listProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.profileStore.List()
	if err != nil {
		h.internalError(w, err)
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
	if err := h.profileStore.Create(&profile); err != nil {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, profile)
}

func (h *Handler) getProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	profile, err := h.profileStore.Get(id)
	if err == storage.ErrProfileNotFound {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "profile not found")
		return
	}
	if err != nil {
		h.internalError(w, err)
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
	profile.ID = id
	if err := h.profileStore.Update(&profile); err != nil {
		if err == storage.ErrProfileNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	h.writeJSON(w, http.StatusOK, profile)
}

func (h *Handler) deleteProfile(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.profileStore.Delete(id); err != nil {
		if err == storage.ErrProfileNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
