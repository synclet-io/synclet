package authservice

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
)

var randomPasswordHash = string(lo.Must(bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)))

// ValidatePassword checks that a password has at least minLength characters and mixes
// letters with digits or symbols.
func ValidatePassword(password string, minLength int) error {
	if utf8.RuneCountInString(password) < minLength {
		return &ValidationError{Message: fmt.Sprintf("password must be at least %d characters", minLength)}
	}

	hasLetter := strings.ContainsFunc(password, unicode.IsLetter)
	hasOther := strings.ContainsFunc(password, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsSpace(r) })

	if !hasLetter || !hasOther {
		return &ValidationError{Message: "password must contain at least one letter and one digit or symbol"}
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
