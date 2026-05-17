package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateMemberRole_PromotesEditorToAdmin(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	targetUserID := uuid.New()
	targetID := uuid.New()

	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: uuid.New(), Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: targetID, WorkspaceID: workspaceID, UserID: targetUserID, Role: MemberRoleEditor})

	require.NoError(t, NewUpdateMemberRole(store, NoopAuditRecorder{}).Execute(ctx, workspaceID, targetUserID, MemberRoleAdmin))

	got, err := store.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{ID: equalsUUID(targetID)})
	require.NoError(t, err)
	assert.Equal(t, MemberRoleAdmin, got.Role)
}

func TestUpdateMemberRole_LastAdminCannotBeDemoted(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	lonelyAdminID := uuid.New()

	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: lonelyAdminID, Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: uuid.New(), Role: MemberRoleEditor})

	err := NewUpdateMemberRole(store, NoopAuditRecorder{}).Execute(ctx, workspaceID, lonelyAdminID, MemberRoleEditor)
	require.ErrorIs(t, err, ErrLastAdminCannotBeDemoted)

	// Member role must remain Admin after the guard fired.
	got, err := store.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{UserID: equalsUUID(lonelyAdminID)})
	require.NoError(t, err)
	assert.Equal(t, MemberRoleAdmin, got.Role)
}

func TestUpdateMemberRole_TwoAdminsAllowsDemotion(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	adminAID := uuid.New()
	adminBID := uuid.New()

	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: adminAID, Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: workspaceID, UserID: adminBID, Role: MemberRoleAdmin})

	require.NoError(t, NewUpdateMemberRole(store, NoopAuditRecorder{}).Execute(ctx, workspaceID, adminAID, MemberRoleViewer))

	got, err := store.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{UserID: equalsUUID(adminAID)})
	require.NoError(t, err)
	assert.Equal(t, MemberRoleViewer, got.Role)
}

func TestUpdateMemberRole_NotAMember(t *testing.T) {
	ctx := context.Background()
	err := NewUpdateMemberRole(newFakeStorage(), NoopAuditRecorder{}).Execute(ctx, uuid.New(), uuid.New(), MemberRoleEditor)
	require.ErrorIs(t, err, ErrWorkspaceMemberNotFound)
}

func TestUpdateMemberRole_InvalidRole(t *testing.T) {
	ctx := context.Background()
	err := NewUpdateMemberRole(newFakeStorage(), NoopAuditRecorder{}).Execute(ctx, uuid.New(), uuid.New(), MemberRole(0))
	require.ErrorIs(t, err, ErrInvalidMemberRole)
}

func TestUpdateMemberRole_NoOpWhenRoleUnchanged(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	userID := uuid.New()
	memberID := uuid.New()

	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: memberID, WorkspaceID: workspaceID, UserID: userID, Role: MemberRoleEditor})

	require.NoError(t, NewUpdateMemberRole(store, NoopAuditRecorder{}).Execute(ctx, workspaceID, userID, MemberRoleEditor))

	got, err := store.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{ID: equalsUUID(memberID)})
	require.NoError(t, err)
	assert.Equal(t, MemberRoleEditor, got.Role)
}
