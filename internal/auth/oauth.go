package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"slices"
	"strings"
)

// GenerateOAuthToken generates a cryptographically random token.
// Returns the plaintext token (base64url-encoded) and its SHA-256 hash for storage.
func GenerateOAuthToken() (plaintext, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plaintext = base64.RawURLEncoding.EncodeToString(b)
	hash = HashToken(plaintext)
	return plaintext, hash, nil
}

// GenerateAuthorizationCode generates a cryptographically random authorization code.
// Returns the plaintext code and its SHA-256 hash for storage.
func GenerateAuthorizationCode() (plaintext, hash string, err error) {
	return GenerateOAuthToken()
}

// HashToken returns the hex-encoded SHA-256 hash of a token.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// ValidatePKCE verifies a code_verifier against a stored code_challenge using S256.
// S256: BASE64URL(SHA256(code_verifier)) == code_challenge
func ValidatePKCE(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}
	if codeVerifier == "" || codeChallenge == "" {
		return false
	}
	h := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(h[:])
	return computed == codeChallenge
}

// ValidateRedirectURI checks if the request redirect URI exactly matches one of the registered URIs.
func ValidateRedirectURI(requestURI string, registeredURIs []string) bool {
	return slices.Contains(registeredURIs, requestURI)
}

// ParseScopes splits a space-delimited OAuth scope string into individual scopes.
func ParseScopes(scope string) []string {
	if scope == "" {
		return nil
	}
	return strings.Fields(scope)
}

// JoinScopes joins scopes into a space-delimited string.
func JoinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}

// IntersectScopes returns scopes that are present in both requested and allowed.
// If requested contains "*", all allowed scopes are returned.
func IntersectScopes(requested, allowed []string) []string {
	if len(requested) == 0 {
		return allowed
	}

	var result []string
	for _, s := range requested {
		if slices.Contains(allowed, s) {
			result = append(result, s)
		}
	}
	return result
}
