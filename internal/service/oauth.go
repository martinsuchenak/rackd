package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"net"
	"net/url"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

var (
	ErrOAuthInvalidClient        = errors.New("invalid client_id")
	ErrOAuthInvalidRedirectURI   = errors.New("invalid redirect_uri")
	ErrOAuthInvalidResponseType  = errors.New("unsupported response_type")
	ErrOAuthInvalidGrantType     = errors.New("unsupported grant_type")
	ErrOAuthInvalidCodeChallenge = errors.New("code_challenge required for public clients")
	ErrOAuthInvalidCodeVerifier  = errors.New("invalid code_verifier")
	ErrOAuthInvalidClientSecret  = errors.New("invalid client_secret")
	ErrOAuthClientNameRequired   = errors.New("client_name is required")
	ErrOAuthRedirectURIRequired  = errors.New("at least one redirect_uri is required")
	ErrOAuthRedirectURINotHTTPS  = errors.New("redirect_uri must be an absolute https URI (http is only allowed for loopback hosts)")
	ErrOAuthScopeRequired        = errors.New("scope is required (an empty scope would grant no access)")
	ErrOAuthScopeNotAllowed      = errors.New("scope must not contain '*'; register explicit scopes")
	ErrOAuthInvalidScope         = errors.New("requested scope exceeds the client's registered scope")
)

type OAuthService struct {
	store           storage.ExtendedStorage
	sessionManager  *auth.SessionManager
	issuerURL       string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	codeExpiry      time.Duration
	stopCleanup     chan struct{}
}

func NewOAuthService(store storage.ExtendedStorage, sm *auth.SessionManager, issuerURL string) *OAuthService {
	return &OAuthService{
		store:           store,
		sessionManager:  sm,
		issuerURL:       issuerURL,
		accessTokenTTL:  1 * time.Hour,
		refreshTokenTTL: 30 * 24 * time.Hour,
		codeExpiry:      10 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}
}

func (s *OAuthService) SetTokenTTLs(accessTTL, refreshTTL time.Duration) {
	s.accessTokenTTL = accessTTL
	s.refreshTokenTTL = refreshTTL
}

func (s *OAuthService) IssuerURL() string {
	return s.issuerURL
}

// isValidRedirectURI enforces the redirect URI policy required for every
// registered and requested redirect URI: it must be an absolute URI using
// the https scheme (http is accepted only for loopback hosts per RFC 8252),
// with no userinfo or fragment component. Scheme-relative, javascript:,
// data: and other custom-scheme URIs are rejected.
func isValidRedirectURI(uri string) bool {
	parsed, err := url.Parse(uri)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" {
		return false
	}
	if parsed.User != nil || parsed.Fragment != "" || parsed.RawFragment != "" {
		return false
	}
	switch parsed.Scheme {
	case "https":
		return true
	case "http":
		host := parsed.Hostname()
		return net.ParseIP(host).IsLoopback() || host == "localhost"
	default:
		return false
	}
}

