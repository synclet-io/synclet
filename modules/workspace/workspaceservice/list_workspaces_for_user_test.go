package workspaceservice

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Minimal in-memory stubs for the storage interfaces. Only the methods the use
// case actually calls are implemented; the rest panic so any new dependency is
// caught loudly in tests.

type stubWorkspacesStorage struct {
	// keyed by workspace ID
	rows map[uuid.UUID]*Workspace
	// optional override: return this error from First instead of looking up rows
	firstErr error
}

func (s *stubWorkspacesStorage) First(_ context.Context, flt *WorkspaceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*Workspace, error) {
	if s.firstErr != nil {
		return nil, s.firstErr
	}

	if flt == nil || flt.ID == nil {
		return nil, errors.New("test stub: filter must include ID")
	}

	idVal, ok := flt.ID.(*filter.EqualsFilter[uuid.UUID])
	if !ok {
		return nil, errors.New("test stub: only Equals(uuid) is supported")
	}

	ws, found := s.rows[idVal.Value]
	if !found {
		return nil, errors.New("workspace not found")
	}

	return ws, nil
}

func (s *stubWorkspacesStorage) Create(_ context.Context, _ *Workspace) (*Workspace, error) {
	panic("Create not implemented")
}
func (s *stubWorkspacesStorage) BatchCreate(_ context.Context, _ []*Workspace) ([]*Workspace, error) {
	panic("BatchCreate not implemented")
}
func (s *stubWorkspacesStorage) Count(_ context.Context, _ *WorkspaceFilter) (int, error) {
	panic("Count not implemented")
}
func (s *stubWorkspacesStorage) Update(_ context.Context, _ *Workspace) (*Workspace, error) {
	panic("Update not implemented")
}
func (s *stubWorkspacesStorage) Save(_ context.Context, _ *Workspace) (*Workspace, error) {
	panic("Save not implemented")
}
func (s *stubWorkspacesStorage) FirstOrCreate(_ context.Context, _ *WorkspaceFilter, _ *Workspace, _ ...optionutil.Option[dbutil.SelectOptions]) (*Workspace, error) {
	panic("FirstOrCreate not implemented")
}
func (s *stubWorkspacesStorage) Find(_ context.Context, _ *WorkspaceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*Workspace, error) {
	panic("Find not implemented")
}
func (s *stubWorkspacesStorage) Delete(_ context.Context, _ *WorkspaceFilter) error {
	panic("Delete not implemented")
}
func (s *stubWorkspacesStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	panic("WithAdvisoryLock not implemented")
}

type stubMembersStorage struct {
	// returned by Find. Other methods panic.
	rows    []*WorkspaceMember
	findErr error
}

func (s *stubMembersStorage) Find(_ context.Context, _ *WorkspaceMemberFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*WorkspaceMember, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}

	return s.rows, nil
}

func (s *stubMembersStorage) Create(_ context.Context, _ *WorkspaceMember) (*WorkspaceMember, error) {
	panic("Create not implemented")
}
func (s *stubMembersStorage) BatchCreate(_ context.Context, _ []*WorkspaceMember) ([]*WorkspaceMember, error) {
	panic("BatchCreate not implemented")
}
func (s *stubMembersStorage) Count(_ context.Context, _ *WorkspaceMemberFilter) (int, error) {
	panic("Count not implemented")
}
func (s *stubMembersStorage) Update(_ context.Context, _ *WorkspaceMember) (*WorkspaceMember, error) {
	panic("Update not implemented")
}
func (s *stubMembersStorage) Save(_ context.Context, _ *WorkspaceMember) (*WorkspaceMember, error) {
	panic("Save not implemented")
}
func (s *stubMembersStorage) First(_ context.Context, _ *WorkspaceMemberFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceMember, error) {
	panic("First not implemented")
}
func (s *stubMembersStorage) FirstOrCreate(_ context.Context, _ *WorkspaceMemberFilter, _ *WorkspaceMember, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceMember, error) {
	panic("FirstOrCreate not implemented")
}
func (s *stubMembersStorage) Delete(_ context.Context, _ *WorkspaceMemberFilter) error {
	panic("Delete not implemented")
}
func (s *stubMembersStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	panic("WithAdvisoryLock not implemented")
}

