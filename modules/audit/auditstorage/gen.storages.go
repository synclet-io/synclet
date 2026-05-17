package auditstorage

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

	auditsvc "github.com/synclet-io/synclet/modules/audit/auditservice"
	// user code 'imports'
	// end user code 'imports'
)

type Storages struct {
	db         *gorm.DB
	logger     *logging.Logger
	processors []txoutbox.MessageProcessor
}

var _ auditsvc.Storage = &Storages{}

func (s Storages) AuditEvents() auditsvc.AuditEventsStorage {
	return NewAuditEventsStorage(s.db, s.logger)
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

func (s Storages) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx auditsvc.Storage) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return cb(ctx, &Storages{db: tx, logger: s.logger, processors: s.processors})
	})
}

func NewStorages(db *gorm.DB, logger *logging.Logger, processors []txoutbox.MessageProcessor) *Storages {
	return &Storages{db: db, logger: logger, processors: processors}
}

func NewAuditEventsStorage(db *gorm.DB, logger *logging.Logger) auditsvc.AuditEventsStorage {
	return dbutil.GormEntityStorage[auditsvc.AuditEvent, dbAuditEvent, auditsvc.AuditEventFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapAuditEventQueryError,
		ConvertToInternal: convertAuditEventToDB,
		ConvertToExternal: convertAuditEventFromDB,
		BuildFilterExpression: func(filter *auditsvc.AuditEventFilter) (clause.Expression, error) {
			return buildAuditEventFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			auditsvc.AuditEventFieldID:            {Name: "id"},
			auditsvc.AuditEventFieldWorkspaceID:   {Name: "workspace_id"},
			auditsvc.AuditEventFieldActorType:     {Name: "actor_type"},
			auditsvc.AuditEventFieldActorID:       {Name: "actor_id"},
			auditsvc.AuditEventFieldActorLabel:    {Name: "actor_label"},
			auditsvc.AuditEventFieldAction:        {Name: "action"},
			auditsvc.AuditEventFieldResourceType:  {Name: "resource_type"},
			auditsvc.AuditEventFieldResourceID:    {Name: "resource_id"},
			auditsvc.AuditEventFieldResourceLabel: {Name: "resource_label"},
			auditsvc.AuditEventFieldDiffJSON:      {Name: "diff_json"},
			auditsvc.AuditEventFieldDiffTruncated: {Name: "diff_truncated"},
			auditsvc.AuditEventFieldCreatedAt:     {Name: "created_at"},
		},
		LockScope: "audit.AuditEvents",
	}
}
