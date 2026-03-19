package secretstorage

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

	secretsvc "github.com/synclet-io/synclet/modules/secret/secretservice"
	// user code 'imports'
	// end user code 'imports'
)

type Storages struct {
	db         *gorm.DB
	logger     *logging.Logger
	processors []txoutbox.MessageProcessor
}

var _ secretsvc.Storage = &Storages{}

func (s Storages) Secrets() secretsvc.SecretsStorage {
	return NewSecretsStorage(s.db, s.logger)
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

func (s Storages) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx secretsvc.Storage) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return cb(ctx, &Storages{db: tx, logger: s.logger, processors: s.processors})
	})
}

func NewStorages(db *gorm.DB, logger *logging.Logger, processors []txoutbox.MessageProcessor) *Storages {
	return &Storages{db: db, logger: logger, processors: processors}
}

func NewSecretsStorage(db *gorm.DB, logger *logging.Logger) secretsvc.SecretsStorage {
	return dbutil.GormEntityStorage[secretsvc.Secret, dbSecret, secretsvc.SecretFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapSecretQueryError,
		ConvertToInternal: convertSecretToDB,
		ConvertToExternal: convertSecretFromDB,
		BuildFilterExpression: func(filter *secretsvc.SecretFilter) (clause.Expression, error) {
			return buildSecretFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			secretsvc.SecretFieldID:             {Name: "id"},
			secretsvc.SecretFieldEncryptedValue: {Name: "encrypted_value"},
			secretsvc.SecretFieldSalt:           {Name: "salt"},
			secretsvc.SecretFieldNonce:          {Name: "nonce"},
			secretsvc.SecretFieldKeyVersion:     {Name: "key_version"},
			secretsvc.SecretFieldOwnerType:      {Name: "owner_type"},
			secretsvc.SecretFieldOwnerID:        {Name: "owner_id"},
			secretsvc.SecretFieldCreatedAt:      {Name: "created_at"},
			secretsvc.SecretFieldUpdatedAt:      {Name: "updated_at"},
		},
		LockScope: "secret.Secrets",
	}
}
