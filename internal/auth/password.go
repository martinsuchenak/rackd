package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 12
)

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
