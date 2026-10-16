package auditeventhandle

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/audit/auditevent"
	"github.com/synclet-io/synclet/modules/audit/auditservice"
)

type stubPersister struct {
	events     []*auditservice.AuditEvent
	executeErr error
}

func (s *stubPersister) Execute(_ context.Context, event *auditservice.AuditEvent) error {
	s.events = append(s.events, event)

	return s.executeErr
}

func newHandlerWithStub(stub *stubPersister) *AuditEventHandler {
	return &AuditEventHandler{
		logger:  (*logging.Logger)(nil),
		persist: stub,
	}
}

func newRecordedEvent() *auditservice.AuditOutboxEvent {
	return &auditservice.AuditOutboxEvent{
		EventType: auditservice.AuditOutboxEventTypeRecorded,
		Data: &auditservice.AuditEventRecordedData{
			Event: &auditservice.AuditEvent{
				ID:            uuid.New(),
				WorkspaceID:   uuid.New(),
				ActorType:     auditservice.ActorTypeUser,
				ActorID:       uuid.New(),
				ActorLabel:    "alice@example.com",
				Action:        auditservice.ActionCreate,
				ResourceType:  auditservice.ResourceTypeDestination,
				ResourceID:    uuid.New(),
				ResourceLabel: "warehouse",
				DiffJSON:      "[]",
				CreatedAt:     time.Now().UTC().Truncate(time.Microsecond),
			},
		},
	}
}

func encodeMessage(t *testing.T, event *auditservice.AuditOutboxEvent) *message.Message {
	t.Helper()

	metadata, body, err := auditevent.EncodeAuditOutboxEvent(event)
	require.NoError(t, err)

	msg := message.NewMessage(uuid.NewString(), body)
	for k, v := range metadata {
		msg.Metadata.Set(k, v)
	}

	return msg
}

func TestHandle_PersistsRecordedEvent(t *testing.T) {
	stub := &stubPersister{}
	handler := newHandlerWithStub(stub)
	event := newRecordedEvent()

	require.NoError(t, handler.Handle(encodeMessage(t, event)))

	require.Len(t, stub.events, 1)

	data, ok := event.Data.(*auditservice.AuditEventRecordedData)
	require.True(t, ok)
	assert.True(t, data.Event.Equals(stub.events[0]))
}

func TestHandle_DecodeFailureIsAcked(t *testing.T) {
	stub := &stubPersister{}
	handler := newHandlerWithStub(stub)

	msg := message.NewMessage(uuid.NewString(), []byte("{}"))
	require.NoError(t, handler.Handle(msg), "decode failure must be acked, not retried")
	assert.Empty(t, stub.events)
}

func TestHandle_ValidationErrorIsAcked(t *testing.T) {
	stub := &stubPersister{executeErr: &auditservice.ValidationError{Message: "audit event id is empty"}}
	handler := newHandlerWithStub(stub)

	require.NoError(t, handler.Handle(encodeMessage(t, newRecordedEvent())), "unstorable events must be acked, not retried")
	assert.Len(t, stub.events, 1)
}

func TestHandle_PersistErrorPropagated(t *testing.T) {
	stub := &stubPersister{executeErr: errors.New("db down")}
	handler := newHandlerWithStub(stub)

	err := handler.Handle(encodeMessage(t, newRecordedEvent()))
	require.ErrorIs(t, err, stub.executeErr, "storage failures must nack so Watermill retries")
}

func TestHandlerMetadata(t *testing.T) {
	handler := newHandlerWithStub(&stubPersister{})
	assert.NotEmpty(t, handler.Name())
	assert.Equal(t, auditevent.AuditEventTopic, handler.Topic())
}
