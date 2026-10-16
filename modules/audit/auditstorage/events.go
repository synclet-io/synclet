package auditstorage

import (
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/txoutbox"

	"github.com/synclet-io/synclet/modules/audit/auditevent"
	auditsvc "github.com/synclet-io/synclet/modules/audit/auditservice"
)

// buildAuditOutboxEventMessage is invoked by the generated
// AuditOutboxEventsOutbox each time a use case sends an audit event. It
// encodes the event into the JSON wire format defined in the auditevent
// package and wraps it in a txoutbox.Message tagged with the audit topic.
// Messages are ordered per workspace and deduplicated by audit event id.
func buildAuditOutboxEventMessage(event *auditsvc.AuditOutboxEvent) (*txoutbox.Message, error) {
	metadata, body, err := auditevent.EncodeAuditOutboxEvent(event)
	if err != nil {
		return nil, fmt.Errorf("encoding audit event: %w", err)
	}

	orderingKey, err := auditevent.OrderingKey(event)
	if err != nil {
		return nil, fmt.Errorf("computing ordering key: %w", err)
	}

	idempotencyKey, err := auditevent.IdempotencyKey(event)
	if err != nil {
		return nil, fmt.Errorf("computing idempotency key: %w", err)
	}

	return &txoutbox.Message{
		Topic:          auditevent.AuditEventTopic,
		OrderingKey:    orderingKey,
		IdempotencyKey: idempotencyKey,
		Data:           body,
		Metadata:       metadata,
	}, nil
}
