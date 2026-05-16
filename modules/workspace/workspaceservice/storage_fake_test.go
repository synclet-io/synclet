package workspaceservice

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
)

// fakeStorage is the in-memory fake implementing Storage for use-case tests.
// Use newFakeStorage(t) to construct.
type fakeStorage struct {
	mu              sync.Mutex
	workspaces      []*Workspace
	members         []*WorkspaceMember
	invites         []*WorkspaceInvite
	workspacesStore *fakeWorkspacesStorage
	membersStore    *fakeMembersStorage
	invitesStore    *fakeInvitesStorage
	txCallCount     int
}

func newFakeStorage() *fakeStorage {
	s := &fakeStorage{}
	s.workspacesStore = &fakeWorkspacesStorage{parent: s}
	s.membersStore = &fakeMembersStorage{parent: s}
	s.invitesStore = &fakeInvitesStorage{parent: s}

	return s
}

func (s *fakeStorage) Workspaces() WorkspacesStorage             { return s.workspacesStore }
func (s *fakeStorage) WorkspaceMembers() WorkspaceMembersStorage { return s.membersStore }
func (s *fakeStorage) WorkspaceInvites() WorkspaceInvitesStorage { return s.invitesStore }
func (s *fakeStorage) IdempotencyKeys() idempotency.Storage {
	panic("IdempotencyKeys not implemented")
}

func (s *fakeStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error {
	s.mu.Lock()
	s.txCallCount++
	s.mu.Unlock()

	return cb(ctx, s)
}

func (s *fakeStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error {
	return nil
}

func (s *fakeStorage) seedWorkspace(w *Workspace) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workspaces = append(s.workspaces, w)
}

func (s *fakeStorage) seedMember(m *WorkspaceMember) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members = append(s.members, m)
}

func (s *fakeStorage) seedInvite(i *WorkspaceInvite) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invites = append(s.invites, i)
}

// equalsUUID is a typed shorthand for filter.Equals[uuid.UUID].
func equalsUUID(v uuid.UUID) filter.Filter[uuid.UUID] {
	return filter.Equals(v)
}

// equalsValue extracts the value held by an Equals filter, returning (value, true)
// when the filter is an Equals predicate.
func equalsValue[T comparable](flt filter.Filter[T]) (T, bool) {
	var zero T
	if flt == nil {
		return zero, false
	}

	if eq, ok := flt.(*filter.EqualsFilter[T]); ok {
		return eq.Value, true
	}

	return zero, false
}

// --- Workspaces --------------------------------------------------------------

type fakeWorkspacesStorage struct {
	parent *fakeStorage
}

func (s *fakeWorkspacesStorage) matches(w *Workspace, flt *WorkspaceFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.ID); ok && w.ID != id {
		return false
	}

	if name, ok := equalsValue[string](flt.Name); ok && w.Name != name {
		return false
	}

	if slug, ok := equalsValue[string](flt.Slug); ok && w.Slug != slug {
		return false
	}

	return true
}

func (s *fakeWorkspacesStorage) Create(_ context.Context, w *Workspace) (*Workspace, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, existing := range s.parent.workspaces {
		if existing.Slug != "" && existing.Slug == w.Slug {
			return nil, ErrWorkspaceAlreadyExists
		}
	}

	s.parent.workspaces = append(s.parent.workspaces, w)

	return w, nil
}

func (s *fakeWorkspacesStorage) BatchCreate(_ context.Context, _ []*Workspace) ([]*Workspace, error) {
	panic("BatchCreate not implemented")
}

func (s *fakeWorkspacesStorage) Count(_ context.Context, flt *WorkspaceFilter) (int, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	count := 0

	for _, w := range s.parent.workspaces {
		if s.matches(w, flt) {
			count++
		}
	}

	return count, nil
}

func (s *fakeWorkspacesStorage) Update(_ context.Context, w *Workspace) (*Workspace, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for i, existing := range s.parent.workspaces {
		if existing.ID == w.ID {
			s.parent.workspaces[i] = w
			return w, nil
		}
	}

	return nil, ErrWorkspaceNotFound
}

func (s *fakeWorkspacesStorage) Save(ctx context.Context, w *Workspace) (*Workspace, error) {
	return s.Update(ctx, w)
}

func (s *fakeWorkspacesStorage) First(_ context.Context, flt *WorkspaceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*Workspace, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, w := range s.parent.workspaces {
		if s.matches(w, flt) {
			return w, nil
		}
	}

	return nil, ErrWorkspaceNotFound
}

