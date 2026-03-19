package notifyservice

import (
	context "context"

	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	// user code 'imports'
	// end user code 'imports'
)

type Storage interface {
	Webhooks() WebhooksStorage
	NotificationChannels() NotificationChannelsStorage
	NotificationRules() NotificationRulesStorage
	IdempotencyKeys() idempotency.Storage
	ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error
	WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error
}
type WebhooksStorage dbutil.EntityStorage[Webhook, WebhookFilter]
type NotificationChannelsStorage dbutil.EntityStorage[NotificationChannel, NotificationChannelFilter]
type NotificationRulesStorage dbutil.EntityStorage[NotificationRule, NotificationRuleFilter]
