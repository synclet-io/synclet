package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveMember_Removes(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: userID, Role: MemberRoleEditor})

	require.NoError(t, NewRemoveMember(store, NoopAuditRecorder{}).Execute(ctx, wsID, userID))

	rows, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{})
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestRemoveMember_MissingIsNotFound(t *testing.T) {
	ctx := context.Background()
	recorder := &capturingAuditRecorder{}
	err := NewRemoveMember(newFakeStorage(), recorder).Execute(ctx, uuid.New(), uuid.New())
	require.ErrorIs(t, err, ErrWorkspaceMemberNotFound)
	assert.Empty(t, recorder.events)
}

func TestRemoveMember_RefusesLastAdmin(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	adminID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: adminID, Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleEditor})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: uuid.New(), UserID: uuid.New(), Role: MemberRoleAdmin})

	err := NewRemoveMember(store, NoopAuditRecorder{}).Execute(ctx, wsID, adminID)
	require.ErrorIs(t, err, ErrLastAdminCannotBeRemoved)

	rows, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{})
	require.NoError(t, err)
	assert.Len(t, rows, 3)
}

func TestRemoveMember_RemovesAdminWhenAnotherRemains(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	adminID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: adminID, Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: uuid.New(), Role: MemberRoleAdmin})

	require.NoError(t, NewRemoveMember(store, NoopAuditRecorder{}).Execute(ctx, wsID, adminID))

	rows, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{})
	require.NoError(t, err)
	assert.Len(t, rows, 1)
}
