package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RefreshOAuthToken rotates an OAuth refresh token.
type RefreshOAuthToken struct {
	storage      Storage
	oauthStorage OAuthStorage
	members      WorkspaceMembershipLister
}

// NewRefreshOAuthToken creates a new RefreshOAuthToken use case.
func NewRefreshOAuthToken(storage Storage, oauthStorage OAuthStorage, members WorkspaceMembershipLister) *RefreshOAuthToken {
	return &RefreshOAuthToken{storage: storage, oauthStorage: oauthStorage, members: members}
}

// Execute spends refreshToken and returns a new refresh token and a new access
// key, deleting the grant's previous key. An unknown or expired token, or one
// issued to another client, returns ErrInvalidGrant. A token presented again
// after it was spent, or one whose user is no longer a member of the grant's
// workspace, ends its whole grant and returns ErrInvalidGrant.
func (uc *RefreshOAuthToken) Execute(ctx context.Context, refreshToken, clientID string) (*OAuthTokens, error) {
	tokenHash := hashToken(refreshToken)

	token, err := uc.storage.OAuthRefreshTokens().First(ctx, &OAuthRefreshTokenFilter{TokenHash: filter.Equals(tokenHash)})
	if errors.Is(err, ErrOAuthRefreshTokenNotFound) {
		return nil, ErrInvalidGrant
	}

	if err != nil {
		return nil, fmt.Errorf("getting refresh token: %w", err)
	}

	grant, err := uc.storage.OAuthGrants().First(ctx, &OAuthGrantFilter{ID: filter.Equals(token.GrantID)})
	if errors.Is(err, ErrOAuthGrantNotFound) {
		return nil, ErrInvalidGrant
	}

	if err != nil {
		return nil, fmt.Errorf("getting OAuth grant: %w", err)
	}

	if grant.ClientID != clientID {
		return nil, ErrInvalidGrant
	}

	if token.UsedAt != nil {
		return nil, uc.revoke(ctx, grant)
	}

	now := time.Now()
	if now.After(token.ExpiresAt) {
		return nil, ErrInvalidGrant
	}

	role, err := workspaceRole(ctx, uc.members, grant.UserID, grant.WorkspaceID)
	if err != nil {
		return nil, err
	}

	if role == WorkspaceRoleUnspecified {
		return nil, uc.revoke(ctx, grant)
	}

	if _, err := uc.oauthStorage.MarkOAuthRefreshTokenUsed(ctx, tokenHash, now); err != nil {
		if errors.Is(err, ErrOAuthRefreshTokenNotFound) {
			return nil, uc.revoke(ctx, grant)
		}

		return nil, fmt.Errorf("spending refresh token: %w", err)
	}

	client, err := uc.storage.OAuthClients().First(ctx, &OAuthClientFilter{ClientID: filter.Equals(clientID)})
	if err != nil {
		return nil, fmt.Errorf("getting OAuth client: %w", err)
	}

	var tokens *OAuthTokens

	err = uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		if err := tx.APIKeys().Delete(ctx, &APIKeyFilter{OAuthGrantID: filter.Equals(&grant.ID)}); err != nil {
			return fmt.Errorf("deleting previous access key: %w", err)
		}

		tokens, _, err = mintOAuthTokens(ctx, tx, grant, client.Name)

		return err
	})
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// revoke ends grant and returns ErrInvalidGrant.
func (uc *RefreshOAuthToken) revoke(ctx context.Context, grant *OAuthGrant) error {
	err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		return deleteOAuthGrants(ctx, tx, &OAuthGrantFilter{ID: filter.Equals(grant.ID)})
	})
	if err != nil {
		return fmt.Errorf("revoking OAuth grant: %w", err)
	}

	return ErrInvalidGrant
}