// RegisterClient handles RFC 7591 dynamic client registration.
func (s *OAuthService) RegisterClient(ctx context.Context, req *model.OAuthClientRegistrationRequest) (*model.OAuthClientRegistrationResponse, error) {
	if req.ClientName == "" {
		return nil, ErrOAuthClientNameRequired
	}
	if len(req.RedirectURIs) == 0 {
		return nil, ErrOAuthRedirectURIRequired
	}
	for _, uri := range req.RedirectURIs {
		if !isValidRedirectURI(uri) {
			return nil, ErrOAuthRedirectURINotHTTPS
		}
	}
	// A wildcard scope must never be self-registered: '*' resolves to an
	// unrestricted caller downstream, so allowing clients to register it via
	// dynamic registration would let anyone mint full-permission tokens.
	requestedScopes := auth.ParseScopes(req.Scope)
	if len(requestedScopes) == 0 {
		return nil, ErrOAuthScopeRequired
	}
	for _, s := range requestedScopes {
		if s == "*" {
			return nil, ErrOAuthScopeNotAllowed
		}
	}
	// Registration scopes must lie inside the advertised scopes_supported
	// catalog; arbitrary strings would otherwise be echoed by the consent
	// screen and stored on the client.
	catalog, err := s.scopeCatalog(ctx)
	if err != nil {
		return nil, err
	}
	if unknown := auth.SubtractScopes(requestedScopes, catalog); len(unknown) > 0 {
		return nil, ErrOAuthScopeNotAllowed
	}

	// Default grant types and response types
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}
	tokenEndpointAuth := req.TokenEndpointAuth
	if tokenEndpointAuth == "" {
		tokenEndpointAuth = "none"
	}

	isConfidential := tokenEndpointAuth == "client_secret_post"

	client := &model.OAuthClient{
		ID:                uuid.New().String(),
		Name:              req.ClientName,
		RedirectURIs:      req.RedirectURIs,
		GrantTypes:        grantTypes,
		ResponseTypes:     responseTypes,
		TokenEndpointAuth: tokenEndpointAuth,
		Scope:             req.Scope,
		ClientURI:         req.ClientURI,
		LogoURI:           req.LogoURI,
		IsConfidential:    isConfidential,
	}

	var clientSecret string
	if isConfidential {
		secret, hash, err := auth.GenerateOAuthToken()
		if err != nil {
			return nil, err
		}
		clientSecret = secret
		client.SecretHash = hash
	}

	if err := s.store.CreateOAuthClient(ctx, client); err != nil {
		return nil, err
	}

	return &model.OAuthClientRegistrationResponse{
		ClientID:          client.ID,
		ClientSecret:      clientSecret,
		ClientName:        client.Name,
		RedirectURIs:      client.RedirectURIs,
		GrantTypes:        client.GrantTypes,
		ResponseTypes:     client.ResponseTypes,
		TokenEndpointAuth: client.TokenEndpointAuth,
		ClientIDIssuedAt:  client.CreatedAt.Unix(),
	}, nil
}

// ValidateAuthRequest validates an authorization request and returns the client
// and the effective scopes for the consent screen.
func (s *OAuthService) ValidateAuthRequest(clientID, redirectURI, responseType, scope, codeChallenge, codeChallengeMethod string) (*model.OAuthClient, []string, error) {
	client, err := s.store.GetOAuthClient(context.Background(), clientID)
	if err != nil {
		return nil, nil, ErrOAuthInvalidClient
	}

	if !auth.ValidateRedirectURI(redirectURI, client.RedirectURIs) {
		return nil, nil, ErrOAuthInvalidRedirectURI
	}
	if !isValidRedirectURI(redirectURI) {
		return nil, nil, ErrOAuthInvalidRedirectURI
	}

	if responseType != "code" {
		return nil, nil, ErrOAuthInvalidResponseType
	}

	// PKCE is required for public clients (OAuth 2.1 mandate)
	if !client.IsConfidential && codeChallenge == "" {
		return nil, nil, ErrOAuthInvalidCodeChallenge
	}
	if codeChallenge != "" && codeChallengeMethod != "S256" {
		return nil, nil, ErrOAuthInvalidCodeChallenge
	}

	requestedScopes := auth.ParseScopes(scope)
	registeredScopes := auth.ParseScopes(client.Scope)
	if len(requestedScopes) == 0 {
		requestedScopes = registeredScopes
	}
	// An authorization request with no effective scope must be rejected
	// instead of issuing a token with an empty scope. An empty scope cannot
	// be displayed on the consent screen and previously resolved to an
	// unrestricted caller, silently granting the user's full RBAC set.
	if len(requestedScopes) == 0 {
		return nil, nil, ErrOAuthScopeRequired
	}
	// The granted scope must be a subset of the client's registered scope:
	// intersect instead of echoing the request verbatim. (Legacy clients
	// that registered "*" before wildcards were rejected keep working; no
	// new client can register "*".)
	if !slices.Contains(registeredScopes, "*") {
		requestedScopes = auth.IntersectScopes(requestedScopes, registeredScopes)
	}
	if len(requestedScopes) == 0 {
		return nil, nil, ErrOAuthInvalidScope
	}
	// Consent-screen scopes are additionally clamped to the advertised
	// scopes_supported catalog, so stale or rogue registered strings are
	// never displayed or echoed back to the client.
	catalog, err := s.scopeCatalog(context.Background())
	if err != nil {
		return nil, nil, err
	}
	requestedScopes = auth.IntersectScopes(requestedScopes, catalog)
	if len(requestedScopes) == 0 {
		return nil, nil, ErrOAuthInvalidScope
	}

	return client, requestedScopes, nil
}

