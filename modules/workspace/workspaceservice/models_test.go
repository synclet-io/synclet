package workspaceservice

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestModels_CopyEquals exercises the generated Copy/Equals helpers so a
// regression in code generation surfaces in CI.
func TestModels_CopyEquals(t *testing.T) {
	now := time.Now()

	ws := &Workspace{ID: uuid.New(), Name: "n", Slug: "s", CreatedAt: now, UpdatedAt: now}
	wsCopy := ws.Copy()
	assert.True(t, ws.Equals(&wsCopy))
	wsCopy.Name = "different"
	assert.False(t, ws.Equals(&wsCopy))

	m := &WorkspaceMember{ID: uuid.New(), WorkspaceID: uuid.New(), UserID: uuid.New(), Role: MemberRoleAdmin, JoinedAt: now}
	mCopy := m.Copy()
	assert.True(t, m.Equals(&mCopy))
	mCopy.Role = MemberRoleEditor
	assert.False(t, m.Equals(&mCopy))

	inv := &WorkspaceInvite{
		ID: uuid.New(), WorkspaceID: uuid.New(), InviterUserID: uuid.New(),
		Email: "x@example.com", Role: MemberRoleViewer, Token: "tok",
		Status: InviteStatusPending, ExpiresAt: now, CreatedAt: now, UpdatedAt: now,
	}
	invCopy := inv.Copy()
	assert.True(t, inv.Equals(&invCopy))
	invCopy.Status = InviteStatusAccepted
	assert.False(t, inv.Equals(&invCopy))
}

func TestErrors_Format(t *testing.T) {
	assert.Equal(t, "Workspace not found", ErrWorkspaceNotFound.Error())
	assert.Equal(t, "Workspace already exists", ErrWorkspaceAlreadyExists.Error())
	assert.Equal(t, "WorkspaceMember not found", ErrWorkspaceMemberNotFound.Error())
	assert.Equal(t, "WorkspaceInvite not found", ErrWorkspaceInviteNotFound.Error())

	// Verify the AlreadyExistsError-typed sentinel renders correctly too.
	assert.Equal(t, "WorkspaceMember already exists", ErrWorkspaceMemberAlreadyExists.Error())
	assert.Equal(t, "WorkspaceInvite already exists", ErrWorkspaceInviteAlreadyExists.Error())

	assert.Equal(t, "last admin cannot be demoted", ErrLastAdminCannotBeDemoted.Error())
	assert.Equal(t, "invalid member role", ErrInvalidMemberRole.Error())
}
