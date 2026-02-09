package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

// UIConfig represents frontend configuration
type UIConfig struct {
	Edition  string    `json:"edition"`
	Features []string  `json:"features"`
	NavItems []NavItem `json:"nav_items"`
	UserInfo *UserInfo `json:"user,omitempty"`
}

type NavItem struct {
	Label               string            `json:"label"`
	Path                string            `json:"path"`
	Icon                string            `json:"icon"`
	Order               int               `json:"order"`
	RequiredPermissions []PermissionCheck `json:"required_permissions,omitempty"`
}

type PermissionCheck struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

type UserInfo struct {
	ID          string             `json:"id"`
	Username    string             `json:"username"`
	Email       string             `json:"email"`
	Roles       []model.Role       `json:"roles"`
	Permissions []model.Permission `json:"permissions"`
}

// UIConfigBuilder collects config from features
type UIConfigBuilder struct {
	config UIConfig
}

func NewUIConfigBuilder() *UIConfigBuilder {
	return &UIConfigBuilder{
		config: UIConfig{
			Edition:  "oss",
			Features: []string{},
			NavItems: []NavItem{},
		},
	}
}

func (b *UIConfigBuilder) SetEdition(edition string) {
	b.config.Edition = edition
}

func (b *UIConfigBuilder) AddFeature(name string) {
	b.config.Features = append(b.config.Features, name)
}

func (b *UIConfigBuilder) AddNavItem(item NavItem) {
	b.config.NavItems = append(b.config.NavItems, item)
}

func (b *UIConfigBuilder) SetUser(user *UserInfo) {
	b.config.UserInfo = user
}

func (b *UIConfigBuilder) Build() UIConfig {
	return b.config
}

func (b *UIConfigBuilder) Handler(sessionManager *auth.SessionManager, store storage.ExtendedStorage) http.HandlerFunc {
	return b.HandlerWithSession(sessionManager, store)
}

// HandlerWithSession returns a handler that optionally populates user info from a session token.
func (b *UIConfigBuilder) HandlerWithSession(sessionManager *auth.SessionManager, store storage.ExtendedStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := b.config
		cfg.UserInfo = nil

		if sessionManager != nil {
			var token string

			// Check session cookie first
			if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
				token = cookie.Value
			} else if authHeader := r.Header.Get("Authorization"); strings.HasPrefix(authHeader, "Bearer ") {
				// Fall back to Authorization header
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}

			if token != "" {
				if session, err := sessionManager.GetSession(token); err == nil {
					ctx := r.Context()

					// Get user roles and permissions
					roles, _ := store.GetUserRoles(ctx, session.UserID)
					permissions, _ := store.GetUserPermissions(ctx, session.UserID)

					cfg.UserInfo = &UserInfo{
						ID:          session.UserID,
						Username:    session.Username,
						Email:       "",
						Roles:       roles,
						Permissions: permissions,
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	}
}