// CreateAuthorizationCode creates an authorization code after user consent.
func (s *OAuthService) CreateAuthorizationCode(ctx context.Context, clientID, userID, redirectURI, scope, codeChallenge, codeChallengeMethod string) (string, error) {
	client, err := s.store.GetOAuthClient(ctx, clientID)
	if err != nil {
		return "", err
	}
	effectiveScopes := auth.ParseScopes(scope)
	if len(effectiveScopes) == 0 {
		// Fail closed: fall back to the client's registered default scope;
		// if the client has none either, refuse to issue a code rather than
		// mint a token with an empty (historically unrestricted) scope.
		effectiveScopes = auth.ParseScopes(client.Scope)
	}
	registeredScopes := auth.ParseScopes(client.Scope)
	if !slices.Contains(registeredScopes, "*") {
		effectiveScopes = auth.IntersectScopes(effectiveScopes, registeredScopes)
	}
	// Clamp to the advertised catalog...
	catalog, err := s.scopeCatalog(ctx)
	if err != nil {
		return "", err
	}
	effectiveScopes = auth.IntersectScopes(effectiveScopes, catalog)
	// ...and to the consenting user's own permissions: a user must never be
	// able to delegate (or be tricked into granting) more than they hold,
	// regardless of what the client registered or the authorize request
	// carried. A legacy "*" registration collapses to the user's permission
	// set by this same intersection.
	userScopes, err := s.userScopeSet(ctx, userID)
	if err != nil {
		return "", err
	}
	effectiveScopes = auth.IntersectScopes(effectiveScopes, userScopes)
	if len(effectiveScopes) == 0 {
		return "", ErrOAuthScopeRequired
	}
	scope = auth.JoinScopes(effectiveScopes)

	plaintext, hash, err := auth.GenerateAuthorizationCode()
	if err != nil {
		return "", err
	}

	code := &model.OAuthAuthorizationCode{
		CodeHash:            hash,
		ClientID:            clientID,
		UserID:              userID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           time.Now().Add(s.codeExpiry),
		CreatedAt:           time.Now().UTC(),
	}

	if err := s.store.CreateAuthorizationCode(ctx, code); err != nil {
		return "", err
	}

	return plaintext, nil
}

// verifyClientSecret authenticates a confidential client per RFC 6749 §4.1.3/§6.
// Public clients are exempt (they must use PKCE instead).
func (s *OAuthService) verifyClientSecret(ctx context.Context, req *model.OAuthTokenRequest) error {
	client, err := s.store.GetOAuthClient(ctx, req.ClientID)
	if err != nil {
		return ErrOAuthInvalidClient
	}
	if !client.IsConfidential {
		return nil
	}
	secretHash := auth.HashToken(req.ClientSecret)
	if subtle.ConstantTimeCompare([]byte(secretHash), []byte(client.SecretHash)) != 1 {
		return ErrOAuthInvalidClientSecret
	}
	return nil
}

