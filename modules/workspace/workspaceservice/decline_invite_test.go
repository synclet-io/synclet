package workspaceservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeclineInvite_TransitionsToDeclined(t *testing.T) {
	ctx := context.Background()
	inviteID := uuid.New()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: inviteID, WorkspaceID: uuid.New(), Email: "x@example.com",
		Token: "tok", Status: InviteStatusPending, ExpiresAt: time.Now().Add(time.Hour),
	})

	require.NoError(t, NewDeclineInvite(store).Execute(ctx, "tok"))

	got, err := store.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{ID: equalsUUID(inviteID)})
	require.NoError(t, err)
	assert.Equal(t, InviteStatusDeclined, got.Status)
}

func TestDeclineInvite_TokenNotFound(t *testing.T) {
	ctx := context.Background()
	err := NewDeclineInvite(newFakeStorage()).Execute(ctx, "missing")
	require.Error(t, err)
}

func TestDeclineInvite_AlreadyAcceptedRejected(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: uuid.New(), Token: "tok", Status: InviteStatusAccepted,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	err := NewDeclineInvite(store).Execute(ctx, "tok")
	require.Error(t, err)
}

func TestDeclineInvite_ExpiredRejected(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: uuid.New(), Token: "tok", Status: InviteStatusPending,
		ExpiresAt: time.Now().Add(-time.Hour),
	})

	err := NewDeclineInvite(store).Execute(ctx, "tok")
	require.Error(t, err)
}
