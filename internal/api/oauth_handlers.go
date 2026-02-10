package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/service"
	"github.com/martinsuchenak/rackd/internal/ui"
)

// oauthRegister handles RFC 7591 Dynamic Client Registration.
// POST /mcp-oauth/register
func (h *Handler) oauthRegister(w http.ResponseWriter, r *http.Request) {
	var req model.OAuthClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	resp, err := h.svc.OAuth.RegisterClient(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOAuthClientNameRequired):
			h.writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "client_name is required")
		case errors.Is(err, service.ErrOAuthRedirectURIRequired):
			h.writeOAuthError(w, http.StatusBadRequest, "invalid_redirect_uri", "at least one redirect_uri is required")
		default:
			log.Error("OAuth client registration failed", "error", err)
			h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "Registration failed")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(resp)
}

// oauthAuthorize handles the authorization endpoint.
// GET /mcp-oauth/authorize
// This endpoint serves JSON consent data for the SPA.
// Browser requests should go through the SPA (served by UI handler).
func (h *Handler) oauthAuthorize(w http.ResponseWriter, r *http.Request) {
	// Check if this is a direct browser navigation (wants HTML)
	// vs SPA fetch request (has credentials: same-origin which sends cookies)
	accept := r.Header.Get("Accept")
	isBrowserNavigation := (strings.Contains(accept, "text/html") || accept == "*/*" || accept == "") &&
		r.Header.Get("X-Requested-With") == "" &&
		!strings.Contains(accept, "application/json")

	// For direct browser navigation, serve the SPA HTML
	// The SPA will then fetch this endpoint to get the consent data
	if isBrowserNavigation {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(ui.IndexHTML())
		return
	}

	// User must be logged in via session cookie
	session := h.getSessionFromCookie(r)
	if session == nil {
		// Redirect to login with return URL
		returnURL := r.URL.String()
		http.Redirect(w, r, "/login?redirect="+url.QueryEscape(returnURL), http.StatusFound)
		return
	}

	q := r.URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	responseType := q.Get("response_type")
	scope := q.Get("scope")
	state := q.Get("state")
	codeChallenge := q.Get("code_challenge")
	codeChallengeMethod := q.Get("code_challenge_method")

	client, scopes, err := h.svc.OAuth.ValidateAuthRequest(clientID, redirectURI, responseType, scope, codeChallenge, codeChallengeMethod)
	if err != nil {
		// If redirect_uri is invalid, don't redirect — show error directly
		if errors.Is(err, service.ErrOAuthInvalidRedirectURI) || errors.Is(err, service.ErrOAuthInvalidClient) {
			h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		// Otherwise redirect with error
		redirectWithError(w, r, redirectURI, "invalid_request", err.Error(), state)
		return
	}

	// Return consent page data as JSON (the SPA will render it)
	h.writeJSON(w, http.StatusOK, map[string]any{
		"client_name":           client.Name,
		"client_uri":            client.ClientURI,
		"scopes":                scopes,
		"client_id":             clientID,
		"redirect_uri":          redirectURI,
		"state":                 state,
		"scope":                 scope,
		"code_challenge":        codeChallenge,
		"code_challenge_method": codeChallengeMethod,
		"user":                  session.Username,
	})
}

// oauthAuthorizeSubmit handles consent form submission.
// POST /mcp-oauth/authorize
func (h *Handler) oauthAuthorizeSubmit(w http.ResponseWriter, r *http.Request) {
	session := h.getSessionFromCookie(r)
	if session == nil {
		h.writeOAuthError(w, http.StatusUnauthorized, "access_denied", "Not authenticated")
		return
	}

	var body struct {
		ClientID            string `json:"client_id"`
		RedirectURI         string `json:"redirect_uri"`
		Scope               string `json:"scope"`
		State               string `json:"state"`
		CodeChallenge       string `json:"code_challenge"`
		CodeChallengeMethod string `json:"code_challenge_method"`
		Approved            bool   `json:"approved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON")
		return
	}

	if !body.Approved {
		redirectWithError(w, r, body.RedirectURI, "access_denied", "User denied the request", body.State)
		return
	}

	// Validate the request again
	_, _, err := h.svc.OAuth.ValidateAuthRequest(body.ClientID, body.RedirectURI, "code", body.Scope, body.CodeChallenge, body.CodeChallengeMethod)
	if err != nil {
		redirectWithError(w, r, body.RedirectURI, "invalid_request", err.Error(), body.State)
		return
	}

	// Create authorization code
	code, err := h.svc.OAuth.CreateAuthorizationCode(
		r.Context(),
		body.ClientID,
		session.UserID,
		body.RedirectURI,
		body.Scope,
		body.CodeChallenge,
		body.CodeChallengeMethod,
	)
	if err != nil {
		log.Error("Failed to create authorization code", "error", err)
		redirectWithError(w, r, body.RedirectURI, "server_error", "Failed to create authorization code", body.State)
		return
	}

	// Return redirect URL (SPA will redirect)
	redirectURL := buildRedirectURL(body.RedirectURI, code, body.State)
	h.writeJSON(w, http.StatusOK, map[string]string{
		"redirect_uri": redirectURL,
	})
}

// oauthToken handles the token endpoint.
// POST /mcp-oauth/token
func (h *Handler) oauthToken(w http.ResponseWriter, r *http.Request) {
	// Parse form-encoded body (OAuth spec requires application/x-www-form-urlencoded)
	if err := r.ParseForm(); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid form data")
		return
	}

	req := &model.OAuthTokenRequest{
		GrantType:    r.FormValue("grant_type"),
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		ClientID:     r.FormValue("client_id"),
		ClientSecret: r.FormValue("client_secret"),
		CodeVerifier: r.FormValue("code_verifier"),
		RefreshToken: r.FormValue("refresh_token"),
		Scope:        r.FormValue("scope"),
	}

	var resp *model.OAuthTokenResponse
	var err error

	switch req.GrantType {
	case "authorization_code":
		resp, err = h.svc.OAuth.ExchangeCode(r.Context(), req)
	case "refresh_token":
		resp, err = h.svc.OAuth.RefreshAccessToken(r.Context(), req)
	case "client_credentials":
		resp, err = h.svc.OAuth.ClientCredentials(r.Context(), req)
	default:
		h.writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "Grant type not supported")
		return
	}

	if err != nil {
		log.Debug("OAuth token request failed", "grant_type", req.GrantType, "error", err)
		switch {
		case errors.Is(err, service.ErrOAuthInvalidClient):
			h.writeOAuthError(w, http.StatusUnauthorized, "invalid_client", err.Error())
		case errors.Is(err, service.ErrOAuthInvalidCodeVerifier):
			h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		case errors.Is(err, service.ErrOAuthInvalidClientSecret):
			h.writeOAuthError(w, http.StatusUnauthorized, "invalid_client", "Invalid client secret")
		default:
			h.writeOAuthError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// oauthRevoke handles token revocation (RFC 7009).
// POST /mcp-oauth/revoke
func (h *Handler) oauthRevoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "Invalid form data")
		return
	}

	token := r.FormValue("token")
	tokenTypeHint := r.FormValue("token_type_hint")

	if token == "" {
		h.writeOAuthError(w, http.StatusBadRequest, "invalid_request", "token is required")
		return
	}

	if err := h.svc.OAuth.RevokeToken(r.Context(), token, tokenTypeHint); err != nil {
		log.Error("OAuth token revocation failed", "error", err)
		h.writeOAuthError(w, http.StatusInternalServerError, "server_error", "Revocation failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// oauthListClients lists registered OAuth clients (admin only).
// GET /api/oauth/clients
func (h *Handler) oauthListClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.svc.OAuth.ListClients(r.Context())
	if err != nil {
		h.handleServiceError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, clients)
}

// oauthDeleteClient deletes an OAuth client (admin only).
// DELETE /api/oauth/clients/{id}
func (h *Handler) oauthDeleteClient(w http.ResponseWriter, r *http.Request) {
	clientID := r.PathValue("id")
	if err := h.svc.OAuth.DeleteClient(r.Context(), clientID); err != nil {
		h.handleServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Helpers ---

func (h *Handler) getSessionFromCookie(r *http.Request) *auth.Session {
	if h.sessionManager == nil {
		return nil
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil
	}
	session, err := h.sessionManager.GetSession(cookie.Value)
	if err != nil {
		return nil
	}
	return session
}

func (h *Handler) writeOAuthError(w http.ResponseWriter, status int, errorCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.OAuthErrorResponse{
		Error:            errorCode,
		ErrorDescription: description,
	})
}

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, errorCode, description, state string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
		return
	}
	q := u.Query()
	q.Set("error", errorCode)
	q.Set("error_description", description)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func buildRedirectURL(redirectURI, code, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
