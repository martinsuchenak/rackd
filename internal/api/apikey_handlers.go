package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
)

// listAPIKeys lists all API keys
func (h *Handler) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	filter := &model.APIKeyFilter{Name: name}

	keys, err := h.store.ListAPIKeys(filter)
	if err != nil {
		h.internalError(w, err)
		return
	}

	// Convert to response format (hide keys)
	responses := make([]model.APIKeyResponse, len(keys))
	for i, key := range keys {
		responses[i] = key.ToResponse()
	}

	h.writeJSON(w, http.StatusOK, responses)
}

// createAPIKey creates a new API key
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

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_NAME", "Name is required")
		return
	}

	// Generate API key
	keyStr, err := auth.GenerateKey()
	if err != nil {
		h.internalError(w, err)
		return
	}

	key := &model.APIKey{
		Name:        req.Name,
		Key:         keyStr,
		Description: req.Description,
		CreatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
	}

	if err := h.store.CreateAPIKey(key); err != nil {
		h.internalError(w, err)
		return
	}

	log.Info("API key created", "name", key.Name, "id", key.ID)

	// Return the full key (including the actual key) only on creation
	h.writeJSON(w, http.StatusCreated, key)
}

// getAPIKey retrieves an API key by ID
func (h *Handler) getAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	key, err := h.store.GetAPIKey(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "API key not found")
		return
	}

	// Return response format (hide key)
	h.writeJSON(w, http.StatusOK, key.ToResponse())
}

// deleteAPIKey deletes an API key
func (h *Handler) deleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.store.DeleteAPIKey(id); err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "API key not found")
		return
	}

	log.Info("API key deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
