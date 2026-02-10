package model

import "time"

// OAuthClient represents a registered OAuth 2.1 client.
type OAuthClient struct {
	ID                string    `json:"client_id" db:"id"`
	Name              string    `json:"client_name" db:"name"`
	SecretHash        string    `json:"-" db:"secret_hash"`
	RedirectURIs      []string  `json:"redirect_uris" db:"-"`
	RedirectURIsJSON  string    `json:"-" db:"redirect_uris"`
	GrantTypes        []string  `json:"grant_types" db:"-"`
	GrantTypesJSON    string    `json:"-" db:"grant_types"`
	ResponseTypes     []string  `json:"response_types" db:"-"`
	ResponseTypesJSON string    `json:"-" db:"response_types"`
	TokenEndpointAuth string    `json:"token_endpoint_auth_method" db:"token_endpoint_auth"`
	Scope             string    `json:"scope,omitempty" db:"scope"`
	ClientURI         string    `json:"client_uri,omitempty" db:"client_uri"`
	LogoURI           string    `json:"logo_uri,omitempty" db:"logo_uri"`
	IsConfidential    bool      `json:"is_confidential" db:"is_confidential"`
	CreatedByUserID   string    `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// OAuthAuthorizationCode represents an OAuth 2.1 authorization code.
type OAuthAuthorizationCode struct {
	CodeHash            string    `db:"code_hash"`
	ClientID            string    `db:"client_id"`
	UserID              string    `db:"user_id"`
	RedirectURI         string    `db:"redirect_uri"`
	Scope               string    `db:"scope"`
	CodeChallenge       string    `db:"code_challenge"`
	CodeChallengeMethod string    `db:"code_challenge_method"`
	ExpiresAt           time.Time `db:"expires_at"`
	CreatedAt           time.Time `db:"created_at"`
	Used                bool      `db:"used"`
}

// OAuthToken represents an OAuth 2.1 access or refresh token.
type OAuthToken struct {
	ID            string     `json:"id" db:"id"`
	TokenType     string     `json:"token_type" db:"token_type"` // "access" or "refresh"
	TokenHash     string     `json:"-" db:"token_hash"`
	ClientID      string     `json:"client_id" db:"client_id"`
	UserID        string     `json:"user_id" db:"user_id"`
	Scope         string     `json:"scope" db:"scope"`
	ExpiresAt     time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	ParentTokenID string     `json:"parent_token_id,omitempty" db:"parent_token_id"`
}

// OAuthClientRegistrationRequest is the RFC 7591 dynamic client registration request.
type OAuthClientRegistrationRequest struct {
	ClientName        string   `json:"client_name"`
	RedirectURIs      []string `json:"redirect_uris"`
	GrantTypes        []string `json:"grant_types,omitempty"`
	ResponseTypes     []string `json:"response_types,omitempty"`
	TokenEndpointAuth string   `json:"token_endpoint_auth_method,omitempty"`
	Scope             string   `json:"scope,omitempty"`
	ClientURI         string   `json:"client_uri,omitempty"`
	LogoURI           string   `json:"logo_uri,omitempty"`
}

// OAuthClientRegistrationResponse is the RFC 7591 dynamic client registration response.
type OAuthClientRegistrationResponse struct {
	ClientID          string   `json:"client_id"`
	ClientSecret      string   `json:"client_secret,omitempty"`
	ClientName        string   `json:"client_name"`
	RedirectURIs      []string `json:"redirect_uris"`
	GrantTypes        []string `json:"grant_types"`
	ResponseTypes     []string `json:"response_types"`
	TokenEndpointAuth string   `json:"token_endpoint_auth_method"`
	ClientIDIssuedAt  int64    `json:"client_id_issued_at"`
}

// OAuthTokenRequest represents the token endpoint request (form-encoded).
type OAuthTokenRequest struct {
	GrantType    string `json:"grant_type"`
	Code         string `json:"code,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	CodeVerifier string `json:"code_verifier,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OAuthTokenResponse represents the token endpoint response.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OAuthErrorResponse represents an OAuth 2.1 error response.
type OAuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}
