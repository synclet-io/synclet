package workspaceservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRevokeInvite_TransitionsToRevoked(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviteID := uuid.New()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: inviteID, WorkspaceID: wsID, Token: "tok",
		Status: InviteStatusPending, ExpiresAt: time.Now().Add(time.Hour),
	})

	require.NoError(t, NewRevokeInvite(store).Execute(ctx, inviteID, wsID))

	got, err := store.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{ID: equalsUUID(inviteID)})
	require.NoError(t, err)
	assert.Equal(t, InviteStatusRevoked, got.Status)
}

func TestRevokeInvite_InviteNotFound(t *testing.T) {
	ctx := context.Background()
	err := NewRevokeInvite(newFakeStorage()).Execute(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
}

func TestRevokeInvite_NotPendingRejected(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviteID := uuid.New()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: inviteID, WorkspaceID: wsID, Token: "tok",
		Status: InviteStatusRevoked, ExpiresAt: time.Now().Add(time.Hour),
	})

	err := NewRevokeInvite(store).Execute(ctx, inviteID, wsID)
	require.Error(t, err)
}
