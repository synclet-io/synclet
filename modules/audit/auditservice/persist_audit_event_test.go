package auditservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
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

func newPendingUserEvent(userID uuid.UUID, action Action) *AuditEvent {
	return &AuditEvent{
		ID:            uuid.New(),
		ActorType:     ActorTypeUser,
		ActorID:       userID,
		ActorLabel:    "new@example.com",
		Action:        action,
		ResourceType:  ResourceTypeUser,
		ResourceID:    userID,
		ResourceLabel: "new@example.com",
		DiffJSON:      "[]",
		CreatedAt:     time.Now(),
	}
}

func TestPersistAuditEvent_JoinAdoptsPendingUserEvents(t *testing.T) {
	cases := map[string]func(userID, workspaceID uuid.UUID) *AuditEvent{
		"workspace created by the user": func(userID, workspaceID uuid.UUID) *AuditEvent {
			return &AuditEvent{
				ID: uuid.New(), WorkspaceID: workspaceID, ActorType: ActorTypeUser, ActorID: userID,
				Action: ActionCreate, ResourceType: ResourceTypeWorkspace, ResourceID: workspaceID,
				DiffJSON: "[]", CreatedAt: time.Now(),
			}
		},
		"invite accepted by the user": func(userID, workspaceID uuid.UUID) *AuditEvent {
			return &AuditEvent{
				ID: uuid.New(), WorkspaceID: workspaceID, ActorType: ActorTypeUser, ActorID: userID,
				Action: ActionCreate, ResourceType: ResourceTypeWorkspaceMember, ResourceID: userID,
				DiffJSON: "[]", CreatedAt: time.Now(),
			}
		},
	}

	for name, joinEvent := range cases {
		t.Run(name, func(t *testing.T) {
			storage := newFakeStorage()
			uc := NewPersistAuditEvent(storage)
			ctx := context.Background()

			userID, otherUserID, workspaceID := uuid.New(), uuid.New(), uuid.New()
			registered := newPendingUserEvent(userID, ActionCreate)
			loggedIn := newPendingUserEvent(userID, ActionLogin)
			otherPending := newPendingUserEvent(otherUserID, ActionLogin)

			for _, event := range []*AuditEvent{registered, loggedIn, otherPending} {
				require.NoError(t, uc.Execute(ctx, event))
			}

			require.NoError(t, uc.Execute(ctx, joinEvent(userID, workspaceID)))

			inWorkspace, err := storage.AuditEvents().Find(ctx, &AuditEventFilter{WorkspaceID: filter.Equals(workspaceID)})
			require.NoError(t, err)

			ids := make([]uuid.UUID, 0, len(inWorkspace))
			for _, event := range inWorkspace {
				ids = append(ids, event.ID)
			}

			assert.Contains(t, ids, registered.ID)
			assert.Contains(t, ids, loggedIn.ID)
			assert.NotContains(t, ids, otherPending.ID)

			pending, err := storage.AuditEvents().Find(ctx, &AuditEventFilter{WorkspaceID: filter.Equals(uuid.Nil)})
			require.NoError(t, err)
			require.Len(t, pending, 1)
			assert.Equal(t, otherPending.ID, pending[0].ID)
		})
	}
}

func TestPersistAuditEvent_NonJoinEventLeavesPendingEvents(t *testing.T) {
	storage := newFakeStorage()
	uc := NewPersistAuditEvent(storage)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, uc.Execute(ctx, newPendingUserEvent(userID, ActionLogin)))

	other := newPersistableEvent()
	other.ActorType = ActorTypeUser
	other.ActorID = userID
	require.NoError(t, uc.Execute(ctx, other))

	pending, err := storage.AuditEvents().Find(ctx, &AuditEventFilter{WorkspaceID: filter.Equals(uuid.Nil)})
	require.NoError(t, err)
	assert.Len(t, pending, 1)
}