// ExchangeCode exchanges an authorization code for tokens (authorization_code grant).
func (s *OAuthService) ExchangeCode(ctx context.Context, req *model.OAuthTokenRequest) (*model.OAuthTokenResponse, error) {
	if req.Code == "" {
		return nil, errors.New("code is required")
	}

	codeHash := auth.HashToken(req.Code)
	code, err := s.store.GetAuthorizationCode(ctx, codeHash)
	if err != nil {
		return nil, err
	}

	// Verify client
	if code.ClientID != req.ClientID {
		return nil, ErrOAuthInvalidClient
	}

	// Authenticate confidential clients (public clients use PKCE instead)
	if err := s.verifyClientSecret(ctx, req); err != nil {
		return nil, err
	}

	// Verify redirect URI
	if code.RedirectURI != req.RedirectURI {
		return nil, ErrOAuthInvalidRedirectURI
	}

	// Mark code as used immediately to prevent any concurrent replay attempts
	if err := s.store.MarkAuthorizationCodeUsed(ctx, codeHash); err != nil {
		return nil, err
	}

	// Verify PKCE
	if code.CodeChallenge != "" {
		if !auth.ValidatePKCE(req.CodeVerifier, code.CodeChallenge, code.CodeChallengeMethod) {
			return nil, ErrOAuthInvalidCodeVerifier
		}
	}

	// Create access token
	accessPlain, accessHash, err := auth.GenerateOAuthToken()
	if err != nil {
		return nil, err
	}

	accessToken := &model.OAuthToken{
		TokenType: "access",
		TokenHash: accessHash,
		ClientID:  code.ClientID,
		UserID:    code.UserID,
		Scope:     code.Scope,
		ExpiresAt: time.Now().Add(s.accessTokenTTL),
	}
	if err := s.store.CreateOAuthToken(ctx, accessToken); err != nil {
		return nil, err
	}

	// Create refresh token
	refreshPlain, refreshHash, err := auth.GenerateOAuthToken()
	if err != nil {
		return nil, err
	}

	refreshToken := &model.OAuthToken{
		TokenType:     "refresh",
		TokenHash:     refreshHash,
		ClientID:      code.ClientID,
		UserID:        code.UserID,
		Scope:         code.Scope,
		ExpiresAt:     time.Now().Add(s.refreshTokenTTL),
		ParentTokenID: accessToken.ID,
	}
	if err := s.store.CreateOAuthToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return &model.OAuthTokenResponse{
		AccessToken:  accessPlain,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
		RefreshToken: refreshPlain,
		Scope:        code.Scope,
	}, nil
}

