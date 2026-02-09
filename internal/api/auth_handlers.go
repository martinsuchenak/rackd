package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
)

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionTTL.Seconds()),
	})
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

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

	if h.svc != nil && h.svc.Auth != nil {
		result, err := h.svc.Auth.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			h.handleServiceError(w, err)
			return
		}

		log.Info("User logged in", "username", req.Username, "user_id", result.User.ID)

		h.setSessionCookie(w, result.Session.Token)

		response := model.LoginResponse{
			User:      result.User,
			ExpiresAt: result.Session.ExpiresAt,
		}

		h.writeJSON(w, http.StatusOK, response)
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
		h.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	if err := auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		log.Warn("Login failed: invalid password", "username", req.Username)
		h.writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password")
		return
	}

	isAdmin, _ := h.store.HasPermission(r.Context(), user.ID, "users", "create")

	session, err := h.sessionManager.CreateSession(user.ID, user.Username, isAdmin)
	if err != nil {
		h.internalError(w, err)
		return
	}

	now := time.Now()
	if err := h.store.UpdateUserLastLogin(user.ID, now); err != nil {
		log.Warn("Failed to update last login", "user_id", user.ID, "error", err)
	}

	log.Info("User logged in", "username", user.Username, "user_id", user.ID)

	h.setSessionCookie(w, session.Token)

	response := model.LoginResponse{
		User:      user.ToResponse(),
		ExpiresAt: session.ExpiresAt,
	}

	h.writeJSON(w, http.StatusOK, response)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if h.svc != nil && h.svc.Auth != nil {
		session, ok := r.Context().Value(contextKey(SessionContextKey)).(*auth.Session)
		if !ok || session == nil {
			h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
			return
		}

		if err := h.svc.Auth.Logout(r.Context(), session.Token); err != nil {
			log.Warn("Failed to invalidate session", "error", err)
		}

		h.clearSessionCookie(w)

		log.Info("User logged out", "username", session.Username, "user_id", session.UserID)

		w.WriteHeader(http.StatusNoContent)
		return
	}

	session, ok := r.Context().Value(contextKey(SessionContextKey)).(*auth.Session)
	if !ok || session == nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if err := h.sessionManager.InvalidateSession(session.Token); err != nil {
		log.Warn("Failed to invalidate session", "error", err)
	}

	h.clearSessionCookie(w)

	log.Info("User logged out", "username", session.Username, "user_id", session.UserID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	if h.svc != nil && h.svc.Auth != nil {
		user, err := h.svc.Auth.GetCurrentUser(r.Context())
		if err != nil {
			h.handleServiceError(w, err)
			return
		}

		h.writeJSON(w, http.StatusOK, user)
		return
	}

	session, ok := r.Context().Value(contextKey(SessionContextKey)).(*auth.Session)
	if !ok || session == nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	user, err := h.store.GetUser(session.UserID)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	roles, err := h.store.GetUserRoles(r.Context(), session.UserID)
	if err != nil {
		h.internalError(w, err)
		return
	}

	response := user.ToResponse()
	response.Roles = make([]model.Role, len(roles))
	for i, role := range roles {
		response.Roles[i] = role
	}

	h.writeJSON(w, http.StatusOK, response)
}
