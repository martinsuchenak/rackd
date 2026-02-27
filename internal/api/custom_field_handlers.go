package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

// normalizeCustomFieldDefinition ensures options is never null
func normalizeCustomFieldDefinition(def *model.CustomFieldDefinition) {
	if def.Options == nil {
		def.Options = []string{}
	}
}

// normalizeCustomFieldDefinitions ensures options is never null for all definitions
func normalizeCustomFieldDefinitions(defs []model.CustomFieldDefinition) {
	for i := range defs {
		if defs[i].Options == nil {
			defs[i].Options = []string{}
		}
	}
}

// listCustomFieldDefinitions returns all custom field definitions
func (h *Handler) listCustomFieldDefinitions(w http.ResponseWriter, r *http.Request) {
	filter := &model.CustomFieldDefinitionFilter{}
	if typeStr := r.URL.Query().Get("type"); typeStr != "" {
		filter.Type = typeStr
	}

	definitions, err := h.svc.CustomFields.ListDefinitions(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	// Ensure we return an empty array, not null
	if definitions == nil {
		definitions = []model.CustomFieldDefinition{}
	}
	normalizeCustomFieldDefinitions(definitions)
	h.writeJSON(w, http.StatusOK, definitions)
}

// getCustomFieldDefinition returns a single custom field definition by ID
func (h *Handler) getCustomFieldDefinition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	def, err := h.svc.CustomFields.GetDefinition(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	normalizeCustomFieldDefinition(def)
	h.writeJSON(w, http.StatusOK, def)
}

// createCustomFieldDefinition creates a new custom field definition
func (h *Handler) createCustomFieldDefinition(w http.ResponseWriter, r *http.Request) {
	var req model.CreateCustomFieldDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	def, err := h.svc.CustomFields.CreateDefinition(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	normalizeCustomFieldDefinition(def)
	h.writeJSON(w, http.StatusCreated, def)
}

// updateCustomFieldDefinition updates an existing custom field definition
func (h *Handler) updateCustomFieldDefinition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req model.UpdateCustomFieldDefinitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "Invalid JSON")
		return
	}

	def, err := h.svc.CustomFields.UpdateDefinition(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	normalizeCustomFieldDefinition(def)
	h.writeJSON(w, http.StatusOK, def)
}

// deleteCustomFieldDefinition deletes a custom field definition
func (h *Handler) deleteCustomFieldDefinition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.CustomFields.DeleteDefinition(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{
		"message": "Custom field definition deleted successfully",
	})
}

// getCustomFieldTypes returns all available custom field types
func (h *Handler) getCustomFieldTypes(w http.ResponseWriter, r *http.Request) {
	types := []map[string]string{
		{"value": string(model.CustomFieldTypeText), "label": "Text"},
		{"value": string(model.CustomFieldTypeNumber), "label": "Number"},
		{"value": string(model.CustomFieldTypeBool), "label": "Boolean"},
		{"value": string(model.CustomFieldTypeSelect), "label": "Select"},
	}
	h.writeJSON(w, http.StatusOK, types)
}
