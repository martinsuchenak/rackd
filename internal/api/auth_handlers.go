package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_JSON", "Invalid JSON")
		return
	}

	if req.Username == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_USERNAME", "Username is required")
		return
	}

	if req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "MISSING_PASSWORD", "Password is required")
		return
	}

	user, err := h.store.GetUserByUsername(req.Username)
	if err != nil {
		log.Warn("Login failed: user not found", "username", req.Username)
		h.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	if !user.IsActive {
		log.Warn("Login failed: user inactive", "username", req.Username)
		h.writeError(w, http.StatusForbidden, "USER_INACTIVE", "User account is inactive")
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		log.Warn("Login failed: invalid password", "username", req.Username)
		h.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	session, err := h.sessionManager.CreateSession(user.ID, user.Username, user.IsAdmin)
	if err != nil {
		h.internalError(w, err)
		return
	}

	now := time.Now()
	if err := h.store.UpdateUserLastLogin(user.ID, now); err != nil {
		log.Warn("Failed to update last login", "user_id", user.ID, "error", err)
	}

	log.Info("User logged in", "username", user.Username, "user_id", user.ID)

	response := model.LoginResponse{
		Token:     session.Token,
		User:      user.ToResponse(),
		ExpiresAt: session.ExpiresAt,
	}

	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(SessionContextKey).(*auth.Session)
	if !ok || session == nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if err := h.sessionManager.InvalidateSession(session.Token); err != nil {
		log.Warn("Failed to invalidate session", "error", err)
	}

	log.Info("User logged out", "username", session.Username, "user_id", session.UserID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(SessionContextKey).(*auth.Session)
	if !ok || session == nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	user, err := h.store.GetUser(session.UserID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	h.writeJSON(w, http.StatusOK, user.ToResponse())
}
