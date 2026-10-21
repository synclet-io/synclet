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
func (s Storages) PasswordResetTokens() authsvc.PasswordResetTokensStorage {
	return NewPasswordResetTokensStorage(s.db, s.logger)
}
func (s Storages) APIKeys() authsvc.APIKeysStorage {
	return NewAPIKeysStorage(s.db, s.logger)
}
func (s Storages) OAuthClients() authsvc.OAuthClientsStorage {
	return NewOAuthClientsStorage(s.db, s.logger)
}
func (s Storages) OAuthCodes() authsvc.OAuthCodesStorage {
	return NewOAuthCodesStorage(s.db, s.logger)
}
func (s Storages) OAuthGrants() authsvc.OAuthGrantsStorage {
	return NewOAuthGrantsStorage(s.db, s.logger)
}
func (s Storages) OAuthRefreshTokens() authsvc.OAuthRefreshTokensStorage {
	return NewOAuthRefreshTokensStorage(s.db, s.logger)
}
func (s Storages) OIDCIdentities() authsvc.OIDCIdentitiesStorage {
	return NewOIDCIdentitiesStorage(s.db, s.logger)
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

func NewPasswordResetTokensStorage(db *gorm.DB, logger *logging.Logger) authsvc.PasswordResetTokensStorage {
	return dbutil.GormEntityStorage[authsvc.PasswordResetToken, dbPasswordResetToken, authsvc.PasswordResetTokenFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapPasswordResetTokenQueryError,
		ConvertToInternal: convertPasswordResetTokenToDB,
		ConvertToExternal: convertPasswordResetTokenFromDB,
		BuildFilterExpression: func(filter *authsvc.PasswordResetTokenFilter) (clause.Expression, error) {
			return buildPasswordResetTokenFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.PasswordResetTokenFieldID:        {Name: "id"},
			authsvc.PasswordResetTokenFieldUserID:    {Name: "user_id"},
			authsvc.PasswordResetTokenFieldTokenHash: {Name: "token_hash"},
			authsvc.PasswordResetTokenFieldExpiresAt: {Name: "expires_at"},
			authsvc.PasswordResetTokenFieldCreatedAt: {Name: "created_at"},
		},
		LockScope: "auth.PasswordResetTokens",
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
			authsvc.APIKeyFieldID:           {Name: "id"},
			authsvc.APIKeyFieldWorkspaceID:  {Name: "workspace_id"},
			authsvc.APIKeyFieldUserID:       {Name: "user_id"},
			authsvc.APIKeyFieldName:         {Name: "name"},
			authsvc.APIKeyFieldKeyHash:      {Name: "key_hash"},
			authsvc.APIKeyFieldLastUsedAt:   {Name: "last_used_at"},
			authsvc.APIKeyFieldExpiresAt:    {Name: "expires_at"},
			authsvc.APIKeyFieldOAuthGrantID: {Name: "oauth_grant_id"},
			authsvc.APIKeyFieldOAuthLevel:   {Name: "oauth_level"},
			authsvc.APIKeyFieldCreatedAt:    {Name: "created_at"},
		},
		LockScope: "auth.APIKeys",
	}
}

func NewOAuthClientsStorage(db *gorm.DB, logger *logging.Logger) authsvc.OAuthClientsStorage {
	return dbutil.GormEntityStorage[authsvc.OAuthClient, dbOAuthClient, authsvc.OAuthClientFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapOAuthClientQueryError,
		ConvertToInternal: convertOAuthClientToDB,
		ConvertToExternal: convertOAuthClientFromDB,
		BuildFilterExpression: func(filter *authsvc.OAuthClientFilter) (clause.Expression, error) {
			return buildOAuthClientFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.OAuthClientFieldID:            {Name: "id"},
			authsvc.OAuthClientFieldClientID:      {Name: "client_id"},
			authsvc.OAuthClientFieldName:          {Name: "name"},
			authsvc.OAuthClientFieldRedirectURIs:  {Name: "redirect_uris"},
			authsvc.OAuthClientFieldLastGrantedAt: {Name: "last_granted_at"},
			authsvc.OAuthClientFieldCreatedAt:     {Name: "created_at"},
		},
		LockScope: "auth.OAuthClients",
	}
}

