package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMembership_FoundReturnsRow(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: userID, Role: MemberRoleAdmin})

	got, err := NewGetMembership(store).Execute(ctx, wsID, userID)
	require.NoError(t, err)
	assert.Equal(t, MemberRoleAdmin, got.Role)
}

func TestGetMembership_NotFound(t *testing.T) {
	ctx := context.Background()
	_, err := NewGetMembership(newFakeStorage()).Execute(ctx, uuid.New(), uuid.New())
	require.ErrorIs(t, err, ErrWorkspaceMemberNotFound)
}
