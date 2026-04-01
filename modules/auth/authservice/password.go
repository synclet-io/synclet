package authservice

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

var randomPasswordHash = string(lo.Must(bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)))

// ValidatePassword checks that a password meets minimum requirements.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return &ValidationError{Message: fmt.Sprintf("password must be at least %d characters", minPasswordLength)}
	}

	return nil
}

// NormalizePassword ensures that password stays within bcrypt's 72-byte limit
func normalizePassword(password string) string {
	shaHash := sha256.New()
	shaHash.Write([]byte(password))

	return hex.EncodeToString(shaHash.Sum(nil))
}

func hashPassword(password string) (string, error) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(normalizePassword(password)), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt: %w", err)
	}

	return string(bcryptHash), nil
}

func comparePasswordWithHash(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(normalizePassword(password)))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}

		return false, fmt.Errorf("bcrypt.CompareHashAndPassword: %w", err)
	}

	return true, nil
}
