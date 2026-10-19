package auditservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPersistAuditEvent(storage *fakeStorage) *PersistAuditEvent {
	return NewPersistAuditEvent(storage, storage, nil)
}

type loggedError struct {
	message string
	fields  map[string]interface{}
}

// capturingDelegate records every Error call with the fields attached to it.
type capturingDelegate struct {
	fields map[string]interface{}
	errors *[]loggedError
}

func newCapturingLogger() (*logging.Logger, *[]loggedError) {
	errs := &[]loggedError{}

	return &logging.Logger{Delegate: capturingDelegate{fields: map[string]interface{}{}, errors: errs}}, errs
}

func (d capturingDelegate) with(fields map[string]interface{}) capturingDelegate {
	merged := make(map[string]interface{}, len(d.fields)+len(fields))
	for k, v := range d.fields {
		merged[k] = v
	}

	for k, v := range fields {
		merged[k] = v
	}

	return capturingDelegate{fields: merged, errors: d.errors}
}

func (capturingDelegate) Info(context.Context, string, ...interface{})  {}
func (capturingDelegate) Warn(context.Context, string, ...interface{})  {}
func (capturingDelegate) Debug(context.Context, string, ...interface{}) {}
func (d capturingDelegate) Error(_ context.Context, msg string, _ ...interface{}) {
	*d.errors = append(*d.errors, loggedError{message: msg, fields: d.fields})
}
func (d capturingDelegate) WithFields(fields map[string]interface{}) logging.Delegate {
	return d.with(fields)
}
func (d capturingDelegate) WithField(key string, value interface{}) logging.Delegate {
	return d.with(map[string]interface{}{key: value})
}
func (d capturingDelegate) WithError(err error) logging.Delegate {
	return d.with(map[string]interface{}{"error": err})
}
func (d capturingDelegate) Named(string) logging.Delegate    { return d }
func (d capturingDelegate) SkipCallers(int) logging.Delegate { return d }

func newPersistableEvent() *AuditEvent {
	return &AuditEvent{
		ID:            uuid.New(),
		WorkspaceID:   uuid.New(),
		ActorType:     ActorTypeAPIKey,
		ActorID:       uuid.New(),
		ActorLabel:    "ci-key",
		Action:        ActionDelete,
		ResourceType:  ResourceTypeWebhook,
		ResourceID:    uuid.New(),
		ResourceLabel: "slack-hook",
		DiffJSON:      "[]",
		CreatedAt:     time.Now(),
	}
}

func TestPersistAuditEvent_InsertsEvent(t *testing.T) {
	storage := newFakeStorage()
	uc := newTestPersistAuditEvent(storage)
	event := newPersistableEvent()

	require.NoError(t, uc.Execute(context.Background(), event))

	require.Len(t, storage.events, 1)
	assert.True(t, event.Equals(storage.events[0]))
}

func TestPersistAuditEvent_DuplicateIDIsSuccess(t *testing.T) {
	storage := newFakeStorage()
	uc := newTestPersistAuditEvent(storage)
	event := newPersistableEvent()

	require.NoError(t, uc.Execute(context.Background(), event))

	redelivered := event.Copy()
	require.NoError(t, uc.Execute(context.Background(), &redelivered))

	require.Len(t, storage.events, 1)
	assert.True(t, event.Equals(storage.events[0]))
}

func TestPersistAuditEvent_RejectsUnstorableEvents(t *testing.T) {
	cases := map[string]*AuditEvent{
		"nil event": nil,
		"empty id": func() *AuditEvent {
			event := newPersistableEvent()
			event.ID = uuid.Nil

			return event
		}(),
		"invalid action": func() *AuditEvent {
			event := newPersistableEvent()
			event.Action = Action(0)

			return event
		}(),
		"invalid resource type": func() *AuditEvent {
			event := newPersistableEvent()
			event.ResourceType = ResourceType(99)

			return event
		}(),
		"invalid actor type": func() *AuditEvent {
			event := newPersistableEvent()
			event.ActorType = ActorType(99)

			return event
		}(),
	}

	for name, event := range cases {
		t.Run(name, func(t *testing.T) {
			storage := newFakeStorage()
			uc := newTestPersistAuditEvent(storage)

			err := uc.Execute(context.Background(), event)

			var validationErr *ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Empty(t, storage.events)
		})
	}
}

