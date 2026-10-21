package authservice

import (
	context "context"

	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	// user code 'imports'
	// end user code 'imports'
)

type Storage interface {
	Users() UsersStorage
	RefreshTokens() RefreshTokensStorage
	PasswordResetTokens() PasswordResetTokensStorage
	APIKeys() APIKeysStorage
	OAuthClients() OAuthClientsStorage
	OAuthCodes() OAuthCodesStorage
	OAuthGrants() OAuthGrantsStorage
	OAuthRefreshTokens() OAuthRefreshTokensStorage
	OIDCIdentities() OIDCIdentitiesStorage
	OIDCStates() OIDCStatesStorage
	IdempotencyKeys() idempotency.Storage
	ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error
	WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error
}
type UsersStorage dbutil.EntityStorage[User, UserFilter]
type RefreshTokensStorage dbutil.EntityStorage[RefreshToken, RefreshTokenFilter]
type PasswordResetTokensStorage dbutil.EntityStorage[PasswordResetToken, PasswordResetTokenFilter]
type APIKeysStorage dbutil.EntityStorage[APIKey, APIKeyFilter]
type OAuthClientsStorage dbutil.EntityStorage[OAuthClient, OAuthClientFilter]
type OAuthCodesStorage dbutil.EntityStorage[OAuthCode, OAuthCodeFilter]
type OAuthGrantsStorage dbutil.EntityStorage[OAuthGrant, OAuthGrantFilter]
type OAuthRefreshTokensStorage dbutil.EntityStorage[OAuthRefreshToken, OAuthRefreshTokenFilter]
type OIDCIdentitiesStorage dbutil.EntityStorage[OIDCIdentity, OIDCIdentityFilter]
type OIDCStatesStorage dbutil.EntityStorage[OIDCState, OIDCStateFilter]
