package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	email := r.URL.Query().Get("email")

	filter := &model.UserFilter{
		Username: username,
		Email:    email,
	}

	users, err := h.svc.Users.List(r.Context(), filter)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, users)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	resp, err := h.svc.Users.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	resp, err := h.svc.Users.Create(r.Context(), &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	log.Info("User created", "username", req.Username, "id", resp.ID)
	h.writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.UpdateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	resp, err := h.svc.Users.Update(r.Context(), id, &req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	log.Info("User updated", "id", id)
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	if err := h.svc.Users.Delete(r.Context(), id); err != nil {
		h.handleServiceError(w, err)
		return
	}

	log.Info("User deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.ChangePasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if err := h.svc.Users.ChangePassword(r.Context(), id, &req); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.sessionManager.InvalidateUserSessions(id)
	log.Info("Password changed", "user_id", id)

	w.WriteHeader(http.StatusNoContent)
}

// resetPassword allows admins to reset a user's password without the old password
func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "INVALID_ID", "ID is required")
		return
	}
	var req model.ResetPasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if err := h.svc.Users.ResetPassword(r.Context(), id, &req); err != nil {
		h.handleServiceError(w, err)
		return
	}

	// Invalidate all sessions for this user
	h.sessionManager.InvalidateUserSessions(id)
	log.Info("Password reset by admin", "user_id", id)

	w.WriteHeader(http.StatusNoContent)
}
