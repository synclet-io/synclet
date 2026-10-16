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
	users := newFakeUserLookup()

	for _, m := range []*WorkspaceMember{
		{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleAdmin},
		{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleEditor},
		{ID: uuid.New(), WorkspaceID: otherWsID, UserID: uuid.New(), Role: MemberRoleAdmin},
	} {
		store.seedMember(m)
		users.seed(&UserInfo{ID: m.UserID, Email: m.UserID.String() + "@example.com"})
	}

	got, err := NewListMembers(store, NewGetMembership(store), users).Execute(ctx, wsID, nil)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestListMembers_NonMemberRejected(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	other := uuid.New()
	store := newFakeStorage()
	users := newFakeUserLookup()

	_, err := NewListMembers(store, NewGetMembership(store), users).Execute(ctx, wsID, &other)
	require.ErrorIs(t, err, ErrWorkspaceNotFound)
}

func TestListMembers_CallerMemberCanList(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: userID, Role: MemberRoleViewer})

	users := newFakeUserLookup()
	users.seed(&UserInfo{ID: userID, Email: "viewer@example.com", Name: "Viewer"})

	got, err := NewListMembers(store, NewGetMembership(store), users).Execute(ctx, wsID, &userID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, userID, got[0].UserID)
}

func TestListMembers_ResolvesUserIdentities(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	adminID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: adminID, Role: MemberRoleAdmin})

	users := newFakeUserLookup()
	users.seed(&UserInfo{ID: adminID, Email: "ada@example.com", Name: "Ada Lovelace"})

	got, err := NewListMembers(store, NewGetMembership(store), users).Execute(ctx, wsID, &adminID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, adminID, got[0].UserID)
	assert.Equal(t, MemberRoleAdmin, got[0].Role)
	assert.Equal(t, "ada@example.com", got[0].Email)
	assert.Equal(t, "Ada Lovelace", got[0].Name)
}

func TestListMembers_UserLookupFailure(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleEditor})

	_, err := NewListMembers(store, NewGetMembership(store), newFakeUserLookup()).Execute(ctx, wsID, nil)
	require.Error(t, err)
}
