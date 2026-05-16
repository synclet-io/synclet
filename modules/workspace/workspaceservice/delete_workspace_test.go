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
	err := NewDeleteWorkspace(newFakeStorage()).Execute(ctx, uuid.New())
	assert.NoError(t, err)
}