func TestPersistAuditEvent_StorageFailurePropagates(t *testing.T) {
	storage := newFakeStorage()
	storage.createErr = errors.New("db down")
	uc := newTestPersistAuditEvent(storage)

	err := uc.Execute(context.Background(), newPersistableEvent())

	require.ErrorIs(t, err, storage.createErr)

	var validationErr *ValidationError
	assert.NotErrorAs(t, err, &validationErr)
	assert.Empty(t, storage.events)
}

func newPendingUserEvent(userID uuid.UUID, action Action) *AuditEvent {
	return &AuditEvent{
		ID:            uuid.New(),
		ActorType:     ActorTypeUser,
		ActorID:       userID,
		ActorLabel:    "new@example.com",
		Action:        action,
		ResourceType:  ResourceTypeUser,
		ResourceID:    userID,
		ResourceLabel: "new@example.com",
		DiffJSON:      "[]",
		CreatedAt:     time.Now(),
	}
}

func TestPersistAuditEvent_JoinAdoptsPendingUserEvents(t *testing.T) {
	cases := map[string]func(userID, workspaceID uuid.UUID) *AuditEvent{
		"workspace created by the user": func(userID, workspaceID uuid.UUID) *AuditEvent {
			return &AuditEvent{
				ID: uuid.New(), WorkspaceID: workspaceID, ActorType: ActorTypeUser, ActorID: userID,
				Action: ActionCreate, ResourceType: ResourceTypeWorkspace, ResourceID: workspaceID,
				DiffJSON: "[]", CreatedAt: time.Now(),
			}
		},
		"invite accepted by the user": func(userID, workspaceID uuid.UUID) *AuditEvent {
			return &AuditEvent{
				ID: uuid.New(), WorkspaceID: workspaceID, ActorType: ActorTypeUser, ActorID: userID,
				Action: ActionCreate, ResourceType: ResourceTypeWorkspaceMember, ResourceID: userID,
				DiffJSON: "[]", CreatedAt: time.Now(),
			}
		},
	}

	for name, joinEvent := range cases {
		t.Run(name, func(t *testing.T) {
			storage := newFakeStorage()
			uc := newTestPersistAuditEvent(storage)
			ctx := context.Background()

			userID, otherUserID, workspaceID := uuid.New(), uuid.New(), uuid.New()
			registered := newPendingUserEvent(userID, ActionCreate)
			loggedIn := newPendingUserEvent(userID, ActionLogin)
			otherPending := newPendingUserEvent(otherUserID, ActionLogin)

			for _, event := range []*AuditEvent{registered, loggedIn, otherPending} {
				require.NoError(t, uc.Execute(ctx, event))
			}

			require.NoError(t, uc.Execute(ctx, joinEvent(userID, workspaceID)))

			inWorkspace, err := storage.AuditEvents().Find(ctx, &AuditEventFilter{WorkspaceID: filter.Equals(workspaceID)})
			require.NoError(t, err)

			ids := make([]uuid.UUID, 0, len(inWorkspace))
			for _, event := range inWorkspace {
				ids = append(ids, event.ID)
			}

			assert.Contains(t, ids, registered.ID)
			assert.Contains(t, ids, loggedIn.ID)
			assert.NotContains(t, ids, otherPending.ID)

			pending, err := storage.AuditEvents().Find(ctx, &AuditEventFilter{WorkspaceID: filter.Equals(uuid.Nil)})
			require.NoError(t, err)
			require.Len(t, pending, 1)
			assert.Equal(t, otherPending.ID, pending[0].ID)
		})
	}
}

