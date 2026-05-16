package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkspace_NilUserSkipsMembershipCheck(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "WS", Slug: "ws"})

	uc := NewGetWorkspace(store, NewGetMembership(store))
	got, err := uc.Execute(ctx, wsID, nil)
	require.NoError(t, err)
	assert.Equal(t, wsID, got.ID)
}

func TestGetWorkspace_MemberCanRead(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	userID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "WS", Slug: "ws"})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: userID, Role: MemberRoleViewer})

	uc := NewGetWorkspace(store, NewGetMembership(store))
	got, err := uc.Execute(ctx, wsID, &userID)
	require.NoError(t, err)
	assert.Equal(t, wsID, got.ID)
}

func TestGetWorkspace_NonMemberRejected(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	other := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "WS", Slug: "ws"})

	uc := NewGetWorkspace(store, NewGetMembership(store))
	_, err := uc.Execute(ctx, wsID, &other)
	require.ErrorIs(t, err, ErrWorkspaceNotFound)
}
