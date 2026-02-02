package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/credentials"
	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
	datacenterID := r.URL.Query().Get("datacenter_id")
	creds, err := h.credStore.List(datacenterID)
	if err != nil {
		h.internalError(w, err)
		return
	}
	responses := make([]model.CredentialResponse, len(creds))
	for i, c := range creds {
		responses[i] = c.ToResponse()
	}
	h.writeJSON(w, http.StatusOK, responses)
}

func (h *Handler) createCredential(w http.ResponseWriter, r *http.Request) {
	var input model.CredentialInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	cred := input.ToCredential()
	if err := h.credStore.Create(cred); err != nil {
		if err == credentials.ErrInvalidCredential {
			h.writeError(w, http.StatusBadRequest, "INVALID_CREDENTIAL", err.Error())
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, cred.ToResponse())
}

func (h *Handler) getCredential(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cred, err := h.credStore.Get(id)
	if err == credentials.ErrCredentialNotFound {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "credential not found")
		return
	}
	if err != nil {
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, cred.ToResponse())
}

func (h *Handler) updateCredential(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var input model.CredentialInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	cred := input.ToCredential()
	cred.ID = id
	if err := h.credStore.Update(cred); err != nil {
		if err == credentials.ErrCredentialNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "credential not found")
			return
		}
		if err == credentials.ErrInvalidCredential {
			h.writeError(w, http.StatusBadRequest, "INVALID_CREDENTIAL", err.Error())
			return
		}
		h.internalError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, cred.ToResponse())
}

func (h *Handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.credStore.Delete(id); err != nil {
		if err == credentials.ErrCredentialNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "credential not found")
			return
		}
		h.internalError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
