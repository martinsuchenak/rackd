package model

import "time"

// APIKey represents an API key for authentication
type APIKey struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Key         string     `json:"key,omitempty"` // Only returned on creation
	UserID      string     `json:"user_id"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// APIKeyFilter represents filter criteria for listing API keys
type APIKeyFilter struct {
	Pagination
	Name   string
	UserID string
}

// APIKeyResponse is the response format for API keys (without the actual key)
type APIKeyResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	UserID      string     `json:"user_id"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ToResponse converts an APIKey to APIKeyResponse (hides the key)
func (k *APIKey) ToResponse() APIKeyResponse {
	return APIKeyResponse{
		ID:          k.ID,
		Name:        k.Name,
		UserID:      k.UserID,
		Description: k.Description,
		CreatedAt:   k.CreatedAt,
		LastUsedAt:  k.LastUsedAt,
		ExpiresAt:   k.ExpiresAt,
	}
}
