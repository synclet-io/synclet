package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWorkspacesForUser_ReturnsWorkspacesWithRoles(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	wsAdminID := uuid.New()
	wsEditorID := uuid.New()
	wsViewerID := uuid.New()

	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsAdminID, Name: "Admin WS"})
	store.seedWorkspace(&Workspace{ID: wsEditorID, Name: "Editor WS"})
	store.seedWorkspace(&Workspace{ID: wsViewerID, Name: "Viewer WS"})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), UserID: userID, WorkspaceID: wsAdminID, Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), UserID: userID, WorkspaceID: wsEditorID, Role: MemberRoleEditor})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), UserID: userID, WorkspaceID: wsViewerID, Role: MemberRoleViewer})

	got, err := NewListWorkspacesForUser(store).Execute(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 3)

	roleByWorkspace := map[uuid.UUID]MemberRole{}

	for _, item := range got {
		require.NotNil(t, item.Workspace)
		roleByWorkspace[item.Workspace.ID] = item.Role
	}

	assert.Equal(t, MemberRoleAdmin, roleByWorkspace[wsAdminID])
	assert.Equal(t, MemberRoleEditor, roleByWorkspace[wsEditorID])
	assert.Equal(t, MemberRoleViewer, roleByWorkspace[wsViewerID])
}

func TestListWorkspacesForUser_SkipsWorkspacesThatFailToLoad(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	goodID := uuid.New()
	orphanID := uuid.New()

	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: goodID, Name: "Good"})
	// orphanID intentionally not seeded as a workspace
	store.seedMember(&WorkspaceMember{ID: uuid.New(), UserID: userID, WorkspaceID: orphanID, Role: MemberRoleAdmin})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), UserID: userID, WorkspaceID: goodID, Role: MemberRoleEditor})

	got, err := NewListWorkspacesForUser(store).Execute(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 1, "orphan membership must be skipped")
	assert.Equal(t, goodID, got[0].Workspace.ID)
	assert.Equal(t, MemberRoleEditor, got[0].Role)
}

func TestListWorkspacesForUser_NoMemberships(t *testing.T) {
	ctx := context.Background()
	got, err := NewListWorkspacesForUser(newFakeStorage()).Execute(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, got)
}