func NewOAuthCodesStorage(db *gorm.DB, logger *logging.Logger) authsvc.OAuthCodesStorage {
	return dbutil.GormEntityStorage[authsvc.OAuthCode, dbOAuthCode, authsvc.OAuthCodeFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapOAuthCodeQueryError,
		ConvertToInternal: convertOAuthCodeToDB,
		ConvertToExternal: convertOAuthCodeFromDB,
		BuildFilterExpression: func(filter *authsvc.OAuthCodeFilter) (clause.Expression, error) {
			return buildOAuthCodeFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.OAuthCodeFieldID:            {Name: "id"},
			authsvc.OAuthCodeFieldCodeHash:      {Name: "code_hash"},
			authsvc.OAuthCodeFieldClientID:      {Name: "client_id"},
			authsvc.OAuthCodeFieldRedirectURI:   {Name: "redirect_uri"},
			authsvc.OAuthCodeFieldCodeChallenge: {Name: "code_challenge"},
			authsvc.OAuthCodeFieldUserID:        {Name: "user_id"},
			authsvc.OAuthCodeFieldWorkspaceID:   {Name: "workspace_id"},
			authsvc.OAuthCodeFieldLevel:         {Name: "level"},
			authsvc.OAuthCodeFieldExpiresAt:     {Name: "expires_at"},
			authsvc.OAuthCodeFieldCreatedAt:     {Name: "created_at"},
		},
		LockScope: "auth.OAuthCodes",
	}
}

func NewOAuthGrantsStorage(db *gorm.DB, logger *logging.Logger) authsvc.OAuthGrantsStorage {
	return dbutil.GormEntityStorage[authsvc.OAuthGrant, dbOAuthGrant, authsvc.OAuthGrantFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapOAuthGrantQueryError,
		ConvertToInternal: convertOAuthGrantToDB,
		ConvertToExternal: convertOAuthGrantFromDB,
		BuildFilterExpression: func(filter *authsvc.OAuthGrantFilter) (clause.Expression, error) {
			return buildOAuthGrantFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.OAuthGrantFieldID:          {Name: "id"},
			authsvc.OAuthGrantFieldClientID:    {Name: "client_id"},
			authsvc.OAuthGrantFieldUserID:      {Name: "user_id"},
			authsvc.OAuthGrantFieldWorkspaceID: {Name: "workspace_id"},
			authsvc.OAuthGrantFieldLevel:       {Name: "level"},
			authsvc.OAuthGrantFieldCreatedAt:   {Name: "created_at"},
		},
		LockScope: "auth.OAuthGrants",
	}
}

func NewOAuthRefreshTokensStorage(db *gorm.DB, logger *logging.Logger) authsvc.OAuthRefreshTokensStorage {
	return dbutil.GormEntityStorage[authsvc.OAuthRefreshToken, dbOAuthRefreshToken, authsvc.OAuthRefreshTokenFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapOAuthRefreshTokenQueryError,
		ConvertToInternal: convertOAuthRefreshTokenToDB,
		ConvertToExternal: convertOAuthRefreshTokenFromDB,
		BuildFilterExpression: func(filter *authsvc.OAuthRefreshTokenFilter) (clause.Expression, error) {
			return buildOAuthRefreshTokenFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			authsvc.OAuthRefreshTokenFieldID:        {Name: "id"},
			authsvc.OAuthRefreshTokenFieldGrantID:   {Name: "grant_id"},
			authsvc.OAuthRefreshTokenFieldTokenHash: {Name: "token_hash"},
			authsvc.OAuthRefreshTokenFieldUsedAt:    {Name: "used_at"},
			authsvc.OAuthRefreshTokenFieldExpiresAt: {Name: "expires_at"},
			authsvc.OAuthRefreshTokenFieldCreatedAt: {Name: "created_at"},
		},
		LockScope: "auth.OAuthRefreshTokens",
	}
}

func NewOIDCIdentitiesStorage(db *gorm.DB, logger *logging.Logger) authsvc.OIDCIdentitiesStorage {
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
		LockScope: "auth.OIDCIdentities",
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
