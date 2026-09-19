package auth

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

const (
	// defaultBcryptCost is the production password hashing cost.
	defaultBcryptCost = 14
	// minBcryptCost is the lowest cost bcrypt accepts.
	minBcryptCost = 4
	// maxBcryptCost is the highest cost bcrypt accepts.
	maxBcryptCost = 31
)

// bcryptCost defaults to defaultBcryptCost and can be overridden via
// RACKD_BCRYPT_COST (validated to 4-31; anything invalid falls back to the
// default). Tests set a low cost because bcrypt dominates test runtime —
// a cost-14 hash takes ~700ms natively and ~7s under the race detector.
//
// This must stay a var initializer rather than an init() function: Go runs
// package-level var initializers in dependency order, so bcryptCost is set
// before the dummyBcryptHash initializer below uses it. init() functions
// run after all var initializers and would be too late.
var bcryptCost = bcryptCostFromEnv()

func bcryptCostFromEnv() int {
	v := os.Getenv("RACKD_BCRYPT_COST")
	if v == "" {
		return defaultBcryptCost
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < minBcryptCost || n > maxBcryptCost {
		return defaultBcryptCost
	}
	return n
}

// dummyBcryptHash is a pre-computed bcrypt hash of a random unused password,
// used to equalize login timing between existing and non-existing usernames.
var dummyBcryptHash = func() string {
	hash, err := bcrypt.GenerateFromPassword([]byte("timing-equalization-dummy"), bcryptCost)
	if err != nil {
		// bcrypt at the configured cost cannot realistically fail here; fall
		// back to a fixed valid hash so VerifyDummyPassword still burns
		// comparable time.
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
