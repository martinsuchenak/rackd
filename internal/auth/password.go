package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 14
)

// dummyBcryptHash is a pre-computed bcrypt hash of a random unused password,
// used to equalize login timing between existing and non-existing usernames.
var dummyBcryptHash = func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte("timing-equalization-dummy"), bcryptCost)
	if err != nil {
		// bcrypt at cost 14 cannot realistically fail here; fall back to a
		// fixed valid hash so VerifyDummyPassword still burns comparable time.
		return "$2a$14$XkdGJp0YUGxQcmpQeSG5nOQxFiFH0GtWv1fLm2SUkWJ0ZGQ0O5O5S"
	}
	return string(hash)
}()

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

func VerifyPassword(hashedPassword, password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	if hashedPassword == "" {
		return fmt.Errorf("hashed password cannot be empty")
	}

	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// VerifyDummyPassword runs a bcrypt comparison against a throwaway hash so
// that login attempts for unknown usernames cost the same time as for known
// ones, preventing username enumeration via response timing.
func VerifyDummyPassword(password string) {
	if password == "" {
		password = "x"
	}
	_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
}
