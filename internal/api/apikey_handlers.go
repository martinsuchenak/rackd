package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/pkg/model"
)

func (h *Handler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	filter := &model.APIKeyFilter{Name: name}
	filter.Pagination = parseOptionalPagination(r)

	keys, err := h.svc.APIKeys.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	responses := make([]model.APIKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = key.ToResponse()
	}

	h.writeJSON(w, http.StatusOK, responses)
}

func (h *Handler) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string     `json:"name"`
		Description string     `json:"description"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
		// A pointer, so an empty list (refused) differs from none (a key
		// with all of the owner's permissions).
		Scopes *[]string `json:"scopes,omitempty"`
	}

	// Unknown fields are refused, not ignored: a request that asks for
	// "permissions" or "role" mustn't silently get a full-access key.
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		if strings.HasPrefix(err.Error(), "json: unknown field") {
			h.badRequest(w, strings.TrimPrefix(err.Error(), "json: ")+"; use scopes to limit what the key can do")
			return
		}
		h.invalidJSON(w)
		return
	}

	newKey := &model.APIKey{
		Name:        req.Name,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	}
	if req.Scopes != nil {
		if len(*req.Scopes) == 0 {
			h.writeValidationErrors(w, ValidationErrors{{Field: "scopes", Message: "give at least one resource:action scope, or leave scopes out for a key with all your permissions"}})
			return
		}
		newKey.Scopes = *req.Scopes
	}

	plaintextKey, err := h.svc.APIKeys.Create(r.Context(), newKey)

	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	log.Info("API key created", "name", req.Name, "id", newKey.ID)

	response := newKey.ToResponse()
	// Expose the raw plaintext key in the response only on creation
	responseKey := struct {
		model.APIKeyResponse
		Key string `json:"key"`
	}{
		APIKeyResponse: response,
		Key:            plaintextKey,
	}

	h.writeJSON(w, http.StatusCreated, responseKey)
}

func (h *Handler) getAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	key, err := h.svc.APIKeys.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, key.ToResponse())
}

func (h *Handler) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.badRequest(w, "ID is required")
		return
	}
	if err := h.svc.APIKeys.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	log.Info("API key deleted", "id", id)

	w.WriteHeader(http.StatusNoContent)
}
