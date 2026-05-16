package workspaceservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInviteByToken_PopulatesWorkspaceAndInviterName(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviterID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Acme", Slug: "acme"})
	store.seedInvite(&WorkspaceInvite{
		ID: uuid.New(), WorkspaceID: wsID, InviterUserID: inviterID,
		Email: "x@example.com", Token: "tok", Status: InviteStatusPending,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	users := newFakeUserLookup()
	users.seed(&UserInfo{ID: inviterID, Email: "admin@example.com", Name: "Alice"})

	got, err := NewGetInviteByToken(store, users).Execute(ctx, "tok")
	require.NoError(t, err)
	assert.Equal(t, "Acme", got.WorkspaceName)
	assert.Equal(t, "Alice", got.InviterName)
	assert.False(t, got.IsExpired)
}

func TestGetInviteByToken_FlagsExpiredPending(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Acme", Slug: "acme"})
	store.seedInvite(&WorkspaceInvite{
		ID: uuid.New(), WorkspaceID: wsID, InviterUserID: uuid.New(),
		Email: "x@example.com", Token: "tok", Status: InviteStatusPending,
		ExpiresAt: time.Now().Add(-time.Hour),
	})

	users := newFakeUserLookup()

	got, err := NewGetInviteByToken(store, users).Execute(ctx, "tok")
	require.NoError(t, err)
	assert.True(t, got.IsExpired)
}

func TestGetInviteByToken_NotFound(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserLookup()
	_, err := NewGetInviteByToken(newFakeStorage(), users).Execute(ctx, "missing")
	require.Error(t, err)
}
