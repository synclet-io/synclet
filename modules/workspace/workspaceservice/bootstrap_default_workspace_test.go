package workspaceservice

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapDefaultWorkspace_CreatesWhenMissing(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()

	got, err := NewBootstrapDefaultWorkspace(store).Execute(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Default", got.Name)
	assert.Equal(t, "default", got.Slug)

	require.Len(t, store.events, 1, "bootstrap must emit workspace.created so subscribers seed registries")
	assert.Equal(t, WorkspaceEventTypeCreated, store.events[0].EventType)
}

func TestBootstrapDefaultWorkspace_IdempotentWhenExisting(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Default", Slug: "default"})

	got, err := NewBootstrapDefaultWorkspace(store).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, wsID, got.ID)

	all, err := store.Workspaces().Find(ctx, &WorkspaceFilter{})
	require.NoError(t, err)
	assert.Len(t, all, 1, "no extra workspace must be created")
	assert.Empty(t, store.events, "no event when bootstrap is a no-op")
}

// staleWorkspacesStorage reports no workspace from First, as a process does
// that looked up the default workspace before another process created it.
type staleWorkspacesStorage struct {
	WorkspacesStorage
}

func (staleWorkspacesStorage) First(context.Context, *WorkspaceFilter, ...optionutil.Option[dbutil.SelectOptions]) (*Workspace, error) {
	return nil, ErrWorkspaceNotFound
}

type staleStorage struct {
	*fakeStorage
	stale bool
}

func (s *staleStorage) Workspaces() WorkspacesStorage {
	if s.stale {
		s.stale = false

		return staleWorkspacesStorage{s.fakeStorage.Workspaces()}
	}

	return s.fakeStorage.Workspaces()
}

func TestBootstrapDefaultWorkspace_LosesCreateRace(t *testing.T) {
	ctx := context.Background()
	wsID := uuid.New()
	store := newFakeStorage()
	store.seedWorkspace(&Workspace{ID: wsID, Name: "Default", Slug: "default"})

	got, err := NewBootstrapDefaultWorkspace(&staleStorage{fakeStorage: store, stale: true}).Execute(ctx)
	require.NoError(t, err)
	assert.Equal(t, wsID, got.ID)
	assert.Empty(t, store.events, "the process that lost the race publishes no workspace.created")
}

var errSendFailed = errors.New("send failed")

type failingEventsOutbox struct{}

func (failingEventsOutbox) Send(context.Context, *WorkspaceEvent) error { return errSendFailed }

// rollbackStorage discards the workspaces written in a transaction whose
// callback fails, and fails every outbox send.
type rollbackStorage struct {
	*fakeStorage
}

func (s *rollbackStorage) WorkspaceEvents() WorkspaceEventsOutbox { return failingEventsOutbox{} }

func (s *rollbackStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error {
	s.mu.Lock()
	snapshot := slices.Clone(s.workspaces)
	s.mu.Unlock()

	if err := cb(ctx, s); err != nil {
		s.mu.Lock()
		s.workspaces = snapshot
		s.mu.Unlock()

		return err
	}

	return nil
}

func TestBootstrapDefaultWorkspace_SendFailureLeavesNoWorkspace(t *testing.T) {
	ctx := context.Background()
	store := newFakeStorage()

	_, err := NewBootstrapDefaultWorkspace(&rollbackStorage{fakeStorage: store}).Execute(ctx)
	require.ErrorIs(t, err, errSendFailed)

	all, err := store.Workspaces().Find(ctx, &WorkspaceFilter{})
	require.NoError(t, err)
	assert.Empty(t, all, "a workspace without its workspace.created event must not be left behind")
}
