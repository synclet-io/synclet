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
	uc := NewListAuditEvents(storage)

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
	uc := NewListAuditEvents(storage)

	events, err := uc.Execute(context.Background(), ListAuditEventsParams{
		WorkspaceID: workspaceID,
		Since:       &now,
		Until:       &until,
	})
	require.NoError(t, err)

	require.Len(t, events, 1)
	assert.Equal(t, atSince, events[0].ID)
}
