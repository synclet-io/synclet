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

func TestRemoveMember_IdempotentWhenMissing(t *testing.T) {
	ctx := context.Background()
	err := NewRemoveMember(newFakeStorage(), NoopAuditRecorder{}).Execute(ctx, uuid.New(), uuid.New())
	assert.NoError(t, err)
}
