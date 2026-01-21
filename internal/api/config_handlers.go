package api

import (
	"encoding/json"
	"net/http"
)

// UIConfig represents frontend configuration
type UIConfig struct {
	Edition  string    `json:"edition"`
	Features []string  `json:"features"`
	NavItems []NavItem `json:"nav_items"`
	UserInfo *UserInfo `json:"user,omitempty"`
}

type NavItem struct {
	Label string `json:"label"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
	Order int    `json:"order"`
}

type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
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

func (b *UIConfigBuilder) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(b.config)
	}
}
