// Package auditeventhandle contains the Watermill handlers that consume audit
// events. Handlers here treat the auditevent package as the contract of the
// audit topic and reach the audit module only through its use cases.
package auditeventhandle

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"

	"github.com/synclet-io/synclet/modules/audit/auditevent"
	"github.com/synclet-io/synclet/modules/audit/auditservice"
)

// auditEventPersister is the narrow port the handler depends on. The real
// implementation is auditservice.PersistAuditEvent; tests substitute a stub.
type auditEventPersister interface {
	Execute(ctx context.Context, event *auditservice.AuditEvent) error
}

// AuditEventHandler persists audit events delivered through the bus. Messages
// that can never be stored (undecodable payloads, invalid events) are logged
// and acked; storage failures are returned so the bus retries the delivery.
type AuditEventHandler struct {
	logger  *logging.Logger
	persist auditEventPersister
}

var (
	_ pnpwatermill.Handler = (*AuditEventHandler)(nil)
	_ auditEventPersister  = (*auditservice.PersistAuditEvent)(nil)
)

// NewAuditEventHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the audit FX module.
func NewAuditEventHandler(logger *logging.Logger, persist *auditservice.PersistAuditEvent) *AuditEventHandler {
	return &AuditEventHandler{
		logger:  logger.Named("audit-event-handler"),
		persist: persist,
	}
}

// Name returns the handler identifier used by pnpwatermill for the
// subscriber. Must be unique per process.
func (h *AuditEventHandler) Name() string {
	return "audit_persist_audit_event"
}

// Topic returns the topic this handler subscribes to.
func (h *AuditEventHandler) Topic() string {
	return auditevent.AuditEventTopic
}

// Handle decodes the message and persists the audit event it carries.
func (h *AuditEventHandler) Handle(msg *message.Message) error {
	ctx := msg.Context()

	event, err := auditevent.DecodeAuditOutboxEvent(msg.Metadata, msg.Payload)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "cannot decode audit event; acking and dropping")

		return nil
	}

	recorded, err := auditevent.RecordedEvent(event)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "audit event carries no audit record; acking and dropping")

		return nil
	}

	err = h.persist.Execute(ctx, recorded)

	var validationErr *auditservice.ValidationError
	if errors.As(err, &validationErr) {
		h.logger.WithError(err).
			WithField("audit_event_id", recorded.ID.String()).
			Error(ctx, "audit event cannot be stored; acking and dropping")

		return nil
	}

	if err != nil {
		return fmt.Errorf("persisting audit event %s: %w", recorded.ID, err)
	}

	return nil
}
