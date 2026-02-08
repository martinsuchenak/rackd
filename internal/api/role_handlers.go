package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	filter := &model.RoleFilter{}
	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = name
	}

	roles, err := h.store.ListRoles(r.Context(), filter)
	if err != nil {
		h.internalError(w, err)
		return
	}

	roleResponses := make([]model.RoleResponse, len(roles))
	for i, role := range roles {
		roleResponses[i] = role.ToResponse()
	}

	h.writeJSON(w, http.StatusOK, roleResponses)
}

func (h *Handler) getRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	role, err := h.store.GetRole(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Role not found")
		return
	}

	h.writeJSON(w, http.StatusOK, role.ToResponse())
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var req model.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.Name == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_NAME", "Name is required")
		return
	}

	existing, _ := h.store.GetRoleByName(r.Context(), req.Name)
	if existing != nil {
		h.writeError(w, http.StatusConflict, "ROLE_EXISTS", "Role with this name already exists")
		return
	}

	role := &model.Role{
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
	}

	if err := h.store.CreateRole(r.Context(), role); err != nil {
		h.internalError(w, err)
		return
	}

	if len(req.Permissions) > 0 {
		if err := h.store.SetRolePermissions(r.Context(), role.ID, req.Permissions); err != nil {
			h.internalError(w, err)
			return
		}
	}

	h.writeJSON(w, http.StatusCreated, role.ToResponse())
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	role, err := h.store.GetRole(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Role not found")
		return
	}

	if role.IsSystem {
		h.writeError(w, http.StatusBadRequest, "SYSTEM_ROLE", "Cannot modify system roles")
		return
	}

	var req model.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	role.Description = req.Description

	if err := h.store.UpdateRole(r.Context(), role); err != nil {
		h.internalError(w, err)
		return
	}

	if req.Permissions != nil {
		if err := h.store.SetRolePermissions(r.Context(), role.ID, req.Permissions); err != nil {
			h.internalError(w, err)
			return
		}
	}

	h.writeJSON(w, http.StatusOK, role.ToResponse())
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.store.DeleteRole(r.Context(), id); err != nil {
		if err.Error() == "cannot delete system role" {
			h.writeError(w, http.StatusBadRequest, "SYSTEM_ROLE", "Cannot delete system roles")
			return
		}
		h.internalError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getRolePermissions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	role, err := h.store.GetRole(r.Context(), id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "Role not found")
		return
	}

	permissions, err := h.store.GetRolePermissions(r.Context(), id)
	if err != nil {
		h.internalError(w, err)
		return
	}

	response := model.RoleWithPermissions{
		Role:        *role,
		Permissions: permissions,
	}

	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	filter := &model.PermissionFilter{}
	if resource := r.URL.Query().Get("resource"); resource != "" {
		filter.Resource = resource
	}
	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}

	permissions, err := h.store.ListPermissions(r.Context(), filter)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, permissions)
}

func (h *Handler) grantRoleToUser(w http.ResponseWriter, r *http.Request) {
	var req model.GrantRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.UserID == "" || req.RoleID == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "User ID and Role ID are required")
		return
	}

	if err := h.store.AssignRoleToUser(r.Context(), req.UserID, req.RoleID); err != nil {
		h.internalError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) revokeRoleFromUser(w http.ResponseWriter, r *http.Request) {
	var req model.RevokeRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.UserID == "" || req.RoleID == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_INPUT", "User ID and Role ID are required")
		return
	}

	if err := h.store.RemoveRoleFromUser(r.Context(), req.UserID, req.RoleID); err != nil {
		h.internalError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getUserRoles(w http.ResponseWriter, r *http.Request) {
	currentUserID := getUserIDFromContext(r)
	if currentUserID == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	requestedUserID := r.PathValue("id")
	if requestedUserID == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "User ID is required")
		return
	}

	// Users can view their own roles; admins can view anyone's
	if currentUserID != requestedUserID {
		isAdmin, err := h.store.HasPermission(r.Context(), currentUserID, "roles", "list")
		if err != nil {
			h.internalError(w, err)
			return
		}
		if !isAdmin {
			h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
			return
		}
	}

	roles, err := h.store.GetUserRoles(r.Context(), requestedUserID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, roles)
}

func (h *Handler) getUserPermissions(w http.ResponseWriter, r *http.Request) {
	currentUserID := getUserIDFromContext(r)
	if currentUserID == "" {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	requestedUserID := r.PathValue("id")
	if requestedUserID == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "User ID is required")
		return
	}

	// Users can view their own permissions; admins can view anyone's
	if currentUserID != requestedUserID {
		isAdmin, err := h.store.HasPermission(r.Context(), currentUserID, "roles", "list")
		if err != nil {
			h.internalError(w, err)
			return
		}
		if !isAdmin {
			h.writeError(w, http.StatusForbidden, "FORBIDDEN", "Forbidden")
			return
		}
	}

	permissions, err := h.store.GetUserPermissions(r.Context(), requestedUserID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, permissions)
}