// RefreshAccessToken exchanges a refresh token for a new access token.
// Implements refresh token rotation (M-4): each refresh token use generates a new refresh token
// and revokes the old one. If a revoked refresh token is reused, it indicates a replay attack
// and all tokens in the chain are revoked.
func (s *OAuthService) RefreshAccessToken(ctx context.Context, req *model.OAuthTokenRequest) (*model.OAuthTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, errors.New("refresh_token is required")
	}

	refreshHash := auth.HashToken(req.RefreshToken)

	// Check if token exists even if revoked (for replay detection)
	refreshToken, err := s.store.GetOAuthTokenByHashIncludingRevoked(ctx, refreshHash)
	if err != nil {
		return nil, err
	}

	if refreshToken.TokenType != "refresh" {
		return nil, storage.ErrOAuthTokenNotFound
	}

	// Reject expired refresh tokens. The lookup above intentionally includes
	// revoked tokens (for replay detection), so the expiry check must happen
	// here: without it an expired refresh token would mint a fresh token pair
	// with a full new TTL, making any leaked refresh token valid forever.
	if time.Now().UTC().After(refreshToken.ExpiresAt) {
		log.Warn("OAuth refresh token expired - refusing refresh",
			"token_id", refreshToken.ID,
			"client_id", refreshToken.ClientID,
			"user_id", refreshToken.UserID,
			"expired_at", refreshToken.ExpiresAt,
		)
		return nil, storage.ErrOAuthTokenExpired
	}

	// Detect replay attack: if the refresh token was already revoked, someone else used it
	// Revoke all tokens in the chain to prevent further abuse
	if refreshToken.RevokedAt != nil {
		log.Warn("OAuth refresh token replay detected - revoking token chain",
			"token_id", refreshToken.ID,
			"client_id", refreshToken.ClientID,
			"user_id", refreshToken.UserID,
		)
		s.store.RevokeOAuthTokenChain(ctx, refreshToken.ID)
		return nil, storage.ErrOAuthTokenRevoked
	}

	// Verify client. If client_id was supplied it must match; if omitted,
	// default it to the token's client so confidential clients are still
	// authenticated below (RFC 6749 §6).
	if req.ClientID == "" {
		req.ClientID = refreshToken.ClientID
	}
	if refreshToken.ClientID != req.ClientID {
		return nil, ErrOAuthInvalidClient
	}

	// Authenticate confidential clients (public clients use PKCE instead)
	if err := s.verifyClientSecret(ctx, req); err != nil {
		return nil, err
	}

	// Determine scope (use refresh token's scope if not specified)
	scope := refreshToken.Scope
	if req.Scope != "" {
		// Requested scope must be a subset of the refresh token's scope
		requestedScopes := auth.ParseScopes(req.Scope)
		allowedScopes := auth.ParseScopes(refreshToken.Scope)
		effectiveScopes := auth.IntersectScopes(requestedScopes, allowedScopes)
		scope = auth.JoinScopes(effectiveScopes)
	}
	// Re-clamp the effective scope on every refresh: to the advertised
	// catalog and to the user's CURRENT permissions. Tokens minted before a
	// permission revocation (or carrying rogue scope strings from a legacy
	// client) lose the excess scopes at the next refresh.
	{
		effectiveScopes := auth.ParseScopes(scope)
		if slices.Contains(effectiveScopes, "*") {
			effectiveScopes = nil
		}
		catalog, err := s.scopeCatalog(ctx)
		if err != nil {
			return nil, err
		}
		effectiveScopes = auth.IntersectScopes(effectiveScopes, catalog)
		userScopes, err := s.userScopeSet(ctx, refreshToken.UserID)
		if err != nil {
			return nil, err
		}
		effectiveScopes = auth.IntersectScopes(effectiveScopes, userScopes)
		if len(effectiveScopes) == 0 {
			return nil, ErrOAuthInvalidScope
		}
		scope = auth.JoinScopes(effectiveScopes)
	}

	// Create new access token
	accessPlain, accessHash, err := auth.GenerateOAuthToken()
	if err != nil {
		return nil, err
	}

	accessToken := &model.OAuthToken{
		TokenType:     "access",
		TokenHash:     accessHash,
		ClientID:      refreshToken.ClientID,
		UserID:        refreshToken.UserID,
		Scope:         scope,
		ExpiresAt:     time.Now().Add(s.accessTokenTTL),
		ParentTokenID: refreshToken.ID,
	}
	if err := s.store.CreateOAuthToken(ctx, accessToken); err != nil {
		return nil, err
	}

	// Refresh token rotation: create new refresh token and revoke the old one
	newRefreshPlain, newRefreshHash, err := auth.GenerateOAuthToken()
	if err != nil {
		return nil, err
	}

	newRefreshToken := &model.OAuthToken{
		TokenType:     "refresh",
		TokenHash:     newRefreshHash,
		ClientID:      refreshToken.ClientID,
		UserID:        refreshToken.UserID,
		Scope:         scope,
		ExpiresAt:     time.Now().Add(s.refreshTokenTTL),
		ParentTokenID: accessToken.ID,
	}
	if err := s.store.CreateOAuthToken(ctx, newRefreshToken); err != nil {
		return nil, err
	}

	// Revoke the old refresh token
	if err := s.store.RevokeOAuthToken(ctx, refreshToken.ID); err != nil {
		log.Error("Failed to revoke old refresh token during rotation", "error", err)
		// Continue anyway - the new token is already created
	}

	return &model.OAuthTokenResponse{
		AccessToken:  accessPlain,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
		RefreshToken: newRefreshPlain,
		Scope:        scope,
	}, nil
}

