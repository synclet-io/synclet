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
	APIKeys() APIKeysStorage
	OIDCIdentitys() OIDCIdentitysStorage
	OIDCStates() OIDCStatesStorage
	IdempotencyKeys() idempotency.Storage
	ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error
	WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error
}
type UsersStorage dbutil.EntityStorage[User, UserFilter]
type RefreshTokensStorage dbutil.EntityStorage[RefreshToken, RefreshTokenFilter]
type APIKeysStorage dbutil.EntityStorage[APIKey, APIKeyFilter]
type OIDCIdentitysStorage dbutil.EntityStorage[OIDCIdentity, OIDCIdentityFilter]
type OIDCStatesStorage dbutil.EntityStorage[OIDCState, OIDCStateFilter]
