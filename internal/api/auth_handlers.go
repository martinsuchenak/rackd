package api

import (
	"encoding/json"
	"net/http"

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
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	session, ok := r.Context().Value(contextKey(SessionContextKey)).(*auth.Session)
	if !ok || session == nil {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if err := h.svc.Auth.Logout(r.Context(), session.Token); err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.clearSessionCookie(w)

	log.Info("User logged out", "username", session.Username, "user_id", session.UserID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := h.svc.Auth.GetCurrentUser(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, user)
}
