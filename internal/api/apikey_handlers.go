package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	filter := &model.APIKeyFilter{Name: name}

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
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	key, err := h.svc.APIKeys.Create(r.Context(), &model.APIKey{
		Name:        req.Name,
		Description: req.Description,
		ExpiresAt:   req.ExpiresAt,
	})

	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	log.Info("API key created", "name", req.Name, "id", key)

	h.writeJSON(w, http.StatusCreated, key)
}

func (h *Handler) getAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	key, err := h.svc.APIKeys.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, key.ToResponse())
}

func (h *Handler) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.APIKeys.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	log.Info("API key deleted", "id", id)

	w.WriteHeader(http.StatusNoContent)
}
