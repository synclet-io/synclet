package authstorage

import (
	context "context"
	strconv "strconv"

	xxhash "github.com/cespare/xxhash"
	logging "github.com/go-pnp/go-pnp/logging"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	txoutbox "github.com/saturn4er/boilerplate-go/lib/txoutbox"
	gorm "gorm.io/gorm"
	clause "gorm.io/gorm/clause"

	authsvc "github.com/synclet-io/synclet/modules/auth/authservice"
	// user code 'imports'
	// end user code 'imports'
)

type Storages struct {
	db         *gorm.DB
	logger     *logging.Logger
	processors []txoutbox.MessageProcessor
}

var _ authsvc.Storage = &Storages{}

func (s Storages) Users() authsvc.UsersStorage {
	return NewUsersStorage(s.db, s.logger)
}
func (s Storages) RefreshTokens() authsvc.RefreshTokensStorage {
	return NewRefreshTokensStorage(s.db, s.logger)
}
func (s Storages) APIKeys() authsvc.APIKeysStorage {
	return NewAPIKeysStorage(s.db, s.logger)
}
func (s Storages) OIDCIdentitys() authsvc.OIDCIdentitysStorage {
	return NewOIDCIdentitysStorage(s.db, s.logger)
}
func (s Storages) OIDCStates() authsvc.OIDCStatesStorage {
	return NewOIDCStatesStorage(s.db, s.logger)
}

func (s Storages) IdempotencyKeys() idempotency.Storage {
	return idempotency.GormStorage{
		DB: s.db,
	}

}

func (s *Storages) WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error {
	hasher := xxhash.New()
	hasher.Write([]byte(scope))
	hasher.Write([]byte{':'})
	hasher.Write(strconv.AppendInt(nil, lockID, 10))

	result := s.db.WithContext(ctx).Exec("SELECT pg_advisory_xact_lock(?)", int64(hasher.Sum64()))
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (s Storages) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx authsvc.Storage) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return cb(ctx, &Storages{db: tx, logger: s.logger, processors: s.processors})
	})
}

func NewStorages(db *gorm.DB, logger *logging.Logger, processors []txoutbox.MessageProcessor) *Storages {
	return &Storages{db: db, logger: logger, processors: processors}
}

func NewUsersStorage(db *gorm.DB, logger *logging.Logger) authsvc.UsersStorage {
	return dbutil.GormEntityStorage[authsvc.User, dbUser, authsvc.UserFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapUserQueryError,
		ConvertToInternal: convertUserToDB,
		ConvertToExternal: convertUserFromDB,
		BuildFilterExpression: func(filter *authsvc.UserFilter) (clause.Expression, error) {
			return buildUserFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.UserFieldID:           {Name: "id"},
			authsvc.UserFieldEmail:        {Name: "email"},
			authsvc.UserFieldPasswordHash: {Name: "password_hash"},
			authsvc.UserFieldName:         {Name: "name"},
			authsvc.UserFieldCreatedAt:    {Name: "created_at"},
		},
		LockScope: "auth.Users",
	}
}

func NewRefreshTokensStorage(db *gorm.DB, logger *logging.Logger) authsvc.RefreshTokensStorage {
	return dbutil.GormEntityStorage[authsvc.RefreshToken, dbRefreshToken, authsvc.RefreshTokenFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapRefreshTokenQueryError,
		ConvertToInternal: convertRefreshTokenToDB,
		ConvertToExternal: convertRefreshTokenFromDB,
		BuildFilterExpression: func(filter *authsvc.RefreshTokenFilter) (clause.Expression, error) {
			return buildRefreshTokenFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.RefreshTokenFieldID:        {Name: "id"},
			authsvc.RefreshTokenFieldUserID:    {Name: "user_id"},
			authsvc.RefreshTokenFieldTokenHash: {Name: "token_hash"},
			authsvc.RefreshTokenFieldExpiresAt: {Name: "expires_at"},
			authsvc.RefreshTokenFieldCreatedAt: {Name: "created_at"},
		},
		LockScope: "auth.RefreshTokens",
	}
}

func NewAPIKeysStorage(db *gorm.DB, logger *logging.Logger) authsvc.APIKeysStorage {
	return dbutil.GormEntityStorage[authsvc.APIKey, dbAPIKey, authsvc.APIKeyFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapAPIKeyQueryError,
		ConvertToInternal: convertAPIKeyToDB,
		ConvertToExternal: convertAPIKeyFromDB,
		BuildFilterExpression: func(filter *authsvc.APIKeyFilter) (clause.Expression, error) {
			return buildAPIKeyFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.APIKeyFieldID:          {Name: "id"},
			authsvc.APIKeyFieldWorkspaceID: {Name: "workspace_id"},
			authsvc.APIKeyFieldUserID:      {Name: "user_id"},
			authsvc.APIKeyFieldName:        {Name: "name"},
			authsvc.APIKeyFieldKeyHash:     {Name: "key_hash"},
			authsvc.APIKeyFieldLastUsedAt:  {Name: "last_used_at"},
			authsvc.APIKeyFieldExpiresAt:   {Name: "expires_at"},
			authsvc.APIKeyFieldCreatedAt:   {Name: "created_at"},
		},
		LockScope: "auth.APIKeys",
	}
}

func NewOIDCIdentitysStorage(db *gorm.DB, logger *logging.Logger) authsvc.OIDCIdentitysStorage {
	return dbutil.GormEntityStorage[authsvc.OIDCIdentity, dbOIDCIdentity, authsvc.OIDCIdentityFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapOIDCIdentityQueryError,
		ConvertToInternal: convertOIDCIdentityToDB,
		ConvertToExternal: convertOIDCIdentityFromDB,
		BuildFilterExpression: func(filter *authsvc.OIDCIdentityFilter) (clause.Expression, error) {
			return buildOIDCIdentityFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.OIDCIdentityFieldID:           {Name: "id"},
			authsvc.OIDCIdentityFieldUserID:       {Name: "user_id"},
			authsvc.OIDCIdentityFieldProviderSlug: {Name: "provider_slug"},
			authsvc.OIDCIdentityFieldSubject:      {Name: "subject"},
			authsvc.OIDCIdentityFieldEmail:        {Name: "email"},
			authsvc.OIDCIdentityFieldCreatedAt:    {Name: "created_at"},
			authsvc.OIDCIdentityFieldLastLoginAt:  {Name: "last_login_at"},
		},
		LockScope: "auth.OIDCIdentitys",
	}
}

func NewOIDCStatesStorage(db *gorm.DB, logger *logging.Logger) authsvc.OIDCStatesStorage {
	return dbutil.GormEntityStorage[authsvc.OIDCState, dbOIDCState, authsvc.OIDCStateFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapOIDCStateQueryError,
		ConvertToInternal: convertOIDCStateToDB,
		ConvertToExternal: convertOIDCStateFromDB,
		BuildFilterExpression: func(filter *authsvc.OIDCStateFilter) (clause.Expression, error) {
			return buildOIDCStateFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.OIDCStateFieldID:           {Name: "id"},
			authsvc.OIDCStateFieldState:        {Name: "state"},
			authsvc.OIDCStateFieldVerifier:     {Name: "verifier"},
			authsvc.OIDCStateFieldProviderSlug: {Name: "provider_slug"},
			authsvc.OIDCStateFieldExpiresAt:    {Name: "expires_at"},
			authsvc.OIDCStateFieldCreatedAt:    {Name: "created_at"},
		},
		LockScope: "auth.OIDCStates",
	}
}
