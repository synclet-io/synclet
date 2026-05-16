package workspaceservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCreateInvite(store *fakeStorage, lookup *fakeUserLookup, email *fakeEmailSender) *CreateInvite {
	cfg := Config{InviteTTL: 24 * time.Hour, FrontendURL: "https://app.example.com"}

	return NewCreateInvite(store, email, lookup, cfg, nil)
}

func TestCreateInvite_NewInviteForUnknownUser(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviter := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Workspace", Slug: "workspace"})

	users := newFakeUserLookup()
	users.seed(&UserInfo{ID: inviter, Email: "admin@example.com", Name: "Admin"})

	mailer := &fakeEmailSender{}

	invite, err := newCreateInvite(store, users, mailer).Execute(ctx, CreateInviteParams{
		WorkspaceID:   wsID,
		InviterUserID: inviter,
		Email:         "  Newcomer@Example.com  ",
		Role:          MemberRoleEditor,
	})
	require.NoError(t, err)
	require.NotNil(t, invite)

	assert.Equal(t, "newcomer@example.com", invite.Email, "email is normalised")
	assert.Equal(t, MemberRoleEditor, invite.Role)
	assert.Equal(t, InviteStatusPending, invite.Status)
	assert.NotEmpty(t, invite.Token, "token must be generated")
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), invite.ExpiresAt, 5*time.Second)
}

func TestCreateInvite_ReplacesExistingPendingInvite(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	inviter := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "WS", Slug: "ws"})
	store.seedInvite(&WorkspaceInvite{
		ID:            uuid.New(),
		WorkspaceID:   wsID,
		InviterUserID: inviter,
		Email:         "person@example.com",
		Role:          MemberRoleViewer,
		Token:         "existing-token",
		Status:        InviteStatusPending,
		ExpiresAt:     time.Now().Add(time.Hour),
	})

	users := newFakeUserLookup()
	users.seed(&UserInfo{ID: inviter, Email: "admin@example.com", Name: "Admin"})

	invite, err := newCreateInvite(store, users, &fakeEmailSender{}).Execute(ctx, CreateInviteParams{
		WorkspaceID:   wsID,
		InviterUserID: inviter,
		Email:         "person@example.com",
		Role:          MemberRoleAdmin,
	})
	require.NoError(t, err)
	assert.Equal(t, MemberRoleAdmin, invite.Role, "role updated")
	assert.Equal(t, "existing-token", invite.Token, "token preserved across replacement")

	all, err := store.WorkspaceInvites().Find(ctx, &WorkspaceInviteFilter{})
	require.NoError(t, err)
	require.Len(t, all, 1, "no duplicate row")
}

func TestCreateInvite_AlreadyMemberRejected(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	memberUser := uuid.New()
	inviter := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "WS", Slug: "ws"})
	store.seedMember(&WorkspaceMember{ID: uuid.New(), WorkspaceID: wsID, UserID: memberUser, Role: MemberRoleEditor})

	users := newFakeUserLookup()
	users.seed(&UserInfo{ID: memberUser, Email: "member@example.com", Name: "Mem"})
	users.seed(&UserInfo{ID: inviter, Email: "admin@example.com", Name: "Admin"})

	_, err := newCreateInvite(store, users, &fakeEmailSender{}).Execute(ctx, CreateInviteParams{
		WorkspaceID:   wsID,
		InviterUserID: inviter,
		Email:         "member@example.com",
		Role:          MemberRoleViewer,
	})
	require.Error(t, err)

	if !strings.Contains(err.Error(), "already a member") {
		t.Fatalf("expected 'already a member' error, got: %v", err)
	}
}
