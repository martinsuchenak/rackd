package auth

import (
	"testing"
	"time"
)

func TestAuthenticator_Authenticate(t *testing.T) {
	auth := NewAuthenticator()

	key1 := &APIKey{
		ID:        "1",
		Name:      "test-key",
		Key:       "test-token-123",
		CreatedAt: time.Now(),
	}
	auth.AddKey(key1)

	tests := []struct {
		name      string
		token     string
		wantValid bool
		wantName  string
	}{
		{
			name:      "Valid token",
			token:     "test-token-123",
			wantValid: true,
			wantName:  "test-key",
		},
		{
			name:      "Invalid token",
			token:     "invalid-token",
			wantValid: false,
		},
		{
			name:      "Empty token",
			token:     "",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, valid := auth.Authenticate(tt.token)
			if valid != tt.wantValid {
				t.Errorf("Authenticate() valid = %v, want %v", valid, tt.wantValid)
			}
			if valid && key.Name != tt.wantName {
				t.Errorf("Authenticate() name = %v, want %v", key.Name, tt.wantName)
			}
		})
	}
}

func TestAuthenticator_Authenticate_Expired(t *testing.T) {
	auth := NewAuthenticator()

	expired := time.Now().Add(-1 * time.Hour)
	key := &APIKey{
		ID:        "1",
		Name:      "expired-key",
		Key:       "expired-token",
		CreatedAt: time.Now(),
		ExpiresAt: &expired,
	}
	auth.AddKey(key)

	_, valid := auth.Authenticate("expired-token")
	if valid {
		t.Error("Expected expired key to be invalid")
	}
}

func TestAuthenticator_Authenticate_UpdatesLastUsed(t *testing.T) {
	auth := NewAuthenticator()

	key := &APIKey{
		ID:        "1",
		Name:      "test-key",
		Key:       "test-token",
		CreatedAt: time.Now(),
	}
	auth.AddKey(key)

	if key.LastUsedAt != nil {
		t.Error("LastUsedAt should be nil initially")
	}

	auth.Authenticate("test-token")

	if key.LastUsedAt == nil {
		t.Error("LastUsedAt should be set after authentication")
	}
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if len(key1) == 0 {
		t.Error("Generated key should not be empty")
	}

	key2, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	if key1 == key2 {
		t.Error("Generated keys should be unique")
	}
}
