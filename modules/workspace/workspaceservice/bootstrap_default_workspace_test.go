package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapDefaultWorkspace_CreatesWhenMissing(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()

	got, err := NewBootstrapDefaultWorkspace(store).Execute(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Default", got.Name)
	assert.Equal(t, "default", got.Slug)

	require.Len(t, store.events, 1, "bootstrap must emit workspace.created so subscribers seed registries")
	assert.Equal(t, WorkspaceEventTypeCreated, store.events[0].EventType)
}

func TestBootstrapDefaultWorkspace_IdempotentWhenExisting(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Default", Slug: "default"})

	got, err := NewBootstrapDefaultWorkspace(store).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, wsID, got.ID)

	all, err := store.Workspaces().Find(ctx, &WorkspaceFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 1, "no extra workspace must be created")
	assert.Empty(t, store.events, "no event when bootstrap is a no-op")
}
