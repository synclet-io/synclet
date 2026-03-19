package authservice

import (
	"context"
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/filter"
	"golang.org/x/crypto/bcrypt"
)

// Login authenticates a user and returns a token pair.
type Login struct {
	storage Storage
	config  Config
}

// NewLogin creates a new Login use case.
func NewLogin(storage Storage, config Config) *Login {
	return &Login{storage: storage, config: config}
}

// Execute authenticates a user by email and password.
func (uc *Login) Execute(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := uc.storage.Users().First(ctx, &UserFilter{
		Email: filter.Equals(email),
	})
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return generateTokenPair(ctx, uc.storage, uc.config, user)
}
