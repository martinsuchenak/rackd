package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/storage"
	"github.com/martinsuchenak/rackd/internal/version"
	"github.com/martinsuchenak/rackd/pkg/extension"
	"github.com/martinsuchenak/rackd/pkg/model"
)

// UIConfig represents frontend configuration
type UIConfig struct {
	Edition string `json:"edition"`
	Version string `json:"version"`
	// Features are the names of the extensions in this binary.
	Features   []string        `json:"features"`
	Extensions []ExtensionInfo `json:"extensions"`
	NavItems   []NavItem       `json:"nav_items"`
	// Pages are extension routes, with the title and permission the router uses.
	Pages    []PageInfo `json:"pages"`
	UserInfo *UserInfo  `json:"user,omitempty"`
}

// ExtensionInfo describes an extension in /api/v1/config.
type ExtensionInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// PageInfo is an extension's web UI route.
type PageInfo struct {
	Path               string           `json:"path"`
	Title              string           `json:"title"`
	RequiredPermission *PermissionCheck `json:"required_permission,omitempty"`
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
	FullName    string             `json:"full_name,omitempty"`
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
			Edition:    "oss",
			Version:    version.Version,
			Features:   []string{},
			Extensions: []ExtensionInfo{},
			NavItems:   []NavItem{},
			Pages:      []PageInfo{},
		},
	}
}

// AddExtension adds an extension's name, navigation entries and pages.
func (b *UIConfigBuilder) AddExtension(m extension.Manifest) {
	b.config.Features = append(b.config.Features, m.Name)
	b.config.Extensions = append(b.config.Extensions, ExtensionInfo{Name: m.Name, Version: m.Version})
	for _, n := range m.UI.Nav {
		item := NavItem{Label: n.Label, Path: n.Path, Icon: n.Icon, Order: n.Order}
		if n.Permission != nil {
			item.RequiredPermissions = []PermissionCheck{{Resource: n.Permission.Resource, Action: n.Permission.Action}}
		}
		b.config.NavItems = append(b.config.NavItems, item)
	}
	for _, p := range m.UI.Pages {
		page := PageInfo{Path: p.Path, Title: p.Title}
		if p.Permission != nil {
			page.RequiredPermission = &PermissionCheck{Resource: p.Permission.Resource, Action: p.Permission.Action}
		}
		b.config.Pages = append(b.config.Pages, page)
	}
}

func (b *UIConfigBuilder) SetEdition(edition string) {
	b.config.Edition = edition
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

					user, err := store.GetUser(ctx, session.UserID)
					if err != nil {
						user = &model.User{
							ID:       session.UserID,
							Username: session.Username,
						}
					}

					// Get user roles and permissions
					roles, _ := store.GetUserRoles(ctx, session.UserID)
					permissions, _ := store.GetUserPermissions(ctx, session.UserID)

					cfg.UserInfo = &UserInfo{
						ID:          user.ID,
						Username:    user.Username,
						Email:       user.Email,
						FullName:    user.FullName,
						Roles:       roles,
						Permissions: permissions,
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(cfg)
	}
}
