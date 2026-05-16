package workspaceservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWorkspace_CreatesWorkspaceAndAdminMember(t *testing.T) {
	ctx := context.Background()
	ownerID := uuid.New()
	store := newFakeStorage()

	ws, err := NewCreateWorkspace(store).Execute(ctx, "Acme Corp", ownerID)
	require.NoError(t, err)
	require.NotNil(t, ws)

	assert.Equal(t, "Acme Corp", ws.Name)
	assert.Equal(t, "acme-corp", ws.Slug)
	assert.NotEqual(t, uuid.Nil, ws.ID)

	members, err := store.WorkspaceMembers().Find(ctx, &WorkspaceMemberFilter{WorkspaceID: equalsUUID(ws.ID)})
	require.NoError(t, err)
	require.Len(t, members, 1)

	assert.Equal(t, ownerID, members[0].UserID)
	assert.Equal(t, MemberRoleAdmin, members[0].Role)
}

func TestCreateWorkspace_EmitsCreatedEvent(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()

	ws, err := NewCreateWorkspace(store).Execute(ctx, "Acme Corp", uuid.New())
	require.NoError(t, err)

	require.Len(t, store.events, 1, "exactly one workspace event must be emitted")
	event := store.events[0]
	assert.Equal(t, WorkspaceEventTypeCreated, event.EventType)

	data, ok := event.Data.(*WorkspaceCreatedEventData)
	require.True(t, ok, "data must be *WorkspaceCreatedEventData")
	require.NotNil(t, data.Workspace)
	assert.Equal(t, ws.ID, data.Workspace.ID)
}

func TestCreateWorkspace_DuplicateSlugFails(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: uuid.New(), Name: "Acme", Slug: "acme"})

	_, err := NewCreateWorkspace(store).Execute(ctx, "acme", uuid.New())
	require.Error(t, err)
}
