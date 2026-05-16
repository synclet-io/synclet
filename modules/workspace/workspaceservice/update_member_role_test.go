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

// fakeMembersStorage is a minimal in-memory implementation of
// WorkspaceMembersStorage. Only the methods used by UpdateMemberRole are
// implemented; the rest panic so any new dependency is caught loudly.
type fakeMembersStorage struct {
	rows []*WorkspaceMember

	firstErr error
	countErr error

	// recorded calls
	updateCalls []*WorkspaceMember
	updateErr   error
}

func (s *fakeMembersStorage) First(_ context.Context, flt *WorkspaceMemberFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceMember, error) {
	if s.firstErr != nil {
		return nil, s.firstErr
	}

	wsID, _ := filterEquals[uuid.UUID](flt.WorkspaceID)
	userID, _ := filterEquals[uuid.UUID](flt.UserID)

	for _, m := range s.rows {
		if m.WorkspaceID == wsID && m.UserID == userID {
			return m, nil
		}
	}

	return nil, ErrWorkspaceMemberNotFound
}

func (s *fakeMembersStorage) Count(_ context.Context, flt *WorkspaceMemberFilter) (int, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}

	wsID, hasWs := filterEquals[uuid.UUID](flt.WorkspaceID)
	role, hasRole := filterEquals[MemberRole](flt.Role)
	count := 0

	for _, m := range s.rows {
		if hasWs && m.WorkspaceID != wsID {
			continue
		}

		if hasRole && m.Role != role {
			continue
		}

		count++
	}

	return count, nil
}

func (s *fakeMembersStorage) Update(_ context.Context, member *WorkspaceMember) (*WorkspaceMember, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}

	s.updateCalls = append(s.updateCalls, member)
	for i, m := range s.rows {
		if m.ID == member.ID {
			s.rows[i] = member

			break
		}
	}

	return member, nil
}

func (s *fakeMembersStorage) Create(_ context.Context, _ *WorkspaceMember) (*WorkspaceMember, error) {
	panic("Create not implemented")
}

func (s *fakeMembersStorage) BatchCreate(_ context.Context, _ []*WorkspaceMember) ([]*WorkspaceMember, error) {
	panic("BatchCreate not implemented")
}

func (s *fakeMembersStorage) Save(_ context.Context, _ *WorkspaceMember) (*WorkspaceMember, error) {
	panic("Save not implemented")
}

func (s *fakeMembersStorage) FirstOrCreate(_ context.Context, _ *WorkspaceMemberFilter, _ *WorkspaceMember, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceMember, error) {
	panic("FirstOrCreate not implemented")
}

func (s *fakeMembersStorage) Find(_ context.Context, _ *WorkspaceMemberFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*WorkspaceMember, error) {
	panic("Find not implemented")
}

func (s *fakeMembersStorage) Delete(_ context.Context, _ *WorkspaceMemberFilter) error {
	panic("Delete not implemented")
}

func (s *fakeMembersStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	panic("WithAdvisoryLock not implemented")
}

// filterEquals returns the value held by an Equals filter, or the zero value
// when the filter is nil or not an Equals filter.
func filterEquals[T comparable](flt filter.Filter[T]) (T, bool) {
	var zero T

	if flt == nil {
		return zero, false
	}

	eq, ok := flt.(*filter.EqualsFilter[T])
	if !ok {
		return zero, false
	}

	return eq.Value, true
}

type updateRoleStorage struct {
	members *fakeMembersStorage
}

func (s *updateRoleStorage) Workspaces() WorkspacesStorage             { panic("Workspaces not implemented") }
func (s *updateRoleStorage) WorkspaceMembers() WorkspaceMembersStorage { return s.members }
func (s *updateRoleStorage) WorkspaceInvites() WorkspaceInvitesStorage {
	panic("WorkspaceInvites not implemented")
}

func (s *updateRoleStorage) IdempotencyKeys() idempotency.Storage {
	panic("IdempotencyKeys not implemented")
}

func (s *updateRoleStorage) ExecuteInTransaction(_ context.Context, _ func(ctx context.Context, tx Storage) error) error {
	panic("ExecuteInTransaction not implemented")
}

