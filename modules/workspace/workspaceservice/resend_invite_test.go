package workspaceservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newResendInvite(store *fakeStorage, lookup *fakeUserLookup, email *fakeEmailSender) *ResendInvite {
	cfg := Config{InviteTTL: 24 * time.Hour, FrontendURL: "https://app.example.com"}

	return NewResendInvite(store, email, lookup, cfg, nil)
}

func TestResendInvite_RefreshesExpiration(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviteID := uuid.New()
	inviterID := uuid.New()
	oldExpiry := time.Now().Add(-time.Minute)

	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "WS", Slug: "ws"})
	store.seedInvite(&WorkspaceInvite{
		ID: inviteID, WorkspaceID: wsID, InviterUserID: inviterID,
		Email: "x@example.com", Token: "tok", Status: InviteStatusPending,
		ExpiresAt: oldExpiry,
	})

	users := newFakeUserLookup()
	users.seed(&UserInfo{ID: inviterID, Email: "admin@example.com", Name: "Admin"})

	require.NoError(t, newResendInvite(store, users, &fakeEmailSender{}).Execute(ctx, inviteID, wsID))

	got, err := store.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{ID: equalsUUID(inviteID)})
	require.NoError(t, err)
	assert.True(t, got.ExpiresAt.After(oldExpiry), "expiration must be pushed forward")
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), got.ExpiresAt, 5*time.Second)
}

func TestResendInvite_NotPendingRejected(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviteID := uuid.New()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: inviteID, WorkspaceID: wsID, Token: "tok", Status: InviteStatusAccepted,
		ExpiresAt: time.Now().Add(time.Hour),
	})

	users := newFakeUserLookup()

	err := newResendInvite(store, users, &fakeEmailSender{}).Execute(ctx, inviteID, wsID)
	require.Error(t, err)
}

func TestResendInvite_InviteNotFound(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserLookup()

	err := newResendInvite(newFakeStorage(), users, &fakeEmailSender{}).Execute(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
}