func TestPersistAuditEvent_NonJoinEventLeavesPendingEvents(t *testing.T) {
	storage := newFakeStorage()
	uc := newTestPersistAuditEvent(storage)
	ctx := context.Background()

	userID := uuid.New()
	require.NoError(t, uc.Execute(ctx, newPendingUserEvent(userID, ActionLogin)))

	other := newPersistableEvent()
	other.ActorType = ActorTypeUser
	other.ActorID = userID
	require.NoError(t, uc.Execute(ctx, other))

	pending, err := storage.AuditEvents().Find(ctx, &AuditEventFilter{WorkspaceID: filter.Equals(uuid.Nil)})
	require.NoError(t, err)
	assert.Len(t, pending, 1)
}

func newWorkspaceCreatedEvent(userID, workspaceID uuid.UUID, at time.Time) *AuditEvent {
	return &AuditEvent{
		ID: uuid.New(), WorkspaceID: workspaceID, ActorType: ActorTypeUser, ActorID: userID,
		Action: ActionCreate, ResourceType: ResourceTypeWorkspace, ResourceID: workspaceID,
		DiffJSON: "[]", CreatedAt: at,
	}
}

func eventWorkspaces(t *testing.T, storage *fakeStorage) map[uuid.UUID]uuid.UUID {
	t.Helper()

	storage.mu.Lock()
	defer storage.mu.Unlock()

	out := make(map[uuid.UUID]uuid.UUID, len(storage.events))
	for _, event := range storage.events {
		out[event.ID] = event.WorkspaceID
	}

	return out
}

func TestPersistAuditEvent_JoinAdoptsOnlyRegistrationAndEarlierLogins(t *testing.T) {
	storage := newFakeStorage()
	uc := newTestPersistAuditEvent(storage)
	ctx := context.Background()

	userID, workspaceID := uuid.New(), uuid.New()
	joinedAt := time.Now()

	registered := newPendingUserEvent(userID, ActionCreate)
	registered.CreatedAt = joinedAt.Add(-time.Hour)
	loggedIn := newPendingUserEvent(userID, ActionLogin)
	loggedIn.CreatedAt = joinedAt.Add(-time.Minute)
	loggedOut := newPendingUserEvent(userID, ActionLogout)
	loggedOut.CreatedAt = joinedAt.Add(-time.Minute)
	passwordChanged := newPendingUserEvent(userID, ActionUpdate)
	passwordChanged.CreatedAt = joinedAt.Add(-time.Minute)
	laterLogin := newPendingUserEvent(userID, ActionLogin)
	laterLogin.CreatedAt = joinedAt.Add(time.Hour)

	for _, event := range []*AuditEvent{registered, loggedIn, loggedOut, passwordChanged, laterLogin} {
		require.NoError(t, uc.Execute(ctx, event))
	}

	require.NoError(t, uc.Execute(ctx, newWorkspaceCreatedEvent(userID, workspaceID, joinedAt)))

	workspaces := eventWorkspaces(t, storage)
	assert.Equal(t, workspaceID, workspaces[registered.ID], "registration")
	assert.Equal(t, workspaceID, workspaces[loggedIn.ID], "login before the join")
	assert.Equal(t, uuid.Nil, workspaces[loggedOut.ID], "logout")
	assert.Equal(t, uuid.Nil, workspaces[passwordChanged.ID], "password change")
	assert.Equal(t, uuid.Nil, workspaces[laterLogin.ID], "login after the join")
}

