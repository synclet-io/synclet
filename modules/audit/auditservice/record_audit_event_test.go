package auditservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validRecordParams() RecordAuditEventParams {
	return RecordAuditEventParams{
		WorkspaceID:   uuid.New(),
		ActorType:     ActorTypeUser,
		ActorID:       uuid.New(),
		ActorLabel:    "alice@example.com",
		Action:        ActionUpdate,
		ResourceType:  ResourceTypeSource,
		ResourceID:    uuid.New(),
		ResourceLabel: "postgres-prod",
		Before:        map[string]any{"name": "old"},
		After:         map[string]any{"name": "new"},
	}
}

func TestRecordAuditEvent_SendsRecordedEventToOutbox(t *testing.T) {
	storage := newFakeStorage()
	uc := NewRecordAuditEvent(storage, nil)
	params := validRecordParams()
	before := time.Now()

	uc.Execute(context.Background(), params)

	require.Len(t, storage.outbox, 1)
	assert.Empty(t, storage.events, "the event is persisted by the bus consumer, not inline")

	sent := storage.outbox[0]
	assert.Equal(t, AuditOutboxEventTypeRecorded, sent.EventType)

	data, ok := sent.Data.(*AuditEventRecordedData)
	require.True(t, ok, "outbox payload must be AuditEventRecordedData, got %T", sent.Data)
	require.NotNil(t, data.Event)

	event := data.Event
	assert.NotEqual(t, uuid.Nil, event.ID)
	assert.Equal(t, params.WorkspaceID, event.WorkspaceID)
	assert.Equal(t, params.ActorType, event.ActorType)
	assert.Equal(t, params.ActorID, event.ActorID)
	assert.Equal(t, params.ActorLabel, event.ActorLabel)
	assert.Equal(t, params.Action, event.Action)
	assert.Equal(t, params.ResourceType, event.ResourceType)
	assert.Equal(t, params.ResourceID, event.ResourceID)
	assert.Equal(t, params.ResourceLabel, event.ResourceLabel)
	assert.JSONEq(t, `[{"path":"name","before":"old","after":"new"}]`, event.DiffJSON)
	assert.False(t, event.DiffTruncated)
	assert.False(t, event.CreatedAt.Before(before))
}

func TestRecordAuditEvent_EachEventGetsOwnID(t *testing.T) {
	storage := newFakeStorage()
	uc := NewRecordAuditEvent(storage, nil)

	uc.Execute(context.Background(), validRecordParams())
	uc.Execute(context.Background(), validRecordParams())

	require.Len(t, storage.outbox, 2)

	first, ok := storage.outbox[0].Data.(*AuditEventRecordedData)
	require.True(t, ok)

	second, ok := storage.outbox[1].Data.(*AuditEventRecordedData)
	require.True(t, ok)

	assert.NotEqual(t, first.Event.ID, second.Event.ID)
}

func TestRecordAuditEvent_SkipsInvalidEnums(t *testing.T) {
	cases := map[string]func(*RecordAuditEventParams){
		"action":        func(p *RecordAuditEventParams) { p.Action = Action(0) },
		"resource type": func(p *RecordAuditEventParams) { p.ResourceType = ResourceType(99) },
		"actor type":    func(p *RecordAuditEventParams) { p.ActorType = ActorType(0) },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			storage := newFakeStorage()
			uc := NewRecordAuditEvent(storage, nil)
			params := validRecordParams()
			mutate(&params)

			uc.Execute(context.Background(), params)

			assert.Empty(t, storage.outbox)
			assert.Empty(t, storage.events)
		})
	}
}

func TestRecordAuditEvent_OutboxFailureIsSwallowed(t *testing.T) {
	storage := newFakeStorage()
	storage.outboxErr = errors.New("db down")
	uc := NewRecordAuditEvent(storage, nil)

	uc.Execute(context.Background(), validRecordParams())

	assert.Empty(t, storage.outbox)
	assert.Empty(t, storage.events)
}
