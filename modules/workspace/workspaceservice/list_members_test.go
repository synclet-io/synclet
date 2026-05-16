package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListMembers_ReturnsAllMembersForWorkspace(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	otherWsID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleEditor})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: otherWsID, UserID: uuid.New(), Role: MemberRoleAdmin})

	got, err := NewListMembers(store, NewGetMembership(store)).Execute(ctx, wsID, nil)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestListMembers_NonMemberRejected(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	other := uuid.New()
	store := newFakeStorage()

	_, err := NewListMembers(store, NewGetMembership(store)).Execute(ctx, wsID, &other)
	require.ErrorIs(t, err, ErrWorkspaceNotFound)
}

func TestListMembers_CallerMemberCanList(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: userID, Role: MemberRoleViewer})

	got, err := NewListMembers(store, NewGetMembership(store)).Execute(ctx, wsID, &userID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, userID, got[0].UserID)
}
