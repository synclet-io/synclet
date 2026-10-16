package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/dbutil"
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
// The token row is locked for the whole rotation, so concurrent refreshes with
// the same token mint one pair and the others fail with ErrInvalidRefreshToken.
func (uc *RefreshTokenUC) Execute(ctx context.Context, refreshToken string) (*TokenPair, error) {
	tokenHash := hashToken(refreshToken)

	var (
		result  *TokenPair
		expired bool
	)

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		storedToken, err := tx.RefreshTokens().First(ctx, &RefreshTokenFilter{
			TokenHash: filter.Equals(tokenHash),
		}, dbutil.WithForUpdate())
		if err != nil {
			if errors.Is(err, ErrRefreshTokenNotFound) {
				return ErrInvalidRefreshToken
			}

			return fmt.Errorf("get refresh token: %w", err)
		}

		if err := tx.RefreshTokens().Delete(ctx, &RefreshTokenFilter{
			ID: filter.Equals(storedToken.ID),
		}); err != nil {
			return fmt.Errorf("delete used refresh token: %w", err)
		}

		if time.Now().After(storedToken.ExpiresAt) {
			expired = true

			return nil
		}

		user, err := tx.Users().First(ctx, &UserFilter{
			ID: filter.Equals(storedToken.UserID),
		})
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				return ErrUserNotFound
			}

			return fmt.Errorf("get refresh token user: %w", err)
		}

		result, err = generateTokenPair(ctx, tx, uc.config, user)

		return err
	}); err != nil {
		return nil, err
	}

	if expired {
		return nil, ErrRefreshTokenExpired
	}

	return result, nil
}
