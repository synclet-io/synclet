package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// CleanupExpiredTokens removes expired refresh tokens and OIDC states.
type CleanupExpiredTokens struct {
	storage    Storage
	stateStore *StateStore
}

// NewCleanupExpiredTokens creates a new CleanupExpiredTokens use case.
func NewCleanupExpiredTokens(storage Storage, stateStore *StateStore) *CleanupExpiredTokens {
	return &CleanupExpiredTokens{storage: storage, stateStore: stateStore}
}

// Execute deletes expired refresh tokens and expired OIDC states.
func (uc *CleanupExpiredTokens) Execute(ctx context.Context) error {
	now := time.Now()

	// Clean up expired refresh tokens.
	tokens, err := uc.storage.RefreshTokens().Find(ctx, &RefreshTokenFilter{})
	if err != nil {
		return fmt.Errorf("listing refresh tokens: %w", err)
	}

	for _, token := range tokens {
		if now.After(token.ExpiresAt) {
			if err := uc.storage.RefreshTokens().Delete(ctx, &RefreshTokenFilter{
				ID: filter.Equals(token.ID),
			}); err != nil {
				return fmt.Errorf("deleting expired refresh token: %w", err)
			}
		}
	}

	// Clean up expired OIDC states.
	if err := uc.stateStore.CleanupExpired(ctx); err != nil {
		return fmt.Errorf("cleaning up expired OIDC states: %w", err)
	}

	return nil
}
