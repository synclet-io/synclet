package authservice

import (
	"context"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// Logout invalidates a refresh token.
type Logout struct {
	storage Storage
}

// NewLogout creates a new Logout use case.
func NewLogout(storage Storage) *Logout {
	return &Logout{storage: storage}
}

// Execute deletes the refresh token matching the given raw token.
func (uc *Logout) Execute(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)

	return uc.storage.RefreshTokens().Delete(ctx, &RefreshTokenFilter{
		TokenHash: filter.Equals(tokenHash),
	})
}
