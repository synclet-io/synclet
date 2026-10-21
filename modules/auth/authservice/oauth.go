package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

const (
	// OAuthCodeTTL is how long an authorization code can be exchanged.
	OAuthCodeTTL = 10 * time.Minute
	// OAuthAccessKeyTTL is the lifetime of the API key issued as an access token.
	OAuthAccessKeyTTL = time.Hour
	// OAuthRefreshTokenTTL is the lifetime of a refresh token.
	OAuthRefreshTokenTTL = 30 * 24 * time.Hour

	// OAuthUnusedClientTTL is how long a registered client that never got a
	// grant is kept.
	OAuthUnusedClientTTL = 24 * time.Hour

	oauthChallengeMethodS256 = "S256"
)

// OAuthStorage holds the OAuth queries the generated storage cannot express.
type OAuthStorage interface {
	// ConsumeOAuthCode deletes the code with codeHash and returns it, or
	// ErrOAuthCodeNotFound, so a code is redeemed at most once.
	ConsumeOAuthCode(ctx context.Context, codeHash string) (*OAuthCode, error)
	// MarkOAuthRefreshTokenUsed sets used_at of the unused refresh token with
	// tokenHash and returns it, or ErrOAuthRefreshTokenNotFound when no unused
	// token has that hash.
	MarkOAuthRefreshTokenUsed(ctx context.Context, tokenHash string, at time.Time) (*OAuthRefreshToken, error)
	// DeleteExpiredOAuthKeys deletes the OAuth-issued API keys that expired
	// before before, except the latest key of a grant that holds an unused
	// refresh token unexpired at before: that key keeps the grant listed and
	// revocable until the grant ends.
	DeleteExpiredOAuthKeys(ctx context.Context, before time.Time) error
	// DeleteLapsedOAuthGrants deletes the refresh tokens whose expiry is
	// earlier than before, then the grants left without a refresh token, with
	// their API keys.
	DeleteLapsedOAuthGrants(ctx context.Context, before time.Time) error
	// DeleteUnusedOAuthClients deletes the clients without a grant that were
	// registered before neverGrantedBefore and never granted, or last granted
	// before lastGrantedBefore.
	DeleteUnusedOAuthClients(ctx context.Context, neverGrantedBefore, lastGrantedBefore time.Time) error
}

// OAuthTokens is the answer of the token endpoint.
type OAuthTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
}

// mintOAuthTokens creates the grant's access key and a refresh token in storage.
func mintOAuthTokens(ctx context.Context, storage Storage, grant *OAuthGrant, clientName string) (*OAuthTokens, *APIKey, error) {
	now := time.Now()
	level := grant.Level
	grantID := grant.ID

	rawKey, apiKey, err := insertAPIKey(ctx, storage, CreateAPIKeyParams{
		WorkspaceID:  grant.WorkspaceID,
		UserID:       grant.UserID,
		Name:         clientName,
		ExpiresAt:    new(now.Add(OAuthAccessKeyTTL)),
		OAuthGrantID: &grantID,
		OAuthLevel:   &level,
	})
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err := generateRandomString(32)
	if err != nil {
		return nil, nil, fmt.Errorf("generating refresh token: %w", err)
	}

	if _, err := storage.OAuthRefreshTokens().Create(ctx, &OAuthRefreshToken{
		ID:        uuid.New(),
		GrantID:   grant.ID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: now.Add(OAuthRefreshTokenTTL),
		CreatedAt: now,
	}); err != nil {
		return nil, nil, fmt.Errorf("creating refresh token: %w", err)
	}

	return &OAuthTokens{AccessToken: rawKey, RefreshToken: refreshToken, ExpiresIn: OAuthAccessKeyTTL}, apiKey, nil
}

// deleteOAuthGrants deletes the grants flt matches with their API keys and
// refresh tokens.
func deleteOAuthGrants(ctx context.Context, storage Storage, flt *OAuthGrantFilter) error {
	grants, err := storage.OAuthGrants().Find(ctx, flt)
	if err != nil {
		return fmt.Errorf("finding OAuth grants: %w", err)
	}

	for _, grant := range grants {
		if err := storage.APIKeys().Delete(ctx, &APIKeyFilter{OAuthGrantID: filter.Equals(&grant.ID)}); err != nil {
			return fmt.Errorf("deleting OAuth grant keys: %w", err)
		}

		if err := storage.OAuthRefreshTokens().Delete(ctx, &OAuthRefreshTokenFilter{GrantID: filter.Equals(grant.ID)}); err != nil {
			return fmt.Errorf("deleting OAuth refresh tokens: %w", err)
		}

		if err := storage.OAuthGrants().Delete(ctx, &OAuthGrantFilter{ID: filter.Equals(grant.ID)}); err != nil {
			return fmt.Errorf("deleting OAuth grant: %w", err)
		}
	}

	return nil
}

// workspaceRole returns the user's role in the workspace, or
// WorkspaceRoleUnspecified when the user is not a member.
func workspaceRole(ctx context.Context, members WorkspaceMembershipLister, userID, workspaceID uuid.UUID) (WorkspaceRole, error) {
	memberships, err := members.ListForUser(ctx, userID)
	if err != nil {
		return WorkspaceRoleUnspecified, fmt.Errorf("listing workspace memberships: %w", err)
	}

	for _, m := range memberships {
		if m.WorkspaceID == workspaceID {
			return m.Role, nil
		}
	}

	return WorkspaceRoleUnspecified, nil
}