func (s *updateRoleStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error {
	panic("WithAdvisoryLock not implemented")
}

func TestUpdateMemberRole_PromotesEditorToAdmin(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	targetUserID := uuid.New()
	adminUserID := uuid.New()
	targetID := uuid.New()

	members := &fakeMembersStorage{
		rows: []*WorkspaceMember{
			{ID: uuid.New(), WorkspaceID: workspaceID, UserID: adminUserID, Role: MemberRoleAdmin},
			{ID: targetID, WorkspaceID: workspaceID, UserID: targetUserID, Role: MemberRoleEditor},
		},
	}
	store := &updateRoleStorage{members: members}

	uc := NewUpdateMemberRole(store)
	require.NoError(t, uc.Execute(ctx, workspaceID, targetUserID, MemberRoleAdmin))

	require.Len(t, members.updateCalls, 1)
	assert.Equal(t, targetID, members.updateCalls[0].ID)
	assert.Equal(t, MemberRoleAdmin, members.updateCalls[0].Role)
}

func TestUpdateMemberRole_LastAdminCannotBeDemoted(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	lonelyAdminID := uuid.New()

	members := &fakeMembersStorage{
		rows: []*WorkspaceMember{
			{ID: uuid.New(), WorkspaceID: workspaceID, UserID: lonelyAdminID, Role: MemberRoleAdmin},
			{ID: uuid.New(), WorkspaceID: workspaceID, UserID: uuid.New(), Role: MemberRoleEditor},
		},
	}
	store := &updateRoleStorage{members: members}

	err := NewUpdateMemberRole(store).Execute(ctx, workspaceID, lonelyAdminID, MemberRoleEditor)
	require.ErrorIs(t, err, ErrLastAdminCannotBeDemoted)
	assert.Empty(t, members.updateCalls, "no update must be issued when guarded")
}

func TestUpdateMemberRole_TwoAdminsAllowsDemotion(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	adminAID := uuid.New()
	adminBID := uuid.New()

	members := &fakeMembersStorage{
		rows: []*WorkspaceMember{
			{ID: uuid.New(), WorkspaceID: workspaceID, UserID: adminAID, Role: MemberRoleAdmin},
			{ID: uuid.New(), WorkspaceID: workspaceID, UserID: adminBID, Role: MemberRoleAdmin},
		},
	}
	store := &updateRoleStorage{members: members}

	require.NoError(t, NewUpdateMemberRole(store).Execute(ctx, workspaceID, adminAID, MemberRoleViewer))
	require.Len(t, members.updateCalls, 1)
	assert.Equal(t, MemberRoleViewer, members.updateCalls[0].Role)
}

func TestUpdateMemberRole_NotAMember(t *testing.T) {
	ctx := context.Background()
	store := &updateRoleStorage{members: &fakeMembersStorage{}}

	err := NewUpdateMemberRole(store).Execute(ctx, uuid.New(), uuid.New(), MemberRoleEditor)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWorkspaceMemberNotFound)
}

func TestUpdateMemberRole_InvalidRole(t *testing.T) {
	ctx := context.Background()
	store := &updateRoleStorage{members: &fakeMembersStorage{}}

	err := NewUpdateMemberRole(store).Execute(ctx, uuid.New(), uuid.New(), MemberRole(0))
	require.ErrorIs(t, err, ErrInvalidMemberRole)
}

func TestUpdateMemberRole_NoOpWhenRoleUnchanged(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	userID := uuid.New()
	members := &fakeMembersStorage{
		rows: []*WorkspaceMember{
			{ID: uuid.New(), WorkspaceID: workspaceID, UserID: userID, Role: MemberRoleEditor},
		},
	}
	store := &updateRoleStorage{members: members}

	require.NoError(t, NewUpdateMemberRole(store).Execute(ctx, workspaceID, userID, MemberRoleEditor))
	assert.Empty(t, members.updateCalls)
}

func TestUpdateMemberRole_PropagatesCountError(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()
	userID := uuid.New()
	members := &fakeMembersStorage{
		rows: []*WorkspaceMember{
			{ID: uuid.New(), WorkspaceID: workspaceID, UserID: userID, Role: MemberRoleAdmin},
		},
		countErr: errors.New("db down"),
	}
	store := &updateRoleStorage{members: members}

	err := NewUpdateMemberRole(store).Execute(ctx, workspaceID, userID, MemberRoleEditor)
	require.Error(t, err)
	assert.Empty(t, members.updateCalls)
}
