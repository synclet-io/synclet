package authservice

import "fmt"

const minPasswordLength = 12

// ValidatePassword checks that a password meets minimum requirements.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return &ValidationError{Message: fmt.Sprintf("password must be at least %d characters", minPasswordLength)}
	}

	return nil
}