type stubStorage struct {
	workspaces *stubWorkspacesStorage
	members    *stubMembersStorage
}

func (s *stubStorage) Workspaces() WorkspacesStorage             { return s.workspaces }
func (s *stubStorage) WorkspaceMembers() WorkspaceMembersStorage { return s.members }
func (s *stubStorage) WorkspaceInvites() WorkspaceInvitesStorage {
	panic("WorkspaceInvites not implemented")
}
func (s *stubStorage) IdempotencyKeys() idempotency.Storage {
	panic("IdempotencyKeys not implemented")
}
func (s *stubStorage) ExecuteInTransaction(_ context.Context, _ func(ctx context.Context, tx Storage) error) error {
	panic("ExecuteInTransaction not implemented")
}
func (s *stubStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error {
	panic("WithAdvisoryLock not implemented")
}

func TestListWorkspacesForUser_ReturnsWorkspacesWithRoles(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	wsAdminID := uuid.New()
	wsEditorID := uuid.New()
	wsViewerID := uuid.New()

	store := &stubStorage{
		workspaces: &stubWorkspacesStorage{
			rows: map[uuid.UUID]*Workspace{
				wsAdminID:  {ID: wsAdminID, Name: "Admin WS"},
				wsEditorID: {ID: wsEditorID, Name: "Editor WS"},
				wsViewerID: {ID: wsViewerID, Name: "Viewer WS"},
			},
		},
		members: &stubMembersStorage{
			rows: []*WorkspaceMember{
				{UserID: userID, WorkspaceID: wsAdminID, Role: MemberRoleAdmin},
				{UserID: userID, WorkspaceID: wsEditorID, Role: MemberRoleEditor},
				{UserID: userID, WorkspaceID: wsViewerID, Role: MemberRoleViewer},
			},
		},
	}

	uc := NewListWorkspacesForUser(store)

	got, err := uc.Execute(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 3)

	// Each entry must pair the right workspace with the right role.
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
	// Membership row references a workspace ID that doesn't exist in storage —
	// the use case must skip it with a warning, not blow up the whole list.
	ctx := context.Background()
	userID := uuid.New()
	goodID := uuid.New()
	orphanID := uuid.New()

	store := &stubStorage{
		workspaces: &stubWorkspacesStorage{
			rows: map[uuid.UUID]*Workspace{
				goodID: {ID: goodID, Name: "Good"},
				// orphanID intentionally absent
			},
		},
		members: &stubMembersStorage{
			rows: []*WorkspaceMember{
				{UserID: userID, WorkspaceID: orphanID, Role: MemberRoleAdmin},
				{UserID: userID, WorkspaceID: goodID, Role: MemberRoleEditor},
			},
		},
	}

	uc := NewListWorkspacesForUser(store)
	got, err := uc.Execute(ctx, userID)
	require.NoError(t, err)
	require.Len(t, got, 1, "orphan membership must be skipped")
	assert.Equal(t, goodID, got[0].Workspace.ID)
	assert.Equal(t, MemberRoleEditor, got[0].Role)
}

func TestListWorkspacesForUser_NoMemberships(t *testing.T) {
	ctx := context.Background()
	store := &stubStorage{
		workspaces: &stubWorkspacesStorage{rows: map[uuid.UUID]*Workspace{}},
		members:    &stubMembersStorage{rows: nil},
	}
	uc := NewListWorkspacesForUser(store)

	got, err := uc.Execute(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestListWorkspacesForUser_PropagatesMembershipLookupError(t *testing.T) {
	ctx := context.Background()
	store := &stubStorage{
		workspaces: &stubWorkspacesStorage{},
		members:    &stubMembersStorage{findErr: errors.New("db unavailable")},
	}
	uc := NewListWorkspacesForUser(store)

	_, err := uc.Execute(ctx, uuid.New())
	require.Error(t, err)
}
