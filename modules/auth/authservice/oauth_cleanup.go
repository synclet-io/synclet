package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// CleanupOAuthGrants deletes what abandoned OAuth clients leave behind:
// expired authorization codes, expired OAuth-issued API keys other than the
// latest key of a grant with a usable refresh token, grants whose
// refresh token has expired, clients that never got a grant within
// OAuthUnusedClientTTL of registering, and clients without a grant whose last
// grant is older than OAuthRefreshTokenTTL.
type CleanupOAuthGrants struct {
	storage      Storage
	oauthStorage OAuthStorage
}

// NewCleanupOAuthGrants creates a new CleanupOAuthGrants use case.
func NewCleanupOAuthGrants(storage Storage, oauthStorage OAuthStorage) *CleanupOAuthGrants {
	return &CleanupOAuthGrants{storage: storage, oauthStorage: oauthStorage}
}

func (uc *CleanupOAuthGrants) Execute(ctx context.Context) error {
	now := time.Now()

	if err := uc.storage.OAuthCodes().Delete(ctx, &OAuthCodeFilter{ExpiresAt: filter.Less(now)}); err != nil {
		return fmt.Errorf("deleting expired authorization codes: %w", err)
	}

	if err := uc.oauthStorage.DeleteExpiredOAuthKeys(ctx, now); err != nil {
		return fmt.Errorf("deleting expired OAuth keys: %w", err)
	}

	if err := uc.oauthStorage.DeleteLapsedOAuthGrants(ctx, now); err != nil {
		return fmt.Errorf("deleting lapsed OAuth grants: %w", err)
	}

	if err := uc.oauthStorage.DeleteUnusedOAuthClients(ctx, now.Add(-OAuthUnusedClientTTL), now.Add(-OAuthRefreshTokenTTL)); err != nil {
		return fmt.Errorf("deleting unused OAuth clients: %w", err)
	}

	return nil
}
