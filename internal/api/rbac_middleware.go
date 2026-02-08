package api

import (
	"context"
	"net/http"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, resource, action string) (bool, error)
}

func RequirePermission(store PermissionChecker, resource, action string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			userID := getUserIDFromContext(r)
			if userID == "" {
				log.Warn("RBAC: no user ID in context", "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Unauthorized","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
				return
			}

			hasPermission, err := store.HasPermission(r.Context(), userID, resource, action)
			if err != nil {
				log.Error("RBAC: permission check error", "error", err, "user_id", userID, "resource", resource, "action", action)
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Internal Server Error","code":"INTERNAL_ERROR"}`, http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				log.Warn("RBAC: permission denied", "user_id", userID, "resource", resource, "action", action, "path", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"Forbidden","code":"FORBIDDEN"}`, http.StatusForbidden)
				return
			}

			next(w, r)
		}
	}
}

func getUserIDFromContext(r *http.Request) string {
	if session := r.Context().Value(SessionContextKey); session != nil {
		if sess, ok := session.(*auth.Session); ok {
			return sess.UserID
		}
	}

	return ""
}