func TestPersistAuditEvent_FormerMemberGetsNothingAdopted(t *testing.T) {
	storage := newFakeStorage()
	uc := newTestPersistAuditEvent(storage)
	ctx := context.Background()

	userID, firstWorkspace, nextWorkspace := uuid.New(), uuid.New(), uuid.New()
	now := time.Now()

	registered := newPendingUserEvent(userID, ActionCreate)
	registered.CreatedAt = now.Add(-3 * time.Hour)
	require.NoError(t, uc.Execute(ctx, registered))
	require.NoError(t, uc.Execute(ctx, newWorkspaceCreatedEvent(userID, firstWorkspace, now.Add(-2*time.Hour))))

	loginAfterRemoval := newPendingUserEvent(userID, ActionLogin)
	loginAfterRemoval.CreatedAt = now.Add(-time.Hour)
	logoutAfterRemoval := newPendingUserEvent(userID, ActionLogout)
	logoutAfterRemoval.CreatedAt = now.Add(-time.Hour)

	require.NoError(t, uc.Execute(ctx, loginAfterRemoval))
	require.NoError(t, uc.Execute(ctx, logoutAfterRemoval))

	require.NoError(t, uc.Execute(ctx, newWorkspaceCreatedEvent(userID, nextWorkspace, now)))

	workspaces := eventWorkspaces(t, storage)
	assert.Equal(t, firstWorkspace, workspaces[registered.ID])
	assert.Equal(t, uuid.Nil, workspaces[loginAfterRemoval.ID])
	assert.Equal(t, uuid.Nil, workspaces[logoutAfterRemoval.ID])
}

func TestPersistAuditEvent_LoginInWorkspaceAdoptsPendingRegistration(t *testing.T) {
	storage := newFakeStorage()
	uc := newTestPersistAuditEvent(storage)
	ctx := context.Background()

	userID, workspaceID := uuid.New(), uuid.New()
	now := time.Now()

	require.NoError(t, uc.Execute(ctx, newWorkspaceCreatedEvent(userID, workspaceID, now.Add(-time.Hour))))

	replayedRegistration := newPendingUserEvent(userID, ActionCreate)
	replayedRegistration.CreatedAt = now.Add(-2 * time.Hour)
	require.NoError(t, uc.Execute(ctx, replayedRegistration))

	login := newPendingUserEvent(userID, ActionLogin)
	login.WorkspaceID = workspaceID
	login.CreatedAt = now
	require.NoError(t, uc.Execute(ctx, login))
	require.NoError(t, uc.Execute(ctx, login))

	assert.Equal(t, workspaceID, eventWorkspaces(t, storage)[replayedRegistration.ID])
}

func TestPersistAuditEvent_MemberRemovalAdoptsNothing(t *testing.T) {
	storage := newFakeStorage()
	uc := newTestPersistAuditEvent(storage)
	ctx := context.Background()

	userID, workspaceID := uuid.New(), uuid.New()
	registered := newPendingUserEvent(userID, ActionCreate)
	registered.CreatedAt = time.Now().Add(-time.Hour)
	require.NoError(t, uc.Execute(ctx, registered))

	require.NoError(t, uc.Execute(ctx, &AuditEvent{
		ID: uuid.New(), WorkspaceID: workspaceID, ActorType: ActorTypeUser, ActorID: uuid.New(),
		Action: ActionDelete, ResourceType: ResourceTypeWorkspaceMember, ResourceID: userID,
		DiffJSON: "[]", CreatedAt: time.Now(),
	}))

	assert.Equal(t, uuid.Nil, eventWorkspaces(t, storage)[registered.ID])
}

func TestPersistAuditEvent_AdoptionFailureLogsTheUser(t *testing.T) {
	storage := newFakeStorage()
	storage.adoptErr = errors.New("db down")
	logger, logged := newCapturingLogger()
	uc := NewPersistAuditEvent(storage, storage, logger)

	userID, workspaceID := uuid.New(), uuid.New()

	err := uc.Execute(context.Background(), newWorkspaceCreatedEvent(userID, workspaceID, time.Now()))

	require.ErrorIs(t, err, storage.adoptErr)
	require.Len(t, *logged, 1)
	assert.Equal(t, userID.String(), (*logged)[0].fields["user_id"])
	assert.Equal(t, workspaceID.String(), (*logged)[0].fields["workspace_id"])
}
