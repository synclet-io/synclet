package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteWorkspace_RemovesWorkspace(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Going", Slug: "going"})

	require.NoError(t, NewDeleteWorkspace(store).Execute(ctx, wsID))

	_, err := store.Workspaces().First(ctx, &WorkspaceFilter{ID: equalsUUID(wsID)})
	require.Error(t, err)
}

func TestDeleteWorkspace_NoOpWhenMissing(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	err := NewDeleteWorkspace(store).Execute(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, store.events, "no event must be emitted when nothing was deleted")
}

func TestDeleteWorkspace_EmitsDeletedEvent(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Going", Slug: "going"})

	require.NoError(t, NewDeleteWorkspace(store).Execute(ctx, wsID))

	require.Len(t, store.events, 1)
	event := store.events[0]
	assert.Equal(t, WorkspaceEventTypeDeleted, event.EventType)

	data, ok := event.Data.(*WorkspaceDeletedEventData)
	require.True(t, ok)
	require.NotNil(t, data.Workspace)
	assert.Equal(t, wsID, data.Workspace.ID)
	assert.Equal(t, "Going", data.Workspace.Name)
}
