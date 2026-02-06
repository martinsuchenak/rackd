package api

import (
	"encoding/json"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	email := r.URL.Query().Get("email")

	filter := &model.UserFilter{
		Username: username,
		Email:    email,
	}

	users, err := h.store.ListUsers(filter)
	if err != nil {
		h.internalError(w, err)
		return
	}

	responses := make([]model.UserResponse, len(users))
	for i, user := range users {
		responses[i] = user.ToResponse()
	}

	h.writeJSON(w, http.StatusOK, responses)
}

func (h *Handler) getUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	user, err := h.store.GetUser(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	h.writeJSON(w, http.StatusOK, user.ToResponse())
}

func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var req model.CreateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.Username == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_USERNAME", "Username is required")
		return
	}

	if len(req.Password) < 8 {
		h.writeError(w, http.StatusBadRequest, "PASSWORD_TOO_SHORT", "Password must be at least 8 characters")
		return
	}

	if req.Email == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_EMAIL", "Email is required")
		return
	}

	existingUser, err := h.store.GetUserByUsername(req.Username)
	if err == nil && existingUser != nil {
		h.writeError(w, http.StatusConflict, "USERNAME_EXISTS", "Username already exists")
		return
	}

	existingUser, err = h.store.GetUserByEmail(req.Email)
	if err == nil && existingUser != nil {
		h.writeError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already exists")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		h.internalError(w, err)
		return
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		FullName:     req.FullName,
		PasswordHash: passwordHash,
		IsActive:     true,
		IsAdmin:      req.IsAdmin,
	}

	if err := h.store.CreateUser(r.Context(), user); err != nil {
		h.internalError(w, err)
		return
	}

	log.Info("User created", "username", user.Username, "id", user.ID)

	h.writeJSON(w, http.StatusCreated, user.ToResponse())
}

func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	user, err := h.store.GetUser(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	var req model.UpdateUserRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.Email != "" {
		existingUser, err := h.store.GetUserByEmail(req.Email)
		if err == nil && existingUser != nil && existingUser.ID != id {
			h.writeError(w, http.StatusConflict, "EMAIL_EXISTS", "Email already exists")
			return
		}
		user.Email = req.Email
	}

	if req.FullName != "" {
		user.FullName = req.FullName
	}

	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	if err := h.store.UpdateUser(r.Context(), user); err != nil {
		h.internalError(w, err)
		return
	}

	log.Info("User updated", "id", user.ID, "username", user.Username)

	h.writeJSON(w, http.StatusOK, user.ToResponse())
}

func (h *Handler) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	user, err := h.store.GetUser(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	session, ok := r.Context().Value(contextKey(SessionContextKey)).(*auth.Session)
	if ok && session != nil && session.UserID == id {
		h.writeError(w, http.StatusBadRequest, "CANNOT_DELETE_SELF", "Cannot delete your own account")
		return
	}

	if err := h.store.DeleteUser(r.Context(), id); err != nil {
		if err == storage.ErrUserNotFound {
			h.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		} else {
			h.internalError(w, err)
		}
		return
	}

	h.sessionManager.InvalidateUserSessions(id)

	log.Info("User deleted", "id", id, "username", user.Username)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	user, err := h.store.GetUser(id)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	var req model.ChangePasswordRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.OldPassword == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_OLD_PASSWORD", "Old password is required")
		return
	}

	if req.NewPassword == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_NEW_PASSWORD", "New password is required")
		return
	}

	if len(req.NewPassword) < 8 {
		h.writeError(w, http.StatusBadRequest, "PASSWORD_TOO_SHORT", "New password must be at least 8 characters")
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.OldPassword); err != nil {
		h.writeError(w, http.StatusUnauthorized, "INVALID_PASSWORD", "Old password is incorrect")
		return
	}

	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		h.internalError(w, err)
		return
	}

	if err := h.store.UpdateUserPassword(id, newPasswordHash); err != nil {
		h.internalError(w, err)
		return
	}

	h.sessionManager.InvalidateUserSessions(id)

	log.Info("Password changed", "user_id", id, "username", user.Username)

	w.WriteHeader(http.StatusNoContent)
}
