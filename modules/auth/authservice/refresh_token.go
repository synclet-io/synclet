package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RefreshTokenUC generates a new token pair from a valid refresh token.
type RefreshTokenUC struct {
	storage Storage
	config  Config
}

// NewRefreshTokenUC creates a new RefreshTokenUC use case.
func NewRefreshTokenUC(storage Storage, config Config) *RefreshTokenUC {
	return &RefreshTokenUC{storage: storage, config: config}
}

// Execute validates the refresh token, rotates it, and returns a new token pair.
func (uc *RefreshTokenUC) Execute(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := hashToken(refreshToken)

	storedToken, err := uc.storage.RefreshTokens().First(ctx, &RefreshTokenFilter{
		TokenHash: filter.Equals(tokenHash),
	})
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if time.Now().After(storedToken.ExpiresAt) {
		// Delete expired token.
		if err := uc.storage.RefreshTokens().Delete(ctx, &RefreshTokenFilter{
			ID: filter.Equals(storedToken.ID),
		}); err != nil {
			return nil, fmt.Errorf("delete expired refresh token: %w", err)
		}

		return nil, ErrRefreshTokenExpired
	}

	user, err := uc.storage.Users().First(ctx, &UserFilter{
		ID: filter.Equals(storedToken.UserID),
	})
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Delete old token and create new pair atomically to prevent reuse and token loss.
	var result *TokenPair

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		if err := tx.RefreshTokens().Delete(ctx, &RefreshTokenFilter{
			ID: filter.Equals(storedToken.ID),
		}); err != nil {
			return fmt.Errorf("delete used refresh token: %w", err)
		}

		var err error
		result, err = generateTokenPair(ctx, tx, uc.config, user)

		return err
	}); err != nil {
		return nil, err
	}

	return result, nil
}
