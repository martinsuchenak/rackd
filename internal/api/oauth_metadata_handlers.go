package api

import (
	"net/http"
)

// oauthProtectedResource returns RFC 9728 Protected Resource Metadata.
// GET /.well-known/oauth-protected-resource
func (h *Handler) oauthProtectedResource(w http.ResponseWriter, r *http.Request) {
	issuerURL := h.svc.OAuth.IssuerURL()
	h.writeJSON(w, http.StatusOK, map[string]any{
		"resource":                issuerURL + "/mcp",
		"authorization_servers":   []string{issuerURL},
		"bearer_methods_supported": []string{"header"},
		"scopes_supported":        h.svc.OAuth.GetAllScopes(),
	})
}

// oauthAuthorizationServerMetadata returns RFC 8414 Authorization Server Metadata.
// GET /.well-known/oauth-authorization-server
func (h *Handler) oauthAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	issuerURL := h.svc.OAuth.IssuerURL()
	h.writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                issuerURL,
		"authorization_endpoint":                issuerURL + "/mcp-oauth/authorize",
		"token_endpoint":                        issuerURL + "/mcp-oauth/token",
		"registration_endpoint":                 issuerURL + "/mcp-oauth/register",
		"revocation_endpoint":                   issuerURL + "/mcp-oauth/revoke",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"token_endpoint_auth_methods_supported": []string{"none", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256"},
		"scopes_supported":                      h.svc.OAuth.GetAllScopes(),
	})
}