// ClientCredentials handles the client_credentials grant for confidential clients.
func (s *OAuthService) ClientCredentials(ctx context.Context, req *model.OAuthTokenRequest) (*model.OAuthTokenResponse, error) {
	client, err := s.store.GetOAuthClient(ctx, req.ClientID)
	if err != nil {
		return nil, ErrOAuthInvalidClient
	}

	if !client.IsConfidential {
		return nil, ErrOAuthInvalidGrantType
	}

	// Verify client secret
	secretHash := auth.HashToken(req.ClientSecret)
	if subtle.ConstantTimeCompare([]byte(secretHash), []byte(client.SecretHash)) != 1 {
		return nil, ErrOAuthInvalidClientSecret
	}

	// For client_credentials, the client must have a created_by_user_id to map to a user
	if client.CreatedByUserID == "" {
		return nil, errors.New("client has no associated user")
	}

	scope := client.Scope
	if req.Scope != "" {
		requestedScopes := auth.ParseScopes(req.Scope)
		allowedScopes := auth.ParseScopes(client.Scope)
		effectiveScopes := auth.IntersectScopes(requestedScopes, allowedScopes)
		scope = auth.JoinScopes(effectiveScopes)
	}
	// Clamp client-credentials grants to the catalog and the associated
	// user's permissions — the client acts on that user's behalf.
	{
		effectiveScopes := auth.ParseScopes(scope)
		if slices.Contains(effectiveScopes, "*") {
			effectiveScopes = nil
		}
		catalog, err := s.scopeCatalog(ctx)
		if err != nil {
			return nil, err
		}
		effectiveScopes = auth.IntersectScopes(effectiveScopes, catalog)
		userScopes, err := s.userScopeSet(ctx, client.CreatedByUserID)
		if err != nil {
			return nil, err
		}
		effectiveScopes = auth.IntersectScopes(effectiveScopes, userScopes)
		if len(effectiveScopes) == 0 {
			return nil, ErrOAuthInvalidScope
		}
		scope = auth.JoinScopes(effectiveScopes)
	}

	accessPlain, accessHash, err := auth.GenerateOAuthToken()
	if err != nil {
		return nil, err
	}

	accessToken := &model.OAuthToken{
		TokenType: "access",
		TokenHash: accessHash,
		ClientID:  client.ID,
		UserID:    client.CreatedByUserID,
		Scope:     scope,
		ExpiresAt: time.Now().Add(s.accessTokenTTL),
	}
	if err := s.store.CreateOAuthToken(ctx, accessToken); err != nil {
		return nil, err
	}

	return &model.OAuthTokenResponse{
		AccessToken: accessPlain,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.accessTokenTTL.Seconds()),
		Scope:       scope,
	}, nil
}

// ValidateAccessToken validates an opaque access token and returns the token record.
func (s *OAuthService) ValidateAccessToken(token string) (*model.OAuthToken, error) {
	tokenHash := auth.HashToken(token)
	oauthToken, err := s.store.GetOAuthTokenByHash(context.Background(), tokenHash)
	if err != nil {
		return nil, err
	}
	if oauthToken.TokenType != "access" {
		return nil, storage.ErrOAuthTokenNotFound
	}
	return oauthToken, nil
}

// ResolveCallerFromOAuthToken builds a Caller from a validated OAuth token.
func (s *OAuthService) ResolveCallerFromOAuthToken(token *model.OAuthToken, remoteAddr string) (*Caller, error) {
	user, err := s.store.GetUser(context.Background(), token.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, errors.New("user is not active")
	}

	scopes := auth.ParseScopes(token.Scope)
	// "*" grants unrestricted access; any other scope list (including an
	// empty one) is enforced as-is. An empty scope must resolve to zero
	// granted scopes, never to the user's full RBAC permission set.
	if slices.Contains(scopes, "*") {
		scopes = nil
	}

	return &Caller{
		Type:      CallerTypeUser,
		UserID:    user.ID,
		Username:  user.Username,
		IPAddress: remoteAddr,
		Source:    "mcp-oauth",
		Scopes:    scopes,
	}, nil
}

