package auditservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAuditEvent_FlagsDeletedUserActor(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	workspaceID := uuid.New()
	liveUser := uuid.New()
	deletedUser := uuid.New()

	addActorEvent := func(actorType ActorType, actorID uuid.UUID) uuid.UUID {
		id := addAuditEvent(storage, workspaceID, now)
		storage.events[len(storage.events)-1].ActorType = actorType
		storage.events[len(storage.events)-1].ActorID = actorID

		return id
	}

	cases := map[string]struct {
		event uuid.UUID
		want  bool
	}{
		"live user":    {event: addActorEvent(ActorTypeUser, liveUser), want: false},
		"deleted user": {event: addActorEvent(ActorTypeUser, deletedUser), want: true},
		"api key":      {event: addActorEvent(ActorTypeAPIKey, uuid.New()), want: false},
		"system":       {event: addActorEvent(ActorTypeSystem, uuid.Nil), want: false},
	}

	users := &fakeUserDirectory{existing: map[uuid.UUID]bool{liveUser: true}}
	uc := NewGetAuditEvent(storage, users)

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			event, err := uc.Execute(context.Background(), GetAuditEventParams{ID: tc.event, WorkspaceID: workspaceID})
			require.NoError(t, err)

			assert.Equal(t, tc.event, event.ID)
			assert.Equal(t, tc.want, event.ActorDeleted)
		})
	}
}
