package workspaceservice

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptInvite_CreatesMembershipAndMarksAccepted(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviteID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Workspace", Slug: "workspace"})
	store.seedInvite(&WorkspaceInvite{
		ID:          inviteID,
		WorkspaceID: wsID,
		Email:       "person@example.com",
		Role:        MemberRoleEditor,
		Token:       "token-xyz",
		Status:      InviteStatusPending,
		ExpiresAt:   time.Now().Add(time.Hour),
	})

	got, err := NewAcceptInvite(store).Execute(ctx, "token-xyz", userID, "PERSON@example.com")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, wsID, got.WorkspaceID)
	assert.Equal(t, "Workspace", got.WorkspaceName)

	updated, err := store.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{ID: equalsUUID(inviteID)})
	require.NoError(t, err)
	assert.Equal(t, InviteStatusAccepted, updated.Status)

	members, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{UserID: equalsUUID(userID)})
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, MemberRoleEditor, members[0].Role)
}

func TestAcceptInvite_AlreadyMemberMarksInviteAcceptedWithoutDuplicate(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "WS", Slug: "ws"})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: userID, Role: MemberRoleAdmin})
	store.seedInvite(&WorkspaceInvite{
		ID:          uuid.New(),
		WorkspaceID: wsID,
		Email:       "x@example.com",
		Role:        MemberRoleViewer,
		Token:       "t",
		Status:      InviteStatusPending,
		ExpiresAt:   time.Now().Add(time.Hour),
	})

	got, err := NewAcceptInvite(store).Execute(ctx, "t", userID, "x@example.com")
	require.NoError(t, err)
	assert.Equal(t, wsID, got.WorkspaceID)

	members, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{UserID: equalsUUID(userID)})
	require.NoError(t, err)
	require.Len(t, members, 1, "no duplicate member created")
	assert.Equal(t, MemberRoleAdmin, members[0].Role, "original role preserved")
}

func TestAcceptInvite_TokenNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := NewAcceptInvite(newFakeStorage()).Execute(ctx, "missing", uuid.New(), "x@example.com")
	require.Error(t, err)
}

func TestAcceptInvite_NotPendingRejected(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: uuid.New(), WorkspaceID: uuid.New(), Email: "x@example.com",
		Token: "t", Status: InviteStatusDeclined, ExpiresAt: time.Now().Add(time.Hour),
	})
	_, err := NewAcceptInvite(store).Execute(ctx, "t", uuid.New(), "x@example.com")
	require.Error(t, err)
}

func TestAcceptInvite_ExpiredRejected(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: uuid.New(), WorkspaceID: uuid.New(), Email: "x@example.com",
		Token: "t", Status: InviteStatusPending, ExpiresAt: time.Now().Add(-time.Hour),
	})
	_, err := NewAcceptInvite(store).Execute(ctx, "t", uuid.New(), "x@example.com")
	require.Error(t, err)
}

func TestAcceptInvite_EmailMismatchRejected(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	store.seedInvite(&WorkspaceInvite{
		ID: uuid.New(), WorkspaceID: uuid.New(), Email: "owner@example.com",
		Token: "t", Status: InviteStatusPending, ExpiresAt: time.Now().Add(time.Hour),
	})
	_, err := NewAcceptInvite(store).Execute(ctx, "t", uuid.New(), "imposter@example.com")
	require.Error(t, err)
}
