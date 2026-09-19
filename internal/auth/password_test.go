package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantErr   bool
		errString string
	}{
		{
			name:     "valid password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "short password",
			password: "short",
			wantErr:  false,
		},
		{
			name:      "empty password",
			password:  "",
			wantErr:   true,
			errString: "password cannot be empty",
		},
		{
			name:     "long password (within bcrypt limit)",
			password: strings.Repeat("a", 72),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)

			if tt.wantErr {
				if err == nil {
					t.Errorf("HashPassword() expected error, got nil")
					return
				}
				if tt.errString != "" && !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("HashPassword() error = %v, want %v", err.Error(), tt.errString)
				}
				return
			}

			if err != nil {
				t.Errorf("HashPassword() unexpected error: %v", err)
				return
			}

			if hash == "" {
				t.Errorf("HashPassword() returned empty hash")
			}

			if hash == tt.password {
				t.Errorf("HashPassword() hash should not equal password")
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "testPassword123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name           string
		hashedPassword string
		password       string
		wantErr        bool
		errString      string
	}{
		{
			name:           "correct password",
			hashedPassword: hash,
			password:       password,
			wantErr:        false,
		},
		{
			name:           "wrong password",
			hashedPassword: hash,
			password:       "wrongPassword",
			wantErr:        true,
		},
		{
			name:           "empty password",
			hashedPassword: hash,
			password:       "",
			wantErr:        true,
			errString:      "password cannot be empty",
		},
		{
			name:           "empty hash",
			hashedPassword: "",
			password:       password,
			wantErr:        true,
			errString:      "hashed password cannot be empty",
		},
		{
			name:           "invalid hash format",
			hashedPassword: "notavalidhash",
			password:       password,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPassword(tt.hashedPassword, tt.password)

			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyPassword() expected error, got nil")
					return
				}
				if tt.errString != "" && !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("VerifyPassword() error = %v, want %v", err.Error(), tt.errString)
				}
				return
			}

			if err != nil {
				t.Errorf("VerifyPassword() unexpected error: %v", err)
			}
		})
	}
}

func TestHashPasswordConsistency(t *testing.T) {
	password := "consistentPassword"

	hash1, err1 := HashPassword(password)
	if err1 != nil {
		t.Fatalf("First hash failed: %v", err1)
	}

	hash2, err2 := HashPassword(password)
	if err2 != nil {
		t.Fatalf("Second hash failed: %v", err2)
	}

	if hash1 == hash2 {
		t.Errorf("HashPassword() should produce different hashes for the same password")
	}

	if err := VerifyPassword(hash1, password); err != nil {
		t.Errorf("VerifyPassword() failed for hash1: %v", err)
	}

	if err := VerifyPassword(hash2, password); err != nil {
		t.Errorf("VerifyPassword() failed for hash2: %v", err)
	}
}

func TestBcryptCostFromEnv(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want int
	}{
		{"unset uses default", "", defaultBcryptCost},
		{"valid low cost", "6", 6},
		{"valid high cost", "31", 31},
		{"below minimum rejected", "3", defaultBcryptCost},
		{"above maximum rejected", "32", defaultBcryptCost},
		{"non-numeric rejected", "strong", defaultBcryptCost},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("RACKD_BCRYPT_COST", tc.env)
			if got := bcryptCostFromEnv(); got != tc.want {
				t.Errorf("bcryptCostFromEnv() with RACKD_BCRYPT_COST=%q = %d, want %d", tc.env, got, tc.want)
			}
		})
	}
}
