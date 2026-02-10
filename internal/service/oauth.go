package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/martinsuchenak/rackd/internal/auth"
	"github.com/martinsuchenak/rackd/internal/log"
	"github.com/martinsuchenak/rackd/internal/model"
	"github.com/martinsuchenak/rackd/internal/storage"
)

var (
	ErrOAuthInvalidClient       = errors.New("invalid client_id")
	ErrOAuthInvalidRedirectURI  = errors.New("invalid redirect_uri")
	ErrOAuthInvalidResponseType = errors.New("unsupported response_type")
	ErrOAuthInvalidGrantType    = errors.New("unsupported grant_type")
	ErrOAuthInvalidCodeChallenge = errors.New("code_challenge required for public clients")
	ErrOAuthInvalidCodeVerifier = errors.New("invalid code_verifier")
	ErrOAuthInvalidClientSecret = errors.New("invalid client_secret")
	ErrOAuthClientNameRequired  = errors.New("client_name is required")
	ErrOAuthRedirectURIRequired = errors.New("at least one redirect_uri is required")
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

// RegisterClient handles RFC 7591 dynamic client registration.
func (s *OAuthService) RegisterClient(ctx context.Context, req *model.OAuthClientRegistrationRequest) (*model.OAuthClientRegistrationResponse, error) {
	if req.ClientName == "" {
		return nil, ErrOAuthClientNameRequired
	}
	if len(req.RedirectURIs) == 0 {
		return nil, ErrOAuthRedirectURIRequired
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
	client, err := s.store.GetOAuthClient(clientID)
	if err != nil {
		return nil, nil, ErrOAuthInvalidClient
	}

	if !auth.ValidateRedirectURI(redirectURI, client.RedirectURIs) {
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
	if len(requestedScopes) == 0 {
		requestedScopes = []string{"*"}
	}

	return client, requestedScopes, nil
}

// CreateAuthorizationCode creates an authorization code after user consent.
func (s *OAuthService) CreateAuthorizationCode(ctx context.Context, clientID, userID, redirectURI, scope, codeChallenge, codeChallengeMethod string) (string, error) {
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

// ExchangeCode exchanges an authorization code for access and refresh tokens.
func (s *OAuthService) ExchangeCode(ctx context.Context, req *model.OAuthTokenRequest) (*model.OAuthTokenResponse, error) {
	if req.Code == "" {
		return nil, errors.New("code is required")
	}

	codeHash := auth.HashToken(req.Code)
	code, err := s.store.GetAuthorizationCode(codeHash)
	if err != nil {
		return nil, err
	}

	// Verify client
	if code.ClientID != req.ClientID {
		return nil, ErrOAuthInvalidClient
	}

	// Verify redirect URI
	if code.RedirectURI != req.RedirectURI {
		return nil, ErrOAuthInvalidRedirectURI
	}

	// Verify PKCE
	if code.CodeChallenge != "" {
		if !auth.ValidatePKCE(req.CodeVerifier, code.CodeChallenge, code.CodeChallengeMethod) {
			// Mark code as used to prevent replay
			s.store.MarkAuthorizationCodeUsed(codeHash)
			return nil, ErrOAuthInvalidCodeVerifier
		}
	}

	// Mark code as used
	if err := s.store.MarkAuthorizationCodeUsed(codeHash); err != nil {
		return nil, err
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
func (s *OAuthService) RefreshAccessToken(ctx context.Context, req *model.OAuthTokenRequest) (*model.OAuthTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, errors.New("refresh_token is required")
	}

	refreshHash := auth.HashToken(req.RefreshToken)
	refreshToken, err := s.store.GetOAuthTokenByHash(refreshHash)
	if err != nil {
		return nil, err
	}

	if refreshToken.TokenType != "refresh" {
		return nil, storage.ErrOAuthTokenNotFound
	}

	// Verify client
	if req.ClientID != "" && refreshToken.ClientID != req.ClientID {
		return nil, ErrOAuthInvalidClient
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

	return &model.OAuthTokenResponse{
		AccessToken: accessPlain,
		TokenType:   "Bearer",
		ExpiresIn:   int(s.accessTokenTTL.Seconds()),
		Scope:       scope,
	}, nil
}

// ClientCredentials handles the client_credentials grant for confidential clients.
func (s *OAuthService) ClientCredentials(ctx context.Context, req *model.OAuthTokenRequest) (*model.OAuthTokenResponse, error) {
	client, err := s.store.GetOAuthClient(req.ClientID)
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

	scope := req.Scope
	if scope == "" {
		scope = client.Scope
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
	oauthToken, err := s.store.GetOAuthTokenByHash(tokenHash)
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
	user, err := s.store.GetUser(token.UserID)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		return nil, errors.New("user is not active")
	}

	scopes := auth.ParseScopes(token.Scope)
	// If scope is "*" or empty, don't restrict (nil scopes = no scope restriction)
	if len(scopes) == 0 || slices.Contains(scopes, "*") {
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
func (s *OAuthService) RevokeToken(ctx context.Context, token, tokenTypeHint string) error {
	tokenHash := auth.HashToken(token)

	// Try to find the token (ignore expiry/revocation errors for revocation endpoint)
	oauthToken, err := s.store.GetOAuthTokenByHash(tokenHash)
	if err != nil {
		// Per RFC 7009, revocation of an invalid token should succeed silently
		log.Debug("OAuth token revocation: token not found", "error", err)
		return nil
	}

	// Revoke the token
	if err := s.store.RevokeOAuthToken(oauthToken.ID); err != nil {
		return err
	}

	// If revoking a refresh token, also revoke associated access tokens
	if oauthToken.TokenType == "refresh" && oauthToken.ParentTokenID != "" {
		s.store.RevokeOAuthToken(oauthToken.ParentTokenID)
	}

	return nil
}

// ListClients lists all registered OAuth clients.
func (s *OAuthService) ListClients(ctx context.Context) ([]model.OAuthClient, error) {
	return s.store.ListOAuthClients("")
}

// DeleteClient deletes an OAuth client and revokes its tokens.
func (s *OAuthService) DeleteClient(ctx context.Context, clientID string) error {
	// Revoke all tokens for this client first
	s.store.RevokeOAuthTokensByClient(clientID)
	return s.store.DeleteOAuthClient(ctx, clientID)
}

// GetAllScopes returns all available permission scopes.
func (s *OAuthService) GetAllScopes() []string {
	ctx := SystemContext(context.Background(), "oauth")
	// Use RBAC storage to get all permissions
	checker, ok := s.store.(interface {
		GetUserPermissions(ctx context.Context, userID string) ([]model.Permission, error)
	})
	if !ok {
		return []string{"*"}
	}

	// Get a superset of permissions by querying with admin
	// Just return a static list of known scopes
	_ = checker
	_ = ctx
	return []string{
		"*",
		"devices:list", "devices:create", "devices:read", "devices:update", "devices:delete",
		"networks:list", "networks:create", "networks:read", "networks:update", "networks:delete",
		"datacenters:list", "datacenters:create", "datacenters:read", "datacenters:update", "datacenters:delete",
		"discovery:list", "discovery:create", "discovery:read", "discovery:delete",
		"pools:list", "pools:create", "pools:read", "pools:update", "pools:delete",
		"relationships:list", "relationships:create", "relationships:read", "relationships:update", "relationships:delete",
	}
}

// StartCleanup starts a background goroutine to clean up expired codes and tokens.
func (s *OAuthService) StartCleanup() {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.store.CleanupExpiredCodes(); err != nil {
					log.Error("Failed to cleanup expired OAuth codes", "error", err)
				}
				if err := s.store.CleanupExpiredTokens(); err != nil {
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
