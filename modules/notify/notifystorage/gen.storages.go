package notifystorage

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

	notifysvc "github.com/synclet-io/synclet/modules/notify/notifyservice"
	// user code 'imports'
	// end user code 'imports'
)

type Storages struct {
	db         *gorm.DB
	logger     *logging.Logger
	processors []txoutbox.MessageProcessor
}

var _ notifysvc.Storage = &Storages{}

func (s Storages) Webhooks() notifysvc.WebhooksStorage {
	return NewWebhooksStorage(s.db, s.logger)
}
func (s Storages) NotificationChannels() notifysvc.NotificationChannelsStorage {
	return NewNotificationChannelsStorage(s.db, s.logger)
}
func (s Storages) NotificationRules() notifysvc.NotificationRulesStorage {
	return NewNotificationRulesStorage(s.db, s.logger)
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

func (s Storages) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx notifysvc.Storage) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return cb(ctx, &Storages{db: tx, logger: s.logger, processors: s.processors})
	})
}

func NewStorages(db *gorm.DB, logger *logging.Logger, processors []txoutbox.MessageProcessor) *Storages {
	return &Storages{db: db, logger: logger, processors: processors}
}

func NewWebhooksStorage(db *gorm.DB, logger *logging.Logger) notifysvc.WebhooksStorage {
	return dbutil.GormEntityStorage[notifysvc.Webhook, dbWebhook, notifysvc.WebhookFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapWebhookQueryError,
		ConvertToInternal: convertWebhookToDB,
		ConvertToExternal: convertWebhookFromDB,
		BuildFilterExpression: func(filter *notifysvc.WebhookFilter) (clause.Expression, error) {
			return buildWebhookFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			notifysvc.WebhookFieldID:          {Name: "id"},
			notifysvc.WebhookFieldWorkspaceID: {Name: "workspace_id"},
			notifysvc.WebhookFieldURL:         {Name: "url"},
			notifysvc.WebhookFieldEvents:      {Name: "events"},
			notifysvc.WebhookFieldSecret:      {Name: "secret"},
			notifysvc.WebhookFieldEnabled:     {Name: "enabled"},
			notifysvc.WebhookFieldCreatedAt:   {Name: "created_at"},
			notifysvc.WebhookFieldUpdatedAt:   {Name: "updated_at"},
		},
		LockScope: "notify.Webhooks",
	}
}

func NewNotificationChannelsStorage(db *gorm.DB, logger *logging.Logger) notifysvc.NotificationChannelsStorage {
	return dbutil.GormEntityStorage[notifysvc.NotificationChannel, dbNotificationChannel, notifysvc.NotificationChannelFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapNotificationChannelQueryError,
		ConvertToInternal: convertNotificationChannelToDB,
		ConvertToExternal: convertNotificationChannelFromDB,
		BuildFilterExpression: func(filter *notifysvc.NotificationChannelFilter) (clause.Expression, error) {
			return buildNotificationChannelFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			notifysvc.NotificationChannelFieldID:          {Name: "id"},
			notifysvc.NotificationChannelFieldWorkspaceID: {Name: "workspace_id"},
			notifysvc.NotificationChannelFieldName:        {Name: "name"},
			notifysvc.NotificationChannelFieldChannelType: {Name: "channel_type"},
			notifysvc.NotificationChannelFieldConfig:      {Name: "config"},
			notifysvc.NotificationChannelFieldEnabled:     {Name: "enabled"},
			notifysvc.NotificationChannelFieldCreatedAt:   {Name: "created_at"},
			notifysvc.NotificationChannelFieldUpdatedAt:   {Name: "updated_at"},
		},
		LockScope: "notify.NotificationChannels",
	}
}

func NewNotificationRulesStorage(db *gorm.DB, logger *logging.Logger) notifysvc.NotificationRulesStorage {
	return dbutil.GormEntityStorage[notifysvc.NotificationRule, dbNotificationRule, notifysvc.NotificationRuleFilter]{
		Logger:            logger,
		DB:                db,
		DBErrorsWrapper:   wrapNotificationRuleQueryError,
		ConvertToInternal: convertNotificationRuleToDB,
		ConvertToExternal: convertNotificationRuleFromDB,
		BuildFilterExpression: func(filter *notifysvc.NotificationRuleFilter) (clause.Expression, error) {
			return buildNotificationRuleFilterExpr(filter)
		},
		FieldMapping: map[any]clause.Column{
			notifysvc.NotificationRuleFieldID:             {Name: "id"},
			notifysvc.NotificationRuleFieldWorkspaceID:    {Name: "workspace_id"},
			notifysvc.NotificationRuleFieldChannelID:      {Name: "channel_id"},
			notifysvc.NotificationRuleFieldConnectionID:   {Name: "connection_id"},
			notifysvc.NotificationRuleFieldCondition:      {Name: "condition"},
			notifysvc.NotificationRuleFieldConditionValue: {Name: "condition_value"},
			notifysvc.NotificationRuleFieldEnabled:        {Name: "enabled"},
			notifysvc.NotificationRuleFieldCreatedAt:      {Name: "created_at"},
			notifysvc.NotificationRuleFieldUpdatedAt:      {Name: "updated_at"},
		},
		LockScope: "notify.NotificationRules",
	}
}
