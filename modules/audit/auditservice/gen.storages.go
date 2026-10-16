package auditservice

import (
	context "context"

	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	txoutbox "github.com/saturn4er/boilerplate-go/lib/txoutbox"
	// user code 'imports'
	// end user code 'imports'
)

type Storage interface {
	AuditEvents() AuditEventsStorage
	AuditRetentionSettings() AuditRetentionSettingsStorage
	AuditOutboxEvents() AuditOutboxEventsOutbox
	IdempotencyKeys() idempotency.Storage
	ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error
	WithAdvisoryLock(ctx context.Context, scope string, lockID int64) error
}
type AuditEventsStorage dbutil.EntityStorage[AuditEvent, AuditEventFilter]
type AuditRetentionSettingsStorage dbutil.EntityStorage[AuditRetentionSetting, AuditRetentionSettingFilter]
type AuditOutboxEventsOutbox txoutbox.Outbox[AuditOutboxEvent]
