package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
	datacenterID := r.URL.Query().Get("datacenter_id")
	creds, err := h.svc.Credentials.List(r.Context(), datacenterID)
	if err != nil {
		h.handleServiceError(w, err)
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
	cred, err := h.svc.Credentials.Create(r.Context(), &input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, cred.ToResponse())
}

func (h *Handler) getCredential(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cred, err := h.svc.Credentials.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
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
	cred, err := h.svc.Credentials.Update(r.Context(), id, &input)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, cred.ToResponse())
}

func (h *Handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.Credentials.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
