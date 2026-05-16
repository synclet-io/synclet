package workspaceservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateWorkspace_UpdatesName(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	originalCreatedAt := time.Now().Add(-time.Hour)
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Old", Slug: "old", CreatedAt: originalCreatedAt})

	name := "New"
	got, err := NewUpdateWorkspace(store).Execute(ctx, UpdateWorkspaceParams{ID: wsID, Name: &name})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "New", got.Name)
	assert.WithinDuration(t, time.Now(), got.UpdatedAt, time.Second)
	assert.True(t, got.UpdatedAt.After(originalCreatedAt))
}

func TestUpdateWorkspace_LeavesNameWhenNil(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Same", Slug: "same"})

	got, err := NewUpdateWorkspace(store).Execute(ctx, UpdateWorkspaceParams{ID: wsID})
	require.NoError(t, err)
	assert.Equal(t, "Same", got.Name)
}

func TestUpdateWorkspace_NotFound(t *testing.T) {
	ctx := context.Background()
	name := "x"
	_, err := NewUpdateWorkspace(newFakeStorage()).Execute(ctx, UpdateWorkspaceParams{ID: uuid.New(), Name: &name})
	require.ErrorIs(t, err, ErrWorkspaceNotFound)
}