func (s *fakeWorkspacesStorage) FirstOrCreate(ctx context.Context, flt *WorkspaceFilter, model *Workspace, _ ...optionutil.Option[dbutil.SelectOptions]) (*Workspace, error) {
	found, err := s.First(ctx, flt)
	if err == nil {
		return found, nil
	}

	if !errors.Is(err, ErrWorkspaceNotFound) {
		return nil, err
	}

	return s.Create(ctx, model)
}

func (s *fakeWorkspacesStorage) Find(_ context.Context, flt *WorkspaceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*Workspace, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	out := []*Workspace{}

	for _, w := range s.parent.workspaces {
		if s.matches(w, flt) {
			out = append(out, w)
		}
	}

	return out, nil
}

func (s *fakeWorkspacesStorage) Delete(_ context.Context, flt *WorkspaceFilter) error {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	kept := s.parent.workspaces[:0]

	for _, w := range s.parent.workspaces {
		if !s.matches(w, flt) {
			kept = append(kept, w)
		}
	}

	s.parent.workspaces = kept

	return nil
}

func (s *fakeWorkspacesStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	return nil
}

// --- Members -----------------------------------------------------------------

type fakeMembersStorage struct {
	parent *fakeStorage
}

func (s *fakeMembersStorage) matches(m *WorkspaceMember, flt *WorkspaceMemberFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.ID); ok && m.ID != id {
		return false
	}

	if wsID, ok := equalsValue[uuid.UUID](flt.WorkspaceID); ok && m.WorkspaceID != wsID {
		return false
	}

	if userID, ok := equalsValue[uuid.UUID](flt.UserID); ok && m.UserID != userID {
		return false
	}

	if role, ok := equalsValue[MemberRole](flt.Role); ok && m.Role != role {
		return false
	}

	return true
}

func (s *fakeMembersStorage) Create(_ context.Context, m *WorkspaceMember) (*WorkspaceMember, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, existing := range s.parent.members {
		if existing.WorkspaceID == m.WorkspaceID && existing.UserID == m.UserID {
			return nil, ErrWorkspaceMemberAlreadyExists
		}
	}

	s.parent.members = append(s.parent.members, m)

	return m, nil
}

func (s *fakeMembersStorage) BatchCreate(_ context.Context, _ []*WorkspaceMember) ([]*WorkspaceMember, error) {
	panic("BatchCreate not implemented")
}

func (s *fakeMembersStorage) Count(_ context.Context, flt *WorkspaceMemberFilter) (int, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	count := 0

	for _, m := range s.parent.members {
		if s.matches(m, flt) {
			count++
		}
	}

	return count, nil
}

func (s *fakeMembersStorage) Update(_ context.Context, m *WorkspaceMember) (*WorkspaceMember, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for i, existing := range s.parent.members {
		if existing.ID == m.ID {
			s.parent.members[i] = m
			return m, nil
		}
	}

	return nil, ErrWorkspaceMemberNotFound
}

func (s *fakeMembersStorage) Save(ctx context.Context, m *WorkspaceMember) (*WorkspaceMember, error) {
	return s.Update(ctx, m)
}

func (s *fakeMembersStorage) First(_ context.Context, flt *WorkspaceMemberFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceMember, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, m := range s.parent.members {
		if s.matches(m, flt) {
			return m, nil
		}
	}

	return nil, ErrWorkspaceMemberNotFound
}

func (s *fakeMembersStorage) FirstOrCreate(ctx context.Context, flt *WorkspaceMemberFilter, model *WorkspaceMember, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceMember, error) {
	found, err := s.First(ctx, flt)
	if err == nil {
		return found, nil
	}

	if !errors.Is(err, ErrWorkspaceMemberNotFound) {
		return nil, err
	}

	return s.Create(ctx, model)
}

func (s *fakeMembersStorage) Find(_ context.Context, flt *WorkspaceMemberFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*WorkspaceMember, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	out := []*WorkspaceMember{}

	for _, m := range s.parent.members {
		if s.matches(m, flt) {
			out = append(out, m)
		}
	}

	return out, nil
}

func (s *fakeMembersStorage) Delete(_ context.Context, flt *WorkspaceMemberFilter) error {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	kept := s.parent.members[:0]

	for _, m := range s.parent.members {
		if !s.matches(m, flt) {
			kept = append(kept, m)
		}
	}

	s.parent.members = kept

	return nil
}

func (s *fakeMembersStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	return nil
}

// --- Invites -----------------------------------------------------------------

type fakeInvitesStorage struct {
	parent *fakeStorage
}

