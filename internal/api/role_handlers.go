package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
)

func getUserIDFromContext(r *http.Request) string {
	caller := service.CallerFrom(r.Context())
	return caller.UserID
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	filter := &model.RoleFilter{}
	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = name
	}

	roles, err := h.svc.Roles.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	roleResponses := make([]model.RoleResponse, len(roles))
	for i, role := range roles {
		permissions, _ := h.svc.Roles.GetPermissions(r.Context(), role.ID)
		roleResponses[i] = role.ToResponseWithPermissions(permissions)
	}

	h.writeJSON(w, http.StatusOK, roleResponses)
}

func (h *Handler) getRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	role, err := h.svc.Roles.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, role.ToResponse())
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var req model.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	role := &model.Role{
		Name:        req.Name,
		Description: req.Description,
		IsSystem:    false,
	}

	if err := h.svc.Roles.Create(r.Context(), role); err != nil {
		h.handleServiceError(w, err)
		return
	}

	if len(req.Permissions) > 0 {
		for _, permID := range req.Permissions {
			if err := h.svc.Roles.GrantPermission(r.Context(), role.ID, permID); err != nil {
				// Log warning but continue
			}
		}
	}

	h.writeJSON(w, http.StatusCreated, role.ToResponse())
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req model.UpdateRoleRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	role, err := h.svc.Roles.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	role.Description = req.Description
	if err := h.svc.Roles.Update(r.Context(), id, role); err != nil {
		h.handleServiceError(w, err)
		return
	}

	if req.Permissions != nil {
		for _, permID := range req.Permissions {
			if err := h.svc.Roles.GrantPermission(r.Context(), id, permID); err != nil {
				// Log warning but continue
			}
		}
	}

	h.writeJSON(w, http.StatusOK, role.ToResponse())
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := h.svc.Roles.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getRolePermissions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	role, err := h.svc.Roles.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	permissions, err := h.svc.Roles.GetPermissions(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
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

	permissions, err := h.svc.Roles.ListPermissions(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, permissions)
}

func (h *Handler) grantRoleToUser(w http.ResponseWriter, r *http.Request) {
	var req model.GrantRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	if err := h.svc.Roles.AssignToUser(r.Context(), req.UserID, req.RoleID); err != nil {
		h.handleServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) revokeRoleFromUser(w http.ResponseWriter, r *http.Request) {
	var req model.RevokeRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.invalidJSON(w)
		return
	}

	if err := h.svc.Roles.RevokeFromUser(r.Context(), req.UserID, req.RoleID); err != nil {
		h.handleServiceError(w, err)
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
		h.badRequest(w, "User ID is required")
		return
	}

	_, err := h.svc.Roles.List(r.Context(), &model.RoleFilter{})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	roles, err := h.svc.Users.GetRoles(r.Context(), requestedUserID)
	if err != nil {
		h.handleServiceError(w, err)
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
		h.badRequest(w, "User ID is required")
		return
	}

	_, err := h.svc.Roles.List(r.Context(), &model.RoleFilter{})
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	permissions, err := h.svc.Users.GetPermissions(r.Context(), requestedUserID)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, permissions)
}
