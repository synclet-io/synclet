package workspaceservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListInvites_ReturnsAllForWorkspace(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	otherID := uuid.New()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{ID: uuid.New(), WorkspaceID: wsID, Token: "t1", Status: InviteStatusPending, ExpiresAt: time.Now().Add(time.Hour)})
	store.seedInvite(&WorkspaceInvite{ID: uuid.New(), WorkspaceID: wsID, Token: "t2", Status: InviteStatusAccepted, ExpiresAt: time.Now().Add(time.Hour)})
	store.seedInvite(&WorkspaceInvite{ID: uuid.New(), WorkspaceID: otherID, Token: "t3", Status: InviteStatusPending, ExpiresAt: time.Now().Add(time.Hour)})

	got, err := NewListInvites(store).Execute(ctx, wsID)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestListInvites_EmptyWorkspace(t *testing.T) {
	ctx := context.Background()
	got, err := NewListInvites(newFakeStorage()).Execute(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, got)
}