func (s *fakeInvitesStorage) matches(i *WorkspaceInvite, flt *WorkspaceInviteFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.ID); ok && i.ID != id {
		return false
	}

	if wsID, ok := equalsValue[uuid.UUID](flt.WorkspaceID); ok && i.WorkspaceID != wsID {
		return false
	}

	if inviterID, ok := equalsValue[uuid.UUID](flt.InviterUserID); ok && i.InviterUserID != inviterID {
		return false
	}

	if email, ok := equalsValue[string](flt.Email); ok && i.Email != email {
		return false
	}

	if token, ok := equalsValue[string](flt.Token); ok && i.Token != token {
		return false
	}

	if status, ok := equalsValue[InviteStatus](flt.Status); ok && i.Status != status {
		return false
	}

	return true
}

func (s *fakeInvitesStorage) Create(_ context.Context, inv *WorkspaceInvite) (*WorkspaceInvite, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, existing := range s.parent.invites {
		if existing.Token != "" && existing.Token == inv.Token {
			return nil, ErrWorkspaceInviteAlreadyExists
		}
	}

	s.parent.invites = append(s.parent.invites, inv)

	return inv, nil
}

func (s *fakeInvitesStorage) BatchCreate(_ context.Context, _ []*WorkspaceInvite) ([]*WorkspaceInvite, error) {
	panic("BatchCreate not implemented")
}

func (s *fakeInvitesStorage) Count(_ context.Context, flt *WorkspaceInviteFilter) (int, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	count := 0

	for _, inv := range s.parent.invites {
		if s.matches(inv, flt) {
			count++
		}
	}

	return count, nil
}

func (s *fakeInvitesStorage) Update(_ context.Context, inv *WorkspaceInvite) (*WorkspaceInvite, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for i, existing := range s.parent.invites {
		if existing.ID == inv.ID {
			s.parent.invites[i] = inv
			return inv, nil
		}
	}

	return nil, ErrWorkspaceInviteNotFound
}

func (s *fakeInvitesStorage) Save(ctx context.Context, inv *WorkspaceInvite) (*WorkspaceInvite, error) {
	return s.Update(ctx, inv)
}

func (s *fakeInvitesStorage) First(_ context.Context, flt *WorkspaceInviteFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceInvite, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, inv := range s.parent.invites {
		if s.matches(inv, flt) {
			return inv, nil
		}
	}

	return nil, ErrWorkspaceInviteNotFound
}

func (s *fakeInvitesStorage) FirstOrCreate(ctx context.Context, flt *WorkspaceInviteFilter, model *WorkspaceInvite, _ ...optionutil.Option[dbutil.SelectOptions]) (*WorkspaceInvite, error) {
	found, err := s.First(ctx, flt)
	if err == nil {
		return found, nil
	}

	if !errors.Is(err, ErrWorkspaceInviteNotFound) {
		return nil, err
	}

	return s.Create(ctx, model)
}

func (s *fakeInvitesStorage) Find(_ context.Context, flt *WorkspaceInviteFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*WorkspaceInvite, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	out := []*WorkspaceInvite{}

	for _, inv := range s.parent.invites {
		if s.matches(inv, flt) {
			out = append(out, inv)
		}
	}

	return out, nil
}

func (s *fakeInvitesStorage) Delete(_ context.Context, flt *WorkspaceInviteFilter) error {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	kept := s.parent.invites[:0]

	for _, inv := range s.parent.invites {
		if !s.matches(inv, flt) {
			kept = append(kept, inv)
		}
	}

	s.parent.invites = kept

	return nil
}

func (s *fakeInvitesStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	return nil
}

// --- Stub adapters -----------------------------------------------------------

type fakeEmailSender struct {
	sent []SendInviteEmailParams
	err  error
}

func (f *fakeEmailSender) SendInviteEmail(_ context.Context, params SendInviteEmailParams) error {
	f.sent = append(f.sent, params)

	return f.err
}

type fakeUserLookup struct {
	byEmail map[string]*UserInfo
	byID    map[uuid.UUID]*UserInfo
}

func newFakeUserLookup() *fakeUserLookup {
	return &fakeUserLookup{
		byEmail: map[string]*UserInfo{},
		byID:    map[uuid.UUID]*UserInfo{},
	}
}

func (f *fakeUserLookup) seed(user *UserInfo) {
	f.byEmail[user.Email] = user
	f.byID[user.ID] = user
}

func (f *fakeUserLookup) GetUserByEmail(_ context.Context, email string) (*UserInfo, error) {
	return f.byEmail[email], nil
}

func (f *fakeUserLookup) GetUserByID(_ context.Context, id uuid.UUID) (*UserInfo, error) {
	user, ok := f.byID[id]
	if !ok {
		return nil, fmt.Errorf("user %s not found", id)
	}

	return user, nil
}
