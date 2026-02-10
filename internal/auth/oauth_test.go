package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateOAuthToken(t *testing.T) {
	plain1, hash1, err := GenerateOAuthToken()
	if err != nil {
		t.Fatalf("GenerateOAuthToken failed: %v", err)
	}
	if plain1 == "" {
		t.Fatal("plaintext token is empty")
	}
	if hash1 == "" {
		t.Fatal("hash is empty")
	}

	// Verify hash matches plaintext
	if HashToken(plain1) != hash1 {
		t.Fatal("HashToken(plain) != returned hash")
	}

	// Verify uniqueness
	plain2, hash2, err := GenerateOAuthToken()
	if err != nil {
		t.Fatalf("second GenerateOAuthToken failed: %v", err)
	}
	if plain1 == plain2 {
		t.Fatal("two generated tokens are identical")
	}
	if hash1 == hash2 {
		t.Fatal("two generated hashes are identical")
	}
}

func TestHashToken(t *testing.T) {
	token := "test-token-value"
	h := sha256.Sum256([]byte(token))
	expected := base64.RawURLEncoding.EncodeToString(h[:])

	got := HashToken(token)
	if got != expected {
		t.Fatalf("HashToken mismatch: got %q, want %q", got, expected)
	}

	// Same input produces same hash
	if HashToken(token) != got {
		t.Fatal("HashToken is not deterministic")
	}
}

func TestValidatePKCE(t *testing.T) {
	// Generate a code verifier and its S256 challenge
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	tests := []struct {
		name     string
		verifier string
		challenge string
		method   string
		want     bool
	}{
		{"valid S256", verifier, challenge, "S256", true},
		{"wrong verifier", "wrong-verifier", challenge, "S256", false},
		{"wrong challenge", verifier, "wrong-challenge", "S256", false},
		{"plain method rejected", verifier, verifier, "plain", false},
		{"empty method rejected", verifier, challenge, "", false},
		{"empty verifier", "", challenge, "S256", false},
		{"empty challenge", verifier, "", "S256", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePKCE(tt.verifier, tt.challenge, tt.method)
			if got != tt.want {
				t.Fatalf("ValidatePKCE() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRedirectURI(t *testing.T) {
	registered := []string{
		"http://localhost:8080/callback",
		"https://example.com/oauth/callback",
	}

	tests := []struct {
		uri  string
		want bool
	}{
		{"http://localhost:8080/callback", true},
		{"https://example.com/oauth/callback", true},
		{"http://localhost:8080/callback/", false}, // trailing slash
		{"http://evil.com/callback", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			got := ValidateRedirectURI(tt.uri, registered)
			if got != tt.want {
				t.Fatalf("ValidateRedirectURI(%q) = %v, want %v", tt.uri, got, tt.want)
			}
		})
	}
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"devices:read", 1},
		{"devices:read devices:write", 2},
		{"devices:read  networks:list", 2}, // extra space
		{"*", 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseScopes(tt.input)
			if len(got) != tt.want {
				t.Fatalf("ParseScopes(%q) returned %d scopes, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestIntersectScopes(t *testing.T) {
	allowed := []string{"devices:read", "devices:write", "networks:list"}

	tests := []struct {
		name      string
		requested []string
		wantLen   int
	}{
		{"empty requested returns all", nil, 3},
		{"wildcard returns all", []string{"*"}, 3},
		{"subset", []string{"devices:read"}, 1},
		{"intersection", []string{"devices:read", "unknown:scope"}, 1},
		{"no overlap", []string{"unknown:scope"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntersectScopes(tt.requested, allowed)
			if len(got) != tt.wantLen {
				t.Fatalf("IntersectScopes() returned %d scopes, want %d", len(got), tt.wantLen)
			}
		})
	}
}
