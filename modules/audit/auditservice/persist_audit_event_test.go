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

func newPersistableEvent() *AuditEvent {
	return &AuditEvent{
		ID:            uuid.New(),
		WorkspaceID:   uuid.New(),
		ActorType:     ActorTypeAPIKey,
		ActorID:       uuid.New(),
		ActorLabel:    "ci-key",
		Action:        ActionDelete,
		ResourceType:  ResourceTypeWebhook,
		ResourceID:    uuid.New(),
		ResourceLabel: "slack-hook",
		DiffJSON:      "[]",
		CreatedAt:     time.Now(),
	}
}

func TestPersistAuditEvent_InsertsEvent(t *testing.T) {
	storage := newFakeStorage()
	uc := NewPersistAuditEvent(storage)
	event := newPersistableEvent()

	require.NoError(t, uc.Execute(context.Background(), event))

	require.Len(t, storage.events, 1)
	assert.True(t, event.Equals(storage.events[0]))
}

func TestPersistAuditEvent_DuplicateIDIsSuccess(t *testing.T) {
	storage := newFakeStorage()
	uc := NewPersistAuditEvent(storage)
	event := newPersistableEvent()

	require.NoError(t, uc.Execute(context.Background(), event))

	redelivered := event.Copy()
	require.NoError(t, uc.Execute(context.Background(), &redelivered))

	require.Len(t, storage.events, 1)
	assert.True(t, event.Equals(storage.events[0]))
}

func TestPersistAuditEvent_RejectsUnstorableEvents(t *testing.T) {
	cases := map[string]*AuditEvent{
		"nil event": nil,
		"empty id": func() *AuditEvent {
			event := newPersistableEvent()
			event.ID = uuid.Nil

			return event
		}(),
		"invalid action": func() *AuditEvent {
			event := newPersistableEvent()
			event.Action = Action(0)

			return event
		}(),
		"invalid resource type": func() *AuditEvent {
			event := newPersistableEvent()
			event.ResourceType = ResourceType(99)

			return event
		}(),
		"invalid actor type": func() *AuditEvent {
			event := newPersistableEvent()
			event.ActorType = ActorType(99)

			return event
		}(),
	}

	for name, event := range cases {
		t.Run(name, func(t *testing.T) {
			storage := newFakeStorage()
			uc := NewPersistAuditEvent(storage)

			err := uc.Execute(context.Background(), event)

			var validationErr *ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Empty(t, storage.events)
		})
	}
}

func TestPersistAuditEvent_StorageFailurePropagates(t *testing.T) {
	storage := newFakeStorage()
	storage.createErr = errors.New("db down")
	uc := NewPersistAuditEvent(storage)

	err := uc.Execute(context.Background(), newPersistableEvent())

	require.ErrorIs(t, err, storage.createErr)

	var validationErr *ValidationError
	assert.NotErrorAs(t, err, &validationErr)
	assert.Empty(t, storage.events)
}
