package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"time"
)

// APIKey represents an API key for authentication
type APIKey struct {
	ID          string
	Name        string
	Key         string
	Description string
	CreatedAt   time.Time
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
}

// Authenticator handles API key authentication
type Authenticator struct {
	keys map[string]*APIKey // key -> APIKey
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		keys: make(map[string]*APIKey),
	}
}

// AddKey adds an API key
func (a *Authenticator) AddKey(key *APIKey) {
	a.keys[key.Key] = key
}

// Authenticate validates an API key and returns the associated key info
func (a *Authenticator) Authenticate(token string) (*APIKey, bool) {
	if token == "" {
		return nil, false
	}

	for keyStr, key := range a.keys {
		if subtle.ConstantTimeCompare([]byte(token), []byte(keyStr)) == 1 {
			// Check expiration
			if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
				return nil, false
			}

			// Update last used
			now := time.Now()
			key.LastUsedAt = &now

			return key, true
		}
	}

	return nil, false
}

// GenerateKey generates a new random API key
func GenerateKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
