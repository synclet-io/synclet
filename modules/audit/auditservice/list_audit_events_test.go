package auditservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAuditEvents_DateRange(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	workspaceID := uuid.New()

	oldest := addAuditEvent(storage, workspaceID, now.Add(-3*time.Hour))
	middle := addAuditEvent(storage, workspaceID, now.Add(-2*time.Hour))
	newest := addAuditEvent(storage, workspaceID, now.Add(-1*time.Hour))
	addAuditEvent(storage, uuid.New(), now.Add(-2*time.Hour))

	since := now.Add(-150 * time.Minute)
	until := now.Add(-90 * time.Minute)
	uc := NewListAuditEvents(storage, &fakeUserDirectory{})

	cases := map[string]struct {
		since *time.Time
		until *time.Time
		want  []uuid.UUID
	}{
		"no bounds":  {want: []uuid.UUID{oldest, middle, newest}},
		"since only": {since: &since, want: []uuid.UUID{middle, newest}},
		"until only": {until: &until, want: []uuid.UUID{oldest, middle}},
		"both":       {since: &since, until: &until, want: []uuid.UUID{middle}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			events, err := uc.Execute(context.Background(), ListAuditEventsParams{
				WorkspaceID: workspaceID,
				Since:       tc.since,
				Until:       tc.until,
			})
			require.NoError(t, err)

			ids := make([]uuid.UUID, 0, len(events))
			for _, event := range events {
				ids = append(ids, event.ID)
			}

			assert.ElementsMatch(t, tc.want, ids)
		})
	}
}

func TestListAuditEvents_BoundsAreHalfOpen(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	workspaceID := uuid.New()

	atSince := addAuditEvent(storage, workspaceID, now)
	addAuditEvent(storage, workspaceID, now.Add(time.Hour))

	until := now.Add(time.Hour)
	uc := NewListAuditEvents(storage, &fakeUserDirectory{})

	events, err := uc.Execute(context.Background(), ListAuditEventsParams{
		WorkspaceID: workspaceID,
		Since:       &now,
		Until:       &until,
	})
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.Equal(t, atSince, events[0].ID)
}

func TestListAuditEvents_OffsetIsExact(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	workspaceID := uuid.New()

	ids := make([]uuid.UUID, 0, 5)
	for i := range 5 {
		ids = append(ids, addAuditEvent(storage, workspaceID, now.Add(-time.Duration(i)*time.Minute)))
	}

	events, err := NewListAuditEvents(storage, &fakeUserDirectory{}).Execute(context.Background(), ListAuditEventsParams{
		WorkspaceID: workspaceID,
		Limit:       2,
		Offset:      3,
	})
	require.NoError(t, err)

	got := make([]uuid.UUID, 0, len(events))
	for _, event := range events {
		got = append(got, event.ID)
	}

	assert.Equal(t, ids[3:5], got)
}

type fakeUserDirectory struct {
	existing map[uuid.UUID]bool
	emails   map[uuid.UUID]string
	lookups  []uuid.UUID
}

func (f *fakeUserDirectory) UserExists(_ context.Context, id uuid.UUID) (bool, error) {
	f.lookups = append(f.lookups, id)

	return f.existing[id], nil
}

func (f *fakeUserDirectory) UserEmail(_ context.Context, id uuid.UUID) (string, error) {
	f.lookups = append(f.lookups, id)

	return f.emails[id], nil
}

func TestListAuditEvents_FlagsDeletedUserActors(t *testing.T) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	storage := newFakeStorage()
	workspaceID := uuid.New()
	liveUser := uuid.New()
	deletedUser := uuid.New()

	addActorEvent := func(actorType ActorType, actorID uuid.UUID, offset time.Duration) uuid.UUID {
		id := addAuditEvent(storage, workspaceID, now.Add(-offset))
		storage.events[len(storage.events)-1].ActorType = actorType
		storage.events[len(storage.events)-1].ActorID = actorID

		return id
	}

	byLive := addActorEvent(ActorTypeUser, liveUser, time.Minute)
	byDeleted := addActorEvent(ActorTypeUser, deletedUser, 2*time.Minute)
	byDeletedAgain := addActorEvent(ActorTypeUser, deletedUser, 3*time.Minute)
	byAPIKey := addActorEvent(ActorTypeAPIKey, uuid.New(), 4*time.Minute)
	bySystem := addActorEvent(ActorTypeSystem, uuid.Nil, 5*time.Minute)

	users := &fakeUserDirectory{existing: map[uuid.UUID]bool{liveUser: true}}

	events, err := NewListAuditEvents(storage, users).Execute(context.Background(), ListAuditEventsParams{
		WorkspaceID: workspaceID,
	})
	require.NoError(t, err)

	deleted := make(map[uuid.UUID]bool, len(events))
	for _, event := range events {
		deleted[event.ID] = event.ActorDeleted
	}

	assert.Equal(t, map[uuid.UUID]bool{
		byLive:         false,
		byDeleted:      true,
		byDeletedAgain: true,
		byAPIKey:       false,
		bySystem:       false,
	}, deleted)
	assert.ElementsMatch(t, []uuid.UUID{liveUser, deletedUser}, users.lookups)
}