// RevokeToken revokes a token by its plaintext value.
func (s *OAuthService) RevokeToken(ctx context.Context, req *model.OAuthTokenRequest) error {
	token := req.Token
	tokenHash := auth.HashToken(token)

	// Try to find the token (ignore expiry/revocation errors for revocation endpoint)
	oauthToken, err := s.store.GetOAuthTokenByHash(ctx, tokenHash)
	if err != nil {
		// Per RFC 7009, revocation of an invalid token should succeed silently
		log.Debug("OAuth token revocation: token not found", "error", err)
		return nil
	}

	// A confidential client may only revoke its own tokens. The token lookup
	// ignores expiry, so re-authenticate the caller here: without this check
	// any party could revoke another client's live tokens.
	if err := s.verifyClientSecret(ctx, req); err != nil {
		return err
	}
	if oauthToken.ClientID != req.ClientID {
		return ErrOAuthInvalidClient
	}

	// Revoke the token
	if err := s.store.RevokeOAuthToken(ctx, oauthToken.ID); err != nil {
		return err
	}

	// If revoking a refresh token, also revoke associated access tokens
	if oauthToken.TokenType == "refresh" && oauthToken.ParentTokenID != "" {
		s.store.RevokeOAuthToken(ctx, oauthToken.ParentTokenID)
	}

	return nil
}

// ListClients lists all registered OAuth clients.
func (s *OAuthService) ListClients(ctx context.Context) ([]model.OAuthClient, error) {
	// OAuth client management is admin-only; gated on users:list to match the UI.
	if err := requirePermission(ctx, s.store, "users", "list"); err != nil {
		return nil, err
	}
	return s.store.ListOAuthClients(ctx, "")
}

// DeleteClient deletes an OAuth client and revokes its tokens.
func (s *OAuthService) DeleteClient(ctx context.Context, clientID string) error {
	// OAuth client management is admin-only; gated on users:delete to match the UI.
	if err := requirePermission(ctx, s.store, "users", "delete"); err != nil {
		return err
	}
	// Revoke all tokens for this client first
	s.store.RevokeOAuthTokensByClient(ctx, clientID)
	return s.store.DeleteOAuthClient(ctx, clientID)
}

// scopeCatalog returns every scope this authorization server may issue,
// derived from the RBAC permission catalog (resource:action). This is the
// single source of truth advertised as scopes_supported and the upper bound
// for any client registration or token grant.
func (s *OAuthService) scopeCatalog(ctx context.Context) ([]string, error) {
	perms, err := s.store.ListPermissions(ctx, nil)
	if err != nil {
		return nil, err
	}
	catalog := make([]string, 0, len(perms))
	for _, p := range perms {
		catalog = append(catalog, p.Resource+":"+p.Action)
	}
	sort.Strings(catalog)
	return catalog, nil
}

// userScopeSet returns the explicit scopes the user may delegate: one
// resource:action entry per permission the user holds.
func (s *OAuthService) userScopeSet(ctx context.Context, userID string) ([]string, error) {
	perms, err := s.store.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	scopes := make([]string, 0, len(perms))
	for _, p := range perms {
		scopes = append(scopes, p.Resource+":"+p.Action)
	}
	return scopes, nil
}

// GetAllScopes returns all available permission scopes (the advertised
// scopes_supported), derived from the permission catalog.
func (s *OAuthService) GetAllScopes() []string {
	ctx := SystemContext(context.Background(), "oauth")
	catalog, err := s.scopeCatalog(ctx)
	if err != nil {
		log.Error("Failed to list permission catalog for scopes_supported", "error", err)
		return []string{}
	}
	return catalog
}

// StartCleanup starts a background goroutine to clean up expired codes and tokens.
func (s *OAuthService) StartCleanup() {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.store.CleanupExpiredCodes(context.Background()); err != nil {
					log.Error("Failed to cleanup expired OAuth codes", "error", err)
				}
				if err := s.store.CleanupExpiredTokens(context.Background()); err != nil {
					log.Error("Failed to cleanup expired OAuth tokens", "error", err)
				}
			case <-s.stopCleanup:
				return
			}
		}
	}()
}

// StopCleanup stops the background cleanup goroutine.
func (s *OAuthService) StopCleanup() {
	close(s.stopCleanup)
}
