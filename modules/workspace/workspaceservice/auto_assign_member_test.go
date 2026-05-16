package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoAssignMember_FirstMemberBecomesAdmin(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Default", Slug: "default"})

	require.NoError(t, NewAutoAssignMember(store).Execute(ctx, userID))

	members, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{WorkspaceID: equalsUUID(wsID)})
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, MemberRoleAdmin, members[0].Role)
	assert.Equal(t, userID, members[0].UserID)
}

func TestAutoAssignMember_SubsequentMembersBecomeViewer(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Default", Slug: "default"})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleAdmin})

	newUserID := uuid.New()
	require.NoError(t, NewAutoAssignMember(store).Execute(ctx, newUserID))

	members, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{UserID: equalsUUID(newUserID)})
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, MemberRoleViewer, members[0].Role)
}

func TestAutoAssignMember_NoDefaultWorkspaceFails(t *testing.T) {
	ctx := context.Background()
	err := NewAutoAssignMember(newFakeStorage()).Execute(ctx, uuid.New())
	require.Error(t, err)
}
